package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"
)

const ContinuousBytesThreshold = (512 * 32) * 64 // 1MiB

func watchKmsg(messages chan<- string, kmsgPath string) {
	// FIXME: sleepでpollingしない

	totalContinuousLength := 0
	firstContinuousAddress := 0
	prevFinalContinuousAddress := 0

	re := regexp.MustCompile(`lun0: file read (\d+) @ (\d+)`)

	for {
		file, err := os.Open(kmsgPath)
		if err != nil {
			fmt.Printf("Failed to open kmsg: %v\n", err)
			time.Sleep(time.Second) // エラーが発生した場合は1秒待機して再試行
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			matches := re.FindStringSubmatch(line)
			if len(matches) == 3 {
				length, err := strconv.Atoi(matches[1])
				if err != nil {
					fmt.Printf("Failed to convert address to int: %v\n", err)
					continue
				}
				address, err := strconv.Atoi(matches[2])
				if err != nil {
					fmt.Printf("Failed to convert address to int: %v\n", err)
					continue
				}

				// 直前のアドレスから連続している、かつ、連続しているバイト数が一定以上の場合は、メッセージを送信
				if address == prevFinalContinuousAddress {
					totalContinuousLength += length
					prevFinalContinuousAddress = address + length
				} else {
					totalContinuousLength = length
					firstContinuousAddress = address
					prevFinalContinuousAddress = address + length
				}

				if totalContinuousLength >= ContinuousBytesThreshold {
					messages <- fmt.Sprintf("USBMSG_READ %d %d", firstContinuousAddress, totalContinuousLength)
					totalContinuousLength = 0
					firstContinuousAddress = 0
					prevFinalContinuousAddress = 0
				}
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading from /dev/kmsg: %v\n", err)
		}

		file.Close()

		time.Sleep(time.Second)
	}
}
