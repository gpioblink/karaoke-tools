package orchestrator_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bytes"
	"net/http"
	"net/http/httptest"
	"os"

	"gpioblink.com/x/karaoke-demon/application"
	"gpioblink.com/x/karaoke-demon/application/eventbus"
	"gpioblink.com/x/karaoke-demon/application/orchestrator"
	"gpioblink.com/x/karaoke-demon/domain/reservation"
	"gpioblink.com/x/karaoke-demon/domain/slot"
	domainSong "gpioblink.com/x/karaoke-demon/domain/song"
	domainVideo "gpioblink.com/x/karaoke-demon/domain/video"
	resInfra "gpioblink.com/x/karaoke-demon/infrastructure/reservation"
	slotInfra "gpioblink.com/x/karaoke-demon/infrastructure/slot"
	videoInfra "gpioblink.com/x/karaoke-demon/infrastructure/video"
)

type fakeVideoRepo struct{}

func (f fakeVideoRepo) FindByRequestNo(requestNo string) (*domainVideo.Video, error) {
	sg, err := domainSong.NewSongInfo(requestNo)
	if err != nil {
		return nil, err
	}
	return domainVideo.NewVideo(sg, requestNo+".mp4")
}
func (f fakeVideoRepo) GetRandomDummyVideo() (*domainVideo.Video, error) {
	sg, err := domainSong.NewSongInfo("dummyfiller")
	if err != nil {
		return nil, err
	}
	return domainVideo.NewVideo(sg, "dummy.mp4")
}
func (f fakeVideoRepo) FindLocalFilesByRequestNo(requestNo string) ([]string, error) {
	return []string{requestNo + ".mp4"}, nil
}

// ダウンロード可否を制御できるテスト用ビデオリポジトリ
// 初期状態では全て未ダウンロード（Findはエラー）。MakeAvailable後に取得可能になる

type downloadGateVideoRepo struct{ available map[string]bool }

func newDownloadGateVideoRepo() *downloadGateVideoRepo {
	return &downloadGateVideoRepo{available: map[string]bool{}}
}

func (f *downloadGateVideoRepo) MakeAvailable(req string) { f.available[req] = true }

func (f *downloadGateVideoRepo) FindByRequestNo(requestNo string) (*domainVideo.Video, error) {
	if !f.available[requestNo] {
		return nil, domainVideo.ErrVideoEmpty
	}
	sg, err := domainSong.NewSongInfo(requestNo)
	if err != nil {
		return nil, err
	}
	return domainVideo.NewVideo(sg, requestNo+".mp4")
}
func (f *downloadGateVideoRepo) GetRandomDummyVideo() (*domainVideo.Video, error) {
	sg, err := domainSong.NewSongInfo("dummyfiller")
	if err != nil {
		return nil, err
	}
	return domainVideo.NewVideo(sg, "dummy.mp4")
}
func (f *downloadGateVideoRepo) FindLocalFilesByRequestNo(requestNo string) ([]string, error) {
	if f.available[requestNo] {
		return []string{requestNo + ".mp4"}, nil
	}
	return []string{}, nil
}

// スロット状態の遷移履歴を記録するラッパ

type recordingSlotRepo struct {
	inner   slot.Repository
	history map[int][]slot.State
}

func newRecordingSlotRepo(inner slot.Repository) *recordingSlotRepo {
	r := &recordingSlotRepo{inner: inner, history: map[int][]slot.State{}}
	slots, _ := inner.List()
	for _, s := range slots {
		r.history[s.Id()] = append(r.history[s.Id()], s.State())
	}
	return r
}

func (r *recordingSlotRepo) record(id int, st slot.State) { r.history[id] = append(r.history[id], st) }

// implement slot.Repository by forwarding
func (r *recordingSlotRepo) Len() int { return r.inner.Len() }
func (r *recordingSlotRepo) AttachReservationById(id int, res *reservation.Reservation) error {
	return r.inner.AttachReservationById(id, res)
}
func (r *recordingSlotRepo) DettachReservationById(id int) error {
	return r.inner.DettachReservationById(id)
}
func (r *recordingSlotRepo) ChangeVideoById(id int, v *domainVideo.Video) error {
	// 書き込み時間を模擬
	time.Sleep(1 * time.Millisecond)
	return r.inner.ChangeVideoById(id, v)
}
func (r *recordingSlotRepo) SetStateById(id int, st slot.State) error {
	r.record(id, st)
	return r.inner.SetStateById(id, st)
}
func (r *recordingSlotRepo) SetSeqById(id int, seq int) error { return r.inner.SetSeqById(id, seq) }
func (r *recordingSlotRepo) SetWritingFlagById(id int, b bool) error {
	return r.inner.SetWritingFlagById(id, b)
}
func (r *recordingSlotRepo) GetFirstSlotByState(st slot.State) (*slot.Slot, error) {
	return r.inner.GetFirstSlotByState(st)
}
func (r *recordingSlotRepo) FindById(id int) (*slot.Slot, error) { return r.inner.FindById(id) }
func (r *recordingSlotRepo) List() ([]*slot.Slot, error)         { return r.inner.List() }

