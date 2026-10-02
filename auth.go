package einocodex

import (
	"errors"
	"sync"
)

type authManager struct {
	authPath string

	mut         sync.RWMutex
	accessToken string
	idToken     string
}

func newAuthManager() *authManager {
	return &authManager{
		authPath: authFilePath(),
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
	am.idToken = f.Tokens.IDToken

	return nil
}

func (am *authManager) AccessToken() string {
	am.mut.Lock()
	defer am.mut.Unlock()
	return am.accessToken
}
