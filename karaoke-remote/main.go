package main

import (
	"fmt"
)

func main() {
	signalCh := make(chan string)

	go receiveIRSignals(signalCh)

	for signal := range signalCh {
		fmt.Println("Received IR Signal:", signal)
	}
}
