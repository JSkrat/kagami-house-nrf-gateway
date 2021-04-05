package main

import (
	"fmt"

	"./NRFTransciever"
	"./RFModel"
	"./UartTransciever"
	"./Redis"
	"./Cache"
	"gopkg.in/ini.v1"
)

func wrapErrPanic(value RFModel.Variant, err error) RFModel.Variant {
	if nil == err {
		return value
	}
	panic(err)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Print(r)
		}
	}()
	settings, err := ini.Load("settings.ini")
	if nil != err {
		panic(fmt.Errorf("unable to load settings.ini, %v", err))
	}
	var model RFModel.RFModel
	switch settings.Section("").Key("rf model").In("nrf", []string{"nrf", "uart master"}) {
	case "nrf":
		var transmitter NRFTransciever.NRFTransmitter
		NRFTransciever.Init(&transmitter, NRFTransciever.TransmitterSettings{
			PortName: settings.Section("nrf").Key("port").String(),
			IrqName:  settings.Section("nrf").Key("irq").String(),
			CEName:   settings.Section("nrf").Key("ce").String(),
			Speed:    float32(wrapErrPanic(settings.Section("nrf").Key("speed").Float64()).(float64)),
		})
		RFModel.Init(&model, &transmitter)
	case "uart master":
		var transmitter UartTransciever.UMTransmitter
		UartTransciever.Init(&transmitter, UartTransciever.TransmitterSettings{
			PortName: settings.Section("uart master").Key("port").String(),
			Speed:    wrapErrPanic(settings.Section("uart master").Key("speed").Int()).(int),
		})
		RFModel.Init(&model, &transmitter)
	}
	defer model.Close()
	var output Redis.Interface
	Redis.Init(&output, settings.Section("").Key("redis").String())
	var cache Cache.Cache
	Cache.Init(&cache, &model, &output)
	uid := RFModel.UID{Address: [5]byte{0xAA, 0xAA, 0xAA, 0xAA, 0x01}, Unit: 1}
	model.WriteFunction(uid, 0x19, byte(0xE1))
	response := model.ReadFunction(uid, 0x18).(byte)
	fmt.Printf("Wrote 0xE1, read %v", response)
}
