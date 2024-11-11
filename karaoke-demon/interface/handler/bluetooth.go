package handler

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"gpioblink.com/x/karaoke-demon/application"
	"gpioblink.com/x/karaoke-demon/domain/song"
)

var ReserveSongResult HandlerFuncWithResponse = func(ctx context.Context, service application.MusicService, req Request) string {
	if len(req.params) != 1 {
		return "params length is not 1"
	}

	requestNo := req.params[0]

	err := service.ReserveSong(song.RequestNo(requestNo))
	if err != nil {
		log.Fatalf("failed to reserve song: %v", err)
		return fmt.Sprintf("failed to reserve song: %v", err)
	}

	return "success"
}

var ListReservations HandlerFuncWithResponse = func(ctx context.Context, service application.MusicService, req Request) string {
	reservations, err := service.ListReservations()
	if err != nil {
		log.Fatalf("failed to list reservations: %v", err)
		return fmt.Sprintf("failed to list reservations: %v", err)
	}

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

var ListSlots HandlerFuncWithResponse = func(ctx context.Context, service application.MusicService, req Request) string {
	slots, err := service.ListSlots()
	if err != nil {
		log.Fatalf("failed to list slots: %v", err)
		return fmt.Sprintf("failed to list slots: %v", err)
	}

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

		slotStr += fmt.Sprintf("id: %d, state: %s, res: seq=%d, songNo=%s, video: %s, isWriting: %t\n", s.Id(), s.State(), seq, songNo, location, s.IsWriting())
	}

	log.Printf("slotStr: %s", slotStr)

	return slotStr
}
