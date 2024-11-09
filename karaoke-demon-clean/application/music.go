package application

import "gpioblink.com/x/karaoke-demon-clean/domain/song"

type reservationModel interface {
	AddSong(requestNo song.RequestNo) error
	RemoveSong(seq int) error
	ListSongs() ([]song.Song, error)
}
