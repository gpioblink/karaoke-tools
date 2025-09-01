package main

import (
	"fmt"
	"log"

	"gpioblink.com/x/karaoke-demon/application"
	"gpioblink.com/x/karaoke-demon/application/eventbus"
	"gpioblink.com/x/karaoke-demon/application/orchestrator"
	"gpioblink.com/x/karaoke-demon/config"
	"gpioblink.com/x/karaoke-demon/infrastructure/reservation"
	"gpioblink.com/x/karaoke-demon/infrastructure/slot"
	"gpioblink.com/x/karaoke-demon/infrastructure/song"
	"gpioblink.com/x/karaoke-demon/infrastructure/video"
	"gpioblink.com/x/karaoke-demon/interface/ble"
	"gpioblink.com/x/karaoke-demon/interface/fifo"
	httpiface "gpioblink.com/x/karaoke-demon/interface/http"
	"gpioblink.com/x/karaoke-demon/interface/ir"
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

	// 赤外線送信機を初期化
	log.Printf("Initializing IR transmitter on GPIO pin %d...", conf.IR_GPIO_PIN)
	irTransmitter, err := ir.NewIRTransmitter(conf.IR_GPIO_PIN)
	if err != nil {
		log.Printf("Warning: failed to initialize IR transmitter: %v", err)
		log.Println("Continuing without IR functionality...")
		irTransmitter = nil
	} else {
		log.Println("IR transmitter initialized successfully")
		defer func() {
			if err := irTransmitter.Close(); err != nil {
				log.Printf("Error closing IR transmitter: %v", err)
			}
		}()
	}

	// イベントバス/オーケストレータ配線
	bus := eventbus.NewInMemoryEventBus()

	// MusicServiceを作成（赤外線送信機付き）
	var musicService *application.MusicService
	if irTransmitter != nil {
		musicService = application.NewMusicServiceWithIR(reservationRepository, slotRepository, videoRepository, bus, irTransmitter)
	} else {
		musicService = application.NewMusicService(reservationRepository, slotRepository, videoRepository, bus)
	}

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
	httpInterface := httpiface.NewHttpInterface(musicService)

	go httpInterface.Run()

	select {}
}
