package main

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

	"gpioblink.com/x/karaoke-demon/watcher"
)

const IMAGE_PATH = "/root/karaoke.img"
const IMAGE_SIZE = "2GiB"
const VIDEO_EXT = "mp4"
const VIDEO_NUM = "3"
const VIDEO_SIZE = "512MiB"

const FIFO_PATH = "/tmp/karaoke-fifo"

var FILLER_VIDEOS_PATH = []string{} // 動画の用意が間に合わない際に利用される

/*
1. 赤外線により、カラオケマシンからの予約情報 → reservedSongsに追加
2. reservedSongsの情報を元に楽曲ファイルがなければをダウンロード (現段階では未実装)
3. FATに次に追加する曲としてキューイング →
*/

/* カラオケマシンの予約状態管理 */
type ReservedSong struct {
	requestNo string
	songName  string // 現段階では未使用
}

var reservedSongs = []ReservedSong{}

/* FATファイルシステム内のスロット管理 */
type SlotState int

const (
	_             SlotState = iota
	SLOT_FREE               // 書き込み可能
	SLOT_OCCUPIED           // 曲が入っている
	SLOT_LOCKED             // 曲が再生中
)

var slotStates = []SlotState{}

// -- 以下TODO

/* 次に再生する曲の管理 */
type VideoState int

const (
	_ VideoState = iota
	VIDEO_REQUESTED
	VIDEO_DOWNLOADED
	VIDEO_INSTALLED
	VIDEO_AVAILABLE
)

type KaraokeVideoNode struct {
	VideoPath string
}

func main() {
	// TODO: 各種初期化処理を適切なパッケージ内へ移動
	setupImage()
	setupFIFO()

	msg := make(chan string, 1)

	fifoWatcher, err := watcher.NewFIFOWatcher(FIFO_PATH, msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer fifoWatcher.Close()

	// FIFOファイルを監視
	var wg sync.WaitGroup

	wg.Add(1)
	go printMsg(msg, &wg)

	wg.Add(1)
	go fifoWatcher.Start(&wg)

	wg.Wait()
}

func printMsg(msg <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		fmt.Println(<-msg)
	}
}

func setupImage() {
	// イメージファイルが存在する場合は削除
	if Exists(IMAGE_PATH) {
		os.Remove(IMAGE_PATH)
	}

	// makemyfatコマンドにより空イメージファイルの作成
	// makemyfat create test1.img 2GiB mp4 3 512MiB 1
	if err := exec.Command("makemyfat", "create",
		IMAGE_PATH, IMAGE_SIZE, VIDEO_EXT, VIDEO_NUM, VIDEO_SIZE, "1").Run(); err != nil {
		// イメージファイルの作成に失敗した場合はエラーを出力
		fmt.Println("Failed to create image file.")
		return
	}
}

func setupFIFO() {
	// FIFOファイルが存在する場合は削除し再作成
	if Exists(FIFO_PATH) {
		os.Remove(FIFO_PATH)
	}

	if err := exec.Command("mkfifo", FIFO_PATH).Run(); err != nil {
		fmt.Println("Failed to create fifo file.")
		return
	}
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
