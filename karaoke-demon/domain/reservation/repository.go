package reservation

import (
	"errors"

	"gpioblink.com/x/karaoke-demon/domain/video"
)

var ErrNotFound = errors.New("reservation not found")

type Repository interface {
	EnQueue(video *video.Video) (seq int, err error)
	DeQueue() (*Reservation, error)
	FindBySeq(seq int) (*Reservation, error)
	FindByQueueIndex(index int) (*Reservation, error)
	RemoveBySeq(seq int) error
	List() ([]*Reservation, error)
}
