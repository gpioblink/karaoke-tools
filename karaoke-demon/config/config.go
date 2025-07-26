package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	IMAGE_PATH         string
	FIFO_PATH          string
	VIDEO_DIR          string
	FILLER_VIDEOS_PATH []string
	WEBHOOK_PORT       int
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
		// Using default values
		fmt.Printf("Failed to load %s file. Using default values.\n", envFile)
		return &Config{
			IMAGE_PATH:         "/home/root/karaoke.img",
			FIFO_PATH:          "/tmp/karaoke-fifo",
			VIDEO_DIR:          "/home/output",
			FILLER_VIDEOS_PATH: []string{"/home/output/dummy.mp4"},
			WEBHOOK_PORT:       8787,
		}, nil
		// return nil, fmt.Errorf("error loading %s file", envFile)
	}

	webhookPort := 8080
	if portStr := os.Getenv("WEBHOOK_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			webhookPort = port
		}
	}

	return &Config{
		IMAGE_PATH:         os.Getenv("IMAGE_PATH"),
		FIFO_PATH:          os.Getenv("FIFO_PATH"),
		VIDEO_DIR:          os.Getenv("VIDEO_DIR"),
		FILLER_VIDEOS_PATH: []string{os.Getenv("DUMMY_VIDEO_PATH")},
		WEBHOOK_PORT:       webhookPort,
	}, nil
}
