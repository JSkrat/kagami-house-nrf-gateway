package main

import (
	"fmt"

	"./RFModel"
	"./UMModel"
	"./nRFModel"
	"gopkg.in/ini.v1"
)

func wrapErrPanic(value RFModel.Variant, err error) RFModel.Variant {
	if nil == err {
		return value
	}
	panic(err)
}

func main() {
	settings, err := ini.Load("settings.ini")
	if nil != err {
		panic(fmt.Errorf("unable to load settings.ini, %v", err))
	}
	var model RFModel.RFModel
	switch settings.Section("").Key("rf model").In("nrf", []string{"nrf", "uart master"}) {
	case "nrf":
		var transmitter nRFModel.NRFTransmitter
		nRFModel.OpenTransmitter(&transmitter, nRFModel.TransmitterSettings{
			PortName: settings.Section("nrf").Key("port").String(),
			IrqName:  settings.Section("nrf").Key("irq").String(),
			CEName:   settings.Section("nrf").Key("ce").String(),
			Speed:    float32(wrapErrPanic(settings.Section("nrf").Key("speed").Float64()).(float64)),
		})
		RFModel.Init(&model, &transmitter)
	case "uart master":
		var transmitter UMModel.UMTransmitter
		UMModel.OpenTransmitter(&transmitter, UMModel.TransmitterSettings{
			PortName: settings.Section("uart master").Key("port").String(),
			Speed:    wrapErrPanic(settings.Section("uart master").Key("speed").Int()).(int),
		})
		RFModel.Init(&model, &transmitter)
	}
	defer model.Close()
	uid := RFModel.UID{Address: [5]byte{0xAA, 0xAA, 0xAA, 0xAA, 0x01}, Unit: 1}
	model.WriteFunction(uid, 0x19, byte(0xE1))
	response := model.ReadFunction(uid, 0x18).(byte)
	fmt.Printf("Wrote 0xE1, read %v", response)
}
