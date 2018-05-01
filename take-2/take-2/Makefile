OPTIONS=-O2 -Wall -pthread -I/usr/local/include -L/usr/local/lib -lrf24
LIBSCGI=../scgi-c-library/scgilib

#allatonce: main.cpp ${LIBSCGI}.c
#	g++ ${OPTIONS} ${LIBSCGI}.c main.cpp -o main

all: main.o scgi-lib.a functions.o
	g++ main.o functions.o ${OPTIONS} -L./ -l:scgi-lib.a -o main

clean:
	rm main *.o

main.o: main.cpp
	g++ -c main.cpp

scgi-lib.a: ${LIBSCGI}.c
	gcc -c ${LIBSCGI}.c -o scgi-lib.o
	ar rcs scgi-lib.a scgi-lib.o

functions.o: functions.cpp
	gcc -c functions.cpp

