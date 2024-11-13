package handler

import (
	"context"
	"fmt"
	"log"

	"gpioblink.com/x/karaoke-demon/application"
	"gpioblink.com/x/karaoke-demon/domain/song"
	"gpioblink.com/x/karaoke-demon/tool"
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

	return tool.TextReservations(reservations)
}

var ListSlots HandlerFuncWithResponse = func(ctx context.Context, service application.MusicService, req Request) string {
	slots, err := service.ListSlots()
	if err != nil {
		log.Fatalf("failed to list slots: %v", err)
		return fmt.Sprintf("failed to list slots: %v", err)
	}

	return tool.TextSlots(slots)
}

var GetStatusJson HandlerFuncWithResponse = func(ctx context.Context, service application.MusicService, req Request) string {
	slots, err := service.ListSlots()
	if err != nil {
		log.Fatalf("failed to list slots: %v", err)
		return fmt.Sprintf("failed to list slots: %v", err)
	}

	reservations, err := service.ListReservations()
	if err != nil {
		log.Fatalf("failed to list reservations: %v", err)
		return fmt.Sprintf("failed to list reservations: %v", err)
	}

	return tool.TextCombinedJson(slots, reservations)
}
