package bluetooth

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"gpioblink.com/x/karaoke-demon-clean/application"
	"gpioblink.com/x/karaoke-demon-clean/interface/handler"
	"tinygo.org/x/bluetooth"
)

type BluetoothInterface struct {
	router        map[string]handler.HandlerFuncWithResponse
	adapter       *bluetooth.Adapter
	advertisement *bluetooth.Advertisement
	serviceUUID   bluetooth.UUID
	rxUUID        bluetooth.UUID
	txUUID        bluetooth.UUID

	musicService  application.MusicService
	receiveBuffer []byte      // Buffer to accumulate data
	bufferLock    sync.Mutex  // Mutex to protect the buffer
	doLock        chan string // Channel to control access to Do function
}

func NewBluetoothInterface(service application.MusicService) *BluetoothInterface {
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
		router: map[string]handler.HandlerFuncWithResponse{
			"REMOTE_SONG":  handler.ReserveSongResult,
			"RESERVATIONS": handler.ListReservations,
			"SLOTS":        handler.ListSlots,
		},
		adapter:       adapter,
		advertisement: adv,
		serviceUUID:   serviceUUID,
		rxUUID:        rxUUID,
		txUUID:        txUUID,
		musicService:  service,
		receiveBuffer: make([]byte, 0),      // Initialize receive buffer
		doLock:        make(chan string, 1), // Initialize channel with buffer size 1
	}
}

func (b *BluetoothInterface) Run() {
	var rxChar bluetooth.Characteristic
	var txChar bluetooth.Characteristic
	service := bluetooth.Service{
		UUID: b.serviceUUID,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &rxChar,
				UUID:   b.rxUUID,
				Flags:  bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					log.Printf("Received chunk: %s", value)

					b.bufferLock.Lock()
					b.receiveBuffer = append(b.receiveBuffer, value...) // Append received bytes to buffer
					b.bufferLock.Unlock()

					// Check if we've received a full command (e.g., ending with a newline)
					if strings.HasSuffix(string(b.receiveBuffer), "\n") {
						b.bufferLock.Lock()
						fullMessage := string(b.receiveBuffer)
						b.receiveBuffer = b.receiveBuffer[:0] // Clear buffer after processing
						b.bufferLock.Unlock()

						// Send fullMessage to the channel
						b.doLock <- strings.TrimSpace(fullMessage)
						b.receiveBuffer = make([]byte, 0) // Initialize receive buffer
					}
				},
			},
			{
				Handle: &txChar,
				UUID:   b.txUUID,
				Flags:  bluetooth.CharacteristicNotifyPermission | bluetooth.CharacteristicReadPermission,
			},
		},
	}
	must("add service", b.adapter.AddService(&service))

	// Start advertising
	must("start adv", b.advertisement.Start())

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
	cmd := strings.Split(line, " ")
	action := cmd[0]

	if handlerFunc, ok := b.router[action]; ok {
		handlerFunc(ctx, b.musicService, *handler.NewRequest(action, cmd[1:]))
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
