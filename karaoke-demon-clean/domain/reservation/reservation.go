package reservation

import (
	"errors"

	"gpioblink.com/x/karaoke-demon-clean/domain/song"
)

var (
	ErrReservationSeqEmpty  = errors.New("empty reservation seq")
	ErrReservationSongEmpty = errors.New("empty reservation song")
)

type Reservation struct {
	seq  int
	song song.Song
}

func (r *Reservation) Seq() int {
	return r.seq
}

func (r *Reservation) Song() song.Song {
	return r.song
}

func NewReservation(seq int, song *song.Song) (*Reservation, error) {
	if seq == 0 {
		return nil, ErrReservationSeqEmpty
	}
	if song == nil {
		return nil, ErrReservationSongEmpty
	}
	return &Reservation{seq: seq, song: *song}, nil
}
