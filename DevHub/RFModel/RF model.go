package RFModel

import (
	"errors"
	"fmt"
	"os"

	"../TranscieverModel"
	"github.com/sirupsen/logrus"
)

type RFModel struct {
	//TranscieverModel.Model
	transmitter TranscieverModel.Transmitter
}

var log = logrus.New()

func Init(rf *RFModel, transmitter TranscieverModel.Transmitter) {
	// logging
	log.Formatter = new(logrus.TextFormatter)
	log.Level = logrus.TraceLevel
	log.Out = os.Stdout
	rf.transmitter = transmitter
	//rf.transmitter.SendMessageStatus = make(chan nRF_model.Message)
	//rf.transmitter.ReceiveMessage = make(chan nRF_model.Message)
}

func (rf *RFModel) Close() {
	rf.transmitter.Close()
}

func checkPayload(payload TranscieverModel.Payload, length int, uid UID, fno FuncNo) {
	if len(payload) != length {
		panic(errors.New(fmt.Sprintf(
			"payload (%v) length does not correspond data type length %v for uid %v fno %v",
			payload, length, uid, fno,
		)))
	}
}

func (rf *RFModel) ReadFunction(uid UID, fno FuncNo) Variant {
	// check all device units and functions data types to cast
	checkDeviceUnits(rf, uid)
	payload := callFunction(rf, uid, fno, []byte{})
	dataType := UnitFunctions[UnitFunctionKey{
		uid: uid,
		fno: fno,
	}].output
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
			} else {
				return true
			}
		}
	case EDByte:
		{
			checkPayload(payload, 1, uid, fno)
			return int8(payload[0])
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
	panic(errors.New(fmt.Sprintf("unexpected data type %v for uid %v fno %v payload %v", dataType, uid, fno, payload)))
}

func (rf *RFModel) WriteFunction(uid UID, fno FuncNo, value Variant) {
	checkDeviceUnits(rf, uid)
	var payload TranscieverModel.Payload
	dataType := UnitFunctions[UnitFunctionKey{
		uid: uid,
		fno: fno,
	}].input
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
			payload = TranscieverModel.Payload{byte(value.(int))}
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
		panic(errors.New(fmt.Sprintf("unexpected input data format %v for uid %v fno %v value %v", dataType, uid, fno, value)))
	}
	callFunction(rf, uid, fno, payload)
}
