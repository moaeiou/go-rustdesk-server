package transport

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

const (
	maxLogSize = 10 << 20
	logBackups = 3
)

var (
	Logger    *log.Logger
	logOutput *rotatingLog
)

type rotatingLog struct {
	mu   sync.Mutex
	path string
	file *os.File
}

func (w *rotatingLog) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if info, err := w.file.Stat(); err == nil && info.Size()+int64(len(data)) > maxLogSize {
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
	_ = os.Remove(w.path + "." + strconv.Itoa(logBackups))
	for index := logBackups - 1; index >= 1; index-- {
		_ = os.Rename(w.path+"."+strconv.Itoa(index), w.path+"."+strconv.Itoa(index+1))
	}
	_ = os.Rename(w.path, w.path+".1")
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err == nil {
		w.file = file
	}
	return err
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
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	logOutput = &rotatingLog{path: logPath, file: f}
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
