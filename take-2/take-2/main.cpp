#include <iostream>
#include "../RF24/RF24.h"
extern "C" {
#include "../scgi-c-library/scgilib.h"
}
#include "functions.h"

using namespace std;

// configuration for the odroid-xu4
// CE is 2, CSN is 10
RF24 radio(2, 10);
#define INTERRUPT_PIN 7
#define RADIO_ADDRESS 0x0000000001

#define SCGIPORT 4000
#define MAX_CONNECTIONS_PER_PERIOD 6

uint8_t deviceAddress[6];

void handler() {
    bool tx_ok, tx_fail, rx;
    radio.whatHappened(tx_ok, tx_fail, rx);


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

            std::string status, rfLevel, payload;
            int statusCode = 0;
            // parse arguments
            printf("query string <%s>\n", req->query_string);
            tArguments arguments = parseQueryString(req->query_string);
            rfLevel = "0";
            if (! arguments.errorCode) {
                radio.openWritingPipe(*(reinterpret_cast<uint64_t*>(arguments.address)));
                status = 'error: no ack';
                if (radio.write(arguments.payload, arguments.payloadLength)) {
                    status = 'error: no response';
                }

                payload = reinterpret_cast<char*>(arguments.payload);
            } else {
                status = "error: bad arguments " + std::to_string(arguments.errorCode);
            }

            std::string response = "Status: 200 OK\r\n"
                       "Content-Type: application/json\r\n\r\n"
                    "{\"status\":" + std::to_string(statusCode) + ""
                    ",\"statusText\":\"" + status + "\""
                    ",\"rf-level\":\"" + rfLevel + "\""
                    ",\"data\":\"" + payload + "\"}"
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
