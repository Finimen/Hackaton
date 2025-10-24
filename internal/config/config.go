package config

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Logging  LoggingConfig  `mapstructure:"login"`
	Security SecurityConfig `mapstructure:"security"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type SecurityConfig struct {
	AgentTokenSecret string        `mapstructure:"agent_token_secret"`
	TokenExpiry      time.Duration `mapstructure:"token_expiry"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("configs")

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		var errViper viper.ConfigFileNotFoundError
		if errors.Is(err, &errViper) {
			slog.Warn("config file not found")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config, %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed, %w", err)
	}

	slog.Info("configuration loaded successfully")
	return &config, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")

	// database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "netscan")
	viper.SetDefault("database.password", "netscan")
	viper.SetDefault("database.dbname", "netscan")
	viper.SetDefault("database.sslmode", "disable")

	// redis defaults
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// login defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")

	// security defaults
	viper.SetDefault("security.agent_token_secret", "change-me-in-production")
	viper.SetDefault("security.token_expiry", "720h") // 30 days
}

func validateConfig(cfg *Config) error {
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port %d", cfg.Server.Port)
	}

	if cfg.Server.Mode != "debug" && cfg.Server.Mode != "release" {
		return fmt.Errorf("invalid server mode %s", cfg.Server.Mode)
	}

	if cfg.Database.Host == "" {
		return errors.New("database host is required")
	}

	if cfg.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	if cfg.Redis.Addr == "" {
		return fmt.Errorf("redis address is required")
	}

	if cfg.Security.AgentTokenSecret == "change-me-in-production" {
		slog.Warn("Using default agent token secret - change this in production!")
	}

	return nil
}

// возвращает DSN строку для PostgreSQL
func (d *DatabaseConfig) GetDNS() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

// возвращает настройки для Redis клиента
func (r *RedisConfig) GetRedisOptions() *redis.Options {
	return &redis.Options{
		Addr:            r.Addr,
		Password:        r.Password,
		DB:              r.DB,
		DisableIdentity: true,
	}
}
