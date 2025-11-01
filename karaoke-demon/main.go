package main

import (
	"fmt"
	"log"

	"gpioblink.com/x/karaoke-demon/application"
	"gpioblink.com/x/karaoke-demon/application/eventbus"
	"gpioblink.com/x/karaoke-demon/application/orchestrator"
	"gpioblink.com/x/karaoke-demon/config"
	"gpioblink.com/x/karaoke-demon/infrastructure/logging"
	"gpioblink.com/x/karaoke-demon/infrastructure/reservation"
	"gpioblink.com/x/karaoke-demon/infrastructure/slot"
	"gpioblink.com/x/karaoke-demon/infrastructure/video"
	"gpioblink.com/x/karaoke-demon/interface/ble"
	"gpioblink.com/x/karaoke-demon/interface/fifo"
	httpiface "gpioblink.com/x/karaoke-demon/interface/http"
)

// pre-commitフックで設定される変数
var (
	Version   = "v0.5.1-5-g501352b-dirty"
	BuildDate = "2025-11-01T11:38:49+09:00"
	BuildUser = "gpioblink"
)

func main() {
	conf, err := config.NewConfig()
	if err != nil {
		fmt.Println(err)
		return
	}

	logging.Setup()

	log.Println("Starting Karaoke Demon...")
	defer log.Println("Karaoke Demon stopped.")

	//songRepository := song.NewMemoryRepository()
	videoRepository := video.NewStorageRepository(conf.VIDEO_DIR, conf.FILLER_VIDEOS_PATH[0])
	reservationRepository := reservation.NewMemoryRepository()
	//slotRepository := slot.NewMemoryRepository()
	slotRepository, err := slot.NewFatRepository(conf.IMAGE_PATH, conf.FILLER_VIDEOS_PATH[0])
	if err != nil {
		log.Fatalf("failed to create fat repository: %v", err)
		panic(err)
	}

	// イベントバス/オーケストレータ配線
	bus := eventbus.NewInMemoryEventBus()
	musicService := application.NewMusicService(reservationRepository, slotRepository, videoRepository, bus)
	_ = orchestrator.New(orchestrator.Dependencies{
		Bus:             bus,
		ReservationRepo: reservationRepository,
		SlotRepo:        slotRepository,
		VideoRepo:       videoRepository,
		DownloadDir:     conf.VIDEO_DIR,
	})

	log.Println("Starting FIFO interface...")
	fifoInterface, err := fifo.NewFifoInterface(musicService, fifo.DefaultRouter, conf.FIFO_PATH)
	if err != nil {
		log.Fatalf("failed to create fifo interface: %v", err)
		panic(err)
	}

	go fifoInterface.Run()

	log.Println("Starting BLE interface...")
	bleInterface := ble.NewBluetoothInterface(musicService, ble.DefaultRouter)

	go bleInterface.Run()

	log.Println("Starting HTTP webhook interface...")
	httpInterface := httpiface.NewHttpInterface(musicService, Version, BuildDate, BuildUser)

	go httpInterface.Run()

	select {}
}
