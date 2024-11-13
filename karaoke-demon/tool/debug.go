package tool

import (
	"fmt"
	"path/filepath"

	"gpioblink.com/x/karaoke-demon/domain/reservation"
	"gpioblink.com/x/karaoke-demon/domain/slot"
)

// FIXME: 不正な値のinjectionチェックをする

func TextSlots(slots []*slot.Slot) string {
	slotStr := ""
	for _, s := range slots {
		songNo := ""
		seq := -1
		location := ""
		if s.Reservation() != nil {
			seq = int(s.Reservation().Seq())

			song, err := s.Reservation().Song()
			if err == nil {
				songNo = string(song.RequestNo())
			}
		}
		if s.Video() != nil {
			location = filepath.Base(s.Video().Location())
		}

		slotStr += fmt.Sprintf("id: %d, state: %s, seq: %d, res: seq=%d, songNo=%s, video: %s, isWriting: %t\n", s.Id(), s.State(), s.Seq(), seq, songNo, location, s.IsWriting())
	}

	return slotStr
}

func TextSlotsJson(slots []*slot.Slot) string {
	// slotのリストをJSON形式に変換して出力
	slotStr := "["
	for i, s := range slots {
		songNo := ""
		seq := -1
		location := ""
		if s.Reservation() != nil {
			seq = int(s.Reservation().Seq())

			song, err := s.Reservation().Song()
			if err == nil {
				songNo = string(song.RequestNo())
			}
		}
		if s.Video() != nil {
			location = filepath.Base(s.Video().Location())
		}

		slotStr += fmt.Sprintf("{\"id\": %d, \"state\": \"%s\", \"seq\": %d, \"res\": {\"seq\": %d, \"songNo\": \"%s\"}, \"video\": \"%s\", \"isWriting\": %t}", s.Id(), s.State(), s.Seq(), seq, songNo, location, s.IsWriting())
		if i < len(slots)-1 {
			slotStr += ","
		}
	}
	slotStr += "]"
	return slotStr
}

func TextReservations(reservations []*reservation.Reservation) string {
	reservationStr := ""
	for _, r := range reservations {
		songNo := ""
		song, err := r.Song()
		if err == nil {
			songNo = string(song.RequestNo())
		}

		reservationStr += fmt.Sprintf("seq: %d, requestNo: %s\n", r.Seq(), songNo)
	}

	return reservationStr
}

func TextReservationsJson(reservations []*reservation.Reservation) string {
	// reservationのリストをJSON形式に変換して出力
	reservationStr := "["
	for i, r := range reservations {
		songNo := ""
		song, err := r.Song()
		if err == nil {
			songNo = string(song.RequestNo())
		}

		reservationStr += fmt.Sprintf("{\"seq\": %d, \"requestNo\": \"%s\"}", r.Seq(), songNo)
		if i < len(reservations)-1 {
			reservationStr += ","
		}
	}
	reservationStr += "]"
	return reservationStr
}

func TextCombinedJson(slots []*slot.Slot, reservations []*reservation.Reservation) string {
	// slotとreservationのリストを結合してJSON形式に変換して出力
	slotStr := TextSlotsJson(slots)
	reservationStr := TextReservationsJson(reservations)
	return fmt.Sprintf("{\"slots\": %s, \"reservations\": %s}", slotStr, reservationStr)
}
