package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

// Условие ТЗ: Переменные окружения должны считываться из файла config.env
const envFile = "./config.env"

type Config struct {
	Env        string `yaml:"env" env-required:"true"`
	Storage    `yaml:"db"`
	HTTPServer `yaml:"http_server"`
	Redis      `yaml:"redis"`
}

type Storage struct {
	User     string `yaml:"user" env-default:"postgres"`
	Password string `env:"DB_PASSWORD"`
	Host     string `yaml:"host" env-default:"localhost"`
	Port     string `yaml:"port" env-default:"5432"`
	Name     string `yaml:"name" env-default:"wallets-db"`
	SSLMode  string `yaml:"sslmode" env-default:"disable"`
}

type Redis struct {
	Addr             string        `yaml:"address" env-default:"redis"`
	Port             string        `yaml:"port" env-default:"6379"`
	Password         string        `env:"REDIS_PASSWORD" env-default:""`
	DB               int           `yaml:"db" env-default:"0"`
	LockExporation   time.Duration `yaml:"lock_exporation" env-default:"500ms"`
	CacheExporation  time.Duration `yaml:"cache_exporation" env-default:"10m"`
	MaxUnlockRetries int           `yaml:"max_unlock_retries" env-default:"3"`
	BaseRetryDelay   time.Duration `yaml:"base_retry_delay" env-default:"50ms"`
}

type HTTPServer struct {
	Address      string        `yaml:"address" env-default:"localhost:8080"`
	Timeout      time.Duration `yaml:"timeout" env-default:"4s"`
	Idle_timeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

func MustLoad() *Config {
	err := godotenv.Overload(envFile)
	if err != nil {
		log.Fatalf("failed to read %s: %s", envFile, err.Error())
	}
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		log.Fatalf("Env file does not exist in root dir: %s", envFile)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("can't read config file: %s", err)
	}

	if err := cleanenv.ReadConfig(envFile, &cfg); err != nil {
		log.Fatalf("can't read env config file: %s", err)
	}

	return &cfg
}
