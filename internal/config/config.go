package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	MTNCollectionSubscriptionKey string
	MTNCollectionAPIUser         string
	MTNCollectionAPIKey          string
	MTNCollectionTargetEnv       string
	MTNCollectionBaseURL         string
	MTNCollectionCallbackURL     string
}

func Load() Config {
	_ = godotenv.Load()

	cfg := Config{
		MTNCollectionSubscriptionKey: strings.TrimSpace(os.Getenv("MTN_COLLECTION_SUBSCRIPTION_KEY")),
		MTNCollectionAPIUser:         strings.TrimSpace(os.Getenv("MTN_COLLECTION_API_USER")),
		MTNCollectionAPIKey:          strings.TrimSpace(os.Getenv("MTN_COLLECTION_API_KEY")),
		MTNCollectionTargetEnv:       strings.TrimSpace(os.Getenv("MTN_COLLECTION_TARGET_ENVIRONMENT")),
		MTNCollectionBaseURL:         strings.TrimSpace(os.Getenv("MTN_COLLECTION_BASE_URL")),
		MTNCollectionCallbackURL:     strings.TrimSpace(os.Getenv("MTN_COLLECTION_CALLBACK_URL")),
	}

	if cfg.MTNCollectionSubscriptionKey == "" {
		log.Fatal("missing MTN_COLLECTION_SUBSCRIPTION_KEY")
	}
	if cfg.MTNCollectionAPIUser == "" {
		log.Fatal("missing MTN_COLLECTION_API_USER")
	}
	if cfg.MTNCollectionAPIKey == "" {
		log.Fatal("missing MTN_COLLECTION_API_KEY")
	}

	if cfg.MTNCollectionTargetEnv == "" {
		cfg.MTNCollectionTargetEnv = "sandbox"
	}

	if cfg.MTNCollectionBaseURL == "" {
		cfg.MTNCollectionBaseURL = "https://sandbox.momodeveloper.mtn.com"
	}

	if cfg.MTNCollectionTargetEnv != "sandbox" && cfg.MTNCollectionCallbackURL == "" {
		log.Fatal("missing MTN_COLLECTION_CALLBACK_URL for non-sandbox environment")
	}

	return cfg
}
