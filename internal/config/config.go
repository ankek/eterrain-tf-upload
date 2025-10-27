package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/ini.v1"
)

// Config holds the application configuration
type Config struct {
	// Server configuration
	Host string
	Port int

	// Storage configuration
	StorageType string // "memory", "csv", "mysql", "dual", etc.
	StoragePath string // Path for file-based storage

	// Database configuration (for MySQL storage)
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string

	// Security
	EnableTLS bool
	CertFile  string
	KeyFile   string
}

// Load loads configuration from backend_service.cfg file
// Falls back to environment variables if config file is not found
func Load() (*Config, error) {
	// Default config file path
	configFile := "backend_service.cfg"

	// Check if config file exists
	if _, err := os.Stat(configFile); err == nil {
		return LoadFromFile(configFile)
	}

	// Fall back to environment variables if config file not found
	config := &Config{
		Host:        getEnv("HOST", "127.0.0.1"),
		Port:        getEnvAsInt("PORT", 7777),
		StorageType: getEnv("STORAGE_TYPE", "csv"),
		StoragePath: getEnv("STORAGE_PATH", "./data"),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnvAsInt("DB_PORT", 3306),
		DBUser:      getEnv("DB_USER", ""),
		DBPassword:  getEnv("DB_PASSWORD", ""),
		DBName:      getEnv("DB_NAME", "data"),
		EnableTLS:   getEnvAsBool("ENABLE_TLS", false),
		CertFile:    getEnv("TLS_CERT_FILE", ""),
		KeyFile:     getEnv("TLS_KEY_FILE", ""),
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// LoadFromFile loads configuration from an INI file
func LoadFromFile(filename string) (*Config, error) {
	// Get absolute path
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Load INI file
	cfg, err := ini.Load(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file %s: %w", absPath, err)
	}

	// Parse server configuration
	serverSection := cfg.Section("server")
	config := &Config{
		Host: serverSection.Key("hostname").MustString("127.0.0.1"),
		Port: serverSection.Key("port").MustInt(7777),
	}

	// Parse storage configuration
	storageSection := cfg.Section("storage")
	config.StorageType = storageSection.Key("type").MustString("csv")
	config.StoragePath = storageSection.Key("path").MustString("./data")

	// Parse security configuration
	securitySection := cfg.Section("security")
	config.EnableTLS = securitySection.Key("enable_tls").MustBool(false)
	config.CertFile = securitySection.Key("cert_file").String()
	config.KeyFile = securitySection.Key("key_file").String()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if c.EnableTLS {
		if c.CertFile == "" {
			return fmt.Errorf("TLS enabled but TLS_CERT_FILE not set")
		}
		if c.KeyFile == "" {
			return fmt.Errorf("TLS enabled but TLS_KEY_FILE not set")
		}
	}

	return nil
}

// Address returns the server address in host:port format
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// DSN returns the MySQL Data Source Name connection string
func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt retrieves an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// getEnvAsBool retrieves an environment variable as a boolean or returns a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
