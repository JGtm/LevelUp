package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
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

func (m *mockSocialRepo) GetFavoriteMatchIDs(_ context.Context, _ string) (map[string]bool, error) {
	result := make(map[string]bool)
	for key, v := range m.favorites {
		if v {
			result[key] = true
		}
	}
	return result, nil
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

// ─── WithWriterAcquirer (commit 5 db-concurrency) ───

// TestSocialService_ToggleMatchFavorite_LeaseBusy_PropagatesErrDBLocked vérifie
// que ToggleMatchFavorite propage ErrDBLocked au caller quand le lease
// shared_social est saturé. Le handler HTTP mappera en 503.
func TestSocialService_ToggleMatchFavorite_LeaseBusy_PropagatesErrDBLocked(t *testing.T) {
	repo := &mockSocialRepo{favorites: make(map[string]bool)}
	acquirer := func() (*dblease.LeasedWriter, error) {
		return nil, fmt.Errorf("simulated lease busy: %w", dblease.ErrDBLocked)
	}
	svc := NewSocialService(repo, WithWriterAcquirer(acquirer))

	err := svc.ToggleMatchFavorite(context.Background(), domain.MatchFavoriteRequest{
		PlayerSlug: "spartan-a",
		MatchID:    "match-001",
		Favorited:  true,
	})
	if err == nil {
		t.Fatal("expected ErrDBLocked, got nil")
	}
	if !errors.Is(err, dblease.ErrDBLocked) {
		t.Errorf("err should wrap dblease.ErrDBLocked, got %v", err)
	}
	if repo.favorites["spartan-a:match-001"] {
		t.Error("favorite should not have been written when lease busy")
	}
}

// TestSocialService_ToggleMatchFavorite_LeaseAcquiredSuccessfully vérifie le
// chemin nominal : un acquéreur qui retourne un writer réel libère le mutex
// après l'opération.
func TestSocialService_ToggleMatchFavorite_LeaseAcquiredSuccessfully(t *testing.T) {
	repo := &mockSocialRepo{favorites: make(map[string]bool)}
	path := "test://" + t.Name() + "/" + time.Now().Format("150405.000000000")
	acquirer := func() (*dblease.LeasedWriter, error) {
		return dblease.AcquireWriter(nil, path, dblease.KindSharedSocial, time.Second)
	}
	svc := NewSocialService(repo, WithWriterAcquirer(acquirer))

	err := svc.ToggleMatchFavorite(context.Background(), domain.MatchFavoriteRequest{
		PlayerSlug: "spartan-a",
		MatchID:    "match-001",
		Favorited:  true,
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !repo.favorites["spartan-a:match-001"] {
		t.Error("favorite should have been written")
	}
	// Vérifier que le writer a été release : aucune fuite.
	dblease.AssertNoLeasedWriters(t)
}

// TestSocialService_NoAcquirer_BehavesLikeBefore — non-régression : les 3
// tests existants passent NewSocialService(repo) sans option, comportement
// strictement identique.
func TestSocialService_NoAcquirer_BehavesLikeBefore(t *testing.T) {
	repo := &mockSocialRepo{favorites: make(map[string]bool)}
	svc := NewSocialService(repo) // pas d'option

	err := svc.ToggleMatchFavorite(context.Background(), domain.MatchFavoriteRequest{
		PlayerSlug: "spartan-a",
		MatchID:    "match-001",
		Favorited:  true,
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !repo.favorites["spartan-a:match-001"] {
		t.Error("favorite should have been written")
	}
}
