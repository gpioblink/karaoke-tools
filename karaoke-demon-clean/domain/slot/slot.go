package slot

import "errors"

var (
	ErrSlotIdEmpty    = errors.New("empty slot id")
	ErrSlotStateEmpty = errors.New("empty slot state")
	ErrorInvalidState = errors.New("invalid slot state")
)

type SlotNum int
type State string

const (
	Empty   = State("empty")
	Reading = State("reading")
	Locked  = State("locked")
)

type Slot struct {
	id    SlotNum
	state State
}

func (s *Slot) Id() int {
	return int(s.id)
}

func (s *Slot) State() State {
	return s.state
}

func NewSlot(id int, state string) (*Slot, error) {
	if id == 0 {
		return nil, ErrSlotIdEmpty
	}
	if state == "" {
		return nil, ErrSlotStateEmpty
	}
	// stateが宣言されている定数のいずれかであることを確認
	switch State(state) {
	case Empty, Reading, Locked:
	default:
		return nil, ErrorInvalidState
	}
	return &Slot{id: SlotNum(id), state: State(state)}, nil
}
