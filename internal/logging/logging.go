package logging

import (
	"io"
	"log"
	"os"
	"sync"
)

var (
	mu      sync.Mutex
	logger  *log.Logger
	enabled bool
	file    *os.File
)

func Init(path string) error {
	mu.Lock()
	defer mu.Unlock()

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	if file != nil {
		file.Close()
	}

	file = f
	logger = log.New(f, "", log.LstdFlags|log.Lmicroseconds)
	enabled = true
	return nil
}

func Debug(format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()
	if enabled && logger != nil {
		logger.Printf(format, args...)
	}
}

func Writer() io.Writer {
	mu.Lock()
	defer mu.Unlock()
	if enabled && file != nil {
		return file
	}
	return io.Discard
}

func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return enabled
}

func Close() {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		file.Close()
		file = nil
	}
	enabled = false
	logger = nil
}
