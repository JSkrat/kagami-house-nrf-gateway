package RFModel

import (
	"../TranscieverModel"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

type DeviceAddress TranscieverModel.Address

// UID unit ID
type UID struct {
	Address DeviceAddress
	Unit    byte
}

// Variant ...
type Variant interface{}

// FuncNo unit function id
type FuncNo byte

// for RF packet parsing
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

// EDataType ...
type EDataType byte

// EDataType enum
const (
	EDNone        EDataType = 0
	EDBool                  = 1
	EDByte                  = 2
	EDInt32                 = 3
	EDString                = 4
	EDByteArray             = 5
	EDUnspecified           = 0xF
)

// RF functions
const (
	// Unit 0, global device functions
	F0SetNewSessionKey    FuncNo = 10
	F0SetMACAddress              = 12
	F0SetRFChannel               = 16
	F0GetDeviceStatistics        = 13
	F0NOP                        = 15
	F0ResetTransactionID         = 14
	F0SetSlaveMode               = 17
	// per Unit functions
	FGetListOfUnitFunctions = 0
	FGetTextDescription     = 1
	FSetTextDescription     = 2
)

type EResponseCode byte

const (
	ERCOk               EResponseCode = 0
	ERCAddressBadLength               = 1

	ERCChBadChannels      = 0x10
	ERCChBadPermissions   = 0x12
	ERCChValidationFailed = 0x13

	ERCNotImplemented              = 0x7F
	ERCBadVersion                  = 0x90
	ERCBadUnitId                   = 0xA0
	ERCNotConsecutiveTransactionId = 0xB0
	ERCBadFunctionId               = 0xC0
	ERCResponseTooBig              = 0xD0
	ERCBadRequestData              = 0xE0
)

var transactionID byte = 0

func serializeRequest(rq *request) TranscieverModel.Payload {
	if MaxDataLengthRq < uint(rq.DataLength) {
		panic(Error{
			Error: fmt.Errorf("RFModel.serializeRequest: too big DataLength %v; ", rq.DataLength),
			Type:  EBadParameter,
		})
	}
	buf := bytes.Buffer{}
	if err := binary.Write(&buf, binary.LittleEndian, rq); err != nil {
		panic(Error{
			Error: fmt.Errorf("RFModel.serializeRequest: binary.Write: %v; ", err.Error()),
			Type:  EGeneral,
		})
	}
	return buf.Bytes()[:PacketLength-MaxDataLengthRq+uint(rq.DataLength)]
}

func parseResponse(r *TranscieverModel.Payload) (ret response) {
	if PacketLength < uint(len(*r)) {
		panic(Error{
			Error: fmt.Errorf("RFModel.parseResponse: too big packet of length %v; ", len(*r)),
			Type:  EPacketValidation,
		})
	}
	buf := bytes.Buffer{}
	buf.Write(*r)
	buf.Write(make([]byte, int(PacketLength+1)-len(*r)))
	if err := binary.Read(&buf, binary.LittleEndian, &ret); err != nil {
		panic(Error{
			Error: fmt.Errorf("RFModel.parseResponse: binary.Read: %v; ", err.Error()),
			Type:  EPacketValidation,
		})
	}
	ret.DataLength = byte(len(*r) - int(ResponseHeaderSize))
	return ret
}

func (r request) Payload() []byte {
	var ret []byte
	copy(ret[:], r.Data[:r.DataLength])
	return ret
}

func (r response) Payload() (ret []byte) {
	//ret = []byte{}
	ret = r.Data[:r.DataLength]
	return ret
}

func createRequest(unitID byte, FunctionID byte, data []byte) request {
	defer func() { transactionID += 1 }()
	var structData [MaxDataLengthRq]byte
	copy(structData[:], data)
	return request{
		Version:       0,
		TransactionID: transactionID,
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
	if transactionID-1 != r.TransactionID {
		return false
	}
	return true
}

func validateResponse(to *DeviceAddress, rq *request, rs *TranscieverModel.Message) (retResp response, retStatus bool) {
	defer func() {
		if r := recover(); r != nil {
			if EPacketValidation == r.(Error).Type {
				retStatus = false
			}
		}
	}()
	retResp = parseResponse(&rs.Payload)
	if !basicValidateResponse(&retResp) {
		panic(Error{
			Error: errors.New("RFModel.validateResponse: basicValidateResponse; "),
			Type:  EPacketValidation,
		})
	}
	if TranscieverModel.Address(*to) != rs.Address {
		// todo count that cases
		panic(Error{
			Error: errors.New("RFModel.validateResponse: unexpected packet from wrong Address; "),
			Type:  EPacketValidation,
		})
	}
	if rq.TransactionID != retResp.TransactionID {
		panic(Error{
			Error: errors.New("RFModel.validateResponse: bad transaction id; "),
			Type:  EPacketValidation,
		})
	}
	return retResp, true
}

// CallFunction is basic api for RFModel
// TODO shall catch and handle all the transceiver panics
// TODO retry 3 times on any error
// may panic by its own
func (rf *RFModel) CallFunction(uid UID, fno FuncNo, payload TranscieverModel.Payload) TranscieverModel.Payload {
	rq := createRequest(uid.Unit, byte(fno), payload)
	rqSerialized := serializeRequest(&rq)
	for i := 3; 0 <= i; i-- {
		log.Debug(fmt.Sprintf("RFModel.CallFunction try %v", i))
		message := rf.transmitter.SendCommand(TranscieverModel.Address(uid.Address), rqSerialized)
		if TranscieverModel.EMSDataPacket == message.Status {
			// message received
			if pm, ok := validateResponse(&uid.Address, &rq, &message); ok {
				// now we have received, parsed and validated message from the device
				if 0 != pm.Code {
					panic(Error{
						Error: fmt.Errorf(
							"RFModel.CallFunction(uid %X, fno 0x%X, payload %s) bad error code 0x%X response %s; ",
							uid, fno, Dump(payload), pm.Code, Dump(pm.Payload()),
						),
						Type: EBadCode,
						Code: pm.Code,
					})
				}
				log.Debug(fmt.Sprintf("RFModel.CallFunction uid %X, FNo 0x%X, payload %s, response %s", uid, fno, Dump(payload), Dump(pm.Payload())))
				return pm.Payload()
			}
		} else {
			log.Debug("RFModel.Protocol.CallFunction: listen timeout")
		}
	}
	panic(Error{
		Error: fmt.Errorf(
			"RFModel.CallFunction.Listen: response timeout 3 times in a row for uid %X, FNo 0x%X, payload %s. Packet is %s",
			uid, fno, Dump(payload), Dump(rqSerialized),
		),
		Type: EDeviceTimeout,
	})
}
