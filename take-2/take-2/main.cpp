#include <iostream>
#include "../RF24/RF24.h"
extern "C" {
#include "../scgi-c-library/scgilib.h"
}
#include "functions.h"
#include <inttypes.h>
#include <string>

using namespace std;

// configuration for the odroid-xu4
// CE is 2, CSN is 10
RF24 radio(2, 10);
#define INTERRUPT_PIN 7
#define RADIO_ADDRESS 0x0000000001

#define SCGIPORT 4000
#define MAX_CONNECTIONS_PER_PERIOD 6

uint8_t deviceAddress[6];

std::string status;
int rfLevel, statusCode;
uint8_t payload[32];
uint8_t payloadLength;


void handler() {
    bool tx_ok, tx_fail, rx;
    // this is resetting the irq flags along
    radio.whatHappened(tx_ok, tx_fail, rx);

    if (rx) {
        // answer received
        payloadLength = 0;
        while (radio.available()) {
            // read one byte to count the payload size
            radio.read(&(payload[payloadLength]), 1);
            payloadLength++;
        }
    } else if (tx_ok) {
        // request sent
        statusCode = 204;
        status = "WIP: packet sent, ack received, waiting for the reply";
    } else if (tx_fail) {
        // no ack received
        statusCode = 203;
        status = "error: no ack";
    }
}

void initRadio() {
    radio.begin();
    radio.setAutoAck(true);

    radio.openReadingPipe(0, RADIO_ADDRESS);


    radio.startListening();
    radio.stopListening();

    // do not mask any irq
    radio.maskIRQ(false, false, false);
    attachInterrupt(INTERRUPT_PIN, INT_EDGE_FALLING, handler);

    radio.printDetails();
}

void initSCGI() {
    int code = scgi_initialize(SCGIPORT);
    if (code) {
        printf("SCGI port %d (return %d)\n", SCGIPORT, code);
    } else {
        printf("Could not listen port %d\n", SCGIPORT);
        exit(1);
    }
}

int main()
{
    initSCGI();
    initRadio();

    while (true) {
        // sleep for 0.5ms
        usleep(500);

        int connections = 0;
        while (MAX_CONNECTIONS_PER_PERIOD > connections) {
            scgi_request *req = scgi_recv();
            int dead;

            // if no more connections go back to sleep
            if (NULL == req) break;

            connections++;
            dead = 0;
            req->dead = &dead;

            // send radio request here, wait for the response, then serialize it and put it into an answer
            // and we can't do anything during that, because we have only one transmitter

            status = ""; statusCode = 0;
            rfLevel = -1; payloadLength = 0;
            // parse arguments
            printf("query string <%s>\n", req->query_string);
            tArguments arguments = parseQueryString(req->query_string);
            if (! arguments.errorCode) {
                radio.openWritingPipe(*(reinterpret_cast<uint64_t*>(arguments.address)));
                status = "error: was no ack - you shouldn't see that";
                statusCode = 201;
                if (radio.write(arguments.payload, arguments.payloadLength)) {
                    status = "error: no interrupts from the transmitter came";
                    statusCode = 202;
                    // listen for the manual response here (not ack!)
                    radio.startListening();
                    /// @TODO redo that with something like mutex to reduce cpu consumption
                    // waiting 2ms for interrupt to come and change the statusCode
                    int timeout = 0;
                    while (202 == statusCode && timeout++ < 4) {
                        usleep(500);
                    }
                    // that's all --- send what irq function prepared
                }

                //payload = reinterpret_cast<char*>(arguments.payload);
            } else {
                status = "error: bad arguments " + std::to_string(arguments.errorCode);
                statusCode = 100 + arguments.errorCode;
            }

            // serialize payload
            std::string payloadStr = "";
            for (int i = 0; i < payloadLength; i++) {
                const char alph[] = {"0123456789ABCDEF"};
                payloadStr += alph[payload[i] >> 4];
                payloadStr += alph[payload[i] & 0x0F];
            }
            std::string response = "Status: 200 OK\r\n"
                       "Content-Type: application/json\r\n\r\n"
                    "{\"status\":" + std::to_string(statusCode) + ""
                    ",\"statusText\":\"" + status + "\""
                    ",\"rf-level\":" + std::to_string(rfLevel) + ""
                    ",\"data\":\"" + payloadStr + "\"}"
                    ;
            if (! scgi_write(req, const_cast<char*>(response.c_str()))) {
                std::cerr << "SCGI response could not be sent, probably not enough RAM\n";
            } else {
                if (1 == dead) {
                    std::cerr << "The connection was killed by SCGI library before response sent back\n";
                }
            }
            freeArguments(arguments);

            if (! dead) {
                req->dead = NULL;
            }
        }
    }
    return 0;
}
