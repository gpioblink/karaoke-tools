package application

import (
	"context"
	"fmt"
	"log"

	"gpioblink.com/x/karaoke-demon/application/eventbus"
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
	bus             eventbus.EventBus
}

func NewMusicService(reservationRepo reservation.Repository, slotRepo slot.Repository, videoRepo video.Repository, bus eventbus.EventBus) *MusicService {
	return &MusicService{
		reservationRepo: reservationRepo,
		slotRepo:        slotRepo,
		videoRepo:       videoRepo,
		bus:             bus,
	}
}

func (s *MusicService) ReserveSong(requestNo song.RequestNo) error {
	// 予約イベントを投げ、オーケストレータに処理させる
	s.bus.Publish(context.Background(), eventbus.ReservationCreated{SongID: string(requestNo)})
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
	// TODO: なんでこの辺のログファイルを残したのか聞く
	fmt.Printf("Handle: slotId: %d\n", id)
	currentSlot, err := s.slotRepo.GetFirstSlotByState(slot.Reading)
	if currentSlot != nil {
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
	if err != nil {
		// まだ一度もreadが来ていない場合、0から始まる場合のみ受け付ける
		if id != 0 {
			fmt.Println("Handle: no read yet. invalid Order")
			return nil
		}
	}

	// Remove the reservation that is previous reading
	// キューからは曲の再生が終わった時点で削除する。そのため1曲も予約してない状態では消えないようにする
	_, err = s.reservationRepo.FindByQueueIndex(0)
	if err == nil && currentSlot != nil {
		_, err = s.reservationRepo.DeQueue()
		if err != nil {
			log.Printf("failed to dequeue reservation: %v", err)
		}
	}

	// 前スロットを開放済みに、現在を読み取りに、次をロックに。完了後に次の割当を促す
	s.bus.Publish(context.Background(), eventbus.SlotReadingAdvanced{ReadingSlotID: id, TotalSlots: s.slotRepo.Len()})

	return nil
}

func (s *MusicService) AttachNextReservationToSlotIfAvailable() error {
	// 互換のために残すが、実体はイベント発行でオーケストレータに委譲
	s.bus.Publish(context.Background(), eventbus.SlotAvailableAppeared{SlotID: -1})
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
