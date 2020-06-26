package RFModel

import "testing"
import "../nRFModel"

func Assert(t *testing.T, condition bool, errorMessage string) {
	if !condition {
		t.Error(errorMessage)
	}
}

func TestBuildRequest(t *testing.T) {
	// test basic creation
	pStruct := request{
		Version:       1,
		TransactionID: 2,
		UnitID:        3,
		FunctionID:    4,
		Data:          [28]byte{},
		DataLength:    0,
	}
	pPacket := serializeRequest(&pStruct)
	Assert(t, pStruct.Version == pPacket[0], "version does not equal")
	Assert(t, pStruct.TransactionID == pPacket[1], "transaction id does not equal")
	Assert(t, pStruct.UnitID == pPacket[2], "unit id does not equal")
	Assert(t, pStruct.FunctionID == pPacket[3], "function id does not equal")
	Assert(t, int(RequestHeaderSize) == len(pPacket), "packet length is not 4 as required for empty data")
	// test data length
	pStruct.DataLength = 14
	pStruct.Data[0] = 12
	pStruct.Data[13] = 11
	pPacket = serializeRequest(&pStruct)
	Assert(t, int(RequestHeaderSize)+14 == len(pPacket), "packet length for 14 bytes data is incorrect")
	Assert(t, 12 == pPacket[RequestHeaderSize], "begin of data is wrong")
	Assert(t, 11 == pPacket[RequestHeaderSize+13], "end of data is wrong")
}

func TestParseResponse(t *testing.T) {
	rBytes := nRF_model.Payload{1, 2, 3}
	rStruct := parseResponse(&rBytes)
	Assert(t, rBytes[0] == rStruct.Version, "version is wrong")
	Assert(t, rBytes[1] == rStruct.TransactionID, "transaction id is wrong")
	Assert(t, rBytes[2] == rStruct.Code, "response code is wrong")
	Assert(t, 0 == rStruct.DataLength, "data length is not 0")
	// test data length
	rBytes = append(rBytes, 4, 5, 6)
	rStruct = parseResponse(&rBytes)
	Assert(t, 3 == rStruct.DataLength, "data length is not 3")
	Assert(t, 4 == rStruct.Data[0], "begin of data is wrong")
	Assert(t, 6 == rStruct.Data[2], "end of data is wrong")
}
