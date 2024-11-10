package application

import (
	"slices"

	"gpioblink.com/x/karaoke-demon-clean/domain/reservation"
	"gpioblink.com/x/karaoke-demon-clean/domain/slot"
	"gpioblink.com/x/karaoke-demon-clean/domain/song"
	"gpioblink.com/x/karaoke-demon-clean/domain/video"
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
	// Make previous slot state available because it is not reserved by any reservation now
	prevSlot, err := s.slotRepo.FindById(calcPositiveModulo(id-1, s.slotRepo.Len()))
	if err != nil {
		return err
	}
	if prevSlot.State() != slot.Waiting {
		if prevSlot.Reservation() != nil {
			err := s.reservationRepo.RemoveBySeq(int(prevSlot.Reservation().Seq()))
			if err != nil {
				return err
			}
			err = s.slotRepo.DettachReservationAndVideoById(calcPositiveModulo(id-1, s.slotRepo.Len()))
			if err != nil {
				return err
			}
		}
		err = s.slotRepo.SetStateById(calcPositiveModulo(id-1, s.slotRepo.Len()), slot.Available)
		if err != nil {
			return err
		}
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
	slots, err := s.slotRepo.List()
	if err != nil {
		return err
	}
	var availableSlot *slot.Slot
	for _, s := range slots {
		// Check no reservation and video is attached to the slot
		if s.Reservation() != nil || s.Video() != nil {
			continue
		}

		if s.State() == slot.Available {
			availableSlot = s
			break
		}
	}
	if availableSlot == nil {
		return nil
	}

	// Attach next reservation to the slot
	reservations, err := s.reservationRepo.List()
	if err != nil {
		return err
	}

	// Find next resercation with ignoring already attached reservations
	var nextReservation *reservation.Reservation
	alreadyAttachedReservationsSeqList := make([]int, 0)
	for _, s := range slots {
		if s.Reservation() != nil {
			alreadyAttachedReservationsSeqList = append(alreadyAttachedReservationsSeqList, int(s.Reservation().Seq()))
		}
	}
	for _, r := range reservations {
		if slices.Contains(alreadyAttachedReservationsSeqList, int(r.Seq())) {
			continue
		}
		nextReservation = r
		break
	}
	if nextReservation == nil {
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
	err = s.slotRepo.AttachReservationAndVideoById(availableSlot.Id(), nextReservation, video)
	if err != nil {
		return err
	}

	// Set the slot state to waiting
	err = s.slotRepo.SetStateById(availableSlot.Id(), slot.Waiting)
	if err != nil {
		return err
	}

	// Writing Video Functionality is in Slot Repository, so it is not implemented here

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
