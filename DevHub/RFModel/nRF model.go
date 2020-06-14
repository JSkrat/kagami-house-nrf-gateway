package nRF_model

import (
	"errors"
	"github.com/sirupsen/logrus"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/conn/spi"
	"periph.io/x/periph/conn/spi/spireg"
	"periph.io/x/periph/host"
	"time"
)

type NRFTransmitter struct {
	port              spi.PortCloser
	connection        spi.Conn
	status            uint8
	channel           uint8
	ce                gpio.PinOut
	irq               gpio.PinIn
	ReceiveMessage    chan Message
	SendMessage       chan Message
	SendMessageStatus chan Message
}

func BV(b Bit) byte {
	return 1 << byte(b)
}

func setCE(rf *NRFTransmitter, value bool) {
	if err := rf.ce.Out(gpio.Level(value)); nil != err {
		panic(errors.New("rf.ce.Out: " + err.Error()))
	}
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

func GetPipeNumberReceived(rf *NRFTransmitter) byte {
	ret := (rf.status & BRxPNoMask) >> BRxPNo
	if 7 == ret {
		panic(errors.New("RX FIFO empty"))
	}
	return ret
}

func Open(rf *NRFTransmitter, portName string, ceName string, irqName string) {
	// Make sure periphery is initialized.
	if _, err := host.Init(); err != nil {
		panic(errors.New("host.Init: " + err.Error()))
	}
	// Use SPI port registry to find the first available SPI bus.
	port, err := spireg.Open(portName)
	if err != nil {
		panic(errors.New("spireg.Open of port " + portName + ": " + err.Error()))
	}
	(*rf).port = port
	// Convert the spi.Port into a spi.Conn so it can be used for communication.
	connection, err := (*rf).port.Connect(10*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		panic(errors.New("port.Connect: " + err.Error()))
	}
	(*rf).connection = connection
	// now gpio
	rf.ce = gpioreg.ByName(ceName)
	if nil == rf.ce {
		panic(errors.New("ce pin <" + ceName + "> was not initialized"))
	}
	rf.irq = gpioreg.ByName(irqName)
	if nil == rf.irq {
		panic(errors.New("irq pin <" + irqName + "> was not initialized"))
	}
	if err := rf.irq.In(gpio.PullNoChange, gpio.RisingEdge); err != nil {
		panic(errors.New("PinIn.In: " + err.Error()))
	}
	initNRF(rf)
	go run(rf)
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
	WriteByteRegister(rf, RRFSetup, 0x03<<byte(BRfPwr))
}

func Close(rf *NRFTransmitter) {
	_ = rf.port.Close()
}

func receiveMessages(rf *NRFTransmitter) {
	// Data Ready RX FIFO interrupt. Asserted when new data arrives in RX FIFO.
	// The RX)DR IRQ is asserted by a new packet arrival event.
	// The procedure for handling this interrupt should be:
	// 1) read payload through SPI
	// 2) clear RX_DR IRQ
	// 3) read FIFO_STATUS to check if there are more payloads available in RX FIFO
	// 4) if there are more data in the RX FIFO, repeat from step 1)
	defer func() {
		// todo we need to distinguish what exactly panic happened
		recover()
	}()
	for {
		var m Message
		m.status = EMSReceived
		m.pipe = GetPipeNumberReceived(rf) // if no messages that will panic
		WriteByteRegister(rf, RStatus, BV(BRxDr))
		// get payload
		payloadLength := sendCommand(rf, CReadRxPayloadWidth, []byte{})[0]
		if 0 == payloadLength {
			m.payload = Payload{}
		} else {
			m.payload = sendCommand(rf, CReadRxPayload, make([]byte, payloadLength))
		}
		// get address
		if 0 == m.pipe {
			copy(m.address[:], ReadRegister(rf, RRxAddrP0))
		} else {
			copy(m.address[:], ReadRegister(rf, RRxAddrP1))
			if 1 < m.pipe {
				m.address[len(m.address)-1] = ReadRegister(rf, Register(RRxAddrP2-2+m.pipe))[0]
			}
		}
		if nil != rf.ReceiveMessage {
			rf.ReceiveMessage <- m
		}
		// update rf status
		sendCommand(rf, CNop, []byte{})
	}
}

func run(rf *NRFTransmitter) {
	// The IRQ pin is activated then TX_DS IRQ, RX_DR IRQ os MAX_RT IRQ are set high
	// by the state machine in the STATUS register
	for rf.irq.WaitForEdge(-1) {
		logrus.Info("IRQ happened")
		//fmt.Println("IRQ happened")
		setCE(rf, false)
		// update status register
		sendCommand(rf, CNop, []byte{})
		var m Message
		if 0 != rf.status&BV(BTxDs) {
			// Data Sent Tx FIFO interrupt. Asserted when the packet is transmitter on TX.
			// If AUTO_ACK is activates, this bit is set high only when ACK is received.
			copy(m.address[:], ReadRegister(rf, RTxAddr))
			m.status = EMSTransmitted
			// reset the flag
			WriteByteRegister(rf, RStatus, BV(BTxDs))
			if nil != rf.SendMessageStatus {
				rf.SendMessageStatus <- m
			}
		} else if 0 != rf.status&BV(BMaxRt) {
			// Maximum number of TX retransmits interrupt
			// If MAX_RT is asserted, it must be cleared to enable further communication.
			copy(m.address[:], ReadRegister(rf, RTxAddr))
			m.status = EMSNoAck
			// TX FIFO does not pop failed element. If we won't clean it, it will be re-sent again.
			sendCommand(rf, CFlushTx, []byte{})
			// reset the flag
			WriteByteRegister(rf, RStatus, BV(BMaxRt))
			if nil != rf.SendMessageStatus {
				rf.SendMessageStatus <- m
			}
		} else if 0 != rf.status&BV(BRxDr) {
			receiveMessages(rf)
		}
	}
}

func Listen(rf *NRFTransmitter, address Address) {
	var config = ReadRegister(rf, RConfig)
	config[0] |= BV(BPrimRx)
	WriteRegister(rf, RConfig, config)
	WriteRegister(rf, RRxAddrP0, address[:])
	setCE(rf, true)
}

func Transmit(rf *NRFTransmitter, a Address, data Payload) {
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
