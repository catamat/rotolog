package rotolog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileDaysRotator defines rotator structure.
type FileDaysRotator struct {
	mu             sync.Mutex
	fileNameFormat string
	fileSuffix     string
	folder         string
	file           *os.File
	maxDays        int
	currDate       string
	closed         bool
	now            func() time.Time
}

// NewFileDaysRotator creates a new rotator that deletes logs after their life exceeds N days.
func NewFileDaysRotator(folder string, days int) (r *FileDaysRotator, err error) {
	if days <= 0 {
		return nil, fmt.Errorf("days must be greater than zero")
	}

	if err := os.MkdirAll(folder, os.ModePerm); err != nil {
		return nil, err
	}

	r = &FileDaysRotator{
		fileNameFormat: "2006-01-02",
		fileSuffix:     ".log",
		folder:         folder,
		maxDays:        days,
		now:            time.Now,
	}

	err = r.delete()
	if err != nil {
		return nil, err
	}

	err = r.rotate()
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Write implements Writer interface.
func (r *FileDaysRotator) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return 0, ErrClosed
	}

	if err := r.rotate(); err != nil {
		return 0, err
	}

	n, err = r.file.Write(p)

	return n, err
}

// rotate handles file rotation and deletes old logs.
func (r *FileDaysRotator) rotate() error {
	if r.closed {
		return ErrClosed
	}

	now := r.currentTime()
	currDate := now.Format(r.fileNameFormat)

	if r.file != nil && r.currDate == currDate {
		return nil
	}

	if r.file != nil {
		if err := r.file.Close(); err != nil {
			return err
		}
		r.file = nil
	}

	if err := r.delete(); err != nil {
		return err
	}

	logName := currDate + r.fileSuffix
	fileName := filepath.Join(r.folder, logName)
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	r.currDate = currDate
	r.file = file

	return nil
}

// delete deletes old logs.
func (r *FileDaysRotator) delete() error {
	now := r.currentTime()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	cutoff := today.AddDate(0, 0, -r.maxDays)
	dateLength := len(now.Format(r.fileNameFormat))

	entries, err := os.ReadDir(r.folder)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != r.fileSuffix {
			continue
		}

		if len(name) != dateLength+len(r.fileSuffix) {
			continue
		}

		logDate, err := time.ParseInLocation(r.fileNameFormat, name[:dateLength], now.Location())
		if err != nil {
			continue
		}

		if !logDate.After(cutoff) {
			fileName := filepath.Join(r.folder, name)
			if err := os.Remove(fileName); err != nil {
				return err
			}
		}
	}

	return nil
}

// Close closes the active log file.
func (r *FileDaysRotator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}

	r.closed = true

	if r.file == nil {
		return nil
	}

	err := r.file.Close()
	r.file = nil

	return err
}

func (r *FileDaysRotator) currentTime() time.Time {
	if r.now != nil {
		return r.now()
	}

	return time.Now()
}
