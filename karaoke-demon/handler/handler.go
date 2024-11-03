package handler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const IMAGE_PATH = "/root/karaoke.img"
const VIDEO_DIR = "/home/gpioblink/Downloads/mvideos/karaoke/output"

var FILLER_VIDEOS_PATH = []string{"/dummy/path/filler-video"} // 動画の用意が間に合わない際に利用される

/* カラオケマシンの予約状態管理 */
type ReservedSong struct {
	requestNo string
}

/* FATファイルシステム内のスロット管理 */
type SlotState int

const (
	_             SlotState = iota
	SLOT_FREE               // 書き込み可能
	SLOT_OCCUPIED           // 曲が入っている
	SLOT_LOCKED             // 曲が再生中
)

type SlotSong struct {
	requestNo string
	videoPath string
}

type KaraokeHandler struct {
	reservedSongs []ReservedSong
	slotStates    []SlotState
	slotSongs     []SlotSong
	messageCh     chan string
}

func NewKaraokeHandler(messageCh chan string) *KaraokeHandler {
	return &KaraokeHandler{
		reservedSongs: []ReservedSong{},
		slotSongs:     []SlotSong{{}, {}, {}},
		slotStates:    []SlotState{SLOT_FREE, SLOT_FREE, SLOT_FREE},
		messageCh:     messageCh,
	}
}

func (kh *KaraokeHandler) printHandler() {
	fmt.Println("<Current Status>")
	fmt.Println("ReservedSongs: ", kh.reservedSongs)
	fmt.Println("SlotStates: ", kh.slotStates)
	fmt.Println("SlotSongs: ", kh.slotSongs)
}

func (kh *KaraokeHandler) handleSongAdded(songId string) {
	fmt.Printf("[HandleSongAdded] songId: %s\n", songId)
	// キューに音楽を追加
	kh.reservedSongs = append(kh.reservedSongs, ReservedSong{requestNo: songId})
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
	// TODO: 決め打ちをなくす

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

	// ファイル番号に関するスロットの状態を更新
	if fileIdx >= 0 && fileIdx < len(kh.slotStates) {
		// ファイル番号を元にスロットの状態を書き換え
		kh.slotStates[fileIdx] = SLOT_LOCKED
		// 現在再生中の前の曲を開放
		lastFileIdx := ((fileIdx-1)%3 + 3) % 3
		if kh.slotStates[lastFileIdx] == SLOT_LOCKED {
			kh.removeSongBySongId(kh.slotSongs[lastFileIdx].requestNo)
			kh.slotStates[lastFileIdx] = SLOT_FREE
		}
	}

	// 曲情報のアップデート要求
	kh.printHandler()
	kh.updateFAT()
	kh.printHandler()
}

func (kh *KaraokeHandler) removeSongBySongId(songId string) {
	fmt.Printf("[RemoveSongBySongId] songId: %s\n", songId)
	for i, song := range kh.reservedSongs {
		if song.requestNo == songId {
			kh.reservedSongs = append(kh.reservedSongs[:i], kh.reservedSongs[i+1:]...)
			return
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

func (kh *KaraokeHandler) updateFAT() {
	fmt.Println("[UpdateFAT]")
	// 空きスロットを探す
	for i, slot := range kh.slotStates {
		if slot == SLOT_FREE {
			// 空きスロットに曲を追加
			if len(kh.reservedSongs) > 0 {
				song := kh.reservedSongs[0]
				// 動画ディレクトリ内の選曲番号から始まるファイル名の曲を探す
				filePath, err := findFileWithPrefix(VIDEO_DIR, song.requestNo)
				if err != nil {
					filePath = FILLER_VIDEOS_PATH[0]
				}
				// FATの書き換え
				fmt.Println("Execute:", "makemyfat", "insert", IMAGE_PATH, filePath, strconv.Itoa(i))
				if err := exec.Command("makemyfat", "insert",
					IMAGE_PATH, filePath, strconv.Itoa(i)).Run(); err != nil {
					// イメージファイルの追加に失敗した場合はエラーを出力
					fmt.Printf("Failed to insert video %s to fileNo %d.", filePath, i)
					return
				}
				// スロットの状態を更新
				kh.slotStates[i] = SLOT_OCCUPIED
				kh.slotSongs[i] = SlotSong{requestNo: song.requestNo, videoPath: filePath}
			}
			return
		}
	}
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
