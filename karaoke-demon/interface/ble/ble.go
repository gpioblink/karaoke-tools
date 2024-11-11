package ble

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"gpioblink.com/x/karaoke-demon/application"
	"gpioblink.com/x/karaoke-demon/interface/handler"
	"tinygo.org/x/bluetooth"
)

type BluetoothInterface struct {
	router        map[string]handler.HandlerFuncWithResponse
	adapter       *bluetooth.Adapter
	advertisement *bluetooth.Advertisement
	serviceUUID   bluetooth.UUID
	rxUUID        bluetooth.UUID
	txUUID        bluetooth.UUID
	txChar        *bluetooth.Characteristic
	rxChar        *bluetooth.Characteristic

	musicService  application.MusicService
	receiveBuffer []byte      // Buffer to accumulate data
	bufferLock    sync.Mutex  // Mutex to protect the buffer
	doLock        chan string // Channel to control access to Do function
}

var DefaultRouter = map[string]handler.HandlerFuncWithResponse{
	"REMOTE_SONG":  handler.ReserveSongResult,
	"RESERVATIONS": handler.ListReservations,
	"SLOTS":        handler.ListSlots,
}

func NewBluetoothInterface(service *application.MusicService, router map[string]handler.HandlerFuncWithResponse) *BluetoothInterface {
	var rxChar bluetooth.Characteristic
	var txChar bluetooth.Characteristic
	adapter := bluetooth.DefaultAdapter
	serviceUUID := bluetooth.NewUUID([16]byte{0x83, 0x71, 0xc4, 0x6d, 0x97, 0x96, 0x4a, 0x80, 0x94, 0x11, 0xcc, 0x73, 0x8d, 0xcd, 0xb5, 0xee})
	rxUUID := bluetooth.CharacteristicUUIDUARTRX
	txUUID := bluetooth.CharacteristicUUIDUARTTX

	must("enable BLE stack", adapter.Enable())

	// Define the peripheral device info.
	adv := adapter.DefaultAdvertisement()
	must("config adv", adv.Configure(bluetooth.AdvertisementOptions{
		LocalName:    "Go Bluetooth for Karaoke USB",
		ServiceUUIDs: []bluetooth.UUID{serviceUUID},
	}))

	return &BluetoothInterface{
		txChar:        &txChar,
		rxChar:        &rxChar,
		router:        router,
		adapter:       adapter,
		advertisement: adv,
		serviceUUID:   serviceUUID,
		rxUUID:        rxUUID,
		txUUID:        txUUID,
		musicService:  *service,
		receiveBuffer: make([]byte, 0),      // Initialize receive buffer
		doLock:        make(chan string, 1), // Initialize channel with buffer size 1
	}
}

func (b *BluetoothInterface) Run() {
	service := bluetooth.Service{
		UUID: b.serviceUUID,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: b.rxChar,
				UUID:   b.rxUUID,
				Flags:  bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					log.Printf("Received BLE: %s", value)

					b.bufferLock.Lock()
					b.receiveBuffer = append(b.receiveBuffer, value...) // Append received bytes to buffer
					b.bufferLock.Unlock()

					// Check if we've received a full command (e.g., ending with a newline)
					if strings.HasSuffix(string(b.receiveBuffer), "\n") {
						log.Printf("Received full message: %s", b.receiveBuffer)
						b.bufferLock.Lock()
						fullMessage := string(b.receiveBuffer[:len(b.receiveBuffer)-1]) // Remove EOF from the end
						b.receiveBuffer = b.receiveBuffer[:0]                           // Clear buffer after processing
						b.bufferLock.Unlock()

						// Send fullMessage to the channel
						b.doLock <- strings.TrimSpace(fullMessage)
					}
				},
			},
			{
				Handle: b.txChar,
				UUID:   b.txUUID,
				Flags:  bluetooth.CharacteristicNotifyPermission | bluetooth.CharacteristicReadPermission,
			},
		},
	}
	must("add service", b.adapter.AddService(&service))

	// Start advertising
	//must("start adv", b.advertisement.Start())
	for err := b.advertisement.Start(); err != nil; err = b.advertisement.Start() {
		log.Printf("Failed to start advertising: %v", err)

		// run: `rfkill unblock all` using cmd
		if err := exec.Command("rfkill", "unblock", "all").Run(); err != nil {
			log.Printf("Failed to unblock rfkill: %v", err)
		}

		time.Sleep(5 * time.Second)
	}

	println("advertising...")

	for {
		select {
		case fullMessage := <-b.doLock:
			ctx := context.Background()
			b.Do(ctx, fullMessage)
		case <-time.After(time.Hour):
			// Do nothing, just keep the loop running
		}
	}
}

func (b *BluetoothInterface) Do(ctx context.Context, line string) {
	log.Printf("Do BLE: %s", line)

	cmd := strings.Split(line, " ")
	action := cmd[0]

	if handlerFunc, ok := b.router[action]; ok {
		res := []byte(handlerFunc(ctx, b.musicService, *handler.NewRequest(action, cmd[1:])))
		res = append(res, 0x0) // Add EOF to the end of the response

		// Send response to the client
		if len(res) > 0 {
			for len(res) > 0 {
				// Send data in chunks of 20 bytes
				chunkSize := 20
				if len(res) < chunkSize {
					chunkSize = len(res)
				}
				chunk := res[:chunkSize]
				res = res[chunkSize:]

				_, err := b.txChar.Write(chunk)
				must(fmt.Sprintf("send response %s", chunk), err)
			}
		}

	} else {
		log.Printf("Unknown command: %s", action)
	}
}

func must(action string, err error) {
	log.Printf("action: %s", action)
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}
