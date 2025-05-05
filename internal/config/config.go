package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the service.
// It contains nested configurations for different components of the application.
type Config struct {
	API   APIConfig
	GRPC  GRPCConfig
	DB    DBConfig
	Redis RedisConfig
	Kafka KafkaConfig
	Rate  RateConfig
	Log   LogConfig
	Trace TraceConfig
}

// APIConfig contains HTTP API server configuration
type APIConfig struct {
	Port int `mapstructure:"API_PORT"` // Port on which the HTTP API server will listen
}

// GRPCConfig contains gRPC server configuration
type GRPCConfig struct {
	Port int `mapstructure:"GRPC_PORT"` // Port on which the gRPC server will listen
}

// DBConfig contains database configuration
type DBConfig struct {
	URL             string        `mapstructure:"DATABASE_URL"`               // Database connection string
	MaxOpenConns    int           `mapstructure:"DATABASE_MAX_OPEN_CONNS"`    // Maximum number of open connections
	MaxIdleConns    int           `mapstructure:"DATABASE_MAX_IDLE_CONNS"`    // Maximum number of idle connections
	ConnMaxLifetime time.Duration `mapstructure:"DATABASE_CONN_MAX_LIFETIME"` // Maximum lifetime of a connection
	ConnMaxIdleTime time.Duration `mapstructure:"DATABASE_CONN_MAX_IDLETIME"` // Maximum idle time of a connection
}

// RedisConfig contains Redis connection configuration
type RedisConfig struct {
	Addr         string        `mapstructure:"REDIS_ADDR"`          // Redis server address
	Password     string        `mapstructure:"REDIS_PASSWORD"`      // Redis password (optional)
	DB           int           `mapstructure:"REDIS_DB"`            // Redis database number
	DialTimeout  time.Duration `mapstructure:"REDIS_DIAL_TIMEOUT"`  // Timeout for connecting to Redis
	ReadTimeout  time.Duration `mapstructure:"REDIS_READ_TIMEOUT"`  // Timeout for reading from Redis
	WriteTimeout time.Duration `mapstructure:"REDIS_WRITE_TIMEOUT"` // Timeout for writing to Redis
	CacheTTL     time.Duration `mapstructure:"REDIS_CACHE_TTL"`     // Time-to-live for cached items
}

// KafkaConfig contains Kafka connection configuration
type KafkaConfig struct {
	Brokers           string        `mapstructure:"KAFKA_BROKERS"`            // Comma-separated list of Kafka brokers
	EventTopic        string        `mapstructure:"KAFKA_USER_EVENTS_TOPIC"`  // Topic for user events
	NumPartitions     int           `mapstructure:"KAFKA_NUM_PARTITIONS"`     // Number of partitions for topics
	ReplicationFactor int           `mapstructure:"KAFKA_REPLICATION_FACTOR"` // Replication factor for topics
	WriteTimeout      time.Duration `mapstructure:"KAFKA_WRITE_TIMEOUT"`      // Timeout for write operations
}

// RateConfig contains rate limiting configuration
type RateConfig struct {
	RequestsPerSecond int `mapstructure:"RATE_LIMIT_RPS"`   // Number of requests allowed per second
	Burst             int `mapstructure:"RATE_LIMIT_BURST"` // Maximum burst size for rate limiting
}

// LogConfig contains logging configuration
type LogConfig struct {
	Level string `mapstructure:"LOG_LEVEL"` // Logging level (debug, info, warn, error)
}

// TraceConfig contains OpenTelemetry tracing configuration
type TraceConfig struct {
	ExporterEndpoint string `mapstructure:"OTEL_EXPORTER_OTLP_ENDPOINT"` // OpenTelemetry collector endpoint
	ServiceName      string `mapstructure:"OTEL_SERVICE_NAME"`           // Service name for tracing
}

