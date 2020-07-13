package RFModel

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"../TranscieverModel"
)

type UID struct {
	Address TranscieverModel.Address
	Unit    byte
}
type Variant interface{}

type FuncNo byte

const (
	PacketLength       uint = 32
	RequestHeaderSize  uint = 4
	ResponseHeaderSize uint = 3
	MaxDataLengthRq         = PacketLength - RequestHeaderSize
	MaxDataLengthRs         = PacketLength - ResponseHeaderSize
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

const (
	// Unit 0, global device functions
	F0SetNewSessionKey    FuncNo = 10
	F0SetMACAddress              = 12
	F0SetRFChannel               = 16
	F0GetDeviceStatistics        = 13
	F0NOP                        = 15
	F0ResetTransactionId         = 14
	F0SetSlaveMode               = 17
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

func serializeRequest(rq *request) TranscieverModel.Payload {
	if MaxDataLengthRq < uint(rq.DataLength) {
		panic(errors.New(fmt.Sprintf("too big DataLength %v", rq.DataLength)))
	}
	buf := bytes.Buffer{}
	if err := binary.Write(&buf, binary.LittleEndian, rq); err != nil {
		panic(errors.New("binary.Write: " + err.Error()))
	}
	return buf.Bytes()[:PacketLength-MaxDataLengthRq+uint(rq.DataLength)]
}

func parseResponse(r *TranscieverModel.Payload) response {
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

func createRequest(unitID byte, FunctionID byte, data []byte) request {
	defer func() { transactionId += 1 }()
	var structData [MaxDataLengthRq]byte
	copy(structData[:], data)
	return request{
		Version:       0,
		TransactionID: transactionId,
		UnitID:        unitID,
		FunctionID:    FunctionID,
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

func validateResponse(to *TranscieverModel.Address, rq *request, rs *TranscieverModel.Message) (retResp response, retStatus bool) {
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

func callFunction(rf *RFModel, uid UID, fno FuncNo, payload TranscieverModel.Payload) TranscieverModel.Payload {
	rq := createRequest(uid.Unit, byte(fno), payload)
	rqSerialized := serializeRequest(&rq)
	for i := 3; 0 <= i; i-- {
		log.Info(fmt.Sprintf("callFunction try %v", i))
		message := rf.transmitter.SendCommand(uid.Address, rqSerialized)
		if TranscieverModel.EMSDataPacket == message.Status {
			// message received
			if pm, ok := validateResponse(&uid.Address, &rq, &message); ok {
				// now we have received, parsed and validated message from the device
				if 0 != pm.Code {
					panic(errors.New(fmt.Sprintf("error code %v", pm.Code)))
				}
				return pm.Payload()
			}
		} else {
			log.Info("RFModel.Protocol.callFucntion: listen timeout")
		}
	}
	panic(errors.New(fmt.Sprintf(
		"callFunction.Listen: response timeout 3 times in a row for uid %v, fno %v, payload %v. Packet is %v",
		uid, fno, payload, rqSerialized,
	)))
}
