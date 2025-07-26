package handler

import (
	"context"
	"fmt"
	"log"

	"gpioblink.com/x/karaoke-demon/application"
	"gpioblink.com/x/karaoke-demon/domain/song"
	"gpioblink.com/x/karaoke-demon/domain/wifi"
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

var ConfigureWiFi HandlerFuncWithResponse = func(ctx context.Context, service application.MusicService, req Request) string {
	if len(req.params) != 2 {
		return "params length is not 2 (expected: SSID PASSWORD)"
	}

	ssid := req.params[0]
	password := req.params[1]

	wifiService := wifi.NewSystemWiFiService()
	config := wifi.WiFiConfig{
		SSID:     ssid,
		Password: password,
	}

	err := wifiService.ConfigureWiFi(config)
	if err != nil {
		log.Printf("failed to configure WiFi: %v", err)
		return fmt.Sprintf("failed to configure WiFi: %v", err)
	}

	log.Printf("successfully configured WiFi: %s", ssid)
	return fmt.Sprintf("success: WiFi configured for %s", ssid)
}

var GetWiFiStatus HandlerFuncWithResponse = func(ctx context.Context, service application.MusicService, req Request) string {
	wifiService := wifi.NewSystemWiFiService()
	
	currentSSID, err := wifiService.GetCurrentConnection()
	if err != nil {
		log.Printf("failed to get WiFi status: %v", err)
		return fmt.Sprintf("failed to get WiFi status: %v", err)
	}

	return fmt.Sprintf("connected to: %s", currentSSID)
}