// ブロッキングできるラッパ（ChangeVideo中に停止し、書き換え中を観測）

type blockingSlotRepo struct {
	inner         slot.Repository
	enteredChange chan struct{}
	resumeChange  chan struct{}
}

func newBlockingSlotRepo(inner slot.Repository) *blockingSlotRepo {
	return &blockingSlotRepo{inner: inner, enteredChange: make(chan struct{}, 1), resumeChange: make(chan struct{})}
}

func (b *blockingSlotRepo) Len() int { return b.inner.Len() }
func (b *blockingSlotRepo) AttachReservationById(id int, res *reservation.Reservation) error {
	return b.inner.AttachReservationById(id, res)
}
func (b *blockingSlotRepo) DettachReservationById(id int) error {
	return b.inner.DettachReservationById(id)
}
func (b *blockingSlotRepo) ChangeVideoById(id int, v *domainVideo.Video) error {
	// 入口通知
	select {
	case b.enteredChange <- struct{}{}:
	default:
	}
	// ブロック
	<-b.resumeChange
	return b.inner.ChangeVideoById(id, v)
}
func (b *blockingSlotRepo) SetStateById(id int, st slot.State) error {
	return b.inner.SetStateById(id, st)
}
func (b *blockingSlotRepo) SetSeqById(id int, seq int) error { return b.inner.SetSeqById(id, seq) }
func (b *blockingSlotRepo) SetWritingFlagById(id int, bflag bool) error {
	return b.inner.SetWritingFlagById(id, bflag)
}
func (b *blockingSlotRepo) GetFirstSlotByState(st slot.State) (*slot.Slot, error) {
	return b.inner.GetFirstSlotByState(st)
}
func (b *blockingSlotRepo) FindById(id int) (*slot.Slot, error) { return b.inner.FindById(id) }
func (b *blockingSlotRepo) List() ([]*slot.Slot, error)         { return b.inner.List() }

func setup(t *testing.T) (*application.MusicService, *eventbus.InMemoryEventBus, orchestrator.Dependencies) {
	videoRepo := fakeVideoRepo{}
	reservationRepo := resInfra.NewMemoryRepository()
	slotRepo := slotInfra.NewMemoryRepository("/tmp/dummy.mp4")

	bus := eventbus.NewInMemoryEventBus()
	service := application.NewMusicService(reservationRepo, slotRepo, videoRepo, bus)

	deps := orchestrator.Dependencies{
		Bus:             bus,
		ReservationRepo: reservationRepo,
		SlotRepo:        slotRepo,
		VideoRepo:       videoRepo,
	}
	_ = orchestrator.New(deps)
	return service, bus, deps
}

func setupWithDownloadRepo(t *testing.T) (*application.MusicService, *eventbus.InMemoryEventBus, orchestrator.Dependencies, *downloadGateVideoRepo) {
	videoRepo := newDownloadGateVideoRepo()
	reservationRepo := resInfra.NewMemoryRepository()
	slotRepo := slotInfra.NewMemoryRepository("/tmp/dummy.mp4")

	bus := eventbus.NewInMemoryEventBus()
	service := application.NewMusicService(reservationRepo, slotRepo, videoRepo, bus)

	deps := orchestrator.Dependencies{
		Bus:             bus,
		ReservationRepo: reservationRepo,
		SlotRepo:        slotRepo,
		VideoRepo:       videoRepo,
	}
	_ = orchestrator.New(deps)
	return service, bus, deps, videoRepo
}

func setupWithRecordingSlotRepo(t *testing.T) (*application.MusicService, *eventbus.InMemoryEventBus, orchestrator.Dependencies, *recordingSlotRepo) {
	videoRepo := fakeVideoRepo{}
	reservationRepo := resInfra.NewMemoryRepository()
	base := slotInfra.NewMemoryRepository("/tmp/dummy.mp4")
	rec := newRecordingSlotRepo(base)

	bus := eventbus.NewInMemoryEventBus()
	service := application.NewMusicService(reservationRepo, rec, videoRepo, bus)

	deps := orchestrator.Dependencies{
		Bus:             bus,
		ReservationRepo: reservationRepo,
		SlotRepo:        rec,
		VideoRepo:       videoRepo,
	}
	_ = orchestrator.New(deps)
	return service, bus, deps, rec
}

