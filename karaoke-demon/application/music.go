package application

import (
	"fmt"
	"log"
	"path/filepath"

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
	fmt.Printf("Handle: slotId: %d\n", id)
	currentSlot, err := s.slotRepo.GetFirstSlotByState(slot.Reading)
	if err == nil {
		fmt.Printf("Handle: currentId: %d\n", currentSlot.Id())
		if currentSlot.Id() == id {
			// 前回の読み込み時点から変わっていなければ何もしない
			fmt.Println("Handle: No Change")
			return nil
		} else if currentSlot.Id() != calcPositiveModulo(id-1, s.slotRepo.Len()) {
			// 前回から連続するIDでない場合は、おかしいので何もしない
			fmt.Println("Handle: invalid Order")
			return nil
		}
	}

	// Remove the reservation that is previous reading
	_, err = s.reservationRepo.DeQueue()
	if err != nil {
		log.Printf("failed to dequeue reservation: %v", err)
	}

	// Make previous slot state available because it is not reserved by any reservation now
	prevSlot, err := s.slotRepo.FindById(calcPositiveModulo(id-1, s.slotRepo.Len()))
	if err != nil {
		return err
	}

	if prevSlot.Reservation() != nil && prevSlot.State() != slot.Waiting {
		err = s.slotRepo.DettachReservationById(calcPositiveModulo(id-1, s.slotRepo.Len()))
		if err != nil {
			return err
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
	readingSlot, err := s.slotRepo.GetFirstSlotByState(slot.Reading)
	if err != nil {
		// Readingがないということは、初期状態であるので、0番のスロットから検索するようにする
		readingSlot, err = s.slotRepo.FindById(0)
		if err != nil {
			return err
		}
	}

	for i := 0; i < s.slotRepo.Len(); i++ {
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
			// 予約がない場合、適当なダミーのビデオを差し込んでおく処理

			// ビデオ名がdummyから始まる場合はすでにフィラーが入っているので無視
			if availableSlot.Video() != nil && filepath.Base(availableSlot.Video().Location())[:5] == "dummy" {
				continue
			}

			// 現時点で予約がない場合は、ダミーのビデオを差し込んでおく
			video, err := s.videoRepo.GetRandomDummyVideo()
			if err != nil {
				return err
			}
			s.slotRepo.ChangeVideoById(availableSlot.Id(), video)

			continue
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
