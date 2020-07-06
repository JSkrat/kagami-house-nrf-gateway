package main

import (
	"./RFModel"
	"./TranscieverModel"
	"./nRFModel"
	"errors"
	"fmt"
	"gopkg.in/ini.v1"
)

func main() {
	settings, err := ini.Load("settings.ini")
	if nil != err {
		panic(errors.New(fmt.Sprintf("unable to load settings.ini, %v", err)))
	}
	var model TranscieverModel.Model
	switch settings.Section("").Key("rf model").In("nrf", []string{"nrf", "usb master"}) {
	case "nrf":
		model = RFModel.RFModel{}
		RFModel.Init(&model, nRF_model.TransmitterSettings{
			PortName: settings.Section("nrf").Key("port").String(),
			IrqName:  settings.Section("nrf").Key("irq").String(),
			CEName:   settings.Section("nrf").Key("ce").String(),
		})
	case "usb master":
		model = RFModel.RFModel{}
	}
	defer model.Close()
	uid := TranscieverModel.UID{Address: [5]byte{0xAA, 0xAA, 0xAA, 0xAA, 0x55}, Unit: 1}
	model.WriteFunction(&model, uid, 0x19, byte(0xE1))
	response := model.ReadFunction(&model, uid, 0x18).(byte)
	fmt.Printf("Wrote 0xE1, read %v", response)
}
