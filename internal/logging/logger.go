package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

var (
	initOnce sync.Once
	initErr  error

	logger *slog.Logger
	path   string
)

func Init() error {
	initOnce.Do(func() {
		exePath, err := os.Executable()
		if err != nil {
			initErr = fmt.Errorf("resolve executable path: %w", err)
			return
		}
		exeDir := filepath.Dir(exePath)
		path = filepath.Join(exeDir, "logging.log")

		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			initErr = fmt.Errorf("open log file %s: %w", path, err)
			return
		}

		handler := slog.NewTextHandler(io.Writer(f), &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		logger = slog.New(handler)
		logger.Info("logger initialized", "path", path)
	})
	return initErr
}

func Path() string {
	return path
}

func Debug(msg string, args ...any) {
	if logger == nil {
		return
	}
	logger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	if logger == nil {
		return
	}
	logger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	if logger == nil {
		return
	}
	logger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	if logger == nil {
		return
	}
	logger.Error(msg, args...)
}
