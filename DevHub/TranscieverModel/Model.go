package TranscieverModel

// Address of the physical device, basically address of the transciever
type Address [5]byte

// Payload in radio packet
type Payload []byte

// Message via radio channel in payload
type Message struct {
	Address Address
	Payload Payload
	Pipe    byte
	Status  EMessageStatus
}

// EMessageStatus of the radio transaction
type EMessageStatus byte

// Message status enum
const (
	EMSNone         EMessageStatus = 0x00
	EMSSlaveTimeout                = 0x01
	EMSAckTimeout                  = 0x02
	EMSDataPacket                  = 0x03
	EMSAckPacket                   = 0x04
)

// Transmitter represents any possible transmitter device
type Transmitter interface {
	Close()
	SendCommand(a Address, data Payload) (ret Message)
}
