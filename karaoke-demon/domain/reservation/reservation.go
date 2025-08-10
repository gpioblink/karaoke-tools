package reservation

import (
	"errors"

	"gpioblink.com/x/karaoke-demon/domain/song"
)

var (
	ErrReservationSeqEmpty  = errors.New("empty reservation seq")
	ErrReservationSongEmpty = errors.New("empty reservation song")
)

type SeqNum int

type State string

const (
	PreparingVideo State = "preparing_video"
	WritingToSlot  State = "writing_to_slot"
	SetInSlot      State = "set_in_slot"
)

type Reservation struct {
	seq   SeqNum
	song  *song.Song
	state State
}

func (r *Reservation) Seq() SeqNum { return r.seq }

func (r *Reservation) Song() (*song.Song, error) {
	if r.song == nil {
		return nil, ErrReservationSongEmpty
	}
	return r.song, nil
}

func (r *Reservation) State() State { return r.state }

func (r *Reservation) SetState(s State) { r.state = s }

func NewReservation(seq SeqNum, song *song.Song) (*Reservation, error) {
	if seq < 0 {
		return nil, ErrReservationSeqEmpty
	}
	if song == nil {
		return nil, ErrReservationSongEmpty
	}
	return &Reservation{seq: seq, song: song, state: PreparingVideo}, nil
}
