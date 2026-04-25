//go:build integration

// Package sync — mock partagé pour les tests d'intégration achievements.
package sync

import "context"

// mockXboxClient implémente XboxAchievementsClient pour les tests d'intégration.
type mockXboxClient struct {
	responses map[string][]PlayerAchievementRaw
	err       error
	callCount map[string]int
}

func newMockXboxClient() *mockXboxClient {
	return &mockXboxClient{
		responses: make(map[string][]PlayerAchievementRaw),
		callCount: make(map[string]int),
	}
}

func (m *mockXboxClient) GetPlayerAchievements(_ context.Context, _, lang string) ([]PlayerAchievementRaw, error) {
	m.callCount[lang]++
	if m.err != nil {
		return nil, m.err
	}
	return m.responses[lang], nil
}
