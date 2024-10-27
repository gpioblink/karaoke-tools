package main

import (
    "bufio"
    "fmt"
    "os"
    "regexp"
    "time"
)

func watchKmsg(messages chan<- string) {
    re := regexp.MustCompile(`lun0: file read (\d+) @ (\d+)`)

    for {
        file, err := os.Open("/dev/kmsg")
        if err != nil {
            fmt.Printf("Failed to open /dev/kmsg: %v\n", err)
            time.Sleep(time.Second) // エラーが発生した場合は1秒待機して再試行
            continue
        }

        scanner := bufio.NewScanner(file)
        for scanner.Scan() {
            line := scanner.Text()
            matches := re.FindStringSubmatch(line)
            if len(matches) == 3 {
                length := matches[1]
                address := matches[2]
                messages <- fmt.Sprintf("READ %s %s", address, length)
            }
        }

        if err := scanner.Err(); err != nil {
            fmt.Printf("Error reading from /dev/kmsg: %v\n", err)
        }

        file.Close()

        time.Sleep(time.Second)
    }
}

