#include "driver.h"
#include <fcntl.h>
#include <unistd.h>
#include <sys/ioctl.h>
#include <linux/types.h>
#include <linux/spi/spidev.h>
#include <vector>
#include <cstring> // for the memset
#include <stdexcept>
#include <fstream>
#include <sys/mman.h>

#include "gpio/source/c_gpio.h"
#include "gpio/source/cpuinfo.h"
#include "gpio/source/odroid.h"

State Driver::send()
{
    int ret = ioctl(this->fileDescriptor, SPI_IOC_MESSAGE(1), &this->tr);
    if (1 == ret) {
        throw std::runtime_error("Driver::send ioctl returned 1");
    }
    this->lastStatus = this->rxBuffer[0];
    return State(this->lastStatus);
}

Driver::Driver(const char* SPIFileName, ReceiveCallback *receiveCallback, SentCallback *sentCallback) :
    Registers({
                        { .name = "CONFIG",     .address = 0x00, .size = 1, .mask = 0xFF },
                        { .name = "EN_AA",      .address = 0x01, .size = 1, .mask = 0x3F },
                        { .name = "EN_RXADDR",  .address = 0x02, .size = 1, .mask = 0x3F },
                        { .name = "SETUP_AW",   .address = 0x03, .size = 1, .mask = 0x03 },
                        { .name = "SETUP_RETR", .address = 0x04, .size = 1, .mask = 0x0F },
                        { .name = "RF_CH",      .address = 0x05, .size = 1, .mask = 0x7F },
                        { .name = "RF_SETUP",   .address = 0x06, .size = 1, .mask = 0xFF },
                        { .name = "STATUS",     .address = 0x07, .size = 1, .mask = 0x7F },
                        { .name = "OBSERVE_TX", .address = 0x08, .size = 1, .mask = 0xFF },
                        { .name = "RPD",        .address = 0x09, .size = 1, .mask = 0x01 },
                        { .name = "RX_ADDR_P0", .address = 0x0A, .size = 5, .mask = 0xFF },
                        { .name = "RX_ADDR_P1", .address = 0x0B, .size = 5, .mask = 0xFF },
                        { .name = "RX_ADDR_P2", .address = 0x0C, .size = 1, .mask = 0xFF },
                        { .name = "RX_ADDR_P3", .address = 0x0D, .size = 1, .mask = 0xFF },
                        { .name = "RX_ADDR_P4", .address = 0x0E, .size = 1, .mask = 0xFF },
                        { .name = "RX_ADDR_P5", .address = 0x0F, .size = 1, .mask = 0xFF },
                        { .name = "TX_ADDR",    .address = 0x10, .size = 5, .mask = 0xFF },
                        { .name = "RX_PW_P0",   .address = 0x11, .size = 1, .mask = 0x3F },
                        { .name = "RX_PW_P1",   .address = 0x12, .size = 1, .mask = 0x3F },
                        { .name = "RX_PW_P2",   .address = 0x13, .size = 1, .mask = 0x3F },
                        { .name = "RX_PW_P3",   .address = 0x14, .size = 1, .mask = 0x3F },
                        { .name = "RX_PW_P4",   .address = 0x15, .size = 1, .mask = 0x3F },
                        { .name = "RX_PW_P5",   .address = 0x16, .size = 1, .mask = 0x3F },
                        { .name = "FIFO_STATUS",.address = 0x17, .size = 1, .mask = 0x73 },
                        { .name = "n/a",        .address = 0x18, .size = 1, .mask = 0xFF },
                        { .name = "n/a",        .address = 0x19, .size = 1, .mask = 0xFF },
                        { .name = "n/a",        .address = 0x1A, .size = 1, .mask = 0xFF },
                        { .name = "n/a",        .address = 0x1B, .size = 1, .mask = 0xFF },
                        { .name = "DYNPD",      .address = 0x1C, .size = 1, .mask = 0x3F },
                        { .name = "FEATURE",    .address = 0x1D, .size = 1, .mask = 0x07 },
                        { .name = "n/a",        .address = 0x1E, .size = 1, .mask = 0xFF },
                        { .name = "n/a",        .address = 0x1F, .size = 1, .mask = 0xFF },
                    })
{
    this->fileDescriptor = open(SPIFileName, O_RDWR);

    // setup
    int mode = SPI_MODE_0; // CPOL = 0 (clk not inverted); CPHA = 0 (bit at rising edge)
    ioctl(this->fileDescriptor, SPI_IOC_WR_MODE, &mode);

    int maxSpeed = 3760000;
    ioctl(this->fileDescriptor, SPI_IOC_WR_MAX_SPEED_HZ, &maxSpeed);
    ioctl(this->fileDescriptor, SPI_IOC_RD_MAX_SPEED_HZ, &maxSpeed);
    printf("SPI max speed: %dHz\n", maxSpeed);

    // MSB
    int lsb_setting = 0;
    ioctl(this->fileDescriptor, SPI_IOC_WR_LSB_FIRST, &lsb_setting);

    // 8 bits per word
    int bits_per_word = 0;
    ioctl(this->fileDescriptor, SPI_IOC_WR_BITS_PER_WORD, &bits_per_word);

    // preinit read-write structure
    // fucking c++ forbids designated initializers for the structure. hate it! >_<
    memset(&(this->tr), 0, sizeof(this->tr));
    this->tr.tx_buf = reinterpret_cast<unsigned long>(this->txBuffer);
    this->tr.rx_buf = reinterpret_cast<unsigned long>(this->rxBuffer);

    this->receiveCallback = receiveCallback;
    this->sentCallback = sentCallback;

    // setup gpio (from py_pgio.c)
    for (i=0; i<=MAXGPIOCOUNT; i++)  //odroid patch
        gpio_direction[i] = -1;
    if (get_rpi_info(&rpiinfo)) {
        throw std::runtime_error("Driver::Driver gpio init: unsupported platform detected");
    }
    if (strstr(rpiinfo.type, "ODROID")) {
        setMappingPtrsOdroid();
    } else {
        bcm_to_odroidgpio = &bcmToOGpioRPi;  //1:1 mapping
        if (rpiinfo.p1_revision == 1) {
            pin_to_gpio = &pin_to_gpio_rev1;
        } else if (rpiinfo.p1_revision == 2) {
            pin_to_gpio = &pin_to_gpio_rev2;
        } else { // assume model B+ or A+ or 2B
            pin_to_gpio = &pin_to_gpio_rev3;
        }
    }
}

