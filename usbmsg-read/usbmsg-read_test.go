package main

import (
	"os"
	"testing"
	"time"
)

func TestWatchKmsg(t *testing.T) {
	messages := make(chan string, 1)

	// Simulate /dev/kmsg content
	kmsgContent := `lun0: file read 16384 @ 0
lun0: file read 16384 @ 16384
lun0: file read 16384 @ 32768
lun0: file read 16384 @ 49152
lun0: file read 16384 @ 65536
lun0: file read 16384 @ 81920
lun0: file read 16384 @ 98304
lun0: file read 16384 @ 114688
lun0: file read 16384 @ 131072
lun0: file read 16384 @ 147456
lun0: file read 16384 @ 163840
lun0: file read 16384 @ 180224
lun0: file read 16384 @ 196608
lun0: file read 16384 @ 212992
lun0: file read 16384 @ 229376
lun0: file read 16384 @ 245760`

	// Create a temporary file to simulate /dev/kmsg
	tmpfile, err := os.CreateTemp("", "kmsg")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(kmsgContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if _, err := tmpfile.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek temp file: %v", err)
	}

	go watchKmsg(messages, tmpfile.Name())

	// Wait for the message to be sent
	select {
	case msg := <-messages:
		expected := "USBMSG_READ 0 262144"
		if msg != expected {
			t.Errorf("Expected message %q, got %q", expected, msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}
