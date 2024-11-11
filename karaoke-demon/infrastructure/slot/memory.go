// this is mock implementation of slot repository. Data is not really stored.
// TODO: process slot with makemyfat

package slot

import (
	"gpioblink.com/x/karaoke-demon/domain/reservation"
	"gpioblink.com/x/karaoke-demon/domain/slot"
	"gpioblink.com/x/karaoke-demon/domain/video"
)

type MemoryRepository struct {
	slots map[int]slot.Slot
}

func NewMemoryRepository() *MemoryRepository {

	// make 3 slot
	slots := make(map[int]slot.Slot)
	for i := 0; i < 3; i++ {
		slots[i] = *slot.NewEmptySlot(i)
	}

	return &MemoryRepository{
		slots: slots,
	}
}

func (m *MemoryRepository) Len() int {
	return len(m.slots)
}

func (m *MemoryRepository) AttachReservationAndVideoById(slotId int, reservation *reservation.Reservation, video *video.Video) error {
	s, ok := m.slots[slotId]
	if !ok {
		return slot.ErrNotFound
	}

	// check not locked and not writing
	if s.State() == slot.Locked {
		return slot.ErrSlotLocked
	}

	if s.IsWriting() {
		return slot.ErrSlotBusy
	}

	newSlot, err := slot.NewSlot(s.Id(), s.State(), reservation, video, false)
	if err != nil {
		return err
	}

	m.slots[slotId] = *newSlot
	return nil
}

func (m *MemoryRepository) ChangeVideoById(slotId int, video *video.Video) error {
	s, ok := m.slots[slotId]
	if !ok {
		return slot.ErrNotFound
	}

	newSlot, err := slot.NewSlot(s.Id(), s.State(), s.Reservation(), video, s.IsWriting())
	if err != nil {
		return err
	}

	m.slots[slotId] = *newSlot
	return nil
}

func (m *MemoryRepository) DettachReservationAndVideoById(slotId int) error {
	s, ok := m.slots[slotId]
	if !ok {
		return slot.ErrNotFound
	}

	newSlot, err := slot.NewSlot(s.Id(), s.State(), nil, nil, false)
	if err != nil {
		return err
	}

	m.slots[slotId] = *newSlot
	return nil
}

func (m *MemoryRepository) SetStateById(slotId int, state slot.State) error {
	s, ok := m.slots[slotId]
	if !ok {
		return slot.ErrNotFound
	}

	newSlot, err := slot.NewSlot(s.Id(), state, s.Reservation(), s.Video(), s.IsWriting())
	if err != nil {
		return err
	}

	m.slots[slotId] = *newSlot
	return nil
}

func (m *MemoryRepository) SetWritingFlagById(slotId int, isWriting bool) error {
	s, ok := m.slots[slotId]
	if !ok {
		return slot.ErrNotFound
	}

	newSlot, err := slot.NewSlot(s.Id(), s.State(), s.Reservation(), s.Video(), isWriting)
	if err != nil {
		return err
	}

	m.slots[slotId] = *newSlot
	return nil
}

func (m *MemoryRepository) FindById(id int) (*slot.Slot, error) {
	s, ok := m.slots[id]
	if !ok {
		return nil, slot.ErrNotFound
	}
	return &s, nil
}

func (m *MemoryRepository) List() ([]*slot.Slot, error) {
	slots := make([]*slot.Slot, len(m.slots))
	for i, s := range m.slots {
		slots[i] = &s
	}
	return slots, nil
}
