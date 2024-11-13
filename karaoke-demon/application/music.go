package application

import (
	"gpioblink.com/x/karaoke-demon/domain/reservation"
	"gpioblink.com/x/karaoke-demon/domain/slot"
	"gpioblink.com/x/karaoke-demon/domain/song"
	"gpioblink.com/x/karaoke-demon/domain/video"
)

type MusicModel interface {
	AddSong(requestNo song.RequestNo) error
	RemoveSong(seq int) error
	ListReservations() ([]reservation.Reservation, error)
}

type MusicService struct {
	reservationRepo reservation.Repository
	slotRepo        slot.Repository
	videoRepo       video.Repository
}

// FIXME: Consider Transactions and Race Conditions

func NewMusicService(reservationRepo reservation.Repository, slotRepo slot.Repository, videoRepo video.Repository) *MusicService {
	return &MusicService{
		reservationRepo: reservationRepo,
		slotRepo:        slotRepo,
		videoRepo:       videoRepo,
	}
}

func (s *MusicService) ReserveSong(requestNo song.RequestNo) error {
	err := s.reservationRepo.EnQueue(string(requestNo))
	if err != nil {
		return err
	}
	err = s.AttachNextReservationToSlotIfAvailable()
	if err != nil {
		return err
	}
	return nil
}

func (s *MusicService) RemoveReservation(seq int) error {
	err := s.reservationRepo.RemoveBySeq(seq)
	if err != nil {
		return err
	}
	// Make the slot free that was reserved by the removed reservation

	return nil
}

func (s *MusicService) ListReservations() ([]*reservation.Reservation, error) {
	reservations, err := s.reservationRepo.List()
	if err != nil {
		return nil, err
	}
	return reservations, nil
}

func (s *MusicService) UpdateSlotStateReadingByReadingSlotId(id int) error {
	// Remove the reservation that is previous reading
	_, err := s.reservationRepo.DeQueue()
	if err != nil {
		return err
	}

	// Make previous slot state available because it is not reserved by any reservation now
	prevSlot, err := s.slotRepo.FindById(calcPositiveModulo(id-1, s.slotRepo.Len()))
	if err != nil {
		return err
	}

	err = s.slotRepo.SetSeqById(calcPositiveModulo(id-1, s.slotRepo.Len()), prevSlot.Seq()+3)
	if err != nil {
		return err
	}

	if prevSlot.Reservation() != nil && prevSlot.State() != slot.Waiting {
		err = s.slotRepo.DettachReservationById(calcPositiveModulo(id-1, s.slotRepo.Len()))
		if err != nil {
			return err
		}
	}

	err = s.slotRepo.SetStateById(calcPositiveModulo(id-1, s.slotRepo.Len()), slot.Available)
	if err != nil {
		return err
	}

	// Make current slot state reading
	err = s.slotRepo.SetStateById(id, slot.Reading)
	if err != nil {
		return err
	}

	// Make next slot state locked
	err = s.slotRepo.SetStateById(calcPositiveModulo(id+1, s.slotRepo.Len()), slot.Locked)
	if err != nil {
		return err
	}

	err = s.AttachNextReservationToSlotIfAvailable()
	if err != nil {
		return err
	}

	return nil
}

func (s *MusicService) AttachNextReservationToSlotIfAvailable() error {
	// Find Slot that state is available
	readingSlot, err := s.slotRepo.GetFirstSlotByState(slot.Reading)
	if err != nil {
		return err
	}

	for i := 1; i < s.slotRepo.Len(); i++ {
		// check if the slot is available
		availableSlot, err := s.slotRepo.FindById(calcPositiveModulo(readingSlot.Id()+i, s.slotRepo.Len()))
		if err != nil {
			return err
		}
		if availableSlot.State() != slot.Available {
			continue
		}

		// Attach next reservation to the slot
		nextReservation, err := s.reservationRepo.FindByQueueIndex(i)
		if err != nil {
			return nil
		}

		// Find Correct Video for the next reservation
		currentSong, err := nextReservation.Song()
		if err != nil {
			return err
		}
		video, err := s.videoRepo.FindByRequestNo(string((currentSong.RequestNo())))
		if err != nil {
			return err
		}

		// Attach next reservation to the slot
		err = s.slotRepo.AttachReservationById(availableSlot.Id(), nextReservation)
		if err != nil {
			return err
		}

		s.slotRepo.ChangeVideoById(availableSlot.Id(), video) // FIXME: Error Handling (現状、失敗してもよいのであえてエラーハンドリングはしていない)

		// Set the slot state to waiting
		err = s.slotRepo.SetStateById(availableSlot.Id(), slot.Waiting)
		if err != nil {
			return err
		}

		// Writing Video Functionality is in Slot Repository, so it is not implemented here
	}

	return nil
}

func (s *MusicService) ListSlots() ([]*slot.Slot, error) {
	slots, err := s.slotRepo.List()
	if err != nil {
		return nil, err
	}
	return slots, nil
}

func calcPositiveModulo(a, b int) int {
	return (a%b + b) % b
}
