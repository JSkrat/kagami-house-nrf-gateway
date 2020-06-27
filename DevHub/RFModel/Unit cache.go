package RFModel

import (
	"../nRFModel"
	"errors"
	"fmt"
	"time"
)

type DeviceKey nRF_model.Address
type Device struct {
	Address      nRF_model.Address
	LastUpdate   time.Time
	UnitCount    uint
	AllFunctions []UnitFunctionKey
}

type UnitFunctionKey struct {
	uid UID
	fno FuncNo
}
type UnitFunction struct {
	input  EDataType
	output EDataType
}

var Devices = map[DeviceKey]*Device{}
var UnitFunctions = map[UnitFunctionKey]UnitFunction{}

func updateDeviceUnits(rf *RFModel, address nRF_model.Address) {
	unitsCountResponse := callFunction(rf, UID{Address: address, Unit: 0}, F0GetNumberOfInternalUnits, []byte{})
	// validation of the request. Don't like that huge chunk here it has to go somewhere else(
	if 1 != len(unitsCountResponse) {
		panic(errors.New(fmt.Sprintf(
			"incorrect response %v from the device %v Unit 0 function get number of internal units %v",
			unitsCountResponse,
			address,
			F0GetNumberOfInternalUnits,
		)))
	}
	// todo get device statistics here too
	deviceKey := DeviceKey(address)
	// delete all Unit functions before re-population
	if _, ok := Devices[deviceKey]; ok {
		for _, v := range Devices[deviceKey].AllFunctions {
			delete(UnitFunctions, v)
		}
	}
	Devices[deviceKey] = &Device{
		Address:      address,
		LastUpdate:   time.Now(),
		UnitCount:    uint(unitsCountResponse[0]),
		AllFunctions: []UnitFunctionKey{},
	}
	for i := 0; i < int(unitsCountResponse[0]); i++ {
		uid := UID{Address: address, Unit: byte(i)}
		functionListResponse := callFunction(rf, uid, FGetListOfUnitFunctions, []byte{})
		// fucking validation, it should go somewhere else(
		if 0 != len(functionListResponse)%3 {
			panic(errors.New(fmt.Sprintf(
				"incorect rsponse %v from the Unit %v function get list of Unit functions %v",
				functionListResponse,
				uid,
				FGetListOfUnitFunctions,
			)))
		}
		// now parse the function list from the slave
		for f := 0; f < len(functionListResponse); f += 3 {
			key := UnitFunctionKey{uid: uid, fno: FuncNo(functionListResponse[f])}
			UnitFunctions[key] = UnitFunction{
				input:  EDataType(functionListResponse[f+1]),
				output: EDataType(functionListResponse[f+2]),
			}
			Devices[deviceKey].AllFunctions = append(Devices[deviceKey].AllFunctions, key)
		}
	}

}

func checkDeviceUnits(rf *RFModel, uid UID) {
	if v, ok := Devices[DeviceKey(uid.Address)]; ok {
		if 1*time.Hour > time.Now().Sub(v.LastUpdate) {
			return
		}
	}
	updateDeviceUnits(rf, uid.Address)
}
