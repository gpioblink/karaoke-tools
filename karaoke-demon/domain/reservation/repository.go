package reservation

import "errors"

var ErrNotFound = errors.New("reservation not found")

type Repository interface {
	EnQueue(requestNo string) error
	DeQueue() (*Reservation, error)
	FindBySeq(seq int) (*Reservation, error)
	FindByQueueIndex(index int) (*Reservation, error)
	RemoveBySeq(seq int) error
	List() ([]*Reservation, error)
}
