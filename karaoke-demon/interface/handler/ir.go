package handler

import (
	"context"
	"log"

	"gpioblink.com/x/karaoke-demon/application"
	"gpioblink.com/x/karaoke-demon/domain/song"
)

var ReserveSong HandlerFunc = func(ctx context.Context, service application.MusicService, req Request) {
	if len(req.params) != 1 {
		return
	}

	requestNo := req.params[0]

	err := service.ReserveSong(song.RequestNo(requestNo))
	if err != nil {
		log.Fatalf("failed to reserve song: %v", err)
		return
	}
}
