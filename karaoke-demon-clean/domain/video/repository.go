package video

import "errors"

var ErrVideoEmpty = errors.New("no such video")

type Repository interface {
	FindByRequestNo(requestNo string) (*Video, error)
}
