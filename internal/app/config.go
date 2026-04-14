package app

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	BaseURL      string
	Username     string
	Password     string
	Token        string
	TimelineType string
	Limit        int
}

func LoadConfig() (Config, error) {
	_ = godotenv.Load()

	baseURL := os.Getenv("FREEFEED_BASE_URL")
	if baseURL == "" {
		baseURL = "https://freefeed.net"
	}

	token := os.Getenv("FREEFEED_APP_TOKEN")
	username := os.Getenv("FREEFEED_USERNAME")
	password := os.Getenv("FREEFEED_PASSWORD")
	if token == "" && (username == "" || password == "") {
		return Config{}, errors.New("missing FREEFEED_APP_TOKEN or FREEFEED_USERNAME/FREEFEED_PASSWORD")
	}

	timelineType := os.Getenv("FREEFEED_TIMELINE")
	if timelineType == "" {
		timelineType = "home"
	}

	limit := 20
	if rawLimit := os.Getenv("FREEFEED_TIMELINE_LIMIT"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	return Config{
		BaseURL:      baseURL,
		Username:     username,
		Password:     password,
		Token:        token,
		TimelineType: timelineType,
		Limit:        limit,
	}, nil
}
