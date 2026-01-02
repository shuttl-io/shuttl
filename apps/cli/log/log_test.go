package log

import (
	"strings"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	logger := New(100)

	if logger == nil {
		t.Fatal("New returned nil")
	}

	if logger.maxSize != 100 {
		t.Errorf("Expected maxSize 100, got %d", logger.maxSize)
	}

	if len(logger.entries) != 0 {
		t.Errorf("Expected empty entries, got %d", len(logger.entries))
	}
}

func TestLog(t *testing.T) {
	logger := New(100)

	t.Run("logs entry with level and message", func(t *testing.T) {
		logger.Log("INFO", "test message")

		entries := logger.GetEntries()
		if len(entries) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(entries))
		}

		if entries[0].Level != "INFO" {
			t.Errorf("Expected level 'INFO', got '%s'", entries[0].Level)
		}

		if entries[0].Message != "test message" {
			t.Errorf("Expected message 'test message', got '%s'", entries[0].Message)
		}

		if entries[0].Timestamp.IsZero() {
			t.Error("Expected non-zero timestamp")
		}
	})

	t.Run("supports format arguments", func(t *testing.T) {
		logger := New(100)
		logger.Log("DEBUG", "value: %d, name: %s", 42, "test")

		entries := logger.GetEntries()
		if len(entries) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(entries))
		}

		if entries[0].Message != "value: 42, name: test" {
			t.Errorf("Expected formatted message, got '%s'", entries[0].Message)
		}
	})
}

func TestMaxSize(t *testing.T) {
	logger := New(3)

	// Log more than max size
	logger.Log("INFO", "message 1")
	logger.Log("INFO", "message 2")
	logger.Log("INFO", "message 3")
	logger.Log("INFO", "message 4")
	logger.Log("INFO", "message 5")

	entries := logger.GetEntries()
	if len(entries) != 3 {
		t.Fatalf("Expected 3 entries (max size), got %d", len(entries))
	}

	// Should have kept the last 3
	if entries[0].Message != "message 3" {
		t.Errorf("Expected first entry 'message 3', got '%s'", entries[0].Message)
	}

	if entries[2].Message != "message 5" {
		t.Errorf("Expected last entry 'message 5', got '%s'", entries[2].Message)
	}
}

func TestGetEntries(t *testing.T) {
	logger := New(100)

	logger.Log("INFO", "test1")
	logger.Log("DEBUG", "test2")

	entries := logger.GetEntries()

	// Verify we get a copy, not the original
	entries[0].Message = "modified"

	originalEntries := logger.GetEntries()
	if originalEntries[0].Message == "modified" {
		t.Error("GetEntries should return a copy, not the original slice")
	}
}

func TestClear(t *testing.T) {
	logger := New(100)

	logger.Log("INFO", "test1")
	logger.Log("INFO", "test2")

	if logger.Len() != 2 {
		t.Fatalf("Expected 2 entries, got %d", logger.Len())
	}

	logger.Clear()

	if logger.Len() != 0 {
		t.Errorf("Expected 0 entries after Clear, got %d", logger.Len())
	}
}

func TestLen(t *testing.T) {
	logger := New(100)

	if logger.Len() != 0 {
		t.Errorf("Expected Len 0, got %d", logger.Len())
	}

	logger.Log("INFO", "test")

	if logger.Len() != 1 {
		t.Errorf("Expected Len 1, got %d", logger.Len())
	}

	logger.Log("INFO", "test2")
	logger.Log("INFO", "test3")

	if logger.Len() != 3 {
		t.Errorf("Expected Len 3, got %d", logger.Len())
	}
}

