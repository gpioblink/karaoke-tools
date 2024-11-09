package song

import "errors"

type Song struct {
	requestNo string // 例 "552501"
}

var ErrRequestNoEmpty = errors.New("empty requestNo")

func (s *Song) RequestNo() string {
	return s.requestNo
}

func NewSongInfo(requestNo string) (*Song, error) {
	if requestNo == "" {
		return nil, ErrRequestNoEmpty
	}
	return &Song{requestNo: requestNo}, nil
}
