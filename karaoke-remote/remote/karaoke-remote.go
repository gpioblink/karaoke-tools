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
	LIRC_GET_FEATURES  = 0x00000000
	LIRC_GET_SEND_MODE = 0x00000001
	LIRC_GET_REC_MODE  = 0x00000002

	LIRC_SET_REC_MODE = 0x00000012

	LIRC_MODE_SCANCODE = 0x00000008
)

type LircScancode struct {
	Timestamp uint64
	Scancode  uint32
	Flags     uint32
}

func ReceiveIRSignals(signalCh chan<- string) {
	// デバイスファイルを開く
	file, err := os.OpenFile(DevicePath, os.O_RDWR, 0)
	if err != nil {
		log.Fatalf("Failed to open LIRC device: %v", err)
	}
	defer file.Close()

	// ioctlで受信モードをLIRC_MODE_SCANCODEに設定
	if err := unix.IoctlSetPointerInt(int(file.Fd()), LIRC_SET_REC_MODE, LIRC_MODE_SCANCODE); err != nil {
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
