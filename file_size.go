package rotolog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileSizeRotator defines rotator structure.
type FileSizeRotator struct {
	mu             sync.Mutex
	fileNameFormat string
	fileSuffix     string
	folder         string
	file           *os.File
	halfSize       int
	currSize       int
	closed         bool
}

// NewFileSizeRotator creates a new rotator which deletes logs after their size exceeds N megabytes.
func NewFileSizeRotator(folder string, size int) (r *FileSizeRotator, err error) {
	if size <= 0 {
		return nil, fmt.Errorf("size must be greater than zero")
	}

	if err := os.MkdirAll(folder, os.ModePerm); err != nil {
		return nil, err
	}

	r = &FileSizeRotator{
		fileNameFormat: "half-",
		fileSuffix:     ".log",
		folder:         folder,
		halfSize:       size * 1000000 / 2,
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
func (r *FileSizeRotator) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return 0, ErrClosed
	}

	if err := r.rotate(); err != nil {
		return 0, err
	}

	n, err = r.file.Write(p)
	r.currSize += n

	return n, err
}

// rotate handles file rotation and deletes big logs.
func (r *FileSizeRotator) rotate() error {
	if r.closed {
		return ErrClosed
	}

	if r.file != nil && r.currSize < r.halfSize {
		return nil
	}

	if r.file != nil {
		if err := r.file.Close(); err != nil {
			return err
		}
		r.file = nil

		if err := r.delete(); err != nil {
			return err
		}
	}

	return r.openCurrentFile()
}

func (r *FileSizeRotator) openCurrentFile() error {
	logName := r.fileNameFormat + "1" + r.fileSuffix
	fileName := filepath.Join(r.folder, logName)
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}

	r.currSize = int(stat.Size())
	r.file = file

	return nil
}

// delete deletes big logs.
func (r *FileSizeRotator) delete() error {
	file1Name := filepath.Join(r.folder, r.fileNameFormat+"1"+r.fileSuffix)
	file2Name := filepath.Join(r.folder, r.fileNameFormat+"2"+r.fileSuffix)

	halfStat, err := os.Stat(file1Name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if int(halfStat.Size()) >= r.halfSize {
		if err := os.Remove(file2Name); err != nil && !os.IsNotExist(err) {
			return err
		}

		err = os.Rename(file1Name, file2Name)
		if err != nil {
			return err
		}
	}

	return nil
}

// Close closes the active log file.
func (r *FileSizeRotator) Close() error {
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
