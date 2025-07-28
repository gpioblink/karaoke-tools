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

var ConfigureWiFi HandlerFuncWithResponse = func(ctx context.Context, service application.MusicService, req Request) string {
	if len(req.params) != 2 {
		return "params length is not 2 (expected: SSID PASSWORD)"
	}

	ssid := req.params[0]
	password := req.params[1]

	wifiService := application.NewSystemWiFiService()
	config := application.WiFiConfig{
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
	wifiService := application.NewSystemWiFiService()

	currentSSID, err := wifiService.GetCurrentConnection()
	if err != nil {
		log.Printf("failed to get WiFi status: %v", err)
		return fmt.Sprintf("failed to get WiFi status: %v", err)
	}

	return fmt.Sprintf("connected to: %s", currentSSID)
}

var StartNgrok HandlerFuncWithResponse = func(ctx context.Context, service application.MusicService, req Request) string {
	ngrokService := application.NewNgrokService(8787)

	err := ngrokService.Start()
	if err != nil {
		log.Printf("failed to start ngrok: %v", err)
		return fmt.Sprintf("failed to start ngrok: %v", err)
	}

	publicURL := ngrokService.GetPublicURL()
	log.Printf("ngrok tunnel started: %s", publicURL)
	return fmt.Sprintf("ngrok started: %s/webhook/reserve", publicURL)
}

var GetWebhookURL HandlerFuncWithResponse = func(ctx context.Context, service application.MusicService, req Request) string {
	ngrokService := application.NewNgrokService(8787)

	publicURL := ngrokService.GetPublicURL()
	if publicURL == "" {
		return "ngrok tunnel not running or not available"
	}

	return fmt.Sprintf("%s/webhook/reserve", publicURL)
}

var ConfigureNgrokToken HandlerFuncWithResponse = func(ctx context.Context, service application.MusicService, req Request) string {
	if len(req.params) != 1 {
		return "params length is not 1 (expected: TOKEN)"
	}

	token := req.params[0]
	ngrokService := application.NewNgrokService(8787)

	err := ngrokService.ConfigureAuthToken(token)
	if err != nil {
		log.Printf("failed to configure ngrok token: %v", err)
		return fmt.Sprintf("failed to configure ngrok token: %v", err)
	}

	log.Printf("successfully configured ngrok auth token")
	return "success: ngrok auth token configured"
}

var ResetWiFiConfig HandlerFuncWithResponse = func(ctx context.Context, service application.MusicService, req Request) string {
	wifiService := application.NewSystemWiFiService()

	err := wifiService.ResetWiFiConfig()
	if err != nil {
		log.Printf("failed to reset WiFi config: %v", err)
		return fmt.Sprintf("failed to reset WiFi config: %v", err)
	}

	log.Printf("successfully reset WiFi configurations")
	return "success: all WiFi configurations removed"
}
