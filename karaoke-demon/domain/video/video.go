package video

import (
	"errors"

	"gpioblink.com/x/karaoke-demon/domain/song"
)

var ErrSongEmpty = errors.New("empty video song")
var ErrFilePathEmpty = errors.New("empty file path")

type Video struct {
	song     song.Song
	location string
}

func (v *Video) Song() song.Song {
	return v.song
}

func (v *Video) Location() string {
	return v.location
}

func NewVideo(song *song.Song, location string) (*Video, error) {
	if song == nil {
		return nil, ErrSongEmpty
	}
	if location == "" {
		return nil, ErrFilePathEmpty
	}
	return &Video{song: *song, location: location}, nil
}
