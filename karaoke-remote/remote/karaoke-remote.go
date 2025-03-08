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

	AEHA_T = 425
	EX     = 150

	PANA_NUM = 0xc0 // 地上D 1ch
)

type Frame struct {
	fixedCode uint32
	data1     uint8
	data2     uint8
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

		if mode != LIRC_MODE2_PULSE && !((AEHA_T*8-EX) < value && value < (AEHA_T*8+EX)) {
			continue
		}

		// 続けてSpaceも適切な時間来るかチェック
		if err := binary.Read(file, binary.LittleEndian, &scancode); err != nil {
			log.Fatalf("Failed to read from LIRC device: %v", err)
		}

		mode = scancode & 0xff000000
		value = scancode & 0x00ffffff
		// fmt.Printf("mode: 0x%x, value: %d\n", mode, value)

		if mode != LIRC_MODE2_SPACE && !((AEHA_T*4-EX) < value && value < (_T*4+EX)) {
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
	var decoded_fixed uint32
	var decoded_remote uint16

	var dataBitRaw = make([]uint32, 2)
	for i := 0; i < 48; i++ {
		if err := binary.Read(file, binary.LittleEndian, &dataBitRaw); err != nil {
			log.Fatalf("Failed to read from LIRC device: %v", err)
		}
		mode0 := dataBitRaw[0] & 0xff000000
		value0 := dataBitRaw[0] & 0x00ffffff
		mode1 := dataBitRaw[1] & 0xff000000
		value1 := dataBitRaw[1] & 0x00ffffff

		if mode0 != LIRC_MODE2_PULSE && !((AEHA_T-EX) < value0 && value0 < (AEHA_T+EX)) {
			return nil, fmt.Errorf("invalid data bit")
		}

		if mode1 == LIRC_MODE2_SPACE {
			if (AEHA_T-EX) < value1 && value1 < (AEHA_T+EX) {
				if i < 32 {
					decoded_fixed |= 0 << uint(i)
				} else {
					decoded_remote |= 0 << uint(i-32)
				}
			} else if (AEHA_T*3-EX) < value1 && value1 < (AEHA_T*3+EX) {
				if i < 32 {
					decoded_fixed |= 1 << uint(i)
				} else {
					decoded_remote |= 1 << uint(i-32)
				}
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
		fixedCode: decoded_fixed,
		data1:     uint8(decoded_remote & 0xff),
		data2:     uint8((decoded_remote >> 8) & 0xff),
	}, nil
}

func ReceiveIRSignals(signalCh chan<- string) {
	// デバイスファイルを開く
	file, err := os.OpenFile(DevicePath, os.O_RDWR, 0)
	if err != nil {
		log.Fatalf("Failed to open LIRC device: %v", err)
	}
	defer file.Close()

	var songNo string

	// 受信ループ
	for {
		frame, err := ReceiveFrame(file)

		if err != nil {
			log.Printf("Failed to receive IR signal: %v", err)
			continue
		}

		// fmt.Printf("Received IR signal - CustomerCode: 0x%x, Data: 0x%x\n", frame.customerCode, frame.data)

		if frame.fixedCode == 0x0220800f {
			if PANA_NUM <= frame.data1 && frame.data1 <= PANA_NUM+9 {
				songNo = fmt.Sprintf("%d", frame.data1-PANA_NUM)
				signalCh <- fmt.Sprintf("REMOTE_SONG %s", songNo)
			}
		}
	}
}
