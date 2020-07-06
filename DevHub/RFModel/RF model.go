package RFModel

import (
	"../TranscieverModel"
	"../nRFModel"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
)

type RFModel struct {
	TranscieverModel.Model
	transmitter nRF_model.NRFTransmitter
}

var log = logrus.New()

func Init(rf *RFModel, settings nRF_model.TransmitterSettings) {
	// logging
	//log.Formatter = new(logrus.JSONFormatter)
	log.Formatter = new(logrus.TextFormatter) //default
	//log.Formatter.(*logrus.TextFormatter).DisableColors = true    // remove colors
	//log.Formatter.(*logrus.TextFormatter).DisableTimestamp = true // remove timestamp from test output
	log.Level = logrus.TraceLevel
	log.Out = os.Stdout
	nRF_model.OpenTransmitter(&rf.transmitter, settings)
	rf.transmitter.SendMessageStatus = make(chan nRF_model.Message)
	rf.transmitter.ReceiveMessage = make(chan nRF_model.Message)
}

func (rf *RFModel) Close() {
	nRF_model.CloseTransmitter(&rf.transmitter)
}

func checkPayload(payload nRF_model.Payload, length int, uid TranscieverModel.UID, fno TranscieverModel.FuncNo) {
	if len(payload) != length {
		panic(errors.New(fmt.Sprintf(
			"payload (%v) length does not correspond data type length %v for uid %v fno %v",
			payload, length, uid, fno,
		)))
	}
}

func (rf *RFModel) ReadFunction(uid TranscieverModel.UID, fno TranscieverModel.FuncNo) TranscieverModel.Variant {
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

func (rf *RFModel) WriteFunction(uid TranscieverModel.UID, fno TranscieverModel.FuncNo, value TranscieverModel.Variant) {
	checkDeviceUnits(rf, uid)
	var payload nRF_model.Payload
	dataType := UnitFunctions[UnitFunctionKey{
		uid: uid,
		fno: fno,
	}].input
	switch dataType {
	case EDBool:
		{
			if true == value {
				payload = nRF_model.Payload{1}
			} else {
				payload = nRF_model.Payload{0}
			}
		}
	case EDByte:
		{
			payload = nRF_model.Payload{byte(value.(int))}
		}
	case EDInt32:
		{
			i := int32(value.(int))
			payload = nRF_model.Payload{
				byte(i & 0xFF),
				byte((i >> 8) & 0xFF),
				byte((i >> 16) & 0xFF),
				byte((i >> 24) & 0xFF),
			}
		}
	case EDString:
		{
			payload = nRF_model.Payload(value.(string))
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
