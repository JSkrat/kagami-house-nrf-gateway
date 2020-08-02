package RFModel

import (
	"fmt"
	"time"

	"../TranscieverModel"
)

// DeviceKey ...
type DeviceKey TranscieverModel.Address

// Device represents a device with one transciever with multiple units in it
type Device struct {
	Address      TranscieverModel.Address
	LastUpdate   time.Time
	UnitCount    uint
	BuildNumber  uint32
	AllFunctions []UnitFunctionKey
}

// UnitFunctionKey ...
type UnitFunctionKey struct {
	uid UID
	fno FuncNo
}

// UnitFunction ...
type UnitFunction struct {
	read  EDataType
	write EDataType
}

// Devices all known devices
var Devices = map[DeviceKey]*Device{}

// UnitFunctions all known functions of all known devices
var UnitFunctions = map[UnitFunctionKey]UnitFunction{}

func updateDeviceUnits(rf *RFModel, address TranscieverModel.Address) {
	unitsCountResponse := callFunction(rf, UID{Address: address, Unit: 0}, FGetListOfUnitFunctions, []byte{})
	// validation of the request. Don't like that huge chunk here it has to go somewhere else(
	if 5 != len(unitsCountResponse) {
		panic(fmt.Errorf(
			"incorrect response %v from the device %v Unit 0 function get number of internal units %v",
			unitsCountResponse,
			address,
			FGetListOfUnitFunctions,
		))
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
	for i := 1; i <= int(unitsCountResponse[0]); i++ {
		uid := UID{Address: address, Unit: byte(i)}
		functionListResponse := callFunction(rf, uid, FGetListOfUnitFunctions, []byte{})
		// fucking validation, it should go somewhere else(
		if 0 != len(functionListResponse)%2 {
			panic(fmt.Errorf(
				"incorect rsponse %v from the Unit %v function get list of Unit functions %v",
				functionListResponse,
				uid,
				FGetListOfUnitFunctions,
			))
		}
		// now parse the function list from the slave
		for f := 0; f < len(functionListResponse); f += 2 {
			key := UnitFunctionKey{uid: uid, fno: FuncNo(functionListResponse[f])}
			UnitFunctions[key] = UnitFunction{
				read:  EDataType(functionListResponse[f+1] >> 4),
				write: EDataType(functionListResponse[f+1] & 0x0F),
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
