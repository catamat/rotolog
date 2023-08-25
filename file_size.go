package rotolog

import (
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
	maxSize        int
	halfSize       int
	currSize       int
}

// NewFileSizeRotator creates a new rotator which deletes logs after their size exceeds N megabytes.
func NewFileSizeRotator(folder string, size int) (r *FileSizeRotator, err error) {
	os.MkdirAll(folder, os.ModePerm)

	r = &FileSizeRotator{
		fileNameFormat: "half-",
		fileSuffix:     ".log",
		folder:         folder,
		maxSize:        size * 1000000,
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

	r.rotate()
	n, err = r.file.Write(p)

	return n, err
}

// rotate handles file rotation and deletes big logs.
func (r *FileSizeRotator) rotate() error {
	if r.currSize >= r.halfSize {
		if r.file != nil {
			r.file.Close()
			r.delete()
		}
	}

	logName := r.fileNameFormat + "1" + r.fileSuffix
	fileName := filepath.Join(r.folder, logName)
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	halfStat, err := os.Stat(fileName)
	if err != nil {
		return err
	}

	r.currSize = int(halfStat.Size())
	r.file = file

	return nil
}

// delete deletes big logs.
func (r *FileSizeRotator) delete() error {
	file1Name := filepath.Join(r.folder, r.fileNameFormat+"1"+r.fileSuffix)
	file2Name := filepath.Join(r.folder, r.fileNameFormat+"2"+r.fileSuffix)

	halfStat, err := os.Stat(file1Name)
	if err != nil {
		return nil
	}

	if int(halfStat.Size()) >= r.halfSize {
		os.Remove(file2Name)

		err = os.Rename(file1Name, file2Name)
		if err != nil {
			return err
		}
	}

	return nil
}
