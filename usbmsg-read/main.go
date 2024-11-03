package main

import (
	"fmt"
	"os"
)

func main() {
	messages := make(chan string)
	go watchKmsg(messages)

	fifo, err := os.OpenFile("/tmp/karaoke-fifo", os.O_WRONLY, 0600)
	if err != nil {
		fmt.Println("Error opening FIFO:", err)
		return
	}
	defer fifo.Close()

	for msg := range messages {
		fmt.Println(msg)

		if _, err := fifo.WriteString(msg + "\n"); err != nil {
			fmt.Println("Error writing to file:", err)
		}
	}
}
