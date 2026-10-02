package einocodex

import (
	"encoding/json/v2"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultCodexHomeEnv = "CODEX_HOME"
	authJSON            = "auth.json"
)

var errAuthFileNotFound = errors.New("auth.json not found")

type tokens struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    string `json:"account_id"`
}

type authFile struct {
	AuthMode    string     `json:"auth_mode"`
	Tokens      *tokens    `json:"tokens"`
	LastRefresh *time.Time `json:"last_refresh"`
}

func (am authFile) isSubscription() bool {
	if am.AuthMode == "apikey" {
		return false
	}
	return am.Tokens != nil
}

func authFilePath() string {
	home := os.Getenv(defaultCodexHomeEnv)
	if home != "" {
		return filepath.Join(home, authJSON)
	}
	d, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, ".codex", authJSON)
}

func loadAuthFile(path string) (authFile, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return authFile{}, errAuthFileNotFound
	}
	if err != nil {
		return authFile{}, err
	}

	var f authFile
	if err := json.Unmarshal(b, &f); err != nil {
		return authFile{}, err
	}
	return f, nil
}
