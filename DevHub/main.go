package main

import (
	"./RFModel"
)

var rf *nRF_model.NRFTransmitter

func main() {
	rf = new(nRF_model.NRFTransmitter)
	nRF_model.Open(rf, "", "2", "7")
	defer nRF_model.Close(rf)

}
