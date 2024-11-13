package slot

import (
	"errors"

	"gpioblink.com/x/karaoke-demon/domain/reservation"
	"gpioblink.com/x/karaoke-demon/domain/video"
)

var ErrNotFound = errors.New("slot not found")

type Repository interface {
	Len() int
	AttachReservationById(slotId int, reservation *reservation.Reservation) error
	DettachReservationById(slotId int) error
	ChangeVideoById(slotId int, video *video.Video) error
	SetStateById(slotId int, state State) error
	SetSeqById(slotId int, seq int) error
	SetWritingFlagById(slotId int, isWriting bool) error
	GetFirstSlotByState(state State) (*Slot, error)
	FindById(id int) (*Slot, error)
	List() ([]*Slot, error)
}
