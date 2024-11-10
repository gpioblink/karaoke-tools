package fifo

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"gpioblink.com/x/karaoke-demon-clean/application"
	"gpioblink.com/x/karaoke-demon-clean/interface/handler"
)

type FifoInterface struct {
	router       map[string]handler.HandlerFunc
	watcher      *fsnotify.Watcher
	fifoFile     *os.File
	musicService application.MusicService
	doChan       chan string
}

var DefaultRouter = map[string]handler.HandlerFunc{
	"REMOTE_SONG": handler.ReserveSong,
	"USBMSG_READ": handler.UpdateReading,
}

func NewFifoInterface(service application.MusicService, router map[string]handler.HandlerFunc, fifoPath string) (*FifoInterface, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	if err := watcher.Add(fifoPath); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to add FIFO to watcher: %w", err)
	}

	fifo, err := os.OpenFile(fifoPath, os.O_RDONLY|syscall.O_NONBLOCK, os.ModeNamedPipe)
	if err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to open FIFO: %w", err)
	}

	fifoInterface := &FifoInterface{
		router:       DefaultRouter,
		watcher:      watcher,
		fifoFile:     fifo,
		musicService: service,
		doChan:       make(chan string),
	}

	go fifoInterface.processDoChan()

	return fifoInterface, nil
}

func (f *FifoInterface) Run() {
	for {
		select {
		case event, ok := <-f.watcher.Events:
			if !ok {
				return
			}

			// 書き込みイベントが発生した場合にデータを読み取る
			if event.Op&fsnotify.Write == fsnotify.Write {
				scanner := bufio.NewScanner(f.fifoFile)
				for scanner.Scan() {
					line := scanner.Text()
					f.doChan <- line
					fmt.Printf("Received: %s\n", line)
				}
				if err := scanner.Err(); err != nil {
					log.Printf("Error reading from FIFO: %v", err)
				}
			}

		case err, ok := <-f.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func (f *FifoInterface) processDoChan() {
	for line := range f.doChan {
		ctx := context.Background()
		f.Do(ctx, line)
	}
}

func (f *FifoInterface) Do(ctx context.Context, line string) {
	cmd := strings.Split(line, " ")
	action := cmd[0]

	if handlerFunc, ok := f.router[action]; ok {
		handlerFunc(ctx, f.musicService, *handler.NewRequest(action, cmd[1:]))
	} else {
		log.Printf("Unknown command: %s", action)
	}
}