// LoadConfig reads configuration from environment variables and .env file
func LoadConfig() (*Config, error) {
	v := viper.New()

	v.SetDefault("DATABASE_MAX_OPEN_CONNS", 60)
	v.SetDefault("DATABASE_MAX_IDLE_CONNS", 30)
	v.SetDefault("DATABASE_CONN_MAX_LIFETIME", "120s")
	v.SetDefault("DATABASE_CONN_MAX_IDLETIME", "20s")

	v.SetDefault("REDIS_DB", 0)
	v.SetDefault("REDIS_DIAL_TIMEOUT", "5s")
	v.SetDefault("REDIS_READ_TIMEOUT", "3s")
	v.SetDefault("REDIS_WRITE_TIMEOUT", "3s")
	v.SetDefault("REDIS_CACHE_TTL", "1h")

	v.SetDefault("KAFKA_USER_EVENTS_TOPIC", "user_events")
	v.SetDefault("KAFKA_NUM_PARTITIONS", 1)
	v.SetDefault("KAFKA_REPLICATION_FACTOR", 1)
	v.SetDefault("KAFKA_WRITE_TIMEOUT", "10s")

	v.SetDefault("RATE_LIMIT_RPS", 10)
	v.SetDefault("RATE_LIMIT_BURST", 20)

	v.SetDefault("API_PORT", 8080)
	v.SetDefault("GRPC_PORT", 50051)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("OTEL_SERVICE_NAME", "user-service")

	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("INFO: No .env file found, relying on environment variables and defaults.")
		} else {
			fmt.Printf("WARNING: Error reading config file (%s): %v\n", v.ConfigFileUsed(), err)
		}
	} else {
		fmt.Printf("INFO: Config loaded from file: %s\n", v.ConfigFileUsed())
	}

	v.AutomaticEnv() // Automatically read matching environment variables

	config := Config{
		API: APIConfig{
			Port: v.GetInt("API_PORT"),
		},
		GRPC: GRPCConfig{
			Port: v.GetInt("GRPC_PORT"),
		},
		DB: DBConfig{
			URL:             v.GetString("DATABASE_URL"),
			MaxOpenConns:    v.GetInt("DATABASE_MAX_OPEN_CONNS"),
			MaxIdleConns:    v.GetInt("DATABASE_MAX_IDLE_CONNS"),
			ConnMaxLifetime: v.GetDuration("DATABASE_CONN_MAX_LIFETIME"),
			ConnMaxIdleTime: v.GetDuration("DATABASE_CONN_MAX_IDLETIME"),
		},
		Redis: RedisConfig{
			Addr:         v.GetString("REDIS_ADDR"),
			Password:     v.GetString("REDIS_PASSWORD"),
			DB:           v.GetInt("REDIS_DB"),
			DialTimeout:  v.GetDuration("REDIS_DIAL_TIMEOUT"),
			ReadTimeout:  v.GetDuration("REDIS_READ_TIMEOUT"),
			WriteTimeout: v.GetDuration("REDIS_WRITE_TIMEOUT"),
			CacheTTL:     v.GetDuration("REDIS_CACHE_TTL"),
		},
		Kafka: KafkaConfig{
			Brokers:           v.GetString("KAFKA_BROKERS"),
			EventTopic:        v.GetString("KAFKA_USER_EVENTS_TOPIC"),
			NumPartitions:     v.GetInt("KAFKA_NUM_PARTITIONS"),
			ReplicationFactor: v.GetInt("KAFKA_REPLICATION_FACTOR"),
			WriteTimeout:      v.GetDuration("KAFKA_WRITE_TIMEOUT"),
		},
		Rate: RateConfig{
			RequestsPerSecond: v.GetInt("RATE_LIMIT_RPS"),
			Burst:             v.GetInt("RATE_LIMIT_BURST"),
		},
		Log: LogConfig{
			Level: v.GetString("LOG_LEVEL"),
		},
		Trace: TraceConfig{
			ExporterEndpoint: v.GetString("OTEL_EXPORTER_OTLP_ENDPOINT"),
			ServiceName:      v.GetString("OTEL_SERVICE_NAME"),
		},
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// validateConfig ensures all required configuration values are present and validates optional ones if set
func validateConfig(config *Config) error {
	var missingVars []string

	if config.DB.URL == "" {
		missingVars = append(missingVars, "DATABASE_URL")
	}
	if config.Redis.Addr == "" {
		missingVars = append(missingVars, "REDIS_ADDR")
	}
	if config.Kafka.Brokers == "" {
		missingVars = append(missingVars, "KAFKA_BROKERS")
	}

	if len(missingVars) > 0 {
		return fmt.Errorf("missing required configuration variables: %v", missingVars)
	}

	if config.Log.Level != "" {
		level := strings.ToLower(config.Log.Level)
		if level != "debug" && level != "info" && level != "warn" && level != "error" {
			return fmt.Errorf("invalid log level: %s", config.Log.Level)
		}
	}

	return nil
}