Driver::~Driver()
{
    close(this->fileDescriptor);
    // close gpio
    cleanup();
}

void Driver::activateCE()
{
    // gpio. somehow.

    // wait 10us
    // (CE must remain active for at least 10us)
    usleep(10);
}

void Driver::deactivateCE()
{

}

State Driver::getLastState()
{
    return State(this->lastStatus);
}

State Driver::readState()
{
    this->tr.len = 1;
    this->txBuffer[0] = 0b11111111; // NOP
    return this->send();
}

tRegister Driver::readRegister(uint8_t address)
{
    if (31 < address) {
        // throw an exception
        throw std::runtime_error("Driver::readRegister register address should not be bigger, than 31, " + to_string(address) + " given");
    }
    this->tr.len = this->Registers[address].size;
    this->txBuffer[0] = 0b00000000 | address;
    this->send();
    tRegister result;
    for (int i = 1; i < this->Registers[address].size; i++) {
        result.push_back(this->rxBuffer[i]);
    }
    return result;
}

State Driver::writeRegister(uint8_t address, uint8_t data)
{
    return this->writeRegister(address, std::vector<uint8_t>(data));
}

unsigned int Driver::readRXPayloadWidth()
{
    this->tr.len = 2;
    this->txBuffer[0] = 0b01100000;
    this->send();
    return this->rxBuffer[1];
}

