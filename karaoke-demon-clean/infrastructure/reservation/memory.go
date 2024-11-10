package reservation

import (
	"gpioblink.com/x/karaoke-demon-clean/domain/reservation"
	"gpioblink.com/x/karaoke-demon-clean/domain/song"
)

type MemoryRepository struct {
	reservations   []reservation.Reservation
	songRepository song.Repository
	currentSeq     int
}

func NewMemoryRepository(songRepository song.Repository) *MemoryRepository {
	return &MemoryRepository{
		reservations:   []reservation.Reservation{},
		songRepository: songRepository,
		currentSeq:     0,
	}
}

func (m *MemoryRepository) EnQueue(requestNo string) error {
	songInfo, err := m.songRepository.FindByRequestNo(requestNo)
	if err != nil {
		return err
	}
	res, err := reservation.NewReservation(reservation.SeqNum(m.currentSeq), songInfo)
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
	for _, res := range m.reservations {
		if int(res.Seq()) == seq {
			return &res, nil
		}
	}
	return nil, reservation.ErrNotFound
}

func (m *MemoryRepository) RemoveBySeq(seq int) error {
	for i, res := range m.reservations {
		if int(res.Seq()) == seq {
			m.reservations = append(m.reservations[:i], m.reservations[i+1:]...)
			return nil
		}
	}
	return reservation.ErrNotFound
}

func (m *MemoryRepository) List() ([]*reservation.Reservation, error) {
	reservations := make([]*reservation.Reservation, len(m.reservations))
	for i, res := range m.reservations {
		reservations[i] = &res
	}
	return reservations, nil
}
