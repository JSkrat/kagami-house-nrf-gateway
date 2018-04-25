#ifndef DRIVER_H
#define DRIVER_H
#include <stdint.h>
#include <vector>
#include <string>
#include <linux/spi/spidev.h>

// gpio registers base address
#define GPIO_BASE 0x20200000
// need only map B4 registers
#define GPIO_LEN 0xB4
//gpio registers
#define GPFSET0 7
#define GPFCLR0 10
#define GPFLEV0 13
#define GPFSEL0 0
#define GPFSEL1 1
#define GPFSEL2 2
#define GPFSEL3 3

class Driver;
using namespace std;
typedef std::vector <uint8_t> tRegister;
typedef struct {
    std::string name;
    uint8_t address;
    uint8_t size;
    uint8_t mask;
} tRegisterDescription;
// typedef for a callback. screw that mindblowing function pointer syntax \0/
//typedef void (*DriverCallback)(Driver* driver);
//typedef void (*ReceiveCallback)(Driver* driver, uint8_t pipe, std::vector <uint8_t> data);
using DriverCallback = void (*)(Driver* driver);
using ReceiveCallback = void (*)(Driver* driver, uint8_t pipe, std::vector <uint8_t> data);
using SentCallback = void (*)(Driver* driver, uint8_t pipe);

class State {
    uint8_t _state;
public:
    State(uint8_t state);
    uint8_t state();
    bool txFull();
    unsigned int rxPipeNumber();
    bool maxRT();
    bool txDataSent();
    bool rxDataReady();
};

class Driver
{
    const tRegisterDescription Registers[32];

    int fileDescriptor;
    uint8_t lastStatus;
    uint8_t rxBuffer[64];
    uint8_t txBuffer[64];
    struct spi_ioc_transfer tr;
    ReceiveCallback *receiveCallback;
    SentCallback *sentCallback;

    State send();
    // gpio stuff
    volatile unsigned int *gpio;
    volatile unsigned int *mapRegAddr(unsigned long baseAddr); //performs mmaping into '/dev/mem'
    void setPinDirection(const unsigned int pinnum, const bool dirOutput);
    bool readPin(const unsigned int pinnum);
    void writePinState(const unsigned int pinnum, const bool pinstate);
        void inline writePinHigh(unsigned int pinnum){*(this->gpio + GPFSET0) = (1 << pinnum);}
        void inline writePinLow(unsigned int pinnum){*(this->gpio + GPFCLR0) = (1 << pinnum);}
public:
    Driver(const char *SPIFileName, ReceiveCallback *receiveCallback, SentCallback *sentCallback);
    ~Driver();

    void activateCE();
    void deactivateCE();
    State getLastState();
    State readState();
    tRegister readRegister(uint8_t address);
    State writeRegister(uint8_t address, tRegister data);
    State writeRegister(uint8_t address, uint8_t data);
    unsigned int readRXPayloadWidth();
    vector <uint8_t> readRXPayload();
    State writeTXPayload(vector <uint8_t> payload);
    State flushTX();
    State flushRX();
    /// call this callback when IRQ from the device has came or by timer
    void processState();

    tRegister getTXAddr();
    tRegister getRXAddr(uint8_t pipe);

    State receive();
    State transmit();
};

#endif // DRIVER_H
