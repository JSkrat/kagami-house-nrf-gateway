#include "functions.h"
#include <inttypes.h>
#include <string>
#include <sstream>

/*
 * converts two hex-chars to the byte value
 * if chars are not hex, it converts anyway
 */
uint8_t hexConv(char mst, char lst) {
    if ('F' < mst) mst -= 'a'-10;
    else if ('A' <= mst) mst -= 'A'-10;
    else mst -= '0';

    if ('F' < lst) lst -= 'a'-10;
    else if ('A' <= lst) lst -= 'A'-10;
    else lst -= '0';

    return (mst<<4) | lst;
}

/*
 * parse query string to a structure
 * all char* should be freed outside
 */
tArguments parseQueryString(char* queryString) {
    tArguments ret = {0, 0, 0};
    std::string query = queryString;
    std::stringstream s(query);
    std::string param;
    while (std::getline(s, param, '&')) {
        std::string name, value;
        std::size_t eq = param.find_first_of('=');
        // we suppose that npos (in case of "not found" condition) equals -1
        name = param.substr(0, eq);
        value = param.substr(eq+1);
        // urldecode value, convert %21 to ! and other binary stuff
        // we may use not all malloc'ed memory, but i don't care, it shouldn't be too big, urlstring is 260-bytes limited anyway
        uint8_t *binValue = reinterpret_cast<uint8_t*>(malloc(value.length()));
        int index = 0;
        for (int i = 0; i < value.length(); i++) {
            if ('%' == value.at(i)) {
                // we ignore unfinished %-sequence at the end of line (% or %1 for example)
                if (i+2 < value.length()) {
                    binValue[index] = hexConv(value.at(i+1), value.at(i+2));
                    i += 2;
                }
            } else {
                binValue[index] = value.at(i);
            }
            index += 1;
        }
        binValue[index] = 0;
        if ("address" == name) {
            ret.address = binValue;
            if (5 != index) {
                printf("functions::parseQueryString address length is wrong\n");
                ret.errorCode = 1;
                return ret;
            }
        } else if ("data" == name) {
            ret.payload = binValue;
            ret.payloadLength = index;
            if (32 < index) {
                printf("functions::parseQueryString payload is longer than 32 byte\n");
                ret.errorCode = 2;
                return ret;
            }
        } else {
            printf("unknown parameter <%s>\n", name.c_str());
            free(binValue);
        }
    }
    return ret;
}

void freeArguments (tArguments arguments) {
    if (NULL != arguments.address) free(arguments.address);
    if (NULL != arguments.payload) free(arguments.payload);
}

