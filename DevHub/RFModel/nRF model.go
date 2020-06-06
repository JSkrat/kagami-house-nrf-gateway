package nRF_model

import (
	"fmt"
	"log"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/conn/spi"
	"periph.io/x/periph/conn/spi/spireg"
	"periph.io/x/periph/host"
)

func Test() {
	fmt.Println("test ok")
}

func Init() {
	// Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Use spireg SPI port registry to find the first available SPI bus.
	p, err := spireg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer p.Close()

	// Convert the spi.Port into a spi.Conn so it can be used for communication.
	c, err := p.Connect(physic.MegaHertz, spi.Mode3, 8)
	if err != nil {
		log.Fatal(err)
	}

	// Prints out the gpio pin used.
	if p, ok := c.(spi.Pins); ok {
		fmt.Printf("  CLK : %s", p.CLK())
		fmt.Printf("  MOSI: %s", p.MOSI())
		fmt.Printf("  MISO: %s", p.MISO())
		fmt.Printf("  CS  : %s", p.CS())
	}
}
