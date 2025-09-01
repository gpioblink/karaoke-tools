package tool

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"gpioblink.com/x/karaoke-demon/interface/ir"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ir_test.go <gpio_pin>")
		fmt.Println("Example: go run ir_test.go 18")
		os.Exit(1)
	}

	gpioPin, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid GPIO pin: %v", err)
	}

	// 赤外線送信機を初期化
	transmitter, err := ir.NewIRTransmitter(gpioPin)
	if err != nil {
		log.Fatalf("Failed to initialize IR transmitter: %v", err)
	}
	defer transmitter.Close()

	fmt.Println("IR Test Program")
	fmt.Println("Commands:")
	fmt.Println("  command <hex>     - Send NEC command (e.g., command 02)")
	fmt.Println("  data <addr> <cmd> - Send NEC data with address and command")
	fmt.Println("  power             - Send power command")
	fmt.Println("  volume_up         - Send volume up command")
	fmt.Println("  volume_down       - Send volume down command")
	fmt.Println("  ch_up             - Send channel up command")
	fmt.Println("  ch_down           - Send channel down command")
	fmt.Println("  quit              - Exit program")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("IR> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}

		command := parts[0]
		switch command {
		case "quit", "exit":
			fmt.Println("Goodbye!")
			return

		case "command":
			if len(parts) != 2 {
				fmt.Println("Usage: command <hex>")
				continue
			}
			err := transmitter.SendCommandString(parts[1])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Sent command: %s\n", parts[1])
			}

		case "data":
			if len(parts) != 3 {
				fmt.Println("Usage: data <addr> <cmd>")
				continue
			}
			err := transmitter.SendDataString(parts[1], parts[2])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Sent data - address: %s, command: %s\n", parts[1], parts[2])
			}

		case "power":
			err := transmitter.SendCommand(ir.NEC_POWER)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("Sent power command")
			}

		case "volume_up":
			err := transmitter.SendCommand(ir.NEC_VOLUME_UP)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("Sent volume up command")
			}

		case "volume_down":
			err := transmitter.SendCommand(ir.NEC_VOLUME_DN)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("Sent volume down command")
			}

		case "ch_up":
			err := transmitter.SendCommand(ir.NEC_CH_UP)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("Sent channel up command")
			}

		case "ch_down":
			err := transmitter.SendCommand(ir.NEC_CH_DN)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("Sent channel down command")
			}

		default:
			fmt.Printf("Unknown command: %s\n", command)
		}
	}
}
