package remote

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"golang.org/x/sys/unix"
)

const (
	// LIRCのデバイスパス
	DevicePath = "/dev/lirc1"

	// NECフォーマットの赤外線コードを受信するためのモード
	LIRC_MODE_SCANCODE = 0x00000004
	LIRC_SET_REC_MODE  = 0x00000003
)

type LircScancode struct {
	Timestamp uint64
	Scancode  uint32
	Flags     uint32
}

func ReceiveIRSignals(signalCh chan<- string) {
	// lircSocket := "/var/run/lirc/lircd"

	// file, err := os.Open(lircSocket)
	// if err != nil {
	// 	log.Fatalf("cannnot open a LIRC socket: %v", err)
	// }
	// defer file.Close()

	// fmt.Println("waiting ir data...")

	// scanner := bufio.NewScanner(file)
	// for scanner.Scan() {
	// 	signalCh <- scanner.Text()
	// }

	// if err := scanner.Err(); err != nil {
	// 	log.Fatalf("LIRC read error: %v", err)
	// }

	// デバイスファイルを開く
	file, err := os.OpenFile(DevicePath, os.O_RDWR, 0)
	if err != nil {
		log.Fatalf("Failed to open LIRC device: %v", err)
	}
	defer file.Close()

	// ioctlで受信モードをLIRC_MODE_SCANCODEに設定
	if err := unix.IoctlSetInt(int(file.Fd()), LIRC_SET_REC_MODE, LIRC_MODE_SCANCODE); err != nil {
		log.Fatalf("Failed to set LIRC mode to SCANCODE: %v", err)
	}

	fmt.Println("LIRC mode set to LIRC_MODE_SCANCODE. Waiting for NEC codes...")

	// 受信ループ
	for {
		// LIRCからスキャンコードを受信する
		var scancode LircScancode
		if err := binary.Read(file, binary.LittleEndian, &scancode); err != nil {
			log.Fatalf("Failed to read from LIRC device: %v", err)
		}

		// 受信したスキャンコードを表示（NECフォーマットの赤外線コード）
		fmt.Printf("Received NEC code - Scancode: 0x%x, Flags: 0x%x\n", scancode.Scancode, scancode.Flags)
		signalCh <- fmt.Sprintf("Received NEC code - Scancode: 0x%x, Flags: 0x%x\n", scancode.Scancode, scancode.Flags)
	}
}
