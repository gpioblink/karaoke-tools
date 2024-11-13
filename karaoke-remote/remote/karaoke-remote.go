package remote

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
)

const (
	DevicePath           = "/dev/lirc1"
	LIRC_MODE2_SPACE     = uint32(0x00000000)
	LIRC_MODE2_PULSE     = uint32(0x01000000)
	LIRC_MODE2_FREQUENCY = uint32(0x02000000)
	LIRC_MODE2_TIMEOUT   = uint32(0x03000000)
	LIRC_MODE2_OVERFLOW  = uint32(0x04000000)

	NEC_T = 562
	EX    = 150

	DAM_CUSTOMER_CODE      = 0xd1
	DAM_CUSTOMER_CODE_INV  = 0x2d
	DAM_START_SENDING_SONG = 0x08
	DAM_STOP_SENDING_SONG  = 0x09
	DAM_NUM                = 0x30
	DAM_DASH               = 0x3c
)

type Frame struct {
	customerCode    uint32
	customerCodeInv uint32
	data            uint32
	dataInv         uint32
}

func FindFrameLeader(file *os.File) error {
	// Frameが来るまで待機
	for {
		var scancode uint32
		// FrameのLeaderのPulseが来るまで待機
		if err := binary.Read(file, binary.LittleEndian, &scancode); err != nil {
			log.Fatalf("Failed to read from LIRC device: %v", err)
		}

		mode := scancode & 0xff000000
		value := scancode & 0x00ffffff
		// fmt.Printf("mode: 0x%x, value: %d\n", mode, value)

		if mode != LIRC_MODE2_PULSE && !((NEC_T*16-EX) < value && value < (NEC_T*16+EX)) {
			continue
		}

		// 続けてSpaceも適切な時間来るかチェック
		if err := binary.Read(file, binary.LittleEndian, &scancode); err != nil {
			log.Fatalf("Failed to read from LIRC device: %v", err)
		}

		mode = scancode & 0xff000000
		value = scancode & 0x00ffffff
		// fmt.Printf("mode: 0x%x, value: %d\n", mode, value)

		if mode != LIRC_MODE2_SPACE && !((NEC_T*8-EX) < value && value < (NEC_T*8+EX)) {
			continue
		}

		break
	}

	return nil
}

func ReceiveFrame(file *os.File) (*Frame, error) {
	err := FindFrameLeader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to find frame leader")
	}

	// データビットをまとめて受信し、データを復号する
	var decoded uint32

	// var dataBitRaw = make([]uint32, 66)
	// if err := binary.Read(file, binary.LittleEndian, &dataBitRaw); err != nil {
	// 	log.Fatalf("Failed to read from LIRC device: %v", err)
	// }
	var dataBitRaw = make([]uint32, 2)
	for i := 0; i < 32; i++ {
		if err := binary.Read(file, binary.LittleEndian, &dataBitRaw); err != nil {
			log.Fatalf("Failed to read from LIRC device: %v", err)
		}
		mode0 := dataBitRaw[0] & 0xff000000
		value0 := dataBitRaw[0] & 0x00ffffff
		mode1 := dataBitRaw[1] & 0xff000000
		value1 := dataBitRaw[1] & 0x00ffffff

		if mode0 != LIRC_MODE2_PULSE && !((NEC_T-EX) < value0 && value0 < (NEC_T+EX)) {
			return nil, fmt.Errorf("invalid data bit")
		}

		if mode1 == LIRC_MODE2_SPACE {
			if (NEC_T-EX) < value1 && value1 < (NEC_T+EX) {
				decoded |= 0 << uint(i)
			} else if (NEC_T*3-EX) < value1 && value1 < (NEC_T*3+EX) {
				decoded |= 1 << uint(i)
			} else {
				return nil, fmt.Errorf("invalid data bit")
			}
		} else {
			return nil, fmt.Errorf("invalid data bit")
		}
	}

	// final 2bit (trash)
	if err := binary.Read(file, binary.LittleEndian, &dataBitRaw); err != nil {
		log.Fatalf("Failed to read from LIRC device: %v", err)
	}

	return &Frame{
		customerCode:    decoded & 0xff,
		customerCodeInv: (decoded >> 8) & 0xff,
		data:            (decoded >> 16) & 0xff,
		dataInv:         (decoded >> 24) & 0xff,
	}, nil
}

func ReceiveIRSignals(signalCh chan<- string) {
	// デバイスファイルを開く
	file, err := os.OpenFile(DevicePath, os.O_RDWR, 0)
	if err != nil {
		log.Fatalf("Failed to open LIRC device: %v", err)
	}
	defer file.Close()

	var isSendingSong bool
	var songNo string

	// 受信ループ
	for {
		frame, err := ReceiveFrame(file)

		if err != nil {
			log.Printf("Failed to receive IR signal: %v", err)
			continue
		}

		// fmt.Printf("Received IR signal - CustomerCode: 0x%x, Data: 0x%x\n", frame.customerCode, frame.data)

		if frame.customerCode == 0xd1 && frame.customerCodeInv == 0x2d {
			switch frame.data {
			case DAM_START_SENDING_SONG:
				isSendingSong = true
				songNo = ""
			case DAM_STOP_SENDING_SONG:
				isSendingSong = false
				signalCh <- fmt.Sprintf("REMOTE_SONG %s", songNo)
			default:
				if isSendingSong {
					if DAM_NUM <= frame.data && frame.data <= DAM_NUM+9 {
						songNo += fmt.Sprintf("%d", frame.data-DAM_NUM)
					}
				}
			}
		}
	}
}
