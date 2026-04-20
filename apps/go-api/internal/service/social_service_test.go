package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// mockSocialRepo implémente port.SocialRepository pour les tests.
type mockSocialRepo struct {
	favorites map[string]bool // clé: playerSlug+":"+matchID
	toggleErr error
	isFavErr  error
}

func (m *mockSocialRepo) ToggleMatchFavorite(_ context.Context, playerSlug, matchID string, favorited bool) error {
	if m.toggleErr != nil {
		return m.toggleErr
	}
	key := playerSlug + ":" + matchID
	if favorited {
		m.favorites[key] = true
	} else {
		delete(m.favorites, key)
	}
	return nil
}

func (m *mockSocialRepo) IsMatchFavorite(_ context.Context, playerSlug, matchID string) (bool, error) {
	if m.isFavErr != nil {
		return false, m.isFavErr
	}
	return m.favorites[playerSlug+":"+matchID], nil
}

func TestSocialService_ToggleMatchFavorite_Add(t *testing.T) {
	repo := &mockSocialRepo{favorites: make(map[string]bool)}
	svc := NewSocialService(repo)

	err := svc.ToggleMatchFavorite(context.Background(), domain.MatchFavoriteRequest{
		PlayerSlug: "spartan-a",
		MatchID:    "match-001",
		Favorited:  true,
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !repo.favorites["spartan-a:match-001"] {
		t.Error("match should be favorited")
	}
}

func TestSocialService_ToggleMatchFavorite_Remove(t *testing.T) {
	repo := &mockSocialRepo{favorites: map[string]bool{"spartan-a:match-001": true}}
	svc := NewSocialService(repo)

	err := svc.ToggleMatchFavorite(context.Background(), domain.MatchFavoriteRequest{
		PlayerSlug: "spartan-a",
		MatchID:    "match-001",
		Favorited:  false,
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if repo.favorites["spartan-a:match-001"] {
		t.Error("match should no longer be favorited")
	}
}

func TestSocialService_ToggleMatchFavorite_RepoError(t *testing.T) {
	repo := &mockSocialRepo{favorites: make(map[string]bool), toggleErr: errors.New("db down")}
	svc := NewSocialService(repo)

	err := svc.ToggleMatchFavorite(context.Background(), domain.MatchFavoriteRequest{
		PlayerSlug: "spartan-a",
		MatchID:    "match-001",
		Favorited:  true,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
