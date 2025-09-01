package reservation

import (
	"errors"

	"gpioblink.com/x/karaoke-demon/domain/reservation"
	"gpioblink.com/x/karaoke-demon/domain/video"
)

var ErrVideoEmpty = errors.New("video is empty")

type MemoryRepository struct {
	reservations []reservation.Reservation
	currentSeq   int
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		reservations: []reservation.Reservation{},
		currentSeq:   0,
	}
}

func (m *MemoryRepository) EnQueue(video *video.Video) error {
	if video == nil {
		return ErrVideoEmpty
	}

	res, err := reservation.NewReservation(reservation.SeqNum(m.currentSeq), video)
	if err != nil {
		return err
	}
	m.reservations = append(m.reservations, *res)

	m.currentSeq++
	return nil
}

func (m *MemoryRepository) DeQueue() (*reservation.Reservation, error) {
	if len(m.reservations) == 0 {
		return nil, reservation.ErrNotFound
	}
	res := m.reservations[0]
	m.reservations = m.reservations[1:]
	return &res, nil
}

func (m *MemoryRepository) FindBySeq(seq int) (*reservation.Reservation, error) {
	for i := range m.reservations {
		if int(m.reservations[i].Seq()) == seq {
			return &m.reservations[i], nil
		}
	}
	return nil, reservation.ErrNotFound
}

func (m *MemoryRepository) FindByQueueIndex(index int) (*reservation.Reservation, error) {
	if index < 0 || index >= len(m.reservations) {
		return nil, reservation.ErrNotFound
	}
	return &m.reservations[index], nil
}

func (m *MemoryRepository) RemoveBySeq(seq int) error {
	for i := range m.reservations {
		if int(m.reservations[i].Seq()) == seq {
			m.reservations = append(m.reservations[:i], m.reservations[i+1:]...)
			return nil
		}
	}
	return reservation.ErrNotFound
}

func (m *MemoryRepository) List() ([]*reservation.Reservation, error) {
	reservations := make([]*reservation.Reservation, len(m.reservations))
	for i := range m.reservations {
		reservations[i] = &m.reservations[i]
	}
	return reservations, nil
}
