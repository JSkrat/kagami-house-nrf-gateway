package RFModel

import (
	"fmt"
	"strconv"
	"strings"
)

type ErrorType string

const (
	EGeneral          ErrorType = "general error"
	EBadParameter               = "bad parameter"
	EBadResponse                = "bad response"
	EPacketValidation           = "packet validation"
	EDeviceTimeout              = "device did not respond 3 times in a row"
	EBadCode                    = "function return code is not 0"
)

type Error struct {
	Error error
	Type  ErrorType
	Code  byte
}

func Dump(b []byte) string {
	var ret string
	for i := range b {
		c := b[i]
		if 16 > c {
			ret += "0"
		}
		ret += fmt.Sprintf("%X ", c)
	}
	return ret
}

func ParseAddress(s string) (ret DeviceAddress) {
	for index, bStr := range strings.Split(s, ":") {
		if index >= len(ret) {
			panic(Error{
				Error: fmt.Errorf("RFModel.ParseAddress: too long address; "),
				Type:  EGeneral,
			})
		}
		b, err := strconv.ParseUint(bStr, 16, 8)
		if nil != err {
			panic(Error{
				Error: fmt.Errorf("RFModel.ParseAddress: strconv.ParseUInt: %v; ", err.Error()),
				Type:  EGeneral,
			})
		}
		ret[index] = byte(b)
	}
	return ret
}

func AddressToString(a DeviceAddress) (ret string) {
	for i := range a[:] {
		c := a[i]
		if 16 > c {
			ret += "0"
		}
		ret += fmt.Sprintf("%X", c)
		if i < len(a)-1 {
			ret += ":"
		}
	}
	return ret
}
