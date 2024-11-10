package song

import "errors"

var ErrNotFound = errors.New("song not found")

type Repository interface {
	FindByRequestNo(requestNo string) (*Song, error)
}
