package main

import (
	"./RFModel"
	"./nRFModel"
	"fmt"
)

func main() {
	model := RFModel.RFModel{}
	RFModel.Init(&model, nRF_model.TransmitterSettings{
		PortName: "/dev/spidev0.0",
		IrqName:  "25",
		CEName:   "24",
	})
	defer RFModel.Close(&model)
	uid := RFModel.UID{Address: [5]byte{0xAA, 0xAA, 0xAA, 0xAA, 0x54}, Unit: 1}
	RFModel.WriteFunction(&model, uid, 0x19, byte(0xE1))
	response := RFModel.ReadFunction(&model, uid, 0x18).(byte)
	fmt.Printf("Wrote 0xE1, read %v", response)
}
