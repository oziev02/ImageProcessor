package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Kafka    KafkaConfig
	Storage  StorageConfig
	Image    ImageConfig
}

type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type KafkaConfig struct {
	Brokers       []string
	Topic         string
	ConsumerGroup string
}

type StorageConfig struct {
	BasePath string
}

type ImageConfig struct {
	MaxFileSize      int64
	ThumbnailWidth   int
	ThumbnailHeight  int
	ProcessedWidth   int
	ProcessedHeight  int
	WatermarkEnabled bool
	WatermarkPath    string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "imageprocessor"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Kafka: KafkaConfig{
			Brokers:       getEnvSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			Topic:         getEnv("KAFKA_TOPIC", "image-processing"),
			ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "image-processor-group"),
		},
		Storage: StorageConfig{
			BasePath: getEnv("STORAGE_BASE_PATH", "./storage"),
		},
		Image: ImageConfig{
			MaxFileSize:      getEnvInt64("IMAGE_MAX_FILE_SIZE", 10*1024*1024), // 10MB
			ThumbnailWidth:   getEnvInt("IMAGE_THUMBNAIL_WIDTH", 200),
			ThumbnailHeight:  getEnvInt("IMAGE_THUMBNAIL_HEIGHT", 200),
			ProcessedWidth:   getEnvInt("IMAGE_PROCESSED_WIDTH", 800),
			ProcessedHeight:  getEnvInt("IMAGE_PROCESSED_HEIGHT", 800),
			WatermarkEnabled: getEnvBool("IMAGE_WATERMARK_ENABLED", false),
			WatermarkPath:    getEnv("IMAGE_WATERMARK_PATH", ""),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Storage.BasePath == "" {
		return fmt.Errorf("storage base path is required")
	}
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("kafka brokers are required")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Split by comma and trim spaces
		var result []string
		parts := strings.Split(value, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}
