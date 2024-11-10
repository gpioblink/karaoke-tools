package bluetooth

import (
	"context"
	"log"
	"strings"
	"time"

	"gpioblink.com/x/karaoke-demon-clean/application"
	"gpioblink.com/x/karaoke-demon-clean/interface/handler"
	"tinygo.org/x/bluetooth"
)

type BluetoothInterface struct {
	router        map[string]handler.HandlerFunc
	adapter       *bluetooth.Adapter
	advertisement *bluetooth.Advertisement
	serviceUUID   bluetooth.UUID
	rxUUID        bluetooth.UUID
	txUUID        bluetooth.UUID

	musicService application.MusicService
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
		router: map[string]handler.HandlerFunc{
			"REMOTE_SONG": handler.ReserveSong,
		},
		adapter:       adapter,
		advertisement: adv,
		serviceUUID:   serviceUUID,
		rxUUID:        rxUUID,
		txUUID:        txUUID,
		musicService:  service,
	}
}

func (b *BluetoothInterface) Run() {

	// Define the peripheral device service.
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
					log.Printf("Received: %s", value)

					ctx := context.Background()
					go b.Do(ctx, string(value))

					var line []byte
					line = []byte("Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a type specimen book. It has survived not only five centuries, but also the leap into electronic typesetting, remaining essentially unchanged. It was popularised in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, and more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum.")
					for {
						sendbuf := line // copy buffer
						// Reset the slice while keeping the buffer in place.
						line = line[:0]

						// Send the sendbuf after breaking it up in pieces.
						for len(sendbuf) != 0 {
							// Chop off up to 20 bytes from the sendbuf.
							partlen := 20
							if len(sendbuf) < 20 {
								partlen = len(sendbuf)
							}
							part := sendbuf[:partlen]
							sendbuf = sendbuf[partlen:]

							// This also sends a notification.
							_, err := txChar.Write(part)
							must("send notification", err)
							// time.Sleep(time.Second)
						}
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
		// Sleep forever.
		time.Sleep(time.Hour)
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
