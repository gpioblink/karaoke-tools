package slot

import "errors"

var ErrNotFound = errors.New("slot not found")

type Repository interface {
	Update(slot *Slot) error
	FindById(id int) (*Slot, error)
	List() ([]*Slot, error)
}
