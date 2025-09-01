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

// SendIRCommand 赤外線コマンドを送信するハンドラー
var SendIRCommand HandlerFunc = func(ctx context.Context, service application.MusicService, req Request) {
	if len(req.params) != 1 {
		log.Printf("SendIRCommand: invalid parameters count: %d", len(req.params))
		return
	}

	commandStr := req.params[0]
	err := service.SendIRCommandString(commandStr)
	if err != nil {
		log.Printf("failed to send IR command %s: %v", commandStr, err)
		return
	}
	log.Printf("Successfully sent IR command: %s", commandStr)
}

// SendIRData 赤外線データを送信するハンドラー
var SendIRData HandlerFunc = func(ctx context.Context, service application.MusicService, req Request) {
	if len(req.params) != 2 {
		log.Printf("SendIRData: invalid parameters count: %d", len(req.params))
		return
	}

	addressStr := req.params[0]
	commandStr := req.params[1]
	err := service.SendIRDataString(addressStr, commandStr)
	if err != nil {
		log.Printf("failed to send IR data - address: %s, command: %s: %v", addressStr, commandStr, err)
		return
	}
	log.Printf("Successfully sent IR data - address: %s, command: %s", addressStr, commandStr)
}
