package orchestrator_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gpioblink.com/x/karaoke-demon/application"
	"gpioblink.com/x/karaoke-demon/application/eventbus"
	"gpioblink.com/x/karaoke-demon/application/orchestrator"
	"gpioblink.com/x/karaoke-demon/config"
	"gpioblink.com/x/karaoke-demon/domain/slot"
	domainSong "gpioblink.com/x/karaoke-demon/domain/song"
	domainVideo "gpioblink.com/x/karaoke-demon/domain/video"
	resInfra "gpioblink.com/x/karaoke-demon/infrastructure/reservation"
	slotInfra "gpioblink.com/x/karaoke-demon/infrastructure/slot"
	videoInfra "gpioblink.com/x/karaoke-demon/infrastructure/video"
)

func setupFat(t *testing.T) (*application.MusicService, *eventbus.InMemoryEventBus, orchestrator.Dependencies, func()) {
	// main.go と同じパスから .env を解決できるように、リポジトリルートへ移動
	root, err := filepath.Abs("../../")
	if err == nil {
		_ = os.Chdir(root)
	}

	os.Setenv("APP_ENV", "dev")
	conf, _ := config.NewConfig()

	// 前提チェック: makemyfat コマンド
	if _, err := exec.LookPath("makemyfat"); err != nil {
		t.Skip("makemyfat not found; skipping FAT scenario test")
	}
	// ダミー動画/動画ディレクトリ存在
	if _, err := os.Stat(conf.FILLER_VIDEOS_PATH[0]); err != nil {
		t.Skip("dummy video not found; set DUMMY_VIDEO_PATH in .env.dev")
	}
	if _, err := os.Stat(conf.VIDEO_DIR); err != nil {
		t.Skip("VIDEO_DIR not found; set VIDEO_DIR in .env.dev")
	}

	videoRepo := videoInfra.NewStorageRepository(conf.VIDEO_DIR, conf.FILLER_VIDEOS_PATH[0])
	reservationRepo := resInfra.NewMemoryRepository()
	slotRepo, err := slotInfra.NewFatRepository(conf.IMAGE_PATH, conf.FILLER_VIDEOS_PATH[0])
	if err != nil {
		t.Fatalf("new fat repo: %v", err)
	}

	bus := eventbus.NewInMemoryEventBus()
	service := application.NewMusicService(reservationRepo, slotRepo, videoRepo, bus)
	deps := orchestrator.Dependencies{Bus: bus, ReservationRepo: reservationRepo, SlotRepo: slotRepo, VideoRepo: videoRepo}
	_ = orchestrator.New(deps)

	cleanup := func() { os.Remove(conf.IMAGE_PATH) }
	return service, bus, deps, cleanup
}

