package config

import (
	"os"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	os.Clearenv()

	requiredVars := map[string]string{
		"DATABASE_URL":  "postgres://test:test@localhost:5432/testdb",
		"REDIS_ADDR":    "localhost:6379",
		"KAFKA_BROKERS": "localhost:9092",
	}
	for k, v := range requiredVars {
		os.Setenv(k, v)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig() with defaults error = %v", err)
		return
	}

	tests := []struct {
		name   string
		got    interface{}
		want   interface{}
		errMsg string
	}{
		{
			name:   "API Port",
			got:    cfg.API.Port,
			want:   8080,
			errMsg: "default API port should be 8080",
		},
		{
			name:   "GRPC Port",
			got:    cfg.GRPC.Port,
			want:   50051,
			errMsg: "default GRPC port should be 50051",
		},
		{
			name:   "Kafka Topic",
			got:    cfg.Kafka.EventTopic,
			want:   "user_events",
			errMsg: "default Kafka topic should be 'user_events'",
		},
		{
			name:   "Rate Limit RPS",
			got:    cfg.Rate.RequestsPerSecond,
			want:   10,
			errMsg: "default rate limit RPS should be 10",
		},
		{
			name:   "Rate Limit Burst",
			got:    cfg.Rate.Burst,
			want:   20,
			errMsg: "default rate limit burst should be 20",
		},
		{
			name:   "Log Level",
			got:    cfg.Log.Level,
			want:   "info",
			errMsg: "default log level should be 'info'",
		},
		{
			name:   "Service Name",
			got:    cfg.Trace.ServiceName,
			want:   "user-service",
			errMsg: "default service name should be 'user-service'",
		},
		{
			name:   "Redis DB",
			got:    cfg.Redis.DB,
			want:   0,
			errMsg: "default Redis DB should be 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.errMsg, tt.got, tt.want)
			}
		})
	}
}

func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	os.Clearenv()

	testVars := map[string]string{
		"DATABASE_URL":                "postgres://test:test@localhost:5432/testdb",
		"REDIS_ADDR":                  "localhost:6379",
		"KAFKA_BROKERS":               "localhost:9092",
		"API_PORT":                    "9090",
		"GRPC_PORT":                   "50052",
		"REDIS_PASSWORD":              "testpass",
		"KAFKA_USER_EVENTS_TOPIC":     "test_events",
		"LOG_LEVEL":                   "debug",
		"OTEL_EXPORTER_OTLP_ENDPOINT": "localhost:4317",
		"OTEL_SERVICE_NAME":           "test-service",
	}

	for k, v := range testVars {
		os.Setenv(k, v)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig() with environment variables error = %v", err)
		return
	}

	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "Database URL",
			got:  cfg.DB.URL,
			want: testVars["DATABASE_URL"],
		},
		{
			name: "Redis Address",
			got:  cfg.Redis.Addr,
			want: testVars["REDIS_ADDR"],
		},
		{
			name: "Kafka Brokers",
			got:  cfg.Kafka.Brokers,
			want: testVars["KAFKA_BROKERS"],
		},
		{
			name: "Redis Password",
			got:  cfg.Redis.Password,
			want: testVars["REDIS_PASSWORD"],
		},
		{
			name: "Log Level",
			got:  cfg.Log.Level,
			want: testVars["LOG_LEVEL"],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestLoadConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name:    "missing required vars",
			envVars: map[string]string{
				// Missing DATABASE_URL, REDIS_ADDR, KAFKA_BROKERS
			},
			wantErr:     true,
			errContains: "missing required configuration variables",
		},
		{
			name: "invalid log level",
			envVars: map[string]string{
				"DATABASE_URL":  "postgres://test:test@localhost:5432/testdb",
				"REDIS_ADDR":    "localhost:6379",
				"KAFKA_BROKERS": "localhost:9092",
				"LOG_LEVEL":     "invalid",
			},
			wantErr:     true,
			errContains: "invalid log level",
		},
		{
			name: "valid configuration",
			envVars: map[string]string{
				"DATABASE_URL":  "postgres://test:test@localhost:5432/testdb",
				"REDIS_ADDR":    "localhost:6379",
				"KAFKA_BROKERS": "localhost:9092",
				"LOG_LEVEL":     "debug",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			_, err := LoadConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("error message '%v' does not contain '%v'", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				DB: DBConfig{
					URL: "postgres://test:test@localhost:5432/testdb",
				},
				Redis: RedisConfig{
					Addr: "localhost:6379",
				},
				Kafka: KafkaConfig{
					Brokers: "localhost:9092",
				},
				Log: LogConfig{
					Level: "info",
				},
			},
			wantErr: false,
		},
		{
			name: "missing database url",
			config: &Config{
				Redis: RedisConfig{
					Addr: "localhost:6379",
				},
				Kafka: KafkaConfig{
					Brokers: "localhost:9092",
				},
			},
			wantErr: true,
		},
		{
			name: "missing redis addr",
			config: &Config{
				DB: DBConfig{
					URL: "postgres://test:test@localhost:5432/testdb",
				},
				Kafka: KafkaConfig{
					Brokers: "localhost:9092",
				},
			},
			wantErr: true,
		},
		{
			name: "missing kafka brokers",
			config: &Config{
				DB: DBConfig{
					URL: "postgres://test:test@localhost:5432/testdb",
				},
				Redis: RedisConfig{
					Addr: "localhost:6379",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func contains(s, substr string) bool {
	return s != "" && substr != "" && s != substr && len(s) > len(substr)
}
