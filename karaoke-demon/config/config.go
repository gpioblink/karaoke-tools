package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	IMAGE_PATH         string
	FIFO_PATH          string
	VIDEO_DIR          string
	FILLER_VIDEOS_PATH []string
}

func NewConfig() (*Config, error) {
	env := os.Getenv("APP_ENV")
	var envFile string

	switch env {
	case "prod":
		envFile = ".env.prod"
	case "dev":
		envFile = ".env.dev"
	default:
		envFile = ".env"
	}

	err := godotenv.Load(envFile)
	if err != nil {
		return nil, fmt.Errorf("error loading %s file", envFile)
	}

	return &Config{
		IMAGE_PATH:         os.Getenv("IMAGE_PATH"),
		FIFO_PATH:          os.Getenv("FIFO_PATH"),
		VIDEO_DIR:          os.Getenv("VIDEO_DIR"),
		FILLER_VIDEOS_PATH: []string{os.Getenv("DUMMY_VIDEO_PATH")},
	}, nil
}
