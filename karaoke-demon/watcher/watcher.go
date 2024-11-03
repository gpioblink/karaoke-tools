package watcher

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sync"
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
		messageCh: messageCh,
	}, nil
}

func (fw *FIFOWatcher) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			// log.Println("event:", event)
			// 書き込みイベントが発生した場合にデータを読み取る
			if event.Op&fsnotify.Write == fsnotify.Write {
				fw.readFIFOData()
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func (fw *FIFOWatcher) Close() {
	fmt.Println("Closing watcher...")
	fw.watcher.Close()
	fw.fifo.Close()
	close(fw.messageCh)
}

func (fw *FIFOWatcher) readFIFOData() {
	scanner := bufio.NewScanner(fw.fifo)
	for scanner.Scan() {
		line := scanner.Text()
		fw.messageCh <- line
		fmt.Printf("Received: %s\n", line)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from FIFO: %v", err)
	}
}
