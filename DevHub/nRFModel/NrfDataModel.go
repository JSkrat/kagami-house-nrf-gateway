package nRFModel

// NRF-related stuff
type Command byte
type Register byte
type Bit byte

// nRF24L01 commands
const (
	CReadRegister        Command = 0x00
	CWriteRegister               = 0x20
	CReadRxPayloadWidth          = 0x60
	CReadRxPayload               = 0x61
	CWriteTxPayload              = 0xA0
	CWriteAckPayload             = 0xA8
	CWriteTxPayloadNoAck         = 0xB0
	CFlushTx                     = 0xE1
	CFlushRx                     = 0xE2
	CReuseTxPl                   = 0xE3
	CNop                         = 0xFF
)

// nRF24L01 registers and bits
const (
	RConfig    Register = 0x00
	BMaskRxDr  Bit      = 6
	BMaskTxDs           = 5
	BMaskMaxRt          = 4
	BEnCrc              = 3
	BCrcO               = 2
	BPwrUp              = 1
	BPrimRx             = 0

	REnAA   Register = 0x01
	BEnAAP5 Bit      = 5
	BEnAAP4          = 4
	BEnAAP3          = 3
	BEnAAP2          = 2
	BEnAAP1          = 1
	BEnAAP0          = 0

	REnRxAddr Register = 0x02
	BEnRxP5   Bit      = 5
	BEnRxP4            = 4
	BEnRxP3            = 3
	BEnRxP2            = 2
	BEnRxP1            = 1
	BEnRxP0            = 0

	RSetupAW Register = 0x03
	BAW      Bit      = 0

	RSetupRetr Register = 0x04
	BARD       Bit      = 4
	BARC                = 0

	RRFCh Register = 0x05

	RRFSetup  Register = 0x06
	BContWave Bit      = 7
	BRfDrLow           = 5
	BPllLock           = 4
	BRfDrHigh          = 3
	BRfPwr             = 1

	RStatus       Register = 0x07
	BRxDr         Bit      = 6
	BTxDs                  = 5
	BMaxRt                 = 4
	BRxPNo                 = 1
	BStatusTxFull          = 0
	BRxPNoMask    byte     = 0x0E

	RObserveTx Register = 0x08
	BPLosCnt   Bit      = 4
	BArcCnt             = 0

	RRPD      Register = 0x09
	RRxAddrP0          = 0x0A
	RRxAddrP1          = 0x0B
	RRxAddrP2          = 0x0C
	RRxAddrP3          = 0x0D
	RRxAddrP4          = 0x0E
	RRxAddrP5          = 0x0F
	RTxAddr            = 0x10
	RRxPwP0            = 0x11
	RRxPwP1            = 0x12
	RRxPwP2            = 0x13
	RRxPwP3            = 0x14
	RRxPwP4            = 0x15
	RRxPwP5            = 0x16

	RFifoStatus Register = 0x17
	BTxReuse    Bit      = 6
	BFifoTxFull          = 5
	BTxEmpty             = 4
	BRxFull              = 1
	BRxEmpty             = 0

	RDynPd Register = 0x1C
	BDplP5 Bit      = 5
	BDplP4          = 4
	BDplP3          = 3
	BDplP2          = 2
	BDplP1          = 1
	BDplP0          = 0

	RFeature  Register = 0x1D
	BEnDpl    Bit      = 2
	BEnAckPay          = 1
	BEnDynAck          = 0
)

var registerLengths = map[Register]byte{
	RConfig:     1,
	REnAA:       1,
	REnRxAddr:   1,
	RSetupAW:    1,
	RSetupRetr:  1,
	RRFCh:       1,
	RRFSetup:    1,
	RStatus:     1,
	RObserveTx:  1,
	RRPD:        1,
	RRxAddrP0:   5,
	RRxAddrP1:   1,
	RRxAddrP2:   1,
	RRxAddrP3:   1,
	RRxAddrP4:   1,
	RRxAddrP5:   1,
	RTxAddr:     5,
	RRxPwP0:     1,
	RRxPwP1:     1,
	RRxPwP2:     1,
	RRxPwP3:     1,
	RRxPwP4:     1,
	RRxPwP5:     1,
	RFifoStatus: 1,
	RDynPd:      1,
	RFeature:    1,
}
