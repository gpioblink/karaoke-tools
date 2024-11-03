package models

import (
	"fmt"
	"path/filepath"
)

/* FATファイルシステム内のスロット管理 */
type SlotState int

const (
	_             SlotState = iota
	SLOT_FREE               // 書き込み可能
	SLOT_OCCUPIED           // 曲が入っている
	SLOT_LOCKED             // 曲が再生中
)

type SlotNode struct {
	state SlotState
	song  SlotSong
}

type SlotSong struct {
	requestNo string
	videoPath string
	seq       int
}

type Slot struct {
	l []SlotNode
}

func NewSlot(n int) Slot {
	l := make([]SlotNode, n)
	for i := range l {
		l[i].state = SLOT_FREE
	}
	return Slot{
		l: l,
	}
}

func NewSlotSong(requestNo, videoPath string, seq int) SlotSong {
	return SlotSong{
		requestNo: requestNo,
		videoPath: videoPath,
		seq:       seq,
	}
}

func (s *Slot) UpdateSlotState(slotNo int, state SlotState) {
	s.l[slotNo].state = state
}

func (s *Slot) UpdateSlotSong(slotNo int, song SlotSong) {
	s.l[slotNo].song = song
}

func (s *Slot) GetSlotSong(slotNo int) SlotSong {
	return s.l[slotNo].song
}

func (s *Slot) GetSlotState(slotNo int) SlotState {
	return s.l[slotNo].state
}

func (s *Slot) FindNextFreeSlot() (int, error) {
	// SLOT_LOCKEDの次のインデックスからSLOT_FREEを探す。ただし、最後まで見つからない場合は先頭から探す
	if idx, err := s.FindLockedSlot(); err == nil {
		for i := 0; i < len(s.l); i++ {
			id := (idx + i) % len(s.l)
			if s.l[id].state == SLOT_FREE {
				return id, nil
			}
		}
	} else {
		for i := 0; i < len(s.l); i++ {
			if s.l[i].state == SLOT_FREE {
				return i, nil
			}
		}
	}

	return -1, fmt.Errorf("no free slot")
}

func (s *Slot) FindLockedSlot() (int, error) {
	for i, slot := range s.l {
		if slot.state == SLOT_LOCKED {
			return i, nil
		}
	}
	return -1, fmt.Errorf("no locked slot")
}

func (s *Slot) IsAllSlotsFree() bool {
	for _, slot := range s.l {
		if slot.state != SLOT_FREE {
			return false
		}
	}
	return true
}

func (ss *SlotSong) GetVideoPath() string {
	return ss.videoPath
}

func (ss *SlotSong) GetRequestNo() string {
	return ss.requestNo
}

func (ss *SlotSong) GetSeq() int {
	return ss.seq
}

func (s *Slot) String() string {
	str := "\n"
	for i, slot := range s.l {
		if slot.song.requestNo == "" {
			str += fmt.Sprintf("%d: EMPTY \n", i)
		} else {
			str += fmt.Sprintf("%d: %s(%d) %d %s \n", i, slot.song.requestNo, slot.song.seq, slot.state, filepath.Base(slot.song.videoPath))
		}
	}
	return str
}
