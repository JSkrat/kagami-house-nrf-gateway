package main

import (
	"./RFModel"
	"./nRFModel"
)

func main() {
	model := RFModel.Init(nRF_model.TransmitterSettings{
		PortName: "",
		IrqName:  "2",
		CEName:   "7",
	})
	defer RFModel.Close(&model)

}
