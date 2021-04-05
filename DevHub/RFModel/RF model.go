package RFModel

import (
	"fmt"
	"os"

	"../TranscieverModel"
	"github.com/sirupsen/logrus"
)

// RFModel handler
type RFModel struct {
	//TranscieverModel.Model
	transmitter TranscieverModel.Transmitter
}

var log = logrus.New()

// Init ...
func Init(rf *RFModel, transmitter TranscieverModel.Transmitter) {
	// logging
	log.Formatter = new(logrus.TextFormatter)
	log.Level = logrus.TraceLevel
	log.Out = os.Stdout
	rf.transmitter = transmitter
	//rf.transmitter.SendMessageStatus = make(chan nRF_model.Message)
	//rf.transmitter.ReceiveMessage = make(chan nRF_model.Message)
}

// Close ...
func (rf *RFModel) Close() {
	rf.transmitter.Close()
}

func checkPayload(payload TranscieverModel.Payload, length int, uid UID, fno FuncNo) {
	if len(payload) != length {
		panic(Error{
			Error: fmt.Errorf(
				"RFModel.checkPayload(payload %s, length %v, uid %X FNo 0x%X) length does not correspond data type; ",
				Dump(payload), length, uid, fno,
			),
			Type: EBadResponse,
		})
	}
}

// ReadFunction read from the unit
// call given function with empty payload and parse result according to the function output type (fn 0 of a given unit)
func (rf *RFModel) ReadFunction(uid UID, fno FuncNo) Variant {
	// check all device units and functions data types to cast
	checkDeviceUnits(rf, uid)
	payload := rf.CallFunction(uid, fno, []byte{})
	dataType := UnitFunctions[UnitFunctionKey{
		UID: uid,
		FNo: fno,
	}].read
	switch dataType {
	case EDNone:
		{
			return 0
		}
	case EDBool:
		{
			checkPayload(payload, 1, uid, fno)
			if 0 == payload[0] {
				return false
			}
			return true
		}
	case EDByte:
		{
			checkPayload(payload, 1, uid, fno)
			return uint8(payload[0])
		}
	case EDInt32:
		{
			checkPayload(payload, 4, uid, fno)
			// todo test against negative values
			return int32(payload[0]) + int32(payload[1])<<8 + int32(payload[2])<<16 + int32(payload[3])<<24
		}
	case EDString:
		{
			// any length is valid
			return string(payload)
		}
	case EDByteArray:
		{
			// any length is valid
			return payload
		}
	}
	panic(Error{
		Error: fmt.Errorf(
			"RFModel.ReadFunction(uid %X FNo 0x%X payload %s) unexpected data type %v; ",
			uid, fno, Dump(payload), dataType,
		),
		Type:  EGeneral,
	})
}

// WriteFunction write to the unit
// call given function with serialized value as a payload according to the function input type
func (rf *RFModel) WriteFunction(uid UID, fno FuncNo, value Variant) {
	checkDeviceUnits(rf, uid)
	var payload TranscieverModel.Payload
	dataType := UnitFunctions[UnitFunctionKey{
		UID: uid,
		FNo: fno,
	}].write
	switch dataType {
	case EDBool:
		{
			if true == value {
				payload = TranscieverModel.Payload{1}
			} else {
				payload = TranscieverModel.Payload{0}
			}
		}
	case EDByte:
		{
			payload = TranscieverModel.Payload{byte(value.(uint8))}
		}
	case EDInt32:
		{
			i := int32(value.(int))
			payload = TranscieverModel.Payload{
				byte(i & 0xFF),
				byte((i >> 8) & 0xFF),
				byte((i >> 16) & 0xFF),
				byte((i >> 24) & 0xFF),
			}
		}
	case EDString:
		{
			payload = TranscieverModel.Payload(value.(string))
		}
	case EDByteArray:
		{
			payload = value.([]byte)
		}
	default:
		panic(Error{
			Error: fmt.Errorf(
				"RFModel.WriteFunction(uid %X FNo 0x%X value %v) unexpected input data format %v; ",
				uid, fno, value, dataType,
			),
			Type:  EGeneral,
		})
	}
	rf.CallFunction(uid, fno, payload)
}
