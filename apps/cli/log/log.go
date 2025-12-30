package log

import (
	"fmt"
	"sync"
	"time"
)

// Entry represents a debug log entry
type Entry struct {
	Timestamp time.Time
	Level     string
	Message   string
}

// Logger is a thread-safe logger that stores logs in a buffer
type Logger struct {
	entries []Entry
	mu      sync.RWMutex
	maxSize int
	mode    LogMode
	level   LogLevel
}

type LogLevel string

var logLevels = map[LogLevel]int{
	LogLevelIPC:   0,
	LogLevelDebug: 1,
	LogLevelInfo:  2,
	LogLevelWarn:  3,
	LogLevelError: 4,
}

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelIPC   LogLevel = "IPC"
)

type LogMode int

const (
	LogToConsole LogMode = iota
	LogToFile
	LogToEntries
)

var (
	// Default is the global debug logger
	Default *Logger
)

func init() {
	// Initialize the global logger
	Default = New(1000)
}

// New creates a new debug logger with the specified max size
func New(maxSize int) *Logger {
	return &Logger{
		entries: make([]Entry, 0, maxSize),
		maxSize: maxSize,
		mode:    LogToConsole,
		level:   LogLevelInfo,
	}
}

func (l *Logger) SetMode(mode LogMode) {
	l.mode = mode
}

func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// Log adds a log entry with the given level
func (l *Logger) Log(level, format string, args ...interface{}) {
	if l.mode == LogToConsole {
		if logLevels[LogLevel(level)] < logLevels[l.level] {
			return
		}
		fmt.Printf("[%s] [%-5s] %s\n",
			time.Now().Format("15:04:05.000"),
			level,
			fmt.Sprintf(format, args...),
		)
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := Entry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   fmt.Sprintf(format, args...),
	}

	l.entries = append(l.entries, entry)

	// Trim if over max size
	if len(l.entries) > l.maxSize {
		l.entries = l.entries[len(l.entries)-l.maxSize:]
	}
}

// GetEntries returns a copy of all log entries
func (l *Logger) GetEntries() []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	entries := make([]Entry, len(l.entries))
	copy(entries, l.entries)
	return entries
}

// Clear clears all log entries
func (l *Logger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = l.entries[:0]
}

// Len returns the number of log entries
func (l *Logger) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.entries)
}

// Format formats all log entries as a string for output
func (l *Logger) Format() string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(l.entries) == 0 {
		return ""
	}

	var result string
	for _, entry := range l.entries {
		result += fmt.Sprintf("[%s] [%-5s] %s\n",
			entry.Timestamp.Format("15:04:05.000"),
			entry.Level,
			entry.Message,
		)
	}
	return result
}

// Info logs a message at info level using the default logger
func Info(format string, args ...interface{}) {
	if Default != nil {
		Default.Log("INFO", format, args...)
	}
}

// Debug logs a message at debug level using the default logger
func Debug(format string, args ...interface{}) {
	if Default != nil {
		Default.Log("DEBUG", format, args...)
	}
}

func DebugWithPrefix(prefix, format string, args ...interface{}) {
	if Default != nil {
		Default.Log("DEBUG", "[%s] %s", prefix, fmt.Sprintf(format, args...))
	}
}

// Warn logs a message at warning level using the default logger
func Warn(format string, args ...interface{}) {
	if Default != nil {
		Default.Log("WARN", format, args...)
	}
}

// Error logs a message at error level using the default logger
func Error(format string, args ...interface{}) {
	if Default != nil {
		Default.Log("ERROR", format, args...)
	}
}

// IPC logs IPC-related messages using the default logger
func IPC(format string, args ...interface{}) {
	if Default != nil {
		Default.Log("IPC", format, args...)
	}
}
