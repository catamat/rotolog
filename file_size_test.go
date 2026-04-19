package rotolog

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestFileSizeRotatorKeepsOpenFileBelowThreshold(t *testing.T) {
	dir := t.TempDir()

	r, err := NewFileSizeRotator(dir, 1)
	if err != nil {
		t.Fatalf("NewFileSizeRotator() error = %v", err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	firstFile := r.file
	if firstFile == nil {
		t.Fatal("expected an open log file after initialization")
	}

	for i := 0; i < 3; i++ {
		if _, err := r.Write([]byte("small log line")); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		if r.file != firstFile {
			t.Fatal("expected writes below the threshold to reuse the current file handle")
		}
	}
}

func TestFileSizeRotatorWriteReturnsRotateError(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")

	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	r := &FileSizeRotator{
		fileNameFormat: "half-",
		fileSuffix:     ".log",
		folder:         blocker,
		halfSize:       10,
	}

	if _, err := r.Write([]byte("hello")); err == nil {
		t.Fatal("expected Write() to return the rotation error")
	}
}

func TestFileSizeRotatorWriteAfterCloseReturnsErrClosed(t *testing.T) {
	dir := t.TempDir()

	r, err := NewFileSizeRotator(dir, 1)
	if err != nil {
		t.Fatalf("NewFileSizeRotator() error = %v", err)
	}

	if err := r.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if _, err := r.Write([]byte("after close")); !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}

func TestNewFileSizeRotatorStartupRollsOversizedHalf1(t *testing.T) {
	dir := t.TempDir()
	oversized := bytes.Repeat([]byte("a"), 600000)

	half1 := filepath.Join(dir, "half-1.log")
	half2 := filepath.Join(dir, "half-2.log")

	if err := os.WriteFile(half1, oversized, 0644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", half1, err)
	}
	if err := os.WriteFile(half2, []byte("stale"), 0644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", half2, err)
	}

	r, err := NewFileSizeRotator(dir, 1)
	if err != nil {
		t.Fatalf("NewFileSizeRotator() error = %v", err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	if _, err := r.Write([]byte("fresh")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	half2Data, err := os.ReadFile(half2)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", half2, err)
	}
	if !bytes.Equal(half2Data, oversized) {
		t.Fatal("expected oversized half-1.log to be preserved as half-2.log on startup")
	}

	half1Data, err := os.ReadFile(half1)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", half1, err)
	}
	if string(half1Data) != "fresh" {
		t.Fatalf("expected startup to create a fresh half-1.log, got %q", half1Data)
	}
}

func TestFileSizeRotatorConcurrentWrites(t *testing.T) {
	dir := t.TempDir()

	r, err := NewFileSizeRotator(dir, 2)
	if err != nil {
		t.Fatalf("NewFileSizeRotator() error = %v", err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	const goroutines = 12
	const writesPerGoroutine = 200
	payload := []byte("size-concurrent-log-line\n")

	var wg sync.WaitGroup
	errCh := make(chan error, goroutines*writesPerGoroutine)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < writesPerGoroutine; j++ {
				if _, err := r.Write(payload); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent Write() error = %v", err)
		}
	}

	data, err := os.ReadFile(filepath.Join(dir, "half-1.log"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	wantBytes := goroutines * writesPerGoroutine * len(payload)
	if len(data) != wantBytes {
		t.Fatalf("expected %d bytes in half-1.log, got %d", wantBytes, len(data))
	}
}