func TestFormat(t *testing.T) {
	t.Run("empty logger returns empty string", func(t *testing.T) {
		logger := New(100)
		result := logger.Format()

		if result != "" {
			t.Errorf("Expected empty string, got '%s'", result)
		}
	})

	t.Run("formats entries correctly", func(t *testing.T) {
		logger := New(100)
		logger.Log("INFO", "test message")
		logger.Log("DEBUG", "debug message")

		result := logger.Format()

		if !strings.Contains(result, "[INFO ]") {
			t.Error("Expected formatted output to contain '[INFO ]'")
		}

		if !strings.Contains(result, "test message") {
			t.Error("Expected formatted output to contain 'test message'")
		}

		if !strings.Contains(result, "[DEBUG]") {
			t.Error("Expected formatted output to contain '[DEBUG]'")
		}

		if !strings.Contains(result, "debug message") {
			t.Error("Expected formatted output to contain 'debug message'")
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	logger := New(1000)
	var wg sync.WaitGroup

	// Spawn multiple goroutines to log concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				logger.Log("INFO", "goroutine %d, iteration %d", id, j)
			}
		}(i)
	}

	wg.Wait()

	// Should have 1000 entries
	if logger.Len() != 1000 {
		t.Errorf("Expected 1000 entries, got %d", logger.Len())
	}
}

func TestGlobalFunctions(t *testing.T) {
	// Save and restore the default logger
	originalDefault := Default
	defer func() { Default = originalDefault }()

	// Create a fresh logger for testing
	Default = New(100)

	t.Run("Info logs at INFO level", func(t *testing.T) {
		Default.Clear()
		Info("info message %d", 1)

		entries := Default.GetEntries()
		if len(entries) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(entries))
		}

		if entries[0].Level != "INFO" {
			t.Errorf("Expected level 'INFO', got '%s'", entries[0].Level)
		}

		if entries[0].Message != "info message 1" {
			t.Errorf("Expected message 'info message 1', got '%s'", entries[0].Message)
		}
	})

	t.Run("Debug logs at DEBUG level", func(t *testing.T) {
		Default.Clear()
		Debug("debug message")

		entries := Default.GetEntries()
		if len(entries) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(entries))
		}

		if entries[0].Level != "DEBUG" {
			t.Errorf("Expected level 'DEBUG', got '%s'", entries[0].Level)
		}
	})

	t.Run("DebugWithPrefix adds prefix", func(t *testing.T) {
		Default.Clear()
		DebugWithPrefix("PREFIX", "prefixed message")

		entries := Default.GetEntries()
		if len(entries) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(entries))
		}

		if !strings.Contains(entries[0].Message, "[PREFIX]") {
			t.Errorf("Expected message to contain '[PREFIX]', got '%s'", entries[0].Message)
		}
	})

	t.Run("Warn logs at WARN level", func(t *testing.T) {
		Default.Clear()
		Warn("warning message")

		entries := Default.GetEntries()
		if len(entries) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(entries))
		}

		if entries[0].Level != "WARN" {
			t.Errorf("Expected level 'WARN', got '%s'", entries[0].Level)
		}
	})

	t.Run("Error logs at ERROR level", func(t *testing.T) {
		Default.Clear()
		Error("error message")

		entries := Default.GetEntries()
		if len(entries) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(entries))
		}

		if entries[0].Level != "ERROR" {
			t.Errorf("Expected level 'ERROR', got '%s'", entries[0].Level)
		}
	})

	t.Run("IPC logs at IPC level", func(t *testing.T) {
		Default.Clear()
		IPC("ipc message")

		entries := Default.GetEntries()
		if len(entries) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(entries))
		}

		if entries[0].Level != "IPC" {
			t.Errorf("Expected level 'IPC', got '%s'", entries[0].Level)
		}
	})
}

func TestNilDefaultLogger(t *testing.T) {
	// Save and restore the default logger
	originalDefault := Default
	defer func() { Default = originalDefault }()

	Default = nil

	// These should not panic
	Info("test")
	Debug("test")
	DebugWithPrefix("PREFIX", "test")
	Warn("test")
	Error("test")
	IPC("test")
}

func TestDefaultLoggerInitialized(t *testing.T) {
	// Verify the default logger is initialized
	if Default == nil {
		t.Fatal("Default logger should be initialized")
	}

	if Default.maxSize != 1000 {
		t.Errorf("Expected default maxSize 1000, got %d", Default.maxSize)
	}
}




