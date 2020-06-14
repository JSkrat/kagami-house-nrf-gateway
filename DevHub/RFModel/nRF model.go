package nRF_model

import (
	"errors"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/conn/spi"
	"periph.io/x/periph/conn/spi/spireg"
	"periph.io/x/periph/host"
)

type NRFTransmitter struct {
	port       spi.PortCloser
	connection spi.Conn
	status     uint8
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
	initNRF()
}

func initNRF() {

}

func Close(rf *NRFTransmitter) {
	rf.port.Close()
}
