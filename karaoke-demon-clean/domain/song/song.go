package song

import "errors"

type RequestNo string

type Song struct {
	requestNo RequestNo // 例 "552501"
}

var ErrRequestNoEmpty = errors.New("empty requestNo")

func (s *Song) RequestNo() RequestNo {
	return s.requestNo
}

func NewSongInfo(requestNo string) (*Song, error) {
	if requestNo == "" {
		return nil, ErrRequestNoEmpty
	}
	return &Song{requestNo: RequestNo(requestNo)}, nil
}
