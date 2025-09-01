package video

import "errors"

var ErrVideoEmpty = errors.New("no such video")
var ErrUrlInvalid = errors.New("url is invalid")

type Repository interface {
	FindByRequestNo(requestNo string) (*Video, error)
	GetRandomDummyVideo() (*Video, error)
	FindLocalFilesByRequestNo(requestNo string) ([]string, error)
	GetVideoDir() string
}
