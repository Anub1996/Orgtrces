package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port     string
	Env      string
	DB       DBConfig
	JWT      JWTConfig
	S3       S3Config
	AgentKey string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	MaxConns string
}

type JWTConfig struct {
	Secret string
	Expiry time.Duration
}

type S3Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
}

func Load() *Config {
	godotenv.Load()

	expiry, _ := time.ParseDuration(getEnv("JWT_EXPIRY", "24h"))

	return &Config{
		Port: getEnv("PORT", "8080"),
		Env:  getEnv("ENV", "development"),
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "orgtraces"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "orgtraces"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
			MaxConns: getEnv("DB_MAX_CONNS", "50"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", ""),
			Expiry: expiry,
		},
		S3: S3Config{
			Endpoint:  getEnv("S3_ENDPOINT", ""),
			Bucket:    getEnv("S3_BUCKET", "orgtraces-screenshots"),
			AccessKey: getEnv("S3_ACCESS_KEY", ""),
			SecretKey: getEnv("S3_SECRET_KEY", ""),
			Region:    getEnv("S3_REGION", "auto"),
		},
		AgentKey: getEnv("AGENT_API_KEY", ""),
	}
}

func (c *Config) DSN() string {
	return "postgres://" + c.DB.User + ":" + c.DB.Password +
		"@" + c.DB.Host + ":" + c.DB.Port +
		"/" + c.DB.Name + "?sslmode=" + c.DB.SSLMode +
		"&pool_max_conns=" + c.DB.MaxConns
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
