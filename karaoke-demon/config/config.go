package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	IMAGE_PATH string
	FIFO_PATH  string
	VIDEO_DIR  string
	DUMMY1     string
	DUMMY2     string
	DUMMY3     string
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
			IMAGE_PATH: "/home/root/karaoke.img",
			FIFO_PATH:  "/tmp/karaoke-fifo",
			VIDEO_DIR:  "/home/output",
			DUMMY1:     "/home/output/dummy.mp4",
			DUMMY2:     "/home/output/dummyfiller1_h264.mp4",
			DUMMY3:     "/home/output/dummyfiller2_h264.mp4",
		}, nil
		// return nil, fmt.Errorf("error loading %s file", envFile)
	}

	return &Config{
		IMAGE_PATH: os.Getenv("IMAGE_PATH"),
		FIFO_PATH:  os.Getenv("FIFO_PATH"),
		VIDEO_DIR:  os.Getenv("VIDEO_DIR"),
		DUMMY1:     os.Getenv("DUMMY1"),
		DUMMY2:     os.Getenv("DUMMY2"),
		DUMMY3:     os.Getenv("DUMMY3"),
	}, nil
}
