package main

import (
	"log"

	"gpioblink.com/x/karaoke-demon-clean/application"
	"gpioblink.com/x/karaoke-demon-clean/infrastructure/reservation"
	"gpioblink.com/x/karaoke-demon-clean/infrastructure/slot"
	"gpioblink.com/x/karaoke-demon-clean/infrastructure/song"
	"gpioblink.com/x/karaoke-demon-clean/infrastructure/video"
	"gpioblink.com/x/karaoke-demon-clean/interface/ble"
	"gpioblink.com/x/karaoke-demon-clean/interface/fifo"
)

func main() {
	log.Println("Starting Karaoke Demon...")
	defer log.Println("Karaoke Demon stopped.")

	songRepository := song.NewMemoryRepository()
	videoRepository := video.NewStorageRepository("/home/gpioblink/Downloads/mvideos/karaoke/output", "/home/gpioblink/Downloads/mvideos/karaoke/output/dummy.mp4")
	reservationRepository := reservation.NewMemoryRepository(songRepository)
	slotRepository := slot.NewMemoryRepository()

	musicService := application.NewMusicService(reservationRepository, slotRepository, videoRepository)

	log.Println("Starting FIFO interface...")
	fifoInterface, err := fifo.NewFifoInterface(musicService, fifo.DefaultRouter, "/tmp/karaoke-fifo")
	if err != nil {
		log.Fatalf("failed to create fifo interface: %v", err)
		panic(err)
	}

	go fifoInterface.Run()

	log.Println("Starting BLE interface...")
	bleInterface := ble.NewBluetoothInterface(musicService, ble.DefaultRouter)

	go bleInterface.Run()

	select {}
}
