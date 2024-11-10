package main

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

	"gpioblink.com/x/karaoke-demon/config"
	"gpioblink.com/x/karaoke-demon/handler"
	"gpioblink.com/x/karaoke-demon/watcher"
)

const IMAGE_SIZE = "1.8GiB"
const VIDEO_EXT = "mp4"
const VIDEO_NUM = "3"
const VIDEO_SIZE = "512MiB"

func setupImage(imagePath string, dummyFilePath string) {
	fmt.Println("Setup image file...")

	// イメージファイルが存在する場合は削除
	if Exists(imagePath) {
		os.Remove(imagePath)
	}

	// makemyfatコマンドにより空イメージファイルの作成
	// makemyfat create test1.img 2GiB mp4 3 512MiB 1
	if err := exec.Command("makemyfat", "create",
		imagePath, IMAGE_SIZE, VIDEO_EXT, VIDEO_NUM, VIDEO_SIZE, "1").Run(); err != nil {
		// イメージファイルの作成に失敗した場合はエラーを出力
		fmt.Println("Failed to create image file.")
		return
	}

	fmt.Println("Insert initial video files...")

	// ビデオ数の分だけダミーファイルを書き込み
	for i := 0; i < 3; i++ {
		if err := exec.Command("makemyfat", "insert",
			imagePath, dummyFilePath, fmt.Sprintf("%d", i)).Run(); err != nil {
			// イメージファイルの追加に失敗した場合はエラーを出力
			fmt.Println("Failed to insert video.")
			return
		}
	}
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func main() {
	fifoCh := make(chan string, 1)
	bleCh := make(chan string, 1)

	// FIXME: この値をそのままexec.Commandに渡しているのを修正したい
	conf, err := config.NewConfig()
	if err != nil {
		fmt.Println(err)
		return
	}

	// TODO: 各種初期化処理を適切なパッケージ内へ移動
	setupImage(conf.IMAGE_PATH, conf.FILLER_VIDEOS_PATH[0])
	// setupFIFO(conf.FIFO_PATH)

	fifoWatcher, err := watcher.NewFIFOWatcher(conf.FIFO_PATH, fifoCh)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer fifoWatcher.Close()

	karaokeHandler := handler.NewKaraokeHandler(fifoCh, conf)

	// FIFO監視とハンドラを同時に起動
	var wg sync.WaitGroup

	wg.Add(1)
	fmt.Println("Start fifo watcher...")
	go karaokeHandler.Start(&wg)

	wg.Add(1)
	fmt.Println("Start karaoke handler...")
	go fifoWatcher.Start(&wg)

	wg.Wait()
}
