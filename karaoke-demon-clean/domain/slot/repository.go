package slot

import (
	"errors"

	"gpioblink.com/x/karaoke-demon-clean/domain/reservation"
	"gpioblink.com/x/karaoke-demon-clean/domain/video"
)

var ErrNotFound = errors.New("slot not found")

type Repository interface {
	Len() int
	AttachReservationAndVideoById(slotId int, reservation *reservation.Reservation, video *video.Video) error
	ChangeVideoById(slotId int, video *video.Video) error
	DettachReservationAndVideoById(slotId int) error
	SetStateById(slotId int, state State) error
	SetWritingFlagById(slotId int, isWriting bool) error
	FindById(id int) (*Slot, error)
	List() ([]*Slot, error)
}
