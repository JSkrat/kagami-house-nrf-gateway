TEMPLATE = app
CONFIG += console c++11
CONFIG -= app_bundle
CONFIG -= qt

SOURCES += main.cpp \
    driver.cpp \
    gpio/source/c_gpio.c \
    gpio/source/cpuinfo.c \
    gpio/source/odroid.c

HEADERS += \
    driver.h \
    gpio/source/c_gpio.h \
    gpio/source/cpuinfo.h \
    gpio/source/odroid.h

DISTFILES += \
    Makefile
