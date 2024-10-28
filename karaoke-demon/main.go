package main

import (
	"fmt"
	"os"
	"os/exec"
)

const IMAGE_PATH = "/root/karaoke.img"
const CTRL_FIFO = "/tmp/karaoke-fifo"

func main() {
	messageChannel := make(chan string)

	// イメージファイルが存在する場合は削除
	if Exists(IMAGE_PATH) {
		os.Remove(IMAGE_PATH)
	}

	// makemyfatコマンドにより空イメージファイルの作成
	cmd := "makemyfat"
	// FIXME: change args
	args := []string{IMAGE_PATH}
	if err := exec.Command(cmd, args...).Run(); err != nil {
		// イメージファイルの作成に失敗した場合はエラーを出力
		fmt.Println("Failed to create image file.")
		return
	}

	// FIFOファイルが存在する場合は削除し再作成
	if Exists(CTRL_FIFO) {
		os.Remove(CTRL_FIFO)
	}

	if err := exec.Command("mkfifo", CTRL_FIFO).Run(); err != nil {
		fmt.Println("Failed to create fifo file.")
		return
	}

	// FIFOファイルの読み込み
	for {
		file, err := os.OpenFile(CTRL_FIFO, os.O_RDONLY, 0666)
		if err != nil {
			fmt.Println("Failed to open fifo file.")
			return
		}

		// メッセージの読み込み
		buf := make([]byte, 1024)
		n, err := file.Read(buf)
		if err != nil {
			fmt.Println("Failed to read message.")
			return
		}

		// メッセージの表示
		message := string(buf[:n])
		fmt.Println("Received message: " + message)

		// メッセージの内容に合わせてハンドラをchannelで実行

	}

}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
