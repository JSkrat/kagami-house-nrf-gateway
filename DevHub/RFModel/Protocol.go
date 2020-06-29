package RFModel

import (
	"../nRFModel"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

const (
	PacketLength       uint = 32
	RequestHeaderSize  uint = 4
	ResponseHeaderSize uint = 3
	MaxDataLengthRq    uint = PacketLength - RequestHeaderSize
	MaxDataLengthRs    uint = PacketLength - ResponseHeaderSize
)

type request struct {
	Version       byte
	TransactionID byte
	UnitID        byte
	FunctionID    byte
	Data          [MaxDataLengthRq]byte
	DataLength    byte
}

type response struct {
	Version       byte
	TransactionID byte
	Code          byte
	Data          [MaxDataLengthRs]byte
	DataLength    byte
}

type EDataType byte

const (
	EDNone        EDataType = 0
	EDBool                  = 1
	EDByte                  = 2
	EDInt32                 = 3
	EDString                = 4
	EDByteArray             = 5
	EDUnspecified           = 0xF
)

type FuncNo byte

const (
	// Unit 0, global device functions
	F0SetNewSessionKey         FuncNo = 0
	F0GetNumberOfInternalUnits        = 1
	F0SetMACAddress                   = 2
	F0SetRFChannel                    = 6
	F0GetDeviceStatistics             = 3
	F0NOP                             = 5
	F0ResetTransactionId              = 4
	F0SetSlaveMode                    = 7
	// per Unit functions
	FGetListOfUnitFunctions = 0
	FGetTextDescription     = 1
	FSetTextDescription     = 2
)

type PacketValidationError error

/*type Payload interface {
	Payload() []byte
}*/

var transactionId byte = 0

func serializeRequest(rq *request) nRF_model.Payload {
	if MaxDataLengthRq < uint(rq.DataLength) {
		panic(errors.New(fmt.Sprintf("too big DataLength %v", rq.DataLength)))
	}
	buf := bytes.Buffer{}
	if err := binary.Write(&buf, binary.LittleEndian, rq); err != nil {
		panic(errors.New("binary.Write: " + err.Error()))
	}
	return buf.Bytes()[:PacketLength-MaxDataLengthRq+uint(rq.DataLength)]
}

func parseResponse(r *nRF_model.Payload) response {
	if PacketLength < uint(len(*r)) {
		panic(errors.New(fmt.Sprintf("too big packet of length %v", len(*r))))
	}
	var ret response
	buf := bytes.Buffer{}
	buf.Write(*r)
	buf.Write(make([]byte, int(PacketLength+1)-len(*r)))
	if err := binary.Read(&buf, binary.LittleEndian, &ret); err != nil {
		panic(errors.New("binary.Read: " + err.Error()))
	}
	ret.DataLength = byte(len(*r) - int(ResponseHeaderSize))
	return ret
}

func (r request) Payload() []byte {
	var ret []byte
	copy(ret[:], r.Data[:r.DataLength])
	return ret
}

func (r response) Payload() []byte {
	var ret []byte
	copy(ret[:], r.Data[:r.DataLength])
	return ret
}

func createRequest(unitID byte, functionId byte, data []byte) request {
	defer func() { transactionId += 1 }()
	var structData [MaxDataLengthRq]byte
	copy(structData[:], data)
	return request{
		Version:       0,
		TransactionID: transactionId,
		UnitID:        unitID,
		FunctionID:    functionId,
		Data:          structData,
		DataLength:    byte(len(data)),
	}
}

func basicValidateResponse(r *response) bool {
	if 0 != r.Version {
		return false
	}
	if transactionId-1 != r.TransactionID {
		return false
	}
	return true
}

func validateResponse(to *nRF_model.Address, rq *request, rs *nRF_model.Message) (retResp response, retStatus bool) {
	retResp = parseResponse(&rs.Payload)
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(PacketValidationError); ok {
				retStatus = false
			}
		}
	}()
	if !basicValidateResponse(&retResp) {
		panic(PacketValidationError(errors.New("basicValidateResponse")))
	}
	if *to != rs.Address {
		// todo count that cases
		panic(PacketValidationError(errors.New("unexpected packet from wrong Address")))
	}
	if rq.TransactionID != retResp.TransactionID {
		panic(PacketValidationError(errors.New("bad transaction id")))
	}
	return retResp, true
}

func callFunction(rf *RFModel, uid UID, fno FuncNo, payload nRF_model.Payload) nRF_model.Payload {
	rq := createRequest(uid.Unit, byte(fno), payload)
	rqSerialized := serializeRequest(&rq)
	for i := 3; 0 <= i; i-- {
		log.Info(fmt.Sprintf("callFunction try %v", i))
		nRF_model.Transmit(&rf.transmitter, uid.Address, rqSerialized)
		// wait for transmission completes
		select {
		case <-rf.transmitter.SendMessageStatus:
			// ok
		case <-time.After(20 * time.Millisecond):
			panic(errors.New(fmt.Sprintf(
				"callFunction.Transmit timeout, no irq TX_DS in 20ms after transmission. UID %v, fno %v, payload %v",
				uid, fno, payload,
			)))
		}
		nRF_model.Listen(&rf.transmitter, uid.Address)
		timeoutChan := make(chan bool)
		go func() {
			<-time.After(50 * time.Millisecond)
			timeoutChan <- true
		}()
		for {
			select {
			case message := <-rf.transmitter.ReceiveMessage:
				// message received
				if pm, ok := validateResponse(&uid.Address, &rq, &message); ok {
					// now we have received, parsed and validated message from the device
					if 0 != pm.Code {
						panic(errors.New(fmt.Sprintf("error code %v", pm.Code)))
					}
					return pm.Payload()
				}
			case <-timeoutChan:
				continue
			}
		}
	}
	panic(errors.New(fmt.Sprintf(
		"callFunction.Listen: response timeout 3 times in a row for uid %v, fno %v, payload %v. Packet is %v",
		uid, fno, payload, rqSerialized,
	)))
}
