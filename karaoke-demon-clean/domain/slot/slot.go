package slot

import "errors"

var (
	ErrSlotIdEmpty    = errors.New("empty slot id")
	ErrSlotStateEmpty = errors.New("empty slot state")
	ErrorInvalidState = errors.New("invalid slot state")
)

type State string

const (
	Empty   = State("empty")
	Reading = State("reading")
	Locked  = State("locked")
)

type Slot struct {
	id    int
	state string
}

func (s *Slot) Id() int {
	return s.id
}

func (s *Slot) State() string {
	return s.state
}

func NewSlot(id int, state State) (*Slot, error) {
	if id == 0 {
		return nil, ErrSlotIdEmpty
	}
	if state == "" {
		return nil, ErrSlotStateEmpty
	}
	// stateが宣言されている定数のいずれかであることを確認
	switch state {
	case Empty, Reading, Locked:
	default:
		return nil, ErrorInvalidState
	}
	return &Slot{id: id, state: string(state)}, nil
}
