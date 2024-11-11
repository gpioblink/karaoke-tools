package slot

import (
	"gpioblink.com/x/karaoke-demon-clean/domain/reservation"
	"gpioblink.com/x/karaoke-demon-clean/domain/video"
)

// type Repository interface {
// 	Len() int
// 	AttachReservationAndVideoById(slotId int, reservation *reservation.Reservation, video *video.Video) error
// 	ChangeVideoById(slotId int, video *video.Video) error
// 	DettachReservationAndVideoById(slotId int) error
// 	SetStateById(slotId int, state State) error
// 	SetWritingFlagById(slotId int, isWriting bool) error
// 	FindById(id int) (*Slot, error)
// 	List() ([]*Slot, error)
// }

type FatRepository struct {
	memoryRepository *MemoryRepository
}

func NewFatRepository() *FatRepository {
	// Memory実装の拡張として実装する
	return &FatRepository{
		memoryRepository: NewMemoryRepository(),
	}
}

func (f *FatRepository) Len() int {
	return f.memoryRepository.Len()
}

func (f *FatRepository) AttachReservationAndVideoById(slotId int, reservation *reservation.Reservation, video *video.Video) error {
	// mkmyfatを利用して、

	return f.memoryRepository.AttachReservationAndVideoById(slotId, reservation, video)
}
