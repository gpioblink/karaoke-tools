package main

import (
	"fmt"
	"os"

	"gpioblink.com/app/karaoke-remote/remote"
)

func main() {
	signalCh := make(chan string)

	go remote.ReceiveIRSignals(signalCh)

	fifo, err := os.OpenFile("/tmp/karaoke-fifo", os.O_WRONLY, 0600)
	if err != nil {
		fmt.Println("Error opening FIFO:", err)
		return
	}
	defer fifo.Close()

	for signal := range signalCh {
		fmt.Println(signal)
		_, err := fifo.WriteString(signal + "\n")
		if err != nil {
			fmt.Println("Error writing to FIFO:", err)
		}
	}
}
