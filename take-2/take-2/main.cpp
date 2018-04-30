#include <iostream>
#include "../RF24/RF24.h"
extern "C" {
#include "../scgi-c-library/scgilib.h"
}

using namespace std;

// configuration for the odroid-xu4
// CE is 2, CSN is 10
RF24 radio(2, 10);

#define SCGIPORT 4000
#define MAX_CONNECTIONS_PER_PERIOD 6

void initRadio() {
    radio.begin();
    radio.setAutoAck(true);

    radio.startListening();
    radio.stopListening();

    radio.printDetails();
}

void initSCGI() {
    if (scgi_initialize(SCGIPORT)) {
        printf("SCGI port %d\n", SCGIPORT);
    } else {
        printf("Could not listen port %d\n", SCGIPORT);
        exit(1);
    }
}

int main()
{
    initRadio();
    initSCGI();

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

            // send radio request here and wait for the response, then serialize it and put it into an answer
            // and we can't do anything during that, because we have only one transmitter

            if (! scgi_write(req,
                             "Status: 200 OK\r\n"
                             "Content-Type: application/json\r\n\r\n"
                             )) {
                std::cerr << "SCGI response could not be sent, probably not enough RAM\n";
            } else {
                if (1 == dead) {
                    std::cerr << "The connection was killed by SCGI library before response sent back\n";
                }
            }

            if (! dead) {
                req->dead = NULL;
            }
        }
    }
    return 0;
}
