// this is mock implementation of slot repository. Data is not really stored.
// TODO: process slot with makemyfat

package slot

import (
	"fmt"
	"log"

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
		slots[i] = *slot.NewEmptySlot(i, i)
	}

	repo := &MemoryRepository{
		slots: slots,
	}

	// repo.SetStateById(0, slot.Available)
	// repo.SetStateById(1, slot.Reading)
	// repo.SetStateById(2, slot.Locked)

	return repo
}

func (m *MemoryRepository) Len() int {
	return len(m.slots)
}

func (m *MemoryRepository) AttachReservationById(slotId int, reservation *reservation.Reservation) error {
	s, ok := m.slots[slotId]
	if !ok {
		return slot.ErrNotFound
	}

	// LockやBusyであっても、Reservationはカラオケ本体の予約リストと一致されるべきなので、通す
	// これらの情報は、あくまでビデオを書き込んで良いかの判定のみに利用する
	// if s.State() == slot.Locked {
	// 	return slot.ErrSlotLocked
	// }

	// if s.IsWriting() {
	// 	return slot.ErrSlotBusy
	// }

	newSlot, err := slot.NewSlot(s.Id(), s.Seq(), s.State(), reservation, s.Video(), false)
	if err != nil {
		return err
	}

	m.slots[slotId] = *newSlot
	return nil
}

func (m *MemoryRepository) SetSeqById(slotId int, seq int) error {
	s, ok := m.slots[slotId]
	if !ok {
		return slot.ErrNotFound
	}

	newSlot, err := slot.NewSlot(s.Id(), seq, s.State(), s.Reservation(), s.Video(), s.IsWriting())
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

	if video == nil {
		return fmt.Errorf("video is nil")
	}

	if s.State() == slot.Locked {
		log.Fatalf("due to being reserved just before playback, slot %d video is not changed for %s\n", slotId, video.Location())
		return slot.ErrSlotLocked
	}

	if s.IsWriting() {
		log.Fatalf("slot %d video is writing by other call\n", slotId)
		return slot.ErrSlotBusy
	}

	newSlot, err := slot.NewSlot(s.Id(), s.Seq(), s.State(), s.Reservation(), video, s.IsWriting())
	if err != nil {
		return err
	}

	m.slots[slotId] = *newSlot
	return nil
}

func (m *MemoryRepository) GetFirstSlotByState(state slot.State) (*slot.Slot, error) {
	for _, s := range m.slots {
		if s.State() == state {
			return &s, nil
		}
	}
	return nil, slot.ErrNotFound
}

func (m *MemoryRepository) DettachReservationById(slotId int) error {
	s, ok := m.slots[slotId]
	if !ok {
		return slot.ErrNotFound
	}

	newSlot, err := slot.NewSlot(s.Id(), s.Seq(), s.State(), nil, s.Video(), s.IsWriting())
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

	newSlot, err := slot.NewSlot(s.Id(), s.Seq(), state, s.Reservation(), s.Video(), s.IsWriting())
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

	newSlot, err := slot.NewSlot(s.Id(), s.Seq(), s.State(), s.Reservation(), s.Video(), isWriting)
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
