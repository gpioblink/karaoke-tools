package logging

import (
	"io"
	"log"
	"os"
	"sync"
	"time"
)

const defaultCapacity = 1024

var (
	defaultCollector = NewCollector(defaultCapacity)
	setupOnce        sync.Once
)

// Entry represents a single log line captured by the collector.
type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

// Collector stores log entries in a fixed-size circular buffer.
type Collector struct {
	mu       sync.RWMutex
	entries  []Entry
	head     int
	size     int
	capacity int
}

// NewCollector creates a Collector that retains up to capacity log entries.
func NewCollector(capacity int) *Collector {
	if capacity <= 0 {
		capacity = defaultCapacity
	}

	return &Collector{
		entries:  make([]Entry, capacity),
		capacity: capacity,
	}
}

// Write implements io.Writer so Collector can be used as a logging sink.
func (c *Collector) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	start := 0
	for start < len(p) {
		// Locate newline to split log lines.
		end := indexByte(p, '\n', start)
		var line string
		if end == -1 {
			if start == len(p) {
				break
			}
			line = string(p[start:])
			start = len(p)
		} else {
			line = string(p[start:end])
			start = end + 1
		}

		if line == "" {
			continue
		}

		c.appendLocked(Entry{
			Timestamp: now,
			Message:   line,
		})
	}

	return len(p), nil
}

func (c *Collector) appendLocked(entry Entry) {
	index := (c.head + c.size) % c.capacity
	c.entries[index] = entry

	if c.size < c.capacity {
		c.size++
		return
	}

	c.head = (c.head + 1) % c.capacity
}

// Snapshot returns up to limit most recent log entries in chronological order.
// Use limit <= 0 to retrieve all retained entries.
func (c *Collector) Snapshot(limit int) []Entry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.size == 0 {
		return nil
	}

	if limit <= 0 || limit > c.size {
		limit = c.size
	}

	result := make([]Entry, limit)
	start := (c.head + c.size - limit) % c.capacity
	for i := 0; i < limit; i++ {
		idx := (start + i) % c.capacity
		result[i] = c.entries[idx]
	}

	return result
}

// Add appends a log message directly to the collector.
func (c *Collector) Add(message string) {
	if message == "" {
		return
	}
	c.mu.Lock()
	c.appendLocked(Entry{
		Timestamp: time.Now(),
		Message:   message,
	})
	c.mu.Unlock()
}

// indexByte finds the index of target starting from offset start.
func indexByte(data []byte, target byte, start int) int {
	for i := start; i < len(data); i++ {
		if data[i] == target {
			return i
		}
	}
	return -1
}

// Setup redirects the standard logger to capture output while still writing to stderr.
func Setup() {
	setupOnce.Do(func() {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
		log.SetOutput(io.MultiWriter(os.Stderr, defaultCollector))
	})
}

// Add appends a message to the default collector.
func Add(message string) {
	defaultCollector.Add(message)
}

// Snapshot returns up to limit log entries from the default collector.
func Snapshot(limit int) []Entry {
	return defaultCollector.Snapshot(limit)
}

// CollectorWriter exposes the default collector as an io.Writer.
func CollectorWriter() io.Writer {
	return defaultCollector
}
