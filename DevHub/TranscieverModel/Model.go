package TranscieverModel

import "../nRFModel"

type UID struct {
	Address nRF_model.Address
	Unit    byte
}
type Variant interface{}

type FuncNo byte

type Model interface {
	Close()
	ReadFunction(uid UID, fno FuncNo) Variant
	WriteFunction(uid UID, fno FuncNo, value Variant)
}
