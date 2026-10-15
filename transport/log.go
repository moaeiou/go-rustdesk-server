package transport

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

var (
	Logger    *log.Logger
	logOutput *rotatingLog
)

type rotatingLog struct {
	mu       sync.Mutex
	path     string
	file     *os.File
	maxSize  int64
	maxFiles int
}

func (w *rotatingLog) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if info, err := w.file.Stat(); err == nil && info.Size()+int64(len(data)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	return w.file.Write(data)
}

func (w *rotatingLog) rotate() error {
	if err := w.file.Close(); err != nil {
		return err
	}
	if err := w.destroyExpiredBackups(); err != nil {
		return err
	}
	backupCount := w.maxFiles - 1
	for index := backupCount - 1; index >= 1; index-- {
		source := w.path + "." + strconv.Itoa(index)
		destination := w.path + "." + strconv.Itoa(index+1)
		if err := os.Rename(source, destination); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if backupCount > 0 {
		if err := os.Rename(w.path, w.path+".1"); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err == nil {
		w.file = file
	}
	return err
}

func (w *rotatingLog) destroyExpiredBackups() error {
	matches, err := filepath.Glob(w.path + ".*")
	if err != nil {
		return err
	}
	for _, path := range matches {
		index, err := strconv.Atoi(strings.TrimPrefix(path, w.path+"."))
		if err != nil || index < w.maxFiles {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (w *rotatingLog) Close() error { w.mu.Lock(); defer w.mu.Unlock(); return w.file.Close() }

func Info(format string, v ...any) {
	if Config.LogLevel == "warn" || Config.LogLevel == "error" {
		return
	}
	msg := fmt.Sprintf("[INFO] "+format, v...)
	logToFile(msg)
	printColored(colorGreen, msg)
}

func Warn(format string, v ...any) {
	if Config.LogLevel == "error" {
		return
	}
	msg := fmt.Sprintf("[WARN] "+format, v...)
	logToFile(msg)
	printColored(colorYellow, msg)
}

func printColored(color, message string) {
	if Config.LogColor {
		fmt.Println(color + message + colorReset)
		return
	}
	fmt.Println(message)
}

func Log() error {
	logPath := Config.LogPath
	if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	logOutput = &rotatingLog{
		path: logPath, file: f, maxSize: Config.MaxLogSize, maxFiles: Config.MaxLogFiles,
	}
	Logger = log.New(logOutput, "", log.LstdFlags)
	return nil
}

func CloseLog() {
	if logOutput != nil {
		_ = logOutput.Close()
	}
}

func logToFile(msg string) {
	if Logger != nil {
		Logger.Println(msg)
	}
}
