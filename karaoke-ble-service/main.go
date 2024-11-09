// peripheral.go
package main

import (
	"fmt"
	"time"

	"tinygo.org/x/bluetooth"
)

var adapter = bluetooth.DefaultAdapter

func main() {
	// Enable BLE interface.
	must("enable BLE stack", adapter.Enable())

	// Define the peripheral device info.
	adv := adapter.DefaultAdvertisement()
	must("config adv", adv.Configure(bluetooth.AdvertisementOptions{
		LocalName: "Go Bluetooth for Karaoke USB",
	}))

	// サービスとキャラクタリスティックのUUIDを定義

	// 8371c46d-9796-4a80-9411-cc738dcdb5ee
	serviceUUID := bluetooth.NewUUID([16]byte{0x83, 0x71, 0xc4, 0x6d, 0x97, 0x96, 0x4a, 0x80, 0x94, 0x11, 0xcc, 0x73, 0x8d, 0xcd, 0xb5, 0xee})
	// d29c26db-e005-4345-ac6b-bb13c363b6b8
	characteristicUUID := bluetooth.NewUUID([16]byte{0xd2, 0x9c, 0x26, 0xdb, 0xe0, 0x05, 0x43, 0x45, 0xac, 0x6b, 0xbb, 0x13, 0xc3, 0x63, 0xb6, 0xb8})

	// サービスを追加
	var karaokeCharacteristic bluetooth.Characteristic
	service := bluetooth.Service{
		UUID: serviceUUID,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &karaokeCharacteristic,
				UUID:   characteristicUUID,
				Value:  []byte("Hello, Bluetooth for Karaoke USB!"),
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					// if offset != 0 || len(value) != 4 {
					// 	return
					// }
					fmt.Printf("Received(%d): %s\n", offset, string(value))
				},
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
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}
