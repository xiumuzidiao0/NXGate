package stats

import (
	"fmt"
	"sync"
	"time"
)

type LogLevel string

const (
	LogLevelInfo    LogLevel = "INFO"
	LogLevelWarning LogLevel = "WARNING"
	LogLevelError   LogLevel = "ERROR"
)

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Module    string    `json:"module"`
	Message   string    `json:"message"`
}

type RingLog struct {
	mu          sync.RWMutex
	capacity    int
	entries     []LogEntry
	head        int
	count       int
	subscribers map[chan LogEntry]struct{}
}

var globalRingLog *RingLog

func InitRingLog(capacity int) *RingLog {
	if capacity <= 0 {
		capacity = 500
	}
	r := &RingLog{
		capacity:    capacity,
		entries:     make([]LogEntry, capacity),
		subscribers: make(map[chan LogEntry]struct{}),
	}
	globalRingLog = r
	return r
}

func GetRingLog() *RingLog {
	if globalRingLog == nil {
		return InitRingLog(500)
	}
	return globalRingLog
}

func (r *RingLog) Log(level LogLevel, module, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Module:    module,
		Message:   msg,
	}

	// Print to stdout as well
	fmt.Printf("[%s] [%s] [%s] %s\n", entry.Timestamp.Format("15:04:05"), entry.Level, entry.Module, entry.Message)

	r.mu.Lock()
	r.entries[r.head] = entry
	r.head = (r.head + 1) % r.capacity
	if r.count < r.capacity {
		r.count++
	}

	// Broadcast to active SSE/WebSocket subscribers
	for ch := range r.subscribers {
		select {
		case ch <- entry:
		default:
			// Non-blocking if subscriber buffer is full
		}
	}
	r.mu.Unlock()
}

func (r *RingLog) Recent(limit int) []LogEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.count == 0 {
		return []LogEntry{}
	}
	if limit <= 0 || limit > r.count {
		limit = r.count
	}
	res := make([]LogEntry, limit)
	start := (r.head - limit + r.capacity) % r.capacity
	if start+limit <= r.capacity {
		copy(res, r.entries[start:start+limit])
	} else {
		first := r.capacity - start
		copy(res[:first], r.entries[start:])
		copy(res[first:], r.entries[:limit-first])
	}
	return res
}

func (r *RingLog) Subscribe() chan LogEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch := make(chan LogEntry, 64)
	r.subscribers[ch] = struct{}{}
	return ch
}

func (r *RingLog) Unsubscribe(ch chan LogEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.subscribers, ch)
	close(ch)
}

func LogInfo(module, format string, args ...any) {
	GetRingLog().Log(LogLevelInfo, module, format, args...)
}

func LogWarn(module, format string, args ...any) {
	GetRingLog().Log(LogLevelWarning, module, format, args...)
}

func LogError(module, format string, args ...any) {
	GetRingLog().Log(LogLevelError, module, format, args...)
}
