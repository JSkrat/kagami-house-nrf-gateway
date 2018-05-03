cmake_minimum_required(VERSION 2.8)

project(take-2)
configure_file (
	"${PROJECT_SOURCE_DIR}/Config.h.in"
	"${PROJECT_BINARY_DIR}/Config.h"
)

include_directories("${PROJECT_BINARY_DIR}")

#add_library(../RF24 

add_executable(${PROJECT_NAME} "main.cpp")

