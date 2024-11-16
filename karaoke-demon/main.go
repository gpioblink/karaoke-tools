package main

import (
	"fmt"
	"log"

	"gpioblink.com/x/karaoke-demon/application"
	"gpioblink.com/x/karaoke-demon/config"
	"gpioblink.com/x/karaoke-demon/infrastructure/reservation"
	"gpioblink.com/x/karaoke-demon/infrastructure/slot"
	"gpioblink.com/x/karaoke-demon/infrastructure/song"
	"gpioblink.com/x/karaoke-demon/infrastructure/video"
	"gpioblink.com/x/karaoke-demon/interface/fifo"
)

func main() {
	conf, err := config.NewConfig()
	if err != nil {
		fmt.Println(err)
		return
	}

	log.Println("Starting Karaoke Demon...")
	defer log.Println("Karaoke Demon stopped.")

	songRepository := song.NewMemoryRepository()
	videoRepository := video.NewStorageRepository(conf.VIDEO_DIR, conf.FILLER_VIDEOS_PATH[0])
	reservationRepository := reservation.NewMemoryRepository(songRepository)
	//slotRepository := slot.NewMemoryRepository()
	slotRepository, err := slot.NewFatRepository(conf.IMAGE_PATH, conf.FILLER_VIDEOS_PATH[0])
	if err != nil {
		log.Fatalf("failed to create fat repository: %v", err)
		panic(err)
	}

	musicService := application.NewMusicService(reservationRepository, slotRepository, videoRepository)

	log.Println("Starting FIFO interface...")
	fifoInterface, err := fifo.NewFifoInterface(musicService, fifo.DefaultRouter, conf.FIFO_PATH)
	if err != nil {
		log.Fatalf("failed to create fifo interface: %v", err)
		panic(err)
	}

	go fifoInterface.Run()

	// log.Println("Starting BLE interface...")
	// bleInterface := ble.NewBluetoothInterface(musicService, ble.DefaultRouter)

	// go bleInterface.Run()

	select {}
}
