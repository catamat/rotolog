package rotolog

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestFileDaysRotatorRotatesWhenMonthChangesOnSameDay(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	r := &FileDaysRotator{
		fileNameFormat: "2006-01-02",
		fileSuffix:     ".log",
		folder:         dir,
		maxDays:        30,
		now: func() time.Time {
			return now
		},
	}

	if err := r.rotate(); err != nil {
		t.Fatalf("rotate() error = %v", err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	firstName := filepath.Base(r.file.Name())
	if firstName != "2026-03-10.log" {
		t.Fatalf("expected first log file to be 2026-03-10.log, got %q", firstName)
	}

	now = time.Date(2026, time.April, 10, 8, 0, 0, 0, time.UTC)

	if _, err := r.Write([]byte("next month")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	secondName := filepath.Base(r.file.Name())
	if secondName != "2026-04-10.log" {
		t.Fatalf("expected log rotation to create 2026-04-10.log, got %q", secondName)
	}
}

func TestFileDaysRotatorDeleteSkipsNonManagedFilesAndSubdirs(t *testing.T) {
	dir := t.TempDir()
	nestedDir := filepath.Join(dir, "nested")

	if err := os.Mkdir(nestedDir, 0755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	oldManagedLog := filepath.Join(dir, "2026-01-01.log")
	foreignLog := filepath.Join(dir, "app.log")
	nestedLog := filepath.Join(nestedDir, "2026-01-01.log")

	for _, file := range []string{oldManagedLog, foreignLog, nestedLog} {
		if err := os.WriteFile(file, []byte("log"), 0644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", file, err)
		}
	}

	r := &FileDaysRotator{
		fileNameFormat: "2006-01-02",
		fileSuffix:     ".log",
		folder:         dir,
		maxDays:        3,
		now: func() time.Time {
			return time.Date(2026, time.January, 10, 12, 0, 0, 0, time.UTC)
		},
	}

	if err := r.delete(); err != nil {
		t.Fatalf("delete() error = %v", err)
	}

	if _, err := os.Stat(oldManagedLog); !os.IsNotExist(err) {
		t.Fatalf("expected managed log %q to be deleted, err = %v", oldManagedLog, err)
	}

	if _, err := os.Stat(foreignLog); err != nil {
		t.Fatalf("expected foreign log %q to be preserved, err = %v", foreignLog, err)
	}

	if _, err := os.Stat(nestedLog); err != nil {
		t.Fatalf("expected nested log %q to be preserved, err = %v", nestedLog, err)
	}
}

func TestFileDaysRotatorDeleteUsesCalendarDaysAcrossDST(t *testing.T) {
	dir := t.TempDir()
	loc, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Skipf("timezone data not available: %v", err)
	}

	dstBoundaryLog := filepath.Join(dir, "2026-03-28.log")
	recentLog := filepath.Join(dir, "2026-03-29.log")

	for _, file := range []string{dstBoundaryLog, recentLog} {
		if err := os.WriteFile(file, []byte("log"), 0644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", file, err)
		}
	}

	r := &FileDaysRotator{
		fileNameFormat: "2006-01-02",
		fileSuffix:     ".log",
		folder:         dir,
		maxDays:        3,
		now: func() time.Time {
			return time.Date(2026, time.March, 31, 12, 0, 0, 0, loc)
		},
	}

	if err := r.delete(); err != nil {
		t.Fatalf("delete() error = %v", err)
	}

	if _, err := os.Stat(dstBoundaryLog); !os.IsNotExist(err) {
		t.Fatalf("expected DST boundary log %q to be deleted, err = %v", dstBoundaryLog, err)
	}

	if _, err := os.Stat(recentLog); err != nil {
		t.Fatalf("expected recent log %q to be preserved, err = %v", recentLog, err)
	}
}

func TestNewFileDaysRotatorStartupDeletesExpiredFilesAndAppendsToToday(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	oldLog := filepath.Join(dir, now.AddDate(0, 0, -10).Format("2006-01-02")+".log")
	todayLog := filepath.Join(dir, now.Format("2006-01-02")+".log")
	foreignLog := filepath.Join(dir, "app.log")

	if err := os.WriteFile(oldLog, []byte("old"), 0644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", oldLog, err)
	}
	if err := os.WriteFile(todayLog, []byte("existing\n"), 0644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", todayLog, err)
	}
	if err := os.WriteFile(foreignLog, []byte("keep"), 0644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", foreignLog, err)
	}

	r, err := NewFileDaysRotator(dir, 3)
	if err != nil {
		t.Fatalf("NewFileDaysRotator() error = %v", err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	if _, err := r.Write([]byte("new\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if _, err := os.Stat(oldLog); !os.IsNotExist(err) {
		t.Fatalf("expected expired log %q to be deleted, err = %v", oldLog, err)
	}

	if _, err := os.Stat(foreignLog); err != nil {
		t.Fatalf("expected foreign log %q to be preserved, err = %v", foreignLog, err)
	}

	data, err := os.ReadFile(todayLog)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", todayLog, err)
	}

	if string(data) != "existing\nnew\n" {
		t.Fatalf("expected constructor to append to today's log, got %q", data)
	}
}

func TestFileDaysRotatorWriteReturnsRotateError(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")

	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	r := &FileDaysRotator{
		fileNameFormat: "2006-01-02",
		fileSuffix:     ".log",
		folder:         blocker,
		maxDays:        1,
		now: func() time.Time {
			return time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)
		},
	}

	if _, err := r.Write([]byte("hello")); err == nil {
		t.Fatal("expected Write() to return the rotation error")
	}
}

func TestFileDaysRotatorWriteAfterCloseReturnsErrClosed(t *testing.T) {
	dir := t.TempDir()

	r, err := NewFileDaysRotator(dir, 7)
	if err != nil {
		t.Fatalf("NewFileDaysRotator() error = %v", err)
	}

	if err := r.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if _, err := r.Write([]byte("after close")); !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}

func TestFileDaysRotatorConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	r := &FileDaysRotator{
		fileNameFormat: "2006-01-02",
		fileSuffix:     ".log",
		folder:         dir,
		maxDays:        30,
		now: func() time.Time {
			return now
		},
	}

	if err := r.rotate(); err != nil {
		t.Fatalf("rotate() error = %v", err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	const goroutines = 12
	const writesPerGoroutine = 200
	payload := []byte("days-concurrent-log-line\n")

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

	logName := filepath.Join(dir, "2026-03-10.log")
	data, err := os.ReadFile(logName)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	wantBytes := goroutines * writesPerGoroutine * len(payload)
	if len(data) != wantBytes {
		t.Fatalf("expected %d bytes in %q, got %d", wantBytes, logName, len(data))
	}
}
