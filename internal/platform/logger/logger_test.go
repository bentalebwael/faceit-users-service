package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/bentalebwael/faceit-users-service/internal/config"
)

// captureOutput replaces os.Stdout with a buffer and returns the buffer
func captureOutput(f func()) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = oldStdout

	_, err = buf.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name          string
		logLevel      string
		wantLevel     slog.Level
		wantAddSource bool
	}{
		{
			name:          "debug level",
			logLevel:      "debug",
			wantLevel:     slog.LevelDebug,
			wantAddSource: true,
		},
		{
			name:          "info level",
			logLevel:      "info",
			wantLevel:     slog.LevelInfo,
			wantAddSource: false,
		},
		{
			name:          "warn level",
			logLevel:      "warn",
			wantLevel:     slog.LevelWarn,
			wantAddSource: false,
		},
		{
			name:          "error level",
			logLevel:      "error",
			wantLevel:     slog.LevelError,
			wantAddSource: false,
		},
		{
			name:          "invalid level defaults to info",
			logLevel:      "invalid",
			wantLevel:     slog.LevelInfo,
			wantAddSource: false,
		},
		{
			name:          "empty level defaults to info",
			logLevel:      "",
			wantLevel:     slog.LevelInfo,
			wantAddSource: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Log: config.LogConfig{
					Level: tt.logLevel,
				},
			}

			buf, err := captureOutput(func() {
				logger := NewLogger(cfg)

				logger.Debug("debug message")
				logger.Info("info message")
				logger.Warn("warn message")
				logger.Error("error message")
			})
			if err != nil {
				t.Fatalf("Failed to capture output: %v", err)
			}

			lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
			for _, line := range lines {
				var logEntry map[string]interface{}
				if err := json.Unmarshal(line, &logEntry); err != nil {
					t.Errorf("Failed to parse log line as JSON: %v", err)
					continue
				}

				if level, ok := logEntry["level"]; ok {
					levelStr, ok := level.(string)
					if !ok {
						t.Errorf("Level is not a string: %v", level)
						continue
					}

					// Check if messages below configured level are not logged
					switch levelStr {
					case "DEBUG":
						if tt.wantLevel > slog.LevelDebug {
							t.Error("Debug message was logged when it should not have been")
						}
					case "INFO":
						if tt.wantLevel > slog.LevelInfo {
							t.Error("Info message was logged when it should not have been")
						}
					case "WARN":
						if tt.wantLevel > slog.LevelWarn {
							t.Error("Warn message was logged when it should not have been")
						}
					case "ERROR":
						if tt.wantLevel > slog.LevelError {
							t.Error("Error message was logged when it should not have been")
						}
					}
				}

				if tt.wantAddSource {
					if _, hasSource := logEntry["source"]; !hasSource {
						t.Error("Source information missing in debug level log")
					}
				}

				if _, hasTime := logEntry["time"]; !hasTime {
					t.Error("Time field missing in log entry")
				}
				if _, hasMsg := logEntry["msg"]; !hasMsg {
					t.Error("Message field missing in log entry")
				}
			}
		})
	}
}

func TestLoggerOutput(t *testing.T) {
	cfg := &config.Config{
		Log: config.LogConfig{
			Level: "info",
		},
	}

	buf, err := captureOutput(func() {
		logger := NewLogger(cfg)
		logger.Info("test message",
			"string_key", "string_value",
			"int_key", 42,
			"bool_key", true,
		)
	})
	if err != nil {
		t.Fatalf("Failed to capture output: %v", err)
	}

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	if len(lines) < 2 {
		t.Fatal("Expected at least 2 log entries (init and test message)")
	}

	// Use the last line which should be our test message
	var logEntry map[string]interface{}
	if err := json.Unmarshal(lines[len(lines)-1], &logEntry); err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v", err)
	}

	tests := []struct {
		key      string
		wantType string
		want     interface{}
	}{
		{"string_key", "string", "string_value"},
		{"int_key", "float64", float64(42)}, // JSON numbers are float64
		{"bool_key", "bool", true},
		{"level", "string", "INFO"},
		{"msg", "string", "test message"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			value, ok := logEntry[tt.key]
			if !ok {
				t.Errorf("Key %s not found in log entry", tt.key)
				return
			}

			valueType := typeOf(value)
			if valueType != tt.wantType {
				t.Errorf("Key %s: got type %s, want %s", tt.key, valueType, tt.wantType)
			}

			if value != tt.want {
				t.Errorf("Key %s: got value %v, want %v", tt.key, value, tt.want)
			}
		})
	}
}

func typeOf(v interface{}) string {
	switch v.(type) {
	case string:
		return "string"
	case float64:
		return "float64"
	case bool:
		return "bool"
	default:
		return "unknown"
	}
}
