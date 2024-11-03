package models

import "fmt"

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
	return Slot{
		l: make([]SlotNode, n),
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
	for i, slot := range s.l {
		if slot.state == SLOT_FREE {
			return i, nil
		}
	}
	return -1, fmt.Errorf("no free slot")
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
