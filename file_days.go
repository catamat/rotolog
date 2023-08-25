package rotolog

import (
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
	currDay        int
}

// NewFileDaysRotator creates a new rotator that deletes logs after their life exceeds N days.
func NewFileDaysRotator(folder string, days int) (r *FileDaysRotator, err error) {
	os.MkdirAll(folder, os.ModePerm)

	r = &FileDaysRotator{
		fileNameFormat: "2006-01-02",
		fileSuffix:     ".log",
		folder:         folder,
		maxDays:        days,
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

	r.rotate()
	n, err = r.file.Write(p)

	return n, err
}

// rotate handles file rotation and deletes old logs.
func (r *FileDaysRotator) rotate() error {
	if r.currDay != time.Now().Day() {
		if r.file != nil {
			r.file.Close()
			r.delete()
		}

		logName := time.Now().Format(r.fileNameFormat) + r.fileSuffix
		fileName := filepath.Join(r.folder, logName)
		file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}

		r.currDay = time.Now().Day()
		r.file = file
	}

	return nil
}

// delete deletes old logs.
func (r *FileDaysRotator) delete() error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	filepath.Walk(r.folder, func(path string, file os.FileInfo, _ error) error {
		if filepath.Ext(path) == r.fileSuffix {
			logDate, err := time.Parse(r.fileNameFormat, file.Name()[:10])
			if err != nil {
				return err
			}

			daysDiff := int(today.Sub(logDate).Hours() / 24)

			if daysDiff >= r.maxDays {
				fileName := filepath.Join(r.folder, file.Name())
				err := os.Remove(fileName)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})

	return nil
}
