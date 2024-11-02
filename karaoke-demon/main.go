package main

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

	"gpioblink.com/x/karaoke-demon/handler"
	"gpioblink.com/x/karaoke-demon/watcher"
)

const IMAGE_PATH = "/root/karaoke.img"
const IMAGE_SIZE = "2GiB"
const VIDEO_EXT = "mp4"
const VIDEO_NUM = "3"
const VIDEO_SIZE = "512MiB"

const FIFO_PATH = "/tmp/karaoke-fifo"

/*
1. 赤外線により、カラオケマシンからの予約情報 → reservedSongsに追加
2. reservedSongsの情報を元に楽曲ファイルがなければをダウンロード (現段階では未実装)
3. FATに次に追加する曲としてキューイング →
*/

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

func main() {
	msg := make(chan string, 1)

	// TODO: 各種初期化処理を適切なパッケージ内へ移動
	setupImage()
	setupFIFO()

	fifoWatcher, err := watcher.NewFIFOWatcher(FIFO_PATH, msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer fifoWatcher.Close()

	karaokeHandler := handler.NewKaraokeHandler(msg)

	// FIFO監視とハンドラを同時に起動
	var wg sync.WaitGroup

	wg.Add(1)
	go karaokeHandler.Start(&wg)

	wg.Add(1)
	go fifoWatcher.Start(&wg)

	wg.Wait()
}