vector<uint8_t> Driver::readRXPayload()
{
    vector<uint8_t> ret;
    unsigned int length = this->readRXPayloadWidth();
    if (! length) return ret;
    this->tr.len = length;
    this->txBuffer[0] = 0b01100001;
    this->send();
    for (int i = 0; i < length; i++) {
        ret.push_back(rxBuffer[i + 1]);
    }
    return ret;
}

State Driver::writeTXPayload(vector<uint8_t> payload)
{
    if (32 < payload.size() || 0 == payload.size()) {
        throw std::runtime_error("Driver::writeTXPayload payload should be up to 32 bytes long, but it's " + to_string(payload.size()) + " bytes long");
    }
    this->tr.len = payload.size();
    this->txBuffer[0] = 0b10100000;
    for (int i = 0; i < payload.size(); i++) {
        this->txBuffer[i + 1] = payload[i];
    }
    return this->send();
}

State Driver::flushTX()
{
    this->tr.len = 1;
    txBuffer[0] = 0b11100001;
    return this->send();
}

State Driver::flushRX()
{
    this->tr.len = 1;
    txBuffer[0] = 0b11100010;
    return this->send();
}

void Driver::processState()
{
    State state = this->readState();
    if (state.rxDataReady()) {
        // state.rxPipeNumber()
        if (NULL != this->receiveCallback) (this->receiveCallback)(this, state.rxPipeNumber(), this->readRXPayload());
        // probably deactivate CE here to return to standby mode from the receive mode
    } else if (state.txDataSent()) {
        // ACK received
        if (NULL != this->sentCallback) (this->sentCallback)(this, 0);
    } else if (state.maxRT()) {
        // maximum number of TX retransmits, receiver not responding
    } else {
        // IDLE
    }
}

tRegister Driver::getTXAddr()
{
    return this->readRegister(0x10);
}

tRegister Driver::getRXAddr(uint8_t pipe)
{
    if (5 < pipe) throw std::runtime_error("Driver::getRXAddr pipe number can not be greater than 5, but it is " + to_string(pipe));
    if (0 == pipe) return this->readRegister(0x0A);
    tRegister ret = this->readRegister(0x0B);
    if (2 <= pipe) {
        ret[0] = this->readRegister(0x0A + pipe);
    }
    return ret;
}

State Driver::receive()
{
    // PWR_UP and PRIM_RX set
    tRegister config = this->readRegister(0x00);
    config[0] |= 0b00000011;
    State ret = this->writeRegister(0x00, config);
    this->activateCE();
    return ret;
}

/// will set "transmission mode" until the TX FIFO empty or CE deactivated
State Driver::transmit()
{
    // PWR_UP set, PRIM_RX unset
    tRegister config = this->readRegister(0x00);
    config[0] &= ~(0b00000011);
    config[0] |= 0b00000010;
    State ret = this->writeRegister(0x00, config);
    this->activateCE();
    return ret;
}

State Driver::writeRegister(uint8_t address, tRegister data)
{
    if (31 < address) {
        throw std::runtime_error("Driver::writeRegister address should be 0-31, " + to_string(address) + " given");
    }
    if (data.size() > this->Registers[address].size) {
        throw std::runtime_error(
                    "Driver::writeRegister register " + to_string(address) + " size is " + to_string(this->Registers[address].size) +
                    ", but given data is " + to_string(data.size()) + " bytes long");
    }
    this->tr.len = data.size();
    txBuffer[0] = 0b00100000 | address;
    for (uint8_t i = 0; i < data.size(); i++) {
        txBuffer[i + 1] = data[i];
    }
    return this->send();
}

/******************************************************************************************************/

State::State(uint8_t state)
{
    this->_state = state;
}

uint8_t State::state()
{
    return this->_state;
}

bool State::txFull()
{
    return (this->_state & 0b00000001);
}

unsigned int State::rxPipeNumber()
{
    return (this->_state & 0b00001110) >> 1;
}

bool State::maxRT()
{
    return (this->_state & 0b00010000);
}

bool State::txDataSent()
{
    return (this->_state & 0b00100000);
}

bool State::rxDataReady()
{
    return (this->_state & 0b01000000);
}
