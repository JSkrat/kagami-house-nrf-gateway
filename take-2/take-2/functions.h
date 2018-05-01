#include <inttypes.h>

typedef struct {
    uint8_t *address;
    uint8_t *payload;
    uint8_t payloadLength;
    uint8_t errorCode;
} tArguments;

uint8_t hexConv(char mst, char lst);
tArguments parseQueryString(char* queryString);
void freeArguments (tArguments arguments);
