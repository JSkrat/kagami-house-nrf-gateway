package UartTransciever

import (
	"fmt"
	"os"
	"sync"
	"time"

	"../TranscieverModel"
	"github.com/sirupsen/logrus"
	"github.com/tarm/serial"
)

var log = logrus.New()

// UMTransmitter handle
type UMTransmitter struct {
	port           *serial.Port
	ReceiveMessage chan TranscieverModel.Message
	//SendMessage       chan Message
	SendMessageStatus chan TranscieverModel.Message
	mutex             sync.Mutex
}

// TransmitterSettings ...
type TransmitterSettings struct {
	PortName string
	Speed    int
}

// Init ...
func Init(tr *UMTransmitter, settings TransmitterSettings) {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()
	log.Formatter = new(logrus.TextFormatter)
	log.Level = logrus.InfoLevel
	log.Out = os.Stdout
	log.Info(fmt.Sprintf("OpenTransmitter begin"))
	c := &serial.Config{Name: settings.PortName, Baud: settings.Speed}
	port, err := serial.OpenPort(c)
	if err != nil {
		panic(fmt.Errorf("serial.OpenPort(%v): %v", settings.PortName, err.Error()))
	}
	tr.port = port
	defer func() {
		if r := recover(); nil != r {
			log.Error(r)
			tr.Close()
			panic(r)
		}
	}()
	go run(tr)
}

// Close port (in current library does nothing)
func (tr *UMTransmitter) Close() {
	/*if nil != tr.port {
		_ = tr.port.Close()
	}*/
}

func run(tr *UMTransmitter) {

}

func uartTransaction(rf *UMTransmitter, data []byte) []byte {
	rf.mutex.Lock()
	defer rf.mutex.Unlock()
	_, err := rf.port.Write(data)
	if nil != err {
		panic(fmt.Errorf("rf.port.Write error: %v", err))
	}
	receive := make(chan []byte)
	go func() {
		var bigBuf []byte
		for {
			buf := make([]byte, 0x100)
			n, err := rf.port.Read(buf)
			if nil != err {
				panic(fmt.Errorf("rf.port.Read error: %v", err))
			}
			bigBuf = append(bigBuf, buf[:n]...)
			if isPacketComplete(bigBuf) {
				break
			}
		}
		log.Debug(fmt.Sprintf("uartTransaction(%v) data %v", data, bigBuf))
		receive <- bigBuf
	}()
	timeout := make(chan bool)
	go func() {
		// TODO read from config
		<-time.After(500 * time.Millisecond)
		timeout <- true
	}()
	select {
	case rs := <-receive:
		return rs
	case <-timeout:
		panic(fmt.Errorf("uartTransaction response timeout. Request %v", data))
	}
}

func transmit(rf *UMTransmitter, a TranscieverModel.Address, data TranscieverModel.Payload) {
	rq := uartRequest{
		command: cTransmit,
		payload: append(a[:], data...),
	}
	response := uartTransaction(rf, stuffPacket(createRequest(rq)))
	rs := parseResponse(unstuffPacket(response))
	if !validateResponse(rs, cTransmit) {
		panic(fmt.Errorf("modem response validation failed. Command %v, payload %v, response %v", cTransmit, data, response))
	}
	if rOk != rs.code {
		panic(fmt.Errorf("modem response code is not ok. Request %v, response %v", rq, rs))
	}
}

func getRxItem(rf *UMTransmitter) (ret TranscieverModel.Message) {
	ret = TranscieverModel.Message{}
	rq := uartRequest{
		command: cGetRxItem,
	}
	response := uartTransaction(rf, stuffPacket(createRequest(rq)))
	rs := parseResponse(unstuffPacket(response))
	if !validateResponse(rs, cGetRxItem) {
		panic(fmt.Errorf("modem response validation failed. Command %v, response %v", cGetRxItem, response))
	}
	codeToStatus := map[responseCode]TranscieverModel.EMessageStatus{
		rNoPackets:            TranscieverModel.EMSNone,
		rSlaveResponseTimeout: TranscieverModel.EMSSlaveTimeout,
		rAckTimeout:           TranscieverModel.EMSAckTimeout,
		rDataPacket:           TranscieverModel.EMSDataPacket,
		rAckPacket:            TranscieverModel.EMSAckPacket,
	}
	ret.Status = codeToStatus[rs.code]
	if len(ret.Address) <= len(rs.payload) {
		copy(ret.Address[:], rs.payload[:len(ret.Address)])
	}
	if len(ret.Address) < len(rs.payload) {
		ret.Payload = rs.payload[len(ret.Address):]
	}
	return ret
}

// SendCommand commands modem to make a transaction to a given slave device and polls for the response
func (tr *UMTransmitter) SendCommand(a TranscieverModel.Address, data TranscieverModel.Payload) (ret TranscieverModel.Message) {
	transmit(tr, a, data)
	response := make(chan TranscieverModel.Message)
	timeout := make(chan bool)
	go func() {
		for {
			msg := getRxItem(tr)
			switch msg.Status {
			default:
				continue
			// wasn't even sent
			case TranscieverModel.EMSAckTimeout:
				timeout <- false
				return
			// we need one of those to return
			case TranscieverModel.EMSDataPacket:
			case TranscieverModel.EMSSlaveTimeout:
			}
			if msg.Address == a {
				response <- msg
				return
			}
			log.Warning(fmt.Sprintf("UMModel.SendCommand(%v, %v) got response from the wrong address %v", a, data, msg))
		}
	}()
	go func() {
		// in case modem won't say anything about timeout
		<-time.After(1000 * time.Millisecond)
		log.Warning(fmt.Sprintf("UMModel.SendCommand(%v, %v) modem did not generated any response packet in 1000ms", a, data))
		timeout <- true
	}()
	select {
	case msg := <-response:
		return msg
	case <-timeout:
		return TranscieverModel.Message{
			Address: a,
			Status:  TranscieverModel.EMSNone,
		}
	}
}
