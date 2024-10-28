package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func receiveIRSignals(signalCh chan<- string) {
	lircSocket := "/var/run/lirc/lircd"

	file, err := os.Open(lircSocket)
	if err != nil {
		log.Fatalf("cannnot open a LIRC socket: %v", err)
	}
	defer file.Close()

	fmt.Println("waiting ir data...")

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		signalCh <- scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("LIRC read error: %v", err)
	}
}