func setupWithBlockingSlotRepo(t *testing.T) (*application.MusicService, *eventbus.InMemoryEventBus, orchestrator.Dependencies, *blockingSlotRepo) {
	videoRepo := fakeVideoRepo{}
	reservationRepo := resInfra.NewMemoryRepository()
	base := slotInfra.NewMemoryRepository("/tmp/dummy.mp4")
	blk := newBlockingSlotRepo(base)

	bus := eventbus.NewInMemoryEventBus()
	service := application.NewMusicService(reservationRepo, blk, videoRepo, bus)

	deps := orchestrator.Dependencies{
		Bus:             bus,
		ReservationRepo: reservationRepo,
		SlotRepo:        blk,
		VideoRepo:       videoRepo,
	}
	_ = orchestrator.New(deps)
	return service, bus, deps, blk
}

func TestReservationAttachToSlot(t *testing.T) {
	service, bus, deps := setup(t)

	// 予約イベント
	if err := service.ReserveSong(domainSong.RequestNo("552501")); err != nil {
		t.Fatalf("reserve error: %v", err)
	}
	bus.Wait()

	logStep(t, "Step: Reserve 552501", deps)

	s0, err := deps.SlotRepo.FindById(0)
	if err != nil {
		t.Fatalf("slot0 not found: %v", err)
	}
	if s0.State() != slot.Set {
		t.Fatalf("slot0 state want %s got %s", slot.Set, s0.State())
	}
	if s0.Reservation() == nil {
		t.Fatalf("slot0 reservation not attached")
	}
}

func TestReadingAdvanceTransitions(t *testing.T) {
	service, bus, deps := setup(t)
	// 1件予約してスロットにセット
	if err := service.ReserveSong(domainSong.RequestNo("552501")); err != nil {
		t.Fatalf("reserve error: %v", err)
	}
	bus.Wait()
	logStep(t, "Step: After first reservation", deps)

	// 読み取りが0へ移動
	if err := service.UpdateSlotStateReadingByReadingSlotId(0); err != nil {
		t.Fatalf("reading advance error: %v", err)
	}
	bus.Wait()
	logStep(t, "Step: Reading slot -> 0", deps)

	s0, _ := deps.SlotRepo.FindById(0)
	s1, _ := deps.SlotRepo.FindById(1)
	if s0.State() != slot.Reading {
		t.Fatalf("slot0 should be reading, got %s", s0.State())
	}
	if s1.State() != slot.Locked {
		t.Fatalf("slot1 should be locked, got %s", s1.State())
	}
}

