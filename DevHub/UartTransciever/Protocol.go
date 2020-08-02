package UartTransciever

import (
	"errors"
)

// Register — nrf transciever internal register
type Register byte

// Address — nrf transciever address
type Address [5]byte
type packet []byte
type command byte
type responseCode byte
type uartRequest struct {
	version byte
	command command
	payload []byte
}
type uartResponse struct {
	version byte
	command command
	code    responseCode
	payload []byte
}

const (
	cEcho                   command = 0x00
	cFWVersion                      = 0x01
	cModemStatus                    = 0x08
	cAddresses                      = 0x09
	cSetRFChannel                   = 0x10
	cSetTxPower                     = 0x11
	cSetBitRate                     = 0x12
	cSetAutoRetransmitDelay         = 0x13
	cSetAutoRetransmitCount         = 0x14
	cClearTxQueue                   = 0x20
	cClearRxQueue                   = 0x21
	cListen                         = 0x30
	cSetMasterSlaveMode             = 0x40
	cSetMasterAddress               = 0x41
	cGetRxItem                      = 0x50
	cTransmit                       = 0x7F
)

const (
	rOk                   responseCode = 0x00
	rNoPackets                         = 0x10
	rSlaveResponseTimeout              = 0x11
	rAckTimeout                        = 0x12
	rDataPacket                        = 0x14
	rAckPacket                         = 0x15
	// fatal errors
	rFail                    = 0x80
	rBadProtocolVersion      = 0x90
	rBadCommand              = 0x91
	rMemoryError             = 0x92
	rArgumentValidationError = 0x93
	rNotImplemented          = 0x94
)

// ModemStatusRegisters response to debug command cModemStatus
type ModemStatusRegisters struct {
	Config            Register
	EnAA              Register
	EnRxAddr          Register
	SetupAW           Register
	SetupRetr         Register
	RfCh              Register
	RfSetup           Register
	Status            Register
	ObserveTx         Register
	RPD               Register
	RxPWP0            Register
	RxPWP1            Register
	RxPwP2            Register
	RxPWP3            Register
	RxPWP4            Register
	RxPWP5            Register
	FifoStatus        Register
	DynPD             Register
	Feature           Register
	BufferPacketCount byte
}

// ModemAddressRegisters response to debug command cAddresses
type ModemAddressRegisters struct {
	RxAddrP0 Address
	RxAddrP1 Address
	RxAddrP2 byte
	RxAddrP3 byte
	RxAddrP4 byte
	RxAddrP5 byte
	TxAddr   Address
}

func stuffPacket(data packet) (ret packet) {
	ret = packet{0xC0}
	for _, v := range data {
		switch v {
		case 0xC0:
			ret = append(ret, 0xDB, 0xDC)
		case 0xDB:
			ret = append(ret, 0xDB, 0xDD)
		default:
			ret = append(ret, v)
		}
	}
	return ret
}

func unstuffPacket(data packet) (ret packet) {
	if 0xC0 != data[0] {
		panic(errors.New("UMModel.unstuffPacket: packet begins with not 0xC0"))
	}
	var esc bool = false
	ret = packet{}
	for _, v := range data[1:] {
		if 0xC0 == v {
			panic(errors.New("UMModel.unstuffPacket: extra 0xC0 inside a single packet"))
		}
		if !esc {
			if 0xDB == v {
				esc = true
			} else {
				ret = append(ret, v)
			}
		} else {
			switch v {
			case 0xDC:
				ret = append(ret, 0xC0)
			case 0xDD:
				ret = append(ret, 0xDB)
			default:
				panic(errors.New("UMModel.unstuffPacket: unexpected escape sequence"))
			}
			esc = false
		}
	}
	if esc {
		panic(errors.New("UMModel.unstuffPacket: unfinished escape sequence at the end of a packet"))
	}
	return ret
}

func createRequest(data uartRequest) (ret packet) {
	ret = packet{data.version, byte(data.command), byte(len(data.payload))}
	return append(ret, data.payload...)
}

func parseResponse(data packet) (ret uartResponse) {
	if 4 > len(data) {
		panic(errors.New("too short response"))
	}
	ret = uartResponse{
		version: data[0],
		command: command(data[1]),
		code:    responseCode(data[2]),
	}
	if 4+int(data[3]) != len(data) {
		panic(errors.New("incorrect response payload length"))
	}
	ret.payload = data[4:]
	return ret
}

func validateResponse(response uartResponse, requestCommand command) bool {
	if response.command != requestCommand|0x80 {
		return false
	}
	return true
}

func isPacketComplete(packet []byte) (ret bool) {
	defer func() {
		if nil != recover() {
			ret = false
		}
	}()
	_ = parseResponse(unstuffPacket(packet))
	return true
}
