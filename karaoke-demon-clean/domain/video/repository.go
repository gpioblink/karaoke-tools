package video

import "errors"

var ErrVideoEmpty = errors.New("no such video")

type Repository interface {
	Store(video *Video) error
	FindById(id int) (*Video, error)
	FindByRequestNo(requestNo string) (*Video, error)
	FindAll() ([]*Video, error)
}