func TestScenario_StateTransitions(t *testing.T) {
	service, bus, deps := setup(t)

	logStep(t, "STEP1: Startup", deps)
	assertSlots(t, deps, []slot.State{slot.Available, slot.Available, slot.Available})
	assertReservationsOrder(t, deps, []string{})
	assertReservationStates(t, deps, []reservation.State{})
	assertSlotVideosReqs(t, deps, []string{"dummy", "dummy", "dummy"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// Step2: 予約1件
	reserve(t, service, bus, "324244")
	logStep(t, "STEP2: Reserve 324244", deps)
	assertSlots(t, deps, []slot.State{slot.Set, slot.Available, slot.Available})
	assertReservationsOrder(t, deps, []string{"324244"})
	assertReservationStates(t, deps, []reservation.State{reservation.SetInSlot})
	assertSlotVideosReqs(t, deps, []string{"324244", "dummy", "dummy"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// Step3: 予約2件目 → 0,1がSet
	reserve(t, service, bus, "321445")
	logStep(t, "STEP3: Reserve 321445", deps)
	assertSlots(t, deps, []slot.State{slot.Set, slot.Set, slot.Available})
	assertReservationsOrder(t, deps, []string{"324244", "321445"})
	assertSlotVideosReqs(t, deps, []string{"324244", "321445", "dummy"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// Step4: 予約3件目 → 0,1,2がSet
	reserve(t, service, bus, "999999")
	logStep(t, "STEP4: Reserve 999999", deps)
	assertSlots(t, deps, []slot.State{slot.Set, slot.Set, slot.Set})
	assertReservationsOrder(t, deps, []string{"324244", "321445", "999999"})
	assertSlotVideosReqs(t, deps, []string{"324244", "321445", "999999"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// Step5: 読み取りが0へ
	reading(t, service, bus, 0)
	logStep(t, "STEP5: Reading -> 0", deps)
	assertSlots(t, deps, []slot.State{slot.Reading, slot.Locked, slot.Set})
	assertReservationsOrder(t, deps, []string{"324244", "321445", "999999"})
	assertSlotVideosReqs(t, deps, []string{"324244", "321445", "999999"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// Step6: 読み取りが1へ（先頭予約デキュー）
	reading(t, service, bus, 1)
	logStep(t, "STEP6: Reading -> 1", deps)
	assertSlots(t, deps, []slot.State{slot.Available, slot.Reading, slot.Locked})
	assertReservationsOrder(t, deps, []string{"321445", "999999"})
	assertSlotVideosReqs(t, deps, []string{"324244", "321445", "999999"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// Step7: 追加予約（きんモザ）→0がSet
	reserve(t, service, bus, "365537")
	logStep(t, "STEP7: Reserve 365537", deps)
	assertSlots(t, deps, []slot.State{slot.Set, slot.Reading, slot.Locked})
	assertReservationsOrder(t, deps, []string{"321445", "999999", "365537"})
	assertSlotVideosReqs(t, deps, []string{"365537", "321445", "999999"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// Step8: 追加予約（のうりん）
	reserve(t, service, bus, "208564")
	logStep(t, "STEP8: Reserve 208564", deps)
	assertSlots(t, deps, []slot.State{slot.Set, slot.Reading, slot.Locked})
	assertReservationsOrder(t, deps, []string{"321445", "999999", "365537", "208564"})
	assertSlotVideosReqs(t, deps, []string{"365537", "321445", "999999"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// Step9: 追加予約（ごちうさ）
	reserve(t, service, bus, "370040")
	logStep(t, "STEP9: Reserve 370040", deps)
	assertSlots(t, deps, []slot.State{slot.Set, slot.Reading, slot.Locked})
	assertReservationsOrder(t, deps, []string{"321445", "999999", "365537", "208564", "370040"})
	assertSlotVideosReqs(t, deps, []string{"365537", "321445", "999999"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// Step10: 読み取りが2へ（先頭予約デキュー、1に208564がセット）
	reading(t, service, bus, 2)
	logStep(t, "STEP10: Reading -> 2", deps)
	assertSlots(t, deps, []slot.State{slot.Locked, slot.Set, slot.Reading})
	assertReservationsOrder(t, deps, []string{"999999", "365537", "208564", "370040"})
	assertSlotVideosReqs(t, deps, []string{"365537", "208564", "999999"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// Step11: 読み取りが0へ（先頭予約デキュー、2に370040がセット）
	reading(t, service, bus, 0)
	logStep(t, "STEP11: Reading -> 0", deps)
	assertSlots(t, deps, []slot.State{slot.Reading, slot.Locked, slot.Set})
	assertReservationsOrder(t, deps, []string{"365537", "208564", "370040"})
	assertSlotVideosReqs(t, deps, []string{"365537", "208564", "370040"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// Step12: 読み取りが1へ（先頭予約デキュー）
	reading(t, service, bus, 1)
	logStep(t, "STEP12: Reading -> 1", deps)
	assertSlots(t, deps, []slot.State{slot.Available, slot.Reading, slot.Locked})
	assertReservationsOrder(t, deps, []string{"208564", "370040"})
	assertSlotVideosReqs(t, deps, []string{"365537", "208564", "370040"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// Step13: 読み取りが2へ（先頭予約デキュー）
	reading(t, service, bus, 2)
	logStep(t, "STEP13: Reading -> 2", deps)
	assertSlots(t, deps, []slot.State{slot.Locked, slot.Available, slot.Reading})
	assertReservationsOrder(t, deps, []string{"370040"})
	assertSlotVideosReqs(t, deps, []string{"365537", "208564", "370040"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})

	// Step14: 読み取りが0へ（先頭予約デキューで空）
	reading(t, service, bus, 0)
	logStep(t, "STEP14: Reading -> 0", deps)
	assertSlots(t, deps, []slot.State{slot.Reading, slot.Locked, slot.Available})
	assertReservationsOrder(t, deps, []string{})
	assertSlotVideosReqs(t, deps, []string{"365537", "208564", "370040"})
	assertSlotVideoStates(t, deps, []domainVideo.State{domainVideo.Ready, domainVideo.Ready, domainVideo.Ready})
}

// --- scenario helpers ---

func reserve(t *testing.T, service *application.MusicService, bus *eventbus.InMemoryEventBus, req string) {
	if err := service.ReserveSong(domainSong.RequestNo(req)); err != nil {
		t.Fatalf("reserve error: %v", err)
	}
	bus.Wait()
}

func reading(t *testing.T, service *application.MusicService, bus *eventbus.InMemoryEventBus, id int) {
	if err := service.UpdateSlotStateReadingByReadingSlotId(id); err != nil {
		t.Fatalf("reading error: %v", err)
	}
	bus.Wait()
}

func assertSlots(t *testing.T, deps orchestrator.Dependencies, want []slot.State) {
	slotsInfo, err := deps.SlotRepo.List()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(slotsInfo) != len(want) {
		t.Fatalf("slot len mismatch: got %d want %d", len(slotsInfo), len(want))
	}
	for i, s := range slotsInfo {
		if s.State() != want[i] {
			t.Fatalf("slot %d state: got %s want %s", i, s.State(), want[i])
		}
	}
}

// --- helpers ---

func logStep(t *testing.T, step string, deps orchestrator.Dependencies) {
	t.Logf("========== %s ==========", step)
	logSlots(t, deps)
	logReservations(t, deps)
}

func logSlots(t *testing.T, deps orchestrator.Dependencies) {
	slotsInfo, err := deps.SlotRepo.List()
	if err != nil {
		t.Logf("[SLOTS] list error: %v", err)
		return
	}
	for _, s := range slotsInfo {
		vidState := "none"
		if s.Video() != nil {
			vidState = string(s.Video().State())
		}
		if s.Reservation() == nil {
			t.Logf("[SLOTS] slot %d: %s (予約なし) video=%v videoState=%s", s.Id(), s.State(), videoPathOrNone(s), vidState)
			continue
		}
		sg, err := s.Reservation().Song()
		if err != nil {
			t.Logf("[SLOTS] slot %d: %s %d (曲取得失敗: %v) video=%v videoState=%s", s.Id(), s.State(), s.Reservation().Seq(), err, videoPathOrNone(s), vidState)
			continue
		}
		t.Logf("[SLOTS] slot %d: %s seq=%d req=%s video=%v videoState=%s", s.Id(), s.State(), s.Reservation().Seq(), sg.RequestNo(), videoPathOrNone(s), vidState)
	}
}

func logReservations(t *testing.T, deps orchestrator.Dependencies) {
	list, err := deps.ReservationRepo.List()
	if err != nil {
		t.Logf("[RES] list error: %v", err)
		return
	}
	if len(list) == 0 {
		t.Log("[RES] (empty)")
	}
	for _, r := range list {
		sg, err := r.Song()
		if err != nil {
			t.Logf("[RES] seq=%d state=%s (曲取得失敗: %v)", r.Seq(), r.State(), err)
			continue
		}
		t.Logf("[RES] seq=%d req=%s state=%s", r.Seq(), sg.RequestNo(), r.State())
	}
}

func videoPathOrNone(s *slot.Slot) string {
	if s.Video() == nil {
		return "<none>"
	}
	return s.Video().Location()
}

func setupWithDownloadAndWrappers(t *testing.T) (*application.MusicService, *eventbus.InMemoryEventBus, orchestrator.Dependencies, *downloadGateVideoRepo, *recordingSlotRepo, *blockingSlotRepo) {
	videoRepo := newDownloadGateVideoRepo()
	reservationRepo := resInfra.NewMemoryRepository()
	base := slotInfra.NewMemoryRepository("/tmp/dummy.mp4")
	rec := newRecordingSlotRepo(base)
	blk := newBlockingSlotRepo(rec)

	bus := eventbus.NewInMemoryEventBus()
	service := application.NewMusicService(reservationRepo, blk, videoRepo, bus)

	deps := orchestrator.Dependencies{
		Bus:             bus,
		ReservationRepo: reservationRepo,
		SlotRepo:        blk,
		VideoRepo:       videoRepo,
	}
	_ = orchestrator.New(deps)
	return service, bus, deps, videoRepo, rec, blk
}

func drainEnteredChange(blk *blockingSlotRepo) {
	for {
		select {
		case <-blk.enteredChange:
			continue
		default:
			return
		}
	}
}

func TestScenario_StateTransitions_WithDownloadWaiting(t *testing.T) {
	service, bus, deps, vrepo, rec, blk := setupWithDownloadAndWrappers(t)

	// STEP1 初期
	logStep(t, "DL-STEP1: Startup (no videos available)", deps)
	assertSlots(t, deps, []slot.State{slot.Available, slot.Available, slot.Available})
	assertReservationsOrder(t, deps, []string{})
	assertReservationStates(t, deps, []reservation.State{})
	assertSlotVideosReqs(t, deps, []string{"dummy", "dummy", "dummy"})

	// STEP2 予約 324244（まだDL不可）
	reserve(t, service, bus, "324244")
	logStep(t, "DL-STEP2: Reserve 324244 (not downloaded)", deps)
	assertReservationsOrder(t, deps, []string{"324244"})
	assertReservationStates(t, deps, []reservation.State{reservation.PreparingVideo})

	// STEP3 324244 ダウンロード完了 -> attachNext で writing に遷移し、ChangeVideo入口でブロック
	vrepo.MakeAvailable("324244")
	bus.Publish(context.Background(), eventbus.VideoDownloaded{ReservationSeq: 0, LocalPath: "324244.mp4"})
	select {
	case <-blk.enteredChange:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("not entered change for 324244")
	}
	// writing観測
	s0, _ := deps.SlotRepo.FindById(0)
	if s0.State() != slot.Writing {
		t.Fatalf("expected slot0 writing, got %s", s0.State())
	}
	assertReservationStates(t, deps, []reservation.State{reservation.WritingToSlot})
	logStep(t, "DL-STEP3: 324244 downloading->writing", deps)
	// ブロック解除→セット完了
	close(blk.resumeChange)
	bus.Wait()
	assertSlots(t, deps, []slot.State{slot.Set, slot.Available, slot.Available})
	assertReservationStates(t, deps, []reservation.State{reservation.SetInSlot})
	assertSlotVideosReqs(t, deps, []string{"324244", "dummy", "dummy"})

	// STEP4 予約 321445（未DL）
	reserve(t, service, bus, "321445")
	logStep(t, "DL-STEP4: Reserve 321445 (not downloaded)", deps)
	assertReservationStates(t, deps, []reservation.State{reservation.SetInSlot, reservation.PreparingVideo})

	// STEP5 321445 ダウンロード完了 -> writing観測
	drainEnteredChange(blk)
	vrepo.MakeAvailable("321445")
	bus.Publish(context.Background(), eventbus.VideoDownloaded{ReservationSeq: 1, LocalPath: "321445.mp4"})
	blk.resumeChange = make(chan struct{})
	select {
	case <-blk.enteredChange:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("not entered change for 321445")
	}
	assertReservationStates(t, deps, []reservation.State{reservation.SetInSlot, reservation.WritingToSlot})
	close(blk.resumeChange)
	bus.Wait()
	assertSlots(t, deps, []slot.State{slot.Set, slot.Set, slot.Available})
	assertReservationStates(t, deps, []reservation.State{reservation.SetInSlot, reservation.SetInSlot})

	// STEP6 予約 999999（未DL）
	reserve(t, service, bus, "999999")
	logStep(t, "DL-STEP6: Reserve 999999 (not downloaded)", deps)
	assertReservationStates(t, deps, []reservation.State{reservation.SetInSlot, reservation.SetInSlot, reservation.PreparingVideo})

	// STEP7 999999 ダウンロード完了 -> writing観測
	drainEnteredChange(blk)
	vrepo.MakeAvailable("999999")
	bus.Publish(context.Background(), eventbus.VideoDownloaded{ReservationSeq: 2, LocalPath: "999999.mp4"})
	blk.resumeChange = make(chan struct{})
	select {
	case <-blk.enteredChange:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("not entered change for 999999")
	}
	assertReservationStates(t, deps, []reservation.State{reservation.SetInSlot, reservation.SetInSlot, reservation.WritingToSlot})
	close(blk.resumeChange)
	bus.Wait()
	assertSlots(t, deps, []slot.State{slot.Set, slot.Set, slot.Set})
	assertReservationStates(t, deps, []reservation.State{reservation.SetInSlot, reservation.SetInSlot, reservation.SetInSlot})

	// STEP8 読み取り進行 -> Releasedが履歴に含まれること
	reading(t, service, bus, 0)
	reading(t, service, bus, 1)
	if !containsOrder(rec.history[0], []slot.State{slot.Released}) {
		t.Fatalf("slot0 history should contain Released, got %v", rec.history[0])
	}
}

func repoMarkDownloadedAndNotify(t *testing.T, vrepo *downloadGateVideoRepo, bus *eventbus.InMemoryEventBus, req string) {
	vrepo.MakeAvailable(req)
	bus.Publish(context.Background(), eventbus.VideoDownloaded{ReservationSeq: 0, LocalPath: req + ".mp4"})
	bus.Wait()
}

func assertReservationsOrder(t *testing.T, deps orchestrator.Dependencies, want []string) {
	list, err := deps.ReservationRepo.List()
	if err != nil {
		t.Fatalf("reservations list error: %v", err)
	}
	if len(list) != len(want) {
		t.Fatalf("reservations len mismatch: got %d want %d", len(list), len(want))
	}
	for i, r := range list {
		sg, err := r.Song()
		if err != nil {
			t.Fatalf("reservation %d song error: %v", i, err)
		}
		if string(sg.RequestNo()) != want[i] {
			t.Fatalf("reservation %d: got %s want %s", i, sg.RequestNo(), want[i])
		}
	}
}

func assertSlotVideosReqs(t *testing.T, deps orchestrator.Dependencies, want []string) {
	slotsInfo, err := deps.SlotRepo.List()
	if err != nil {
		t.Fatalf("slots list error: %v", err)
	}
	if len(slotsInfo) != len(want) {
		t.Fatalf("slots len mismatch: got %d want %d", len(slotsInfo), len(want))
	}
	for i, s := range slotsInfo {
		got := videoReqTagOrDummy(s)
		if got != want[i] {
			t.Fatalf("slot %d video tag: got %s want %s", i, got, want[i])
		}
	}
}

func assertReservationStates(t *testing.T, deps orchestrator.Dependencies, want []reservation.State) {
	list, err := deps.ReservationRepo.List()
	if err != nil {
		t.Fatalf("reservations list error: %v", err)
	}
	if len(list) != len(want) {
		t.Fatalf("reservations len mismatch: got %d want %d", len(list), len(want))
	}
	for i, r := range list {
		if r.State() != want[i] {
			t.Fatalf("reservation %d state: got %s want %s", i, r.State(), want[i])
		}
	}
}

func assertSlotVideoStates(t *testing.T, deps orchestrator.Dependencies, want []domainVideo.State) {
	slotsInfo, err := deps.SlotRepo.List()
	if err != nil {
		t.Fatalf("slots list error: %v", err)
	}
	if len(slotsInfo) != len(want) {
		t.Fatalf("slots len mismatch: got %d want %d", len(slotsInfo), len(want))
	}
	for i, s := range slotsInfo {
		st := domainVideo.Ready
		if s.Video() != nil {
			st = s.Video().State()
		}
		if st != want[i] {
			t.Fatalf("slot %d video state: got %s want %s", i, st, want[i])
		}
	}
}

func videoReqTagOrDummy(s *slot.Slot) string {
	if s.Video() == nil {
		return "none"
	}
	base := filepath.Base(s.Video().Location())
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if strings.HasPrefix(name, "dummy") {
		return "dummy"
	}
	return name
}

func TestWritingStateDuringVideoChange(t *testing.T) {
	service, bus, deps, rec := setupWithRecordingSlotRepo(t)
	reserve(t, service, bus, "324244")

	h := rec.history[0]
	// 期待: 初期available -> writing -> set を含む
	if !containsOrder(h, []slot.State{slot.Available, slot.Writing, slot.Set}) {
		t.Fatalf("slot0 history does not contain Available->Writing->Set, got %v", h)
	}
	logStep(t, "WRITING: after first reservation", deps)
}

func TestWritingStateIsObservable(t *testing.T) {
	service, bus, deps, blk := setupWithBlockingSlotRepo(t)

	// 予約を投げる → attachNext で Writing に遷移後、ChangeVideoでブロック
	go func() { _ = service.ReserveSong(domainSong.RequestNo("324244")) }()
	// ChangeVideo入口を待つ
	select {
	case <-blk.enteredChange:
		// 少し待ってstate反映
		time.Sleep(1 * time.Millisecond)
		logStep(t, "OBSERVE WRITING: during change", deps)
		// スロット0はwriting、予約はwriting_to_slotのはず
		s0, _ := deps.SlotRepo.FindById(0)
		if s0.State() != slot.Writing {
			t.Fatalf("expected slot0 writing, got %s", s0.State())
		}
		resList, _ := deps.ReservationRepo.List()
		if len(resList) < 1 || resList[0].State() != reservation.WritingToSlot {
			t.Fatalf("expected reservation writing_to_slot, got %v", func() reservation.State {
				if len(resList) > 0 {
					return resList[0].State()
				}
				return ""
			}())
		}
		// ブロック解除
		close(blk.resumeChange)
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("did not enter ChangeVideoById in time")
	}

	bus.Wait()
	logStep(t, "OBSERVE WRITING: after change", deps)
	// 最終的にset_in_slotに遷移
	s0, _ := deps.SlotRepo.FindById(0)
	if s0.State() != slot.Set {
		t.Fatalf("expected slot0 set, got %s", s0.State())
	}
	resList, _ := deps.ReservationRepo.List()
	if len(resList) < 1 || resList[0].State() != reservation.SetInSlot {
		t.Fatalf("expected reservation set_in_slot, got %v", func() reservation.State {
			if len(resList) > 0 {
				return resList[0].State()
			}
			return ""
		}())
	}
}

func containsOrder(history []slot.State, seq []slot.State) bool {
	idx := 0
	for _, st := range history {
		if st == seq[idx] {
			idx++
			if idx == len(seq) {
				return true
			}
		}
	}
	return false
}

func TestReservationCreated_WithURL_WaitsUntilDownloaded(t *testing.T) {
	videoRepo := newDownloadGateVideoRepo()
	reservationRepo := resInfra.NewMemoryRepository()
	base := slotInfra.NewMemoryRepository("/tmp/dummy.mp4")

	bus := eventbus.NewInMemoryEventBus()
	service := application.NewMusicService(reservationRepo, base, videoRepo, bus)
	deps := orchestrator.Dependencies{Bus: bus, ReservationRepo: reservationRepo, SlotRepo: base, VideoRepo: videoRepo, DownloadDir: "/tmp"}
	_ = orchestrator.New(deps)

	// URL付きで予約（ダウンロード可能になるまでSetされない）
	_ = service.ReserveSongWithVideo(domainSong.RequestNo("777777"), &application.VideoInfo{Type: "download", Filename: "777777-title.mp4", URL: "http://example.test/777777.mp4"})
	// すぐにはセットされない
	time.Sleep(5 * time.Millisecond)
	s0, _ := deps.SlotRepo.FindById(0)
	if s0.State() != slot.Available {
		t.Fatalf("slot should remain available until download, got %s", s0.State())
	}

	// ダウンロード完了を通知（テスト用ビデオリポジトリに可用化）
	videoRepo.MakeAvailable("777777")
	bus.Publish(context.Background(), eventbus.VideoDownloaded{ReservationSeq: 0, LocalPath: filepath.Join("/tmp", "777777-title.mp4")})
	bus.Wait()

	s0, _ = deps.SlotRepo.FindById(0)
	if s0.State() != slot.Set {
		t.Fatalf("slot should be set after download, got %s", s0.State())
	}
}

func TestDownloadIntegration_WithThrottledHTTP10MB(t *testing.T) {
	// テンポラリな動画ディレクトリとダミー動画を用意
	tmpDir := t.TempDir()
	dummyPath := filepath.Join(tmpDir, "dummy.mp4")
	if err := os.WriteFile(dummyPath, []byte("dummy"), 0644); err != nil {
		t.Fatalf("write dummy: %v", err)
	}

	// 10MBを帯域制限付きで返すハンドラ
	const totalBytes = 10 * 1024 * 1024
	const chunkSize = 64 * 1024        // 64KB/チャンク
	rateBytesPerSec := 5 * 1024 * 1024 // 約5MB/s
	intervalPerChunk := time.Duration(int64(time.Second) * int64(chunkSize) / int64(rateBytesPerSec))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		written := 0
		buf := bytes.Repeat([]byte{'A'}, chunkSize)
		for written < totalBytes {
			n := chunkSize
			if totalBytes-written < chunkSize {
				n = totalBytes - written
			}
			if _, err := w.Write(buf[:n]); err != nil {
				return
			}
			written += n
			time.Sleep(intervalPerChunk)
		}
	}))
	defer ts.Close()

	// 実配線（ストレージビデオリポジトリでダウンロード結果を検出）
	videoRepo := videoInfra.NewStorageRepository(tmpDir, dummyPath)
	reservationRepo := resInfra.NewMemoryRepository()
	slotRepo := slotInfra.NewMemoryRepository(dummyPath)

	bus := eventbus.NewInMemoryEventBus()
	service := application.NewMusicService(reservationRepo, slotRepo, videoRepo, bus)
	deps := orchestrator.Dependencies{Bus: bus, ReservationRepo: reservationRepo, SlotRepo: slotRepo, VideoRepo: videoRepo, DownloadDir: tmpDir}
	_ = orchestrator.New(deps)

	// URL付き予約を投げる
	reqNo := domainSong.RequestNo("123456")
	filename := "123456-title.mp4"
	url := ts.URL + "/video.mp4"
	if err := service.ReserveSongWithVideo(reqNo, &application.VideoInfo{Type: "download", Filename: filename, URL: url}); err != nil {
		t.Fatalf("reserve with video: %v", err)
	}

	// 直後はAvailableのまま（DL待ち）である可能性が高い
	s0, _ := deps.SlotRepo.FindById(0)
	if s0.State() != slot.Available && s0.State() != slot.Set { // 環境によっては即時完了する可能性もあるため許容
		t.Fatalf("unexpected initial slot0 state: %s", s0.State())
	}

	// 完了まで待ち、Setになることを確認
	deadline := time.Now().Add(8 * time.Second)
	for {
		s0, _ = deps.SlotRepo.FindById(0)
		if s0.State() == slot.Set {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for slot0 Set, last state=%s", s0.State())
		}
		time.Sleep(50 * time.Millisecond)
	}

	// 保存先ファイルが存在し、スロットにその動画が紐づいている
	saved := filepath.Join(tmpDir, filename)
	if _, err := os.Stat(saved); err != nil {
		t.Fatalf("downloaded file not found: %v", err)
	}
	if s0.Video() == nil || s0.Video().Location() != saved {
		t.Fatalf("slot video location mismatch: got %v want %v", func() string {
			if s0.Video() != nil {
				return s0.Video().Location()
			}
			return "<nil>"
		}(), saved)
	}
}
