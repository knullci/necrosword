package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Executor ExecutorConfig `mapstructure:"executor"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// ExecutorConfig holds process executor configuration
type ExecutorConfig struct {
	AllowedTools   []string `mapstructure:"allowed_tools"`
	DefaultTimeout int      `mapstructure:"default_timeout"` // in seconds
	MaxConcurrent  int      `mapstructure:"max_concurrent"`
	WorkspaceBase  string   `mapstructure:"workspace_base"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"` // json or console
}

// Load reads configuration from file and environment
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8081)
	v.SetDefault("executor.allowed_tools", []string{"git", "npm", "mvn", "docker", "kubectl", "go", "make"})
	v.SetDefault("executor.default_timeout", 3600) // 1 hour
	v.SetDefault("executor.max_concurrent", 10)
	v.SetDefault("executor.workspace_base", "workspace")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	// Config file settings
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/necrosword")

	// Environment variables with NECROSWORD prefix
	v.SetEnvPrefix("NECROSWORD")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Try to read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, using defaults and env vars
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	// Check for shared KNULL_WORKSPACE environment variable
	// This allows both Knull and Necrosword to share the same workspace path
	// Priority: NECROSWORD_EXECUTOR_WORKSPACE_BASE > KNULL_WORKSPACE > config file > default
	if knullWorkspace := os.Getenv("KNULL_WORKSPACE"); knullWorkspace != "" {
		// Only use KNULL_WORKSPACE if NECROSWORD_EXECUTOR_WORKSPACE_BASE is not set
		if os.Getenv("NECROSWORD_EXECUTOR_WORKSPACE_BASE") == "" {
			cfg.Executor.WorkspaceBase = knullWorkspace
		}
	}

	return &cfg, nil
}

// Address returns the server address string
func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// IsToolAllowed checks if a tool is in the allowed list
func (c *ExecutorConfig) IsToolAllowed(tool string) bool {
	for _, t := range c.AllowedTools {
		if strings.EqualFold(t, tool) {
			return true
		}
	}
	return false
}
