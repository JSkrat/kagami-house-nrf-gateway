package RFModel

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
