#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

#include <fcntl.h>
#include <unistd.h>
#include <sys/ioctl.h>
#include <linux/types.h>
#include <linux/spi/spidev.h>

static void pabort(const char *s)
{
	perror(s);
	abort();
}


int main(int argc, char **argv) {
	int fd = open("/dev/spidev1.0", O_RDWR);

	// setup
	int mode = SPI_MODE_0; // CPOL = 0 (clk not inverted); CPHA = 0 (bit at rising edge)
	ioctl(fd, SPI_IOC_WR_MODE, &mode);

	int maxSpeed = 3760000;
	ioctl(fd, SPI_IOC_WR_MAX_SPEED_HZ, &maxSpeed);
	ioctl(fd, SPI_IOC_RD_MAX_SPEED_HZ, &maxSpeed);
	printf("Max speed: %dHz\n", maxSpeed);

	// MSB
	int lsb_setting = 0;
	ioctl(fd, SPI_IOC_WR_LSB_FIRST, &lsb_setting);

	// 8 bits per word
	int bits_per_word = 0;
	ioctl(fd, SPI_IOC_WR_BITS_PER_WORD, &bits_per_word);

	// read register 0
	uint8_t tx[] = {
		0x00, 0x00
	};
	uint8_t rx[sizeof(tx)];

	struct spi_ioc_transfer tr = {
		.tx_buf = (unsigned long) tx,
		.rx_buf = (unsigned long) rx,
		.len = sizeof(tx),
		.delay_usecs = 0,
		.speed_hz = 0,
		.bits_per_word = 0,
	};

	int ret = ioctl(fd, SPI_IOC_MESSAGE(1), &tr);
	if (1 == ret) {
		pabort("can't send spi message");
	}

	printf("Device status 0x%02X, config byte 0x%02X\n", rx[0], rx[1]);
	return 0;
}
