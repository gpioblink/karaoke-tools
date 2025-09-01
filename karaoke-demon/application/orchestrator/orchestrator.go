package orchestrator

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gpioblink.com/x/karaoke-demon/application/eventbus"
	"gpioblink.com/x/karaoke-demon/domain/reservation"
	"gpioblink.com/x/karaoke-demon/domain/slot"
	"gpioblink.com/x/karaoke-demon/domain/song"
	"gpioblink.com/x/karaoke-demon/domain/video"
)

type Dependencies struct {
	Bus             eventbus.EventBus
	ReservationRepo reservation.Repository
	SlotRepo        slot.Repository
	VideoRepo       video.Repository
	DownloadDir     string
}

type Orchestrator struct {
	d Dependencies
}

// New は依存関係を受け取り、イベント購読を登録した Orchestrator を返す
func New(d Dependencies) *Orchestrator {
	o := &Orchestrator{d: d}
	o.wire()
	return o
}

// 互換: 既存呼び出しのための薄いラッパ
func Wire(d Dependencies) { _ = New(d) }

// wire はイベント購読をセットアップする
func (o *Orchestrator) wire() {
	// 予約作成時: キューへ積み、空スロットに割当。
	o.d.Bus.Subscribe(eventbus.EventReservationCreated, func(ctx context.Context, e eventbus.Event) {
		v := e.(eventbus.ReservationCreated)
		eSong, err := song.NewSongInfo(v.SongID)
		if err != nil {
			log.Printf("song creation error: %v", err)
			return
		}
		evideo, err := video.NewNetworkVideo(eSong, v.VideoTitle, v.WithVideoURL)
		if err != nil {
			log.Printf("video creation error: %v", err)
			return
		}
		eSeq, err := o.d.ReservationRepo.EnQueue(evideo)
		if err != nil {
			log.Printf("enqueue error: %v", err)
			return
		}
		// URL付きならダウンロード完了後に割当。URLが無ければ即割当。
		if strings.TrimSpace(v.WithVideoURL) != "" {
			go func(url, title string, seq int) {
				// 保存先ファイル名: 指定タイトルがあればそれを使用。無ければSongID.mp4
				fileName := title
				if strings.TrimSpace(fileName) == "" {
					fileName = v.SongID + ".mp4"
				}
				target := filepath.Join(o.d.DownloadDir, fileName)
				if err := downloadFile(ctx, url, target); err != nil {
					log.Printf("video download failed: %v", err)
					return
				}
				// ダウンロード完了を通知
				o.d.Bus.Publish(context.Background(), eventbus.VideoDownloaded{ReservationSeq: seq, LocalPath: target})
			}(v.WithVideoURL, v.VideoTitle, eSeq)
			return
		}
		o.attachNext()
	})

	// スロットの読み取り進行 -> 前スロット開放、現在Reading、次スロットLock
	o.d.Bus.Subscribe(eventbus.EventSlotReadingAdvanced, func(ctx context.Context, e eventbus.Event) {
		v := e.(eventbus.SlotReadingAdvanced)
		current, _ := o.d.SlotRepo.GetFirstSlotByState(slot.Reading)
		if current != nil && current.Id() == v.ReadingSlotID {
			return
		}
		prevID := calcPositiveModulo(v.ReadingSlotID-1, o.d.SlotRepo.Len())
		// 前スロットを開放済みに
		_ = o.d.SlotRepo.DettachReservationById(prevID)
		_ = o.d.SlotRepo.SetStateById(prevID, slot.Released)
		// 現在を読み取り(=再生中)へ
		_ = o.d.SlotRepo.SetStateById(v.ReadingSlotID, slot.Reading)
		// 次スロットをロック
		nextID := calcPositiveModulo(v.ReadingSlotID+1, o.d.SlotRepo.Len())
		_ = o.d.SlotRepo.SetStateById(nextID, slot.Locked)
		o.attachNext()
	})

	// 動画ダウンロード完了
	o.d.Bus.Subscribe(eventbus.EventVideoDownloaded, func(ctx context.Context, e eventbus.Event) {
		v := e.(eventbus.VideoDownloaded)
		res, err := o.d.ReservationRepo.FindBySeq(v.ReservationSeq)
		if err != nil {
			log.Printf("reservation find error: %v", err)
			return
		}
		vid := res.Video()
		if vid == nil {
			log.Printf("video not found: %v", v.ReservationSeq)
			return
		}
		vid.SetLocation(v.LocalPath)
		vid.SetState(video.Ready)
		// 予約キュー -> スロット割り当ては attachNext に集約
		o.attachNext()
	})
}

func (o *Orchestrator) attachNext() {
	reading, err := o.d.SlotRepo.GetFirstSlotByState(slot.Reading)
	if err != nil {
		reading, err = o.d.SlotRepo.FindById(0)
		if err != nil {
			return
		}
	}
	for i := 0; i < o.d.SlotRepo.Len(); i++ {
		sl, err := o.d.SlotRepo.FindById(calcPositiveModulo(reading.Id()+i, o.d.SlotRepo.Len()))
		if err != nil {
			return
		}
		if sl.State() != slot.Available && sl.State() != slot.Released && sl.State() != slot.Set {
			continue
		}

		// Released は未使用に戻す
		if sl.State() == slot.Released {
			_ = o.d.SlotRepo.SetStateById(sl.Id(), slot.Available)
		}

		res, err := o.d.ReservationRepo.FindByQueueIndex(i)
		if err != nil {
			continue
		}
		// 予約: Preparing -> Writing -> Set
		sg, err := res.Song()
		if err != nil {
			continue
		}
		vd, err := o.d.VideoRepo.FindByRequestNo(string(sg.RequestNo()))
		if err != nil {
			continue
		}

		// すでに同じ予約曲/同じ動画がセット済みなら再書き込みしない
		if sl.State() == slot.Set && sl.Reservation() != nil {
			if curSong, err := sl.Reservation().Song(); err == nil && curSong.RequestNo() == sg.RequestNo() {
				if sl.Video() != nil && sl.Video().Location() == vd.Location() {
					continue
				}
			}
		}

		res.SetState(reservation.WritingToSlot)
		_ = o.d.SlotRepo.AttachReservationById(sl.Id(), res)
		_ = o.d.SlotRepo.SetStateById(sl.Id(), slot.Writing)
		o.d.SlotRepo.ChangeVideoById(sl.Id(), vd)
		_ = o.d.SlotRepo.SetStateById(sl.Id(), slot.Set)
		res.SetState(reservation.SetInSlot)
	}
}

func calcPositiveModulo(a, b int) int { return (a%b + b) % b }

func downloadFile(ctx context.Context, url string, dest string) error {
	// 親ディレクトリ作成
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	// GET
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed: status %s", resp.Status)
	}
	// 書き出し
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
