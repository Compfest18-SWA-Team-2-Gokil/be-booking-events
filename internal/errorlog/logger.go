package errorlog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type contextKey struct{}

type errorSlot struct {
	err error
}

type Logger struct {
	mu   sync.Mutex
	file *os.File
}

func NewLogger() (*Logger, error) {
	dir := filepath.Join(".", "log")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("errorlog: gagal buat direktori log: %w", err)
	}

	path := filepath.Join(dir, "error.txt")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("errorlog: gagal buka file log: %w", err)
	}

	return &Logger{file: f}, nil
}

func WithErrorSlot(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey{}, &errorSlot{})
}

func SetError(ctx context.Context, err error) {
	if slot, ok := ctx.Value(contextKey{}).(*errorSlot); ok && slot != nil {
		slot.err = err
	}
}

func GetError(ctx context.Context) error {
	if slot, ok := ctx.Value(contextKey{}).(*errorSlot); ok && slot != nil {
		return slot.err
	}
	return nil
}

func (l *Logger) LogEntry(method, path string, statusCode int, err error, responseBody string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	ts := time.Now().Format("2006-01-02 15:04:05")
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	entry := fmt.Sprintf("[%s] %s %s -> %d | error: %s | response: %s\n", ts, method, path, statusCode, errMsg, responseBody)

	if _, writeErr := io.WriteString(l.file, entry); writeErr != nil {
		slog.Error("errorlog: gagal tulis ke file log", "err", writeErr)
	}
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
