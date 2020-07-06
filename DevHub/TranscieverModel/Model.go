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
	ReadFunction(rf *Model, uid UID, fno FuncNo) Variant
	WriteFunction(rf *Model, uid UID, fno FuncNo, value Variant)
}
