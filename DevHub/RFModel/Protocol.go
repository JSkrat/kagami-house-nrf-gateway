package RFModel

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	PacketLength       uint = 32
	RequestHeaderSize  uint = 4
	ResponseHeaderSize uint = 3
	MaxDataLengthRq    uint = PacketLength - RequestHeaderSize
	MaxDataLengthRs    uint = PacketLength - ResponseHeaderSize
)

type packet []byte

type request struct {
	Version       byte
	TransactionID byte
	UnitID        byte
	FunctionID    byte
	Data          [MaxDataLengthRq]byte
	DataLength    byte
}

type response struct {
	Version       byte
	TransactionID byte
	Code          byte
	Data          [MaxDataLengthRs]byte
	DataLength    byte
}

type Payload interface {
	Payload() []byte
}

func makeRequest(rq request) packet {
	if MaxDataLengthRq < uint(rq.DataLength) {
		panic(errors.New(fmt.Sprintf("too big DataLength %v", rq.DataLength)))
	}
	buf := bytes.Buffer{}
	if err := binary.Write(&buf, binary.LittleEndian, rq); err != nil {
		panic(errors.New("binary.Write: " + err.Error()))
	}
	return buf.Bytes()[:PacketLength-MaxDataLengthRq+uint(rq.DataLength)]
}

func parseResponse(r packet) response {
	if PacketLength < uint(len(r)) {
		panic(errors.New(fmt.Sprintf("too big packet of length %v", len(r))))
	}
	var ret response
	buf := bytes.Buffer{}
	buf.Write(r)
	buf.Write(make([]byte, int(PacketLength+1)-len(r)))
	if err := binary.Read(&buf, binary.LittleEndian, &ret); err != nil {
		panic(errors.New("binary.Read: " + err.Error()))
	}
	ret.DataLength = byte(len(r) - int(ResponseHeaderSize))
	return ret
}

func (r request) Payload() []byte {
	var ret []byte
	copy(ret[:], r.Data[:r.DataLength])
	return ret
}

func (r response) Payload() []byte {
	var ret []byte
	copy(ret[:], r.Data[:r.DataLength])
	return ret
}
