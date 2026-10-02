package einocodex

import (
	"encoding/json/jsontext"
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
	return am.AuthMode == "chatgpt" && am.Tokens != nil
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

func replaceAuthFile(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".auth.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func syncTokens(path string, expectedRefreshToken string, t tokens, nowFn func() time.Time) error {
	if t.AccessToken == "" {
		return errors.New("access token is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var file map[string]jsontext.Value
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}

	var saved map[string]jsontext.Value
	if err := json.Unmarshal(file["tokens"], &saved); err != nil {
		return err
	}
	if saved == nil {
		return errors.New("tokens are empty")
	}

	var savedRefresh string
	if err := json.Unmarshal(saved["refresh_token"], &savedRefresh); err != nil {
		return err
	}
	if savedRefresh != expectedRefreshToken {
		return errors.New("token values changed during refresh")
	}

	for key, val := range map[string]string{
		"access_token":  t.AccessToken,
		"refresh_token": t.RefreshToken,
		"id_token":      t.IDToken,
	} {
		if val == "" {
			continue
		}
		encoded, err := json.Marshal(val)
		if err != nil {
			return err
		}
		saved[key] = jsontext.Value(encoded)
	}

	tokenJSON, err := json.Marshal(saved)
	if err != nil {
		return err
	}
	file["tokens"] = jsontext.Value(tokenJSON)

	timestampJSON, err := json.Marshal(nowFn().UTC())
	if err != nil {
		return err
	}
	file["last_refresh"] = jsontext.Value(timestampJSON)

	updated, err := json.Marshal(file)
	if err != nil {
		return err
	}

	return replaceAuthFile(path, updated)
}
