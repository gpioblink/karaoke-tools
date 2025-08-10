package video

import (
	"errors"

	"gpioblink.com/x/karaoke-demon/domain/song"
)

var ErrSongEmpty = errors.New("empty video song")
var ErrFilePathEmpty = errors.New("empty file path")

// VideoState は動画の状態
// url_waiting: URL待ち、downloading: ダウンロード中、ready: 使用可能

type State string

const (
	URLWaiting  State = "url_waiting"
	Downloading State = "downloading"
	Ready       State = "ready"
)

type Video struct {
	song     song.Song
	location string
	state    State
}

func (v *Video) Song() song.Song  { return v.song }
func (v *Video) Location() string { return v.location }
func (v *Video) State() State     { return v.state }

func NewVideo(song *song.Song, location string) (*Video, error) {
	if song == nil {
		return nil, ErrSongEmpty
	}
	if location == "" {
		return nil, ErrFilePathEmpty
	}
	return &Video{song: *song, location: location, state: Ready}, nil
}
