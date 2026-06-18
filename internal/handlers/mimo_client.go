package handlers

import (
	"context"
	"fmt"

	"mimo-webui/internal/db"
	"mimo-webui/internal/mimo"
)

// MiMoSession holds the per-user MiMo client and model configuration.
type MiMoSession struct {
	Client       *mimo.Client
	ModelVersion string // e.g. "mimo-v2.5"
}

// getMiMoSession returns a MiMo client + model version from user settings.
// Returns error if api_key or base_url are not configured.
func getMiMoSession(database *db.DB, userID int64) (*MiMoSession, error) {
	settings, err := db.GetSettings(context.Background(), database, userID)
	if err != nil {
		return nil, fmt.Errorf("load settings: %w", err)
	}

	apiKey, _ := settings["api_key"]
	baseURL, _ := settings["base_url"]
	modelVersion, _ := settings["model_version"]

	if apiKey == "" {
		return nil, fmt.Errorf("请先在设置中配置 API Key")
	}
	if baseURL == "" {
		return nil, fmt.Errorf("请先在设置中配置 API Base URL")
	}
	if modelVersion == "" {
		modelVersion = "mimo-v2.5" // fallback default
	}

	return &MiMoSession{
		Client:       mimo.NewClient(baseURL, apiKey),
		ModelVersion: modelVersion,
	}, nil
}
