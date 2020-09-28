package RFModel

import (
	"fmt"
	"time"
)

// Device represents a device with one transceiver with multiple units in it
type Device struct {
	Address      DeviceAddress
	LastUpdate   time.Time
	UnitCount    uint
	BuildNumber  uint32
	AllFunctions []UnitFunctionKey
}

// UnitFunctionKey ...
type UnitFunctionKey struct {
	UID UID
	FNo FuncNo
}

// UnitFunction ...
type UnitFunction struct {
	read  EDataType
	write EDataType
}

// Devices all known devices
var Devices = map[DeviceAddress]*Device{}

// UnitFunctions all known functions of all known devices
var UnitFunctions = map[UnitFunctionKey]UnitFunction{}

func updateDeviceUnits(rf *RFModel, address DeviceAddress) {
	unitsCountResponse := rf.CallFunction(UID{Address: address, Unit: 0}, FGetListOfUnitFunctions, []byte{})
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
	// delete all Unit functions before re-population
	if _, ok := Devices[address]; ok {
		for _, v := range Devices[address].AllFunctions {
			delete(UnitFunctions, v)
		}
	}
	Devices[address] = &Device{
		Address:      address,
		LastUpdate:   time.Now(),
		UnitCount:    uint(unitsCountResponse[0]),
		AllFunctions: []UnitFunctionKey{},
	}
	for i := 1; i <= int(unitsCountResponse[0]); i++ {
		uid := UID{Address: address, Unit: byte(i)}
		functionListResponse := rf.CallFunction(uid, FGetListOfUnitFunctions, []byte{})
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
			key := UnitFunctionKey{UID: uid, FNo: FuncNo(functionListResponse[f])}
			UnitFunctions[key] = UnitFunction{
				read:  EDataType(functionListResponse[f+1] >> 4),
				write: EDataType(functionListResponse[f+1] & 0x0F),
			}
			Devices[address].AllFunctions = append(Devices[address].AllFunctions, key)
		}
	}

}

func checkDeviceUnits(rf *RFModel, uid UID) {
	if v, ok := Devices[uid.Address]; ok {
		if 1*time.Hour > time.Now().Sub(v.LastUpdate) {
			return
		}
	}
	updateDeviceUnits(rf, uid.Address)
}
