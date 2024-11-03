package main

import (
	"fmt"

	"gpioblink.com/app/karaoke-remote/remote"
)

func main() {
	signalCh := make(chan string)

	go remote.ReceiveIRSignals(signalCh)

	for signal := range signalCh {
		fmt.Println("Received IR Signal:", signal)
	}
}
