package einocodex

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	codexClientID     = "app_EMoamEEZ73f0CkXaXp7hrann"
	codexTokenURL     = "https://auth.openai.com/oauth/token"
	expBufferDuration = time.Minute * 5
)

type authManager struct {
	authPath   string
	httpClient *http.Client

	nowFn func() time.Time

	mut          sync.Mutex
	accessToken  string
	refreshToken string
}

func newAuthManager(nowFn func() time.Time) *authManager {
	return &authManager{
		authPath: authFilePath(),
		nowFn:    nowFn,
		httpClient: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (am *authManager) ensureSession() error {
	am.mut.Lock()
	defer am.mut.Unlock()
	f, err := loadAuthFile(am.authPath)
	if err != nil {
		return err
	}

	if !f.isSubscription() {
		return errors.New("auth is not subscription based")
	}

	am.accessToken = f.Tokens.AccessToken
	am.refreshToken = f.Tokens.RefreshToken

	return nil
}

func (am *authManager) token(ctx context.Context) (string, error) {
	am.mut.Lock()
	defer am.mut.Unlock()

	f, err := loadAuthFile(am.authPath)
	if err != nil {
		return "", err
	}
	if !f.isSubscription() {
		return "", errors.New("invalid auth.json")
	}
	am.accessToken = f.Tokens.AccessToken
	am.refreshToken = f.Tokens.RefreshToken

	t, _, err := jwt.NewParser().ParseUnverified(am.accessToken, jwt.MapClaims{})
	if err != nil {
		return "", err
	}

	exp, err := t.Claims.GetExpirationTime()
	if err != nil {
		return "", err
	}
	if exp == nil {
		return "", fmt.Errorf("access token must have an expiration time")
	}

	if exp.Before(am.nowFn().Add(expBufferDuration)) {
		// refresh
		t, err := am.refreshAccessToken(ctx)
		if err != nil {
			return "", err
		}
		am.accessToken = t.AccessToken
		if t.RefreshToken != "" {
			am.refreshToken = t.RefreshToken
		}
	}

	return am.accessToken, nil
}

func (am *authManager) refreshAccessToken(ctx context.Context) (tokens, error) {
	body, err := json.Marshal(map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     codexClientID,
		"refresh_token": am.refreshToken,
	})
	if err != nil {
		return tokens{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, codexTokenURL, bytes.NewReader(body))
	if err != nil {
		return tokens{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := am.httpClient.Do(req)
	if err != nil {
		return tokens{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return tokens{}, fmt.Errorf("token refresh failed: HTTP %d", res.StatusCode)
	}

	var got tokens
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return tokens{}, err
	}
	if err := json.Unmarshal(b, &got); err != nil {
		return tokens{}, err
	}
	if got.AccessToken == "" {
		return tokens{}, errors.New("token refresh produced empty token")
	}

	if err := syncTokens(am.authPath, am.refreshToken, got, am.nowFn); err != nil {
		return tokens{}, err
	}

	return got, nil
}
