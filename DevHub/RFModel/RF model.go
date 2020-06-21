package RFModel

import (
	"../nRFModel"
)

type UID struct {
	address nRF_model.Address
	unit    byte
}
type FuncNo byte
type Variant interface{}
type RFModel struct {
	transmitter nRF_model.NRFTransmitter
}

func Init(settings nRF_model.TransmitterSettings) RFModel {
	rf := RFModel{}
	rf.transmitter = nRF_model.OpenTransmitter(settings)
	return rf
}

func Close(rf *RFModel) {
	nRF_model.CloseTransmitter(&rf.transmitter)
}

func ReadFunction(uid UID, fno FuncNo) Variant {
	return 0
}