func TestScenario_StateTransitions_Fat(t *testing.T) {
	service, bus, deps, cleanup := setupFat(t)
	defer cleanup()

	// STEP1
	logStep(t, "FAT-STEP1: Startup", deps)
	assertSlots(t, deps, []slot.State{slot.Available, slot.Available, slot.Available})
	assertReservationsOrder(t, deps, []string{})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// STEP2: 予約1件
	reserve(t, service, bus, string(domainSong.RequestNo("324244")))
	logStep(t, "FAT-STEP2: Reserve 324244", deps)
	assertSlots(t, deps, []slot.State{slot.Set, slot.Available, slot.Available})
	assertReservationsOrder(t, deps, []string{"324244"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// STEP3: 予約2件目
	reserve(t, service, bus, string(domainSong.RequestNo("321445")))
	logStep(t, "FAT-STEP3: Reserve 321445", deps)
	assertSlots(t, deps, []slot.State{slot.Set, slot.Set, slot.Available})
	assertReservationsOrder(t, deps, []string{"324244", "321445"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// STEP4: 予約3件目
	reserve(t, service, bus, string(domainSong.RequestNo("999999")))
	logStep(t, "FAT-STEP4: Reserve 999999", deps)
	assertSlots(t, deps, []slot.State{slot.Set, slot.Set, slot.Set})
	assertReservationsOrder(t, deps, []string{"324244", "321445", "999999"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// STEP5: 読み取りが0へ
	reading(t, service, bus, 0)
	logStep(t, "FAT-STEP5: Reading -> 0", deps)
	assertSlots(t, deps, []slot.State{slot.Reading, slot.Locked, slot.Set})
	assertReservationsOrder(t, deps, []string{"324244", "321445", "999999"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// STEP6: 読み取りが1へ（先頭予約デキュー）
	reading(t, service, bus, 1)
	logStep(t, "FAT-STEP6: Reading -> 1", deps)
	assertSlots(t, deps, []slot.State{slot.Available, slot.Reading, slot.Locked})
	assertReservationsOrder(t, deps, []string{"321445", "999999"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// STEP7: 追加予約（きんモザ）→0がSet
	reserve(t, service, bus, string(domainSong.RequestNo("365537")))
	logStep(t, "FAT-STEP7: Reserve 365537", deps)
	assertSlots(t, deps, []slot.State{slot.Set, slot.Reading, slot.Locked})
	assertReservationsOrder(t, deps, []string{"321445", "999999", "365537"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// STEP8: 追加予約（のうりん）
	reserve(t, service, bus, string(domainSong.RequestNo("208564")))
	logStep(t, "FAT-STEP8: Reserve 208564", deps)
	assertSlots(t, deps, []slot.State{slot.Set, slot.Reading, slot.Locked})
	assertReservationsOrder(t, deps, []string{"321445", "999999", "365537", "208564"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// STEP9: 追加予約（ごちうさ）
	reserve(t, service, bus, string(domainSong.RequestNo("370040")))
	logStep(t, "FAT-STEP9: Reserve 370040", deps)
	assertSlots(t, deps, []slot.State{slot.Set, slot.Reading, slot.Locked})
	assertReservationsOrder(t, deps, []string{"321445", "999999", "365537", "208564", "370040"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// STEP10: 読み取りが2へ（先頭予約デキュー、1に208564がセット）
	reading(t, service, bus, 2)
	logStep(t, "FAT-STEP10: Reading -> 2", deps)
	assertSlots(t, deps, []slot.State{slot.Locked, slot.Set, slot.Reading})
	assertReservationsOrder(t, deps, []string{"999999", "365537", "208564", "370040"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// STEP11: 読み取りが0へ（先頭予約デキュー、2に370040がセット）
	reading(t, service, bus, 0)
	logStep(t, "FAT-STEP11: Reading -> 0", deps)
	assertSlots(t, deps, []slot.State{slot.Reading, slot.Locked, slot.Set})
	assertReservationsOrder(t, deps, []string{"365537", "208564", "370040"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// STEP12: 読み取りが1へ（先頭予約デキュー）
	reading(t, service, bus, 1)
	logStep(t, "FAT-STEP12: Reading -> 1", deps)
	assertSlots(t, deps, []slot.State{slot.Available, slot.Reading, slot.Locked})
	assertReservationsOrder(t, deps, []string{"208564", "370040"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// STEP13: 読み取りが2へ（先頭予約デキュー）
	reading(t, service, bus, 2)
	logStep(t, "FAT-STEP13: Reading -> 2", deps)
	assertSlots(t, deps, []slot.State{slot.Locked, slot.Available, slot.Reading})
	assertReservationsOrder(t, deps, []string{"370040"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// STEP14: 読み取りが0へ（先頭予約デキューで空）
	reading(t, service, bus, 0)
	logStep(t, "FAT-STEP14: Reading -> 0", deps)
	assertSlots(t, deps, []slot.State{slot.Reading, slot.Locked, slot.Available})
	assertReservationsOrder(t, deps, []string{})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})
}
