package slot

import (
	"errors"

	"gpioblink.com/x/karaoke-demon/domain/reservation"
	"gpioblink.com/x/karaoke-demon/domain/video"
)

var (
	ErrSlotIdEmpty    = errors.New("empty slot id")
	ErrSlotStateEmpty = errors.New("empty slot state")
	ErrInvalidState   = errors.New("invalid slot state")
	ErrSlotBusy       = errors.New("slot is busy")
	ErrSlotLocked     = errors.New("slot is locked")
)

type SlotNum int
type State string

const (
	Available = State("available")
	Waiting   = State("waiting")
	Reading   = State("reading")
	Locked    = State("locked")
)

type Slot struct {
	id          SlotNum
	seq         int
	state       State
	reservation *reservation.Reservation
	video       *video.Video
	isWriting   bool
}

func (s *Slot) Id() int {
	return int(s.id)
}

func (s *Slot) Seq() int {
	return s.seq
}

func (s *Slot) State() State {
	return s.state
}

func (s *Slot) Reservation() *reservation.Reservation {
	return s.reservation
}

func (s *Slot) Video() *video.Video {
	return s.video
}

func (s *Slot) IsWriting() bool {
	return s.isWriting
}

func NewEmptySlot(id int, seq int) *Slot {
	return &Slot{id: SlotNum(id), seq: seq, state: Available, reservation: nil, video: nil, isWriting: false}
}

func NewSlot(id int, seq int, state State, reservation *reservation.Reservation, video *video.Video, isWriting bool) (*Slot, error) {
	if id < 0 {
		return nil, ErrSlotIdEmpty
	}
	if state == "" {
		return nil, ErrSlotStateEmpty
	}
	// stateが宣言されている定数のいずれかであることを確認
	switch State(state) {
	case Available, Waiting, Reading, Locked:
	default:
		return nil, ErrInvalidState
	}
	return &Slot{id: SlotNum(id), seq: seq, state: State(state), reservation: reservation, video: video, isWriting: isWriting}, nil
}
