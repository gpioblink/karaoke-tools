package handler

import (
	"context"
	"fmt"
	"log"

	"gpioblink.com/x/karaoke-demon-clean/application"
	"gpioblink.com/x/karaoke-demon-clean/domain/song"
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
		song := r.Song()
		reservationStr += fmt.Sprintf("seq: %d, requestNo: %s\n", r.Seq(), string(song.RequestNo()))
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
		slotStr += fmt.Sprintf("id: %d, state: %s\n", s.Id(), s.State())
	}

	return slotStr
}
