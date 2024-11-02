package watcher

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/fsnotify/fsnotify"
)

type FIFOWatcher struct {
	watcher   *fsnotify.Watcher
	fifo      *os.File
	messageCh chan string
}

func NewFIFOWatcher(fifoPath string, messageCh chan string) (*FIFOWatcher, error) {
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

	return &FIFOWatcher{
		watcher:   watcher,
		fifo:      fifo,
		messageCh: make(chan string),
	}, nil
}

func (fw *FIFOWatcher) StartWatchingFIFO() error {
	go func() {
		defer fw.watcher.Close()
		for {
			log.Printf("Waiting for events...")
			select {
			case event, ok := <-fw.watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				// 書き込みイベントが発生した場合にデータを読み取る
				if event.Op&fsnotify.Write == fsnotify.Write {
					fw.readFIFOData(fw.fifo)
				}

			case err, ok := <-fw.watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)
			}
		}
	}()

	select {}
}

func (fw *FIFOWatcher) Close() {
	fw.watcher.Close()
	fw.fifo.Close()
	close(fw.messageCh)
}

func (fw *FIFOWatcher) readFIFOData(fifo *os.File) {
	scanner := bufio.NewScanner(fifo)
	for scanner.Scan() {
		line := scanner.Text()
		fw.messageCh <- line
		fmt.Printf("Received: %s\n", line)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from FIFO: %v", err)
	}
}
