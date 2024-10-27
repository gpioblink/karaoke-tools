package main

import "fmt"

func main() {
    messages := make(chan string)
    go watchKmsg(messages)
    for msg := range messages {
        fmt.Println(msg)
    }
}

