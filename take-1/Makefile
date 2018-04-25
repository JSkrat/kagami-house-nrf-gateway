#FLAGS = -std=gnu++11
FLAGS = -std=c++14

clean:
	rm -rf *.o take-1

driver.o: driver.cpp
	g++ $(FLAGS) -c driver.cpp


main.o: main.cpp
	g++ $(FLAGS) -c main.cpp

take-1: main.o driver.o
	g++ main.o driver.o -o take-1

all: take-1

