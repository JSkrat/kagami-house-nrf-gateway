#include <stdio.h>
#include <fcntl.h>
#include <stdlib.h>
#include <unistd.h>

#define DEVICE_FILE "/dev/spidev1.0"


char wr_buf[] = {0x15};
char rd_buf[2];
int main(int argc, char** argv) {
	int fd = open(DEVICE_FILE, O_RDWR);

	if (1 != write(fd, wr_buf, 1)) {
		perror("write error");
	}

	if (1 > read(fd, rd_buf, 1)) {
		perror("read error");
	}

	printf("pipe length is 0x%02X\n", rd_buf[0]);

	close(fd);
	return 0;
}
