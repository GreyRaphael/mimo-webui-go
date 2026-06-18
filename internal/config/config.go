package config

import (
	"os"
	"strconv"

	toml "github.com/pelletier/go-toml/v2"
)

type Config struct {
	Server    ServerConfig
	Auth      AuthConfig
	Upload    UploadConfig
	RateLimit RateLimitConfig
	Database  DatabaseConfig
}

type ServerConfig struct {
	Host string
	Port int
}

type AuthConfig struct {
	JWTSecret        string `toml:"jwt_secret"`
	JWTExpiryHours   int    `toml:"jwt_expiry_hours"`
	OpenRegistration bool   `toml:"open_registration"`
	MaxUsers         int    `toml:"max_users"`
	AdminPassword    string `toml:"admin_password"`
}

type UploadConfig struct {
	MaxImageMB         int    `toml:"max_image_mb"`
	MaxAudioMB         int    `toml:"max_audio_mb"`
	MaxVideoMB         int    `toml:"max_video_mb"`
	TempDir            string `toml:"temp_dir"`
	CleanupIntervalMin int    `toml:"cleanup_interval_minutes"`
	FileExpiryMin      int    `toml:"file_expiry_minutes"`
}

type RateLimitConfig struct {
	RequestsPerMinute int `toml:"requests_per_minute"`
}

type DatabaseConfig struct {
	Path string
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{Host: "0.0.0.0", Port: 3000},
		Auth: AuthConfig{
			JWTSecret:      "change-me-to-random-string",
			JWTExpiryHours: 24,
			MaxUsers:       10,
		},
		Upload: UploadConfig{
			MaxImageMB:         50,
			MaxAudioMB:         100,
			MaxVideoMB:         500,
			TempDir:            "/tmp/mimo-uploads",
			CleanupIntervalMin: 30,
			FileExpiryMin:      60,
		},
		RateLimit: RateLimitConfig{RequestsPerMinute: 30},
		Database:  DatabaseConfig{Path: "mimo-webui.db"},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			applyEnvOverrides(cfg)
			return cfg, nil
		}
		return nil, err
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	applyEnvOverrides(cfg)
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	if v := os.Getenv("PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.Database.Path = v
	}
}
