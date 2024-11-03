package models

import "fmt"

/* カラオケマシンの予約状態管理 */
type KaraokeSong struct {
	requestNo  string
	isAttached bool
	seq        int
}

type ReservedSongs struct {
	l   []KaraokeSong
	cnt int
}

func NewReservedSongs() ReservedSongs {
	return ReservedSongs{
		l:   []KaraokeSong{},
		cnt: 0,
	}
}

func (rs *ReservedSongs) AddSong(requestNo string) {
	rs.l = append(rs.l, KaraokeSong{requestNo: requestNo, isAttached: false, seq: rs.cnt})
	rs.cnt++
}

func (rs *ReservedSongs) AttachBySeq(seq int) {
	for i, song := range rs.l {
		if song.seq == seq {
			rs.l[i].isAttached = true
			break
		}
	}
}

func (rs *ReservedSongs) RemoveSongBySeq(seq int) { // Detachを兼ねている
	for i, song := range rs.l {
		if song.seq == seq {
			rs.l = append(rs.l[:i], rs.l[i+1:]...)
			break
		}
	}
}

func (rs *ReservedSongs) FindNextAttachableSong() (KaraokeSong, error) {
	for _, song := range rs.l {
		if !song.isAttached {
			return song, nil
		}
	}
	return KaraokeSong{}, fmt.Errorf("no attachable song")
}

func (rs *ReservedSongs) GetSongBySeq(seq int) (KaraokeSong, error) {
	for _, song := range rs.l {
		if song.seq == seq {
			return song, nil
		}
	}
	return KaraokeSong{}, fmt.Errorf("no song")
}

func (ks *KaraokeSong) GetRequestNo() string {
	return ks.requestNo
}

func (ks *KaraokeSong) GetSeq() int {
	return ks.seq
}
