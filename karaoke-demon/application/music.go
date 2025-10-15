package application

import (
	"context"
	"errors"
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

// VideoInfo は予約時のビデオ情報を表します
type VideoInfo struct {
	Type     string // "local" または "download"
	Filename string
	URL      string // ダウンロードの場合のみ
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
	// ビデオを検索
	var video *video.Video
	video, err := s.videoRepo.FindByRequestNo(string(requestNo))
	if err != nil {
		// ビデオが見つからない場合は、デフォルトのビデオを使用
		video, err = s.videoRepo.GetRandomDummyVideo()
		if err != nil {
			return err
		}
	}

	// 予約イベントを投げ、オーケストレータに処理させる
	s.bus.Publish(context.Background(), eventbus.ReservationCreated{SongID: string(requestNo), VideoTitle: video.Location()})
	return nil
}

// ReserveSongWithVideo はビデオ情報付きで予約を作成します
func (s *MusicService) ReserveSongWithVideo(requestNo song.RequestNo, videoInfo *VideoInfo) error {
	if videoInfo == nil {
		// ビデオ情報がない場合は通常の予約
		return s.ReserveSong(requestNo)
	}

	// ビデオ情報付きの予約イベントを発行
	event := eventbus.ReservationCreated{
		SongID: string(requestNo),
	}

	switch videoInfo.Type {
	case "local":
		// ローカルファイルの場合
		event.VideoTitle = videoInfo.Filename
		event.WithVideoURL = "" // ローカルファイルなのでURLは空
	case "download":
		// ダウンロードの場合
		event.VideoTitle = videoInfo.Filename
		event.WithVideoURL = videoInfo.URL
	default:
		return fmt.Errorf("unsupported video type: %s", videoInfo.Type)
	}

	s.bus.Publish(context.Background(), event)
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
	totalSlots := s.slotRepo.Len()
	if totalSlots == 0 {
		return fmt.Errorf("Handle: slot repository is empty")
	}

	currentSlot, err := s.slotRepo.GetFirstSlotByState(slot.Reading)
	if err != nil && !errors.Is(err, slot.ErrNotFound) {
		return err
	}

	targetID := id
	if currentSlot != nil {
		fmt.Printf("Handle: currentId: %d\n", currentSlot.Id())
		if currentSlot.Id() == targetID {
			// 前回の読み込み時点から変わっていなければ何もしない
			fmt.Println("Handle: No Change")
			return nil
		}

		expectedPrev := calcPositiveModulo(targetID-1, totalSlots)
		if currentSlot.Id() != expectedPrev {
			// 3スロット前提での読み取り順に合わせるため、次スロットへ補正
			expectedNext := calcPositiveModulo(currentSlot.Id()+1, totalSlots)
			fmt.Printf("Handle: unexpected Order (current=%d, expectedPrev=%d). adjust to %d\n", currentSlot.Id(), expectedPrev, expectedNext)
			targetID = expectedNext
		}
	} else if targetID != 0 {
		// まだ一度もreadが来ていない場合、0から始まる場合のみ受け付ける
		fmt.Println("Handle: no read yet. invalid Order")
		return nil
	}

	// Remove the reservation that is previous reading
	// キューからは曲の再生が終わった時点で削除する。そのため1曲も予約してない状態では消えないようにする
	if _, err := s.reservationRepo.FindByQueueIndex(0); err == nil && currentSlot != nil {
		if _, err := s.reservationRepo.DeQueue(); err != nil {
			log.Printf("failed to dequeue reservation: %v", err)
		}
	}

	// 前スロットを開放済みに、現在を読み取りに、次をロックに。完了後に次の割当を促す
	s.bus.Publish(context.Background(), eventbus.SlotReadingAdvanced{ReadingSlotID: targetID, TotalSlots: totalSlots})

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

func (s *MusicService) FindLocalFilesByRequestNo(requestNo string) ([]string, error) {
	files, err := s.videoRepo.FindLocalFilesByRequestNo(requestNo)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (s *MusicService) GetReservationWithSlotInfo() ([]*reservation.Reservation, []*slot.Slot, error) {
	reservations, err := s.reservationRepo.List()
	if err != nil {
		return nil, nil, err
	}

	slots, err := s.slotRepo.List()
	if err != nil {
		return nil, nil, err
	}

	return reservations, slots, nil
}

func calcPositiveModulo(a, b int) int {
	return (a%b + b) % b
}
