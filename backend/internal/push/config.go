package push

import (
	"errors"
	"os"
	"strings"
)

type RuntimeConfig struct {
	TokenKeyID             string
	TokenKeyBase64         string
	FirebaseProjectID      string
	FirebaseCredentialsFile string
}

func LoadRuntimeConfig() (RuntimeConfig, error) {
	cfg := RuntimeConfig{
		TokenKeyID: strings.TrimSpace(os.Getenv("LINKUP_PUSH_TOKEN_KEY_ID")),
		TokenKeyBase64: strings.TrimSpace(os.Getenv("LINKUP_PUSH_TOKEN_KEY_BASE64")),
		FirebaseProjectID: strings.TrimSpace(os.Getenv("LINKUP_FIREBASE_PROJECT_ID")),
		FirebaseCredentialsFile: strings.TrimSpace(os.Getenv("LINKUP_FIREBASE_SERVICE_ACCOUNT_FILE")),
	}
	keyConfigured := cfg.TokenKeyID != "" || cfg.TokenKeyBase64 != ""
	if keyConfigured && (cfg.TokenKeyID == "" || cfg.TokenKeyBase64 == "") {
		return RuntimeConfig{}, errors.New("LINKUP_PUSH_TOKEN_KEY_ID and LINKUP_PUSH_TOKEN_KEY_BASE64 must be configured together")
	}
	firebaseConfigured := cfg.FirebaseProjectID != "" || cfg.FirebaseCredentialsFile != ""
	if firebaseConfigured && (cfg.FirebaseProjectID == "" || cfg.FirebaseCredentialsFile == "") {
		// Project id alone is allowed while device registration is being enabled;
		// actual outbound FCM delivery remains disabled until credentials exist.
		if cfg.FirebaseCredentialsFile != "" {
			return RuntimeConfig{}, errors.New("LINKUP_FIREBASE_PROJECT_ID is required with LINKUP_FIREBASE_SERVICE_ACCOUNT_FILE")
		}
	}
	return cfg, nil
}

func (c RuntimeConfig) RegistrationEnabled() bool {
	return c.TokenKeyID != "" && c.TokenKeyBase64 != ""
}

func (c RuntimeConfig) DeliveryEnabled() bool {
	return c.RegistrationEnabled() && c.FirebaseProjectID != "" && c.FirebaseCredentialsFile != ""
}
