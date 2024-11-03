package handler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gpioblink.com/x/karaoke-demon/config"
	"gpioblink.com/x/karaoke-demon/models"
)

type KaraokeHandler struct {
	reservedSongs models.ReservedSongs
	slot          models.Slot
	messageCh     chan string
	conf          *config.Config
}

func NewKaraokeHandler(messageCh chan string, conf *config.Config) *KaraokeHandler {
	return &KaraokeHandler{
		reservedSongs: models.NewReservedSongs(),
		slot:          models.NewSlot(3), // TODO: とりあえず3スロットに決め打ち
		messageCh:     messageCh,
		conf:          conf,
	}
}

func (kh *KaraokeHandler) printHandler() {
	fmt.Println("<Current Status>")
	fmt.Println("ReservedSongs: ", kh.reservedSongs)
	fmt.Println("Slot: ", kh.slot)
}

func (kh *KaraokeHandler) handleSongAdded(songId string) {
	fmt.Printf("[HandleSongAdded] songId: %s\n", songId)
	// キューに音楽を追加
	kh.reservedSongs.AddSong(songId)
	kh.printHandler()
	kh.updateFAT()
	kh.printHandler()
}

/*
$ makemyfat create test1.img 2GiB mp4 3 512MiB 1
imagePath test1.img, fileSize 2147483648, fileExt mp4, numOfFiles 3, eachFileSize 536870912, isMBR true
***** Root File List (MBR Shifted) *****
0       MP4[536870912bytes]: LBA 0x00002814-0x00102814 0x0000000000502800-0x0000000020502800 clus=3
1       MP4[536870912bytes]: LBA 0x00102814-0x00202814 0x0000000020502800-0x0000000040502800 clus=131075
2       MP4[536870912bytes]: LBA 0x00202814-0x00302814 0x0000000040502800-0x0000000060502800 clus=262147
*/
func (kh *KaraokeHandler) handleMsgRead(addr uint64) {
	fmt.Printf("[HandleMsgRead] addr: %d\n", addr)
	// TODO: ハードコードによる決め打ちをなくす
	// アドレスを元にファイル番号を特定
	fileIdx := -1
	if addr > 0x00002814 && addr < 0x00102814 {
		fileIdx = 0
	} else if addr < 0x00202814 {
		fileIdx = 1
	} else if addr < 0x00302814 {
		fileIdx = 2
	} else {
		return
	}

	if fileIdx > 0 {
		// ファイル番号に関するスロットの状態を更新
		currentIdx := fileIdx
		lastIdx := ((fileIdx-1)%3 + 3) % 3

		kh.slot.UpdateSlotState(currentIdx, models.SLOT_LOCKED)

		// 現在が再生中の場合かつ、前の曲が再生終了している場合、前の曲は再生終了しているので解放処理
		if kh.slot.GetSlotState(lastIdx) == models.SLOT_LOCKED {
			lastSong := kh.slot.GetSlotSong(lastIdx)
			kh.reservedSongs.RemoveSongBySeq(lastSong.GetSeq())
			kh.slot.UpdateSlotState(lastIdx, models.SLOT_FREE)
		}
	}

	// 曲情報のアップデート要求
	kh.updateFAT()
	kh.printHandler()
}

func (kh *KaraokeHandler) updateFAT() {
	fmt.Println("[UpdateFAT]")
	// 空きスロットを探す
	freeSlotNum, err := kh.slot.FindNextFreeSlot()
	if err != nil {
		return
	}

	// 予約された曲の中から次に再生可能な曲を探す
	song, err := kh.reservedSongs.FindNextAttachableSong()
	if err != nil {
		return
	}

	// 動画ディレクトリ内の選曲番号から始まるファイル名の曲を探す
	filePath, err := findFileWithPrefix(kh.conf.VIDEO_DIR, song.GetRequestNo())
	if err != nil {
		filePath = kh.conf.FILLER_VIDEOS_PATH[0]
	}

	// FATの書き換え
	fmt.Println("Execute:", "makemyfat", "insert", kh.conf.IMAGE_PATH, filePath, fmt.Sprintf("%d", freeSlotNum))
	if err := exec.Command("makemyfat", "insert",
		kh.conf.IMAGE_PATH, filePath, fmt.Sprintf("%d", freeSlotNum)).Run(); err != nil {
		// イメージファイルの追加に失敗した場合はエラーを出力
		fmt.Printf("Failed to insert video %s to fileNo %d.\n", filePath, freeSlotNum)
		return
	}

	// スロットの状態を更新
	kh.slot.UpdateSlotState(freeSlotNum, models.SLOT_OCCUPIED)
	kh.slot.UpdateSlotSong(freeSlotNum, models.NewSlotSong(song.GetRequestNo(), filePath, song.GetSeq()))

	kh.reservedSongs.AttachBySeq(song.GetSeq())
}

func (kh *KaraokeHandler) Start(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		res := strings.Split(<-kh.messageCh, " ")

		command := res[0]
		switch command {
		case "REMOTE_SONG":
			songId := res[1]
			kh.handleSongAdded(songId)
		case "USBMSG_READ":
			addr, err := strconv.ParseUint(res[1], 10, 64)
			if err != nil {
				continue
			}
			// length, err := strconv.ParseUint(res[2], 10, 64)
			// if err != nil {
			// 	continue
			// }
			kh.handleMsgRead(addr)
		}
	}
}

func findFileWithPrefix(dir string, prefix string) (string, error) {
	fmt.Printf("[FindFileWithPrefix] dir: %s, prefix: %s\n", dir, prefix)
	// ディレクトリ内のファイルを取得
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	// ファイル名がprefixから始まるファイルを探す
	for _, entry := range files {
		if strings.HasPrefix(entry.Name(), prefix) {
			return filepath.Join(dir, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("file starting with %s is not found", prefix)
}
