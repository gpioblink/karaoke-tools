// peripheral.go
package main

import (
	"log"
	"time"

	"tinygo.org/x/bluetooth"
)

var adapter = bluetooth.DefaultAdapter

func main() {
	// サービスとキャラクタリスティックのUUIDを定義

	// 8371c46d-9796-4a80-9411-cc738dcdb5ee
	serviceUUID := bluetooth.NewUUID([16]byte{0x83, 0x71, 0xc4, 0x6d, 0x97, 0x96, 0x4a, 0x80, 0x94, 0x11, 0xcc, 0x73, 0x8d, 0xcd, 0xb5, 0xee})
	// d29c26db-e005-4345-ac6b-bb13c363b6b8
	// characteristicUUID := bluetooth.NewUUID([16]byte{0xd2, 0x9c, 0x26, 0xdb, 0xe0, 0x05, 0x43, 0x45, 0xac, 0x6b, 0xbb, 0x13, 0xc3, 0x63, 0xb6, 0xb8})
	rxUUID := bluetooth.CharacteristicUUIDUARTRX
	txUUID := bluetooth.CharacteristicUUIDUARTTX

	// Enable BLE interface.
	must("enable BLE stack", adapter.Enable())

	// Define the peripheral device info.
	adv := adapter.DefaultAdvertisement()
	must("config adv", adv.Configure(bluetooth.AdvertisementOptions{
		LocalName:    "Go Bluetooth for Karaoke USB",
		ServiceUUIDs: []bluetooth.UUID{serviceUUID},
	}))

	// サービスを追加
	var rxChar bluetooth.Characteristic
	var txChar bluetooth.Characteristic
	service := bluetooth.Service{
		UUID: serviceUUID,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &rxChar,
				UUID:   rxUUID,
				Flags:  bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					txChar.Write(value)
					for _, c := range value {
						log.Printf("Received: %c", c)
					}

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
							//time.Sleep(time.Second)
							must("send notification", err)
						}
					}
				},
			},
			{
				Handle: &txChar,
				UUID:   txUUID,
				Flags:  bluetooth.CharacteristicNotifyPermission | bluetooth.CharacteristicReadPermission,
			},
		},
	}
	must("add service", adapter.AddService(&service))

	// Start advertising
	must("start adv", adv.Start())

	println("advertising...")

	for {
		// Sleep forever.
		time.Sleep(time.Hour)
	}
}

func must(action string, err error) {
	log.Printf("action: %s", action)
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}
