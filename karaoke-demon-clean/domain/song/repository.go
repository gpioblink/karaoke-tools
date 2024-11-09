package song

import "errors"

var ErrNotFound = errors.New("song not found")

type Repository interface {
	Store(song *Song) error
	FindById(id int) (*Song, error)
	FindAll() ([]*Song, error)
}
