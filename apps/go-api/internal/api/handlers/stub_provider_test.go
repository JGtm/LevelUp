// Package handlers_test — stub_provider_test.go : stub TokenProvider pour tests unitaires.
//
// StubTokenProvider permet de tester AuthHandler sans dépendance MSAL/réseau.
package handlers_test

import (
	"context"
	"errors"

	auth_platform "levelup/go-api/internal/platform/auth"
)

// stubTokenProvider implémente auth_platform.TokenProvider pour les tests.
type stubTokenProvider struct {
	initFlowErr    error
	initFlowFlow   auth_platform.DeviceFlow
	exchangeErr    error
	exchangeResult *auth_platform.ExchangeResult
}

// Vérification compile-time : stubTokenProvider implémente TokenProvider.
var _ auth_platform.TokenProvider = (*stubTokenProvider)(nil)

func (s *stubTokenProvider) InitDeviceFlow(_ context.Context) (auth_platform.DeviceFlow, error) {
	if s.initFlowErr != nil {
		return nil, s.initFlowErr
	}
	if s.initFlowFlow != nil {
		return s.initFlowFlow, nil
	}
	return auth_platform.NewStubDeviceFlow("STUB1234", "https://microsoft.com/devicelogin", "Stub message", 0, "msal"), nil
}

func (s *stubTokenProvider) TryOAuthRefresh(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (s *stubTokenProvider) TryOAuthRefreshWithRotation(_ context.Context, _ string) (string, string, error) {
	return "", "", nil
}

func (s *stubTokenProvider) Exchange(_ context.Context, _ string) (*auth_platform.ExchangeResult, error) {
	if s.exchangeErr != nil {
		return nil, s.exchangeErr
	}
	if s.exchangeResult != nil {
		return s.exchangeResult, nil
	}
	return nil, errors.New("stub: Exchange non configuré")
}
