#ifndef DRIVER_H
#define DRIVER_H
#include <stdint.h>
#include <vector>
#include <string>
#include <linux/spi/spidev.h>

#include "gpio/source/cpuinfo.h"

#define MODE_UNKNOWN -1
#define BOARD        10
#define BCM          11
#define SERIAL       40
#define SPI          41
#define I2C          42
#define PWM          43

int gpio_mode;
#define MAXPINCOUNT 40  //odroid added
const int pin_to_gpio_rev1[MAXPINCOUNT+1];
const int pin_to_gpio_rev2[MAXPINCOUNT+1];
const int pin_to_gpio_rev3[MAXPINCOUNT+1];
const int (*pin_to_gpio)[MAXPINCOUNT+1];
#define MAXGPIOCOUNT 255  //odroid added
int gpio_direction[MAXGPIOCOUNT+1];  //odroid change 54->255 to accommodate XU4 gpio numbers as index
rpi_info rpiinfo;

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
