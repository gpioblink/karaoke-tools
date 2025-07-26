package handler

import (
	"context"
	"log"

	"gpioblink.com/x/karaoke-demon/application"
	"gpioblink.com/x/karaoke-demon/domain/song"
)

var ReserveSongWebhook HandlerFunc = func(ctx context.Context, service application.MusicService, req Request) {
	if len(req.params) != 1 {
		log.Printf("invalid webhook parameters: expected 1, got %d", len(req.params))
		return
	}

	requestNo := req.params[0]

	err := service.ReserveSong(song.RequestNo(requestNo))
	if err != nil {
		log.Printf("failed to reserve song via webhook: %v", err)
		return
	}

	log.Printf("successfully reserved song via webhook: %s", requestNo)
}