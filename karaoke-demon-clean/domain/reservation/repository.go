package reservation

import "errors"

var ErrNotFound = errors.New("reservation not found")

type Repository interface {
	EnQueue(reservation *Reservation) error
	DeQueue() (*Reservation, error)
	FindBySeq(seq int) (*Reservation, error)
	RemoveBySeq(seq int) error
	List() ([]*Reservation, error)
}
