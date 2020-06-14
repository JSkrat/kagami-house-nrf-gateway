package nRF_model

import (
	"errors"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/conn/spi"
	"periph.io/x/periph/conn/spi/spireg"
	"periph.io/x/periph/host"
	"time"
)

type NRFTransmitter struct {
	port       spi.PortCloser
	connection spi.Conn
	status     uint8
	channel    uint8
}

func BV(b Bit) byte {
	return 1 << byte(b)
}

func setCE(rf *NRFTransmitter, value bool) {

}

/**
 * @brief
 * The serial shifting SPI commands is in the following format:
 * <Command word: MSBit to LSBit (one byte)>
 * <Data bytes: LSByte to MSByte, MSBit in each byte first>
 * @param rf
 * @param command
 * @param data length of data array determines how much bytes would be read and written
 */
func sendCommand(rf *NRFTransmitter, command Command, data []byte) []byte {
	var write = make([]byte, 1+len(data))
	var read = make([]byte, len(write))
	write[0] = byte(command)
	_ = append(write, data...)
	if err := (*rf).connection.Tx(write, read); err != nil {
		panic(errors.New("sendCommand.Tx: " + err.Error()))
	}
	(*rf).status = read[0]
	return read[1:]
}

func ReadRegister(rf *NRFTransmitter, register Register) []byte {
	return sendCommand(rf, Command(byte(CReadRegister)|byte(register)), make([]byte, registerLengths[register]))
}

func WriteRegister(rf *NRFTransmitter, r Register, data []byte) {
	if len(data) > int(registerLengths[r]) {
		panic(errors.New("data is bigger than register size"))
	}
	sendCommand(rf, Command(byte(CWriteRegister)|byte(r)), data)
}

func WriteByteRegister(rf *NRFTransmitter, r Register, data byte) {
	WriteRegister(rf, r, []byte{data})
}

func Open(rf *NRFTransmitter) {
	// Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		panic(errors.New("host.Init: " + err.Error()))
	}
	// Use spireg SPI port registry to find the first available SPI bus.
	port, err := spireg.Open("")
	if err != nil {
		panic(errors.New("spireg.Open: " + err.Error()))
	}
	(*rf).port = port
	// Convert the spi.Port into a spi.Conn so it can be used for communication.
	connection, err := (*rf).port.Connect(10*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		panic(errors.New("port.Connect: " + err.Error()))
	}
	(*rf).connection = connection
	initNRF(rf)
}

func initNRF(rf *NRFTransmitter) {
	// we do not use nrf pipes 2-5
	setCE(rf, false)
	sendCommand(rf, CFlushRx, []byte{})
	sendCommand(rf, CFlushTx, []byte{})
	// clear all interrupts
	WriteByteRegister(rf, RStatus, BV(BRxDr)|BV(BTxDs)|BV(BMaxRt))
	WriteByteRegister(rf, RConfig, BV(BEnCrc)|BV(BCrcO)|BV(BPwrUp)|BV(BPrimRx))
	// disable auto ack
	WriteByteRegister(rf, REnAA, 0)
	WriteByteRegister(rf, RDynPd, BV(BDplP0)|BV(BDplP1))
	WriteByteRegister(rf, RFeature, BV(BEnDpl))
	WriteByteRegister(rf, REnRxAddr, BV(BEnRxP0))
	// 1Mbps, max power
	WriteByteRegister(rf, RRFSetup, 3<<byte(BRfPwr))
}

func Close(rf *NRFTransmitter) {
	_ = rf.port.Close()
}

func Listen(rf *NRFTransmitter, address Address) {
	var config = ReadRegister(rf, RConfig)
	config[0] |= BV(BPrimRx)
	WriteRegister(rf, RConfig, config)
	WriteRegister(rf, RRxAddrP0, address[:])
	setCE(rf, true)
}

func Transmit(rf *NRFTransmitter, a Address, data []byte) {
	if 32 < len(data) {
		panic(errors.New("too big payload, " + string(len(data))))
	}
	// without a CE changing from low to high transmission won't start
	setCE(rf, false)
	time.Sleep(10 * time.Microsecond)
	// clear interrupts
	WriteByteRegister(rf, RStatus, BV(BTxDs)|BV(BMaxRt))
	WriteRegister(rf, RTxAddr, a[:])
	WriteRegister(rf, RRxAddrP0, a[:])
	sendCommand(rf, CWriteTxPayload, data)
	var config = ReadRegister(rf, RConfig)
	config[0] &^= BV(BPrimRx)
	WriteRegister(rf, RConfig, config)
	setCE(rf, true)
}

func GoIdle(rf *NRFTransmitter) {
	setCE(rf, false)
}

func ValidateRfChannel(channel byte) bool {
	return channel < 128
}

func SetRfChannel(rf *NRFTransmitter, channel byte) {
	if !ValidateRfChannel(channel) {
		panic(errors.New("incorrect channel " + string(channel)))
	}
	rf.channel = channel
	WriteByteRegister(rf, RRFCh, channel)
}
