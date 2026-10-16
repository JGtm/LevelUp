package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

// buildPoolSharing construit le bootstrap d'une session et retourne pool_sharing.
func buildPoolSharing(t *testing.T, svc *BootstrapService, sess *domain.SessionData) string {
	t.Helper()
	resp, err := svc.Build(context.Background(), sess)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return resp.PoolSharing
}

// Sans résolveur câblé, l'état est « unavailable » : le front ne pose aucune question.
func TestBuild_PoolSharing_NoResolver_Unavailable(t *testing.T) {
	svc := NewBootstrapService(&config.AppConfig{}, &mockBootRepo{})
	sess := &domain.SessionData{LinkedHaloIdentity: &domain.HaloIdentity{Gamertag: "GT", XUID: "123"}}

	if got := buildPoolSharing(t, svc, sess); got != domain.PoolSharingUnavailable {
		t.Fatalf("pool_sharing = %q, want unavailable", got)
	}
}

// Sans identité propre (compte password sans Xbox, session vide), le résolveur n'est même
// pas consulté : rien à mettre en commun.
func TestBuild_PoolSharing_NoOwnIdentity_Unavailable(t *testing.T) {
	called := false
	svc := NewBootstrapService(&config.AppConfig{}, &mockBootRepo{}).
		WithPoolSharingResolver(func(string) string { called = true; return domain.PoolSharingUndecided })

	if got := buildPoolSharing(t, svc, &domain.SessionData{}); got != domain.PoolSharingUnavailable {
		t.Fatalf("pool_sharing = %q, want unavailable", got)
	}
	if called {
		t.Fatal("le résolveur ne doit pas être appelé sans xuid propre")
	}
}

// L'état rendu est celui du résolveur, pour chacun des quatre états.
func TestBuild_PoolSharing_PassesResolverState(t *testing.T) {
	for _, state := range []string{
		domain.PoolSharingGranted, domain.PoolSharingDeclined,
		domain.PoolSharingUndecided, domain.PoolSharingUnavailable,
	} {
		t.Run(state, func(t *testing.T) {
			svc := NewBootstrapService(&config.AppConfig{}, &mockBootRepo{}).
				WithPoolSharingResolver(func(string) string { return state })
			sess := &domain.SessionData{LinkedHaloIdentity: &domain.HaloIdentity{Gamertag: "GT", XUID: "123"}}

			if got := buildPoolSharing(t, svc, sess); got != state {
				t.Fatalf("pool_sharing = %q, want %q", got, state)
			}
		})
	}
}

// Comme reauth_required : l'état est celui du COMPTE CONNECTÉ (son xuid), jamais celui du
// joueur affiché. Un admin qui consulte un autre joueur ne décide pas à sa place.
func TestBuild_PoolSharing_ScopedToOwnAccount(t *testing.T) {
	const ownXUID = "2533274823110022"
	const otherXUID = "2535469190789936"
	admin := &domain.User{Username: "jgtm_xbox", XUID: ownXUID, Role: domain.RoleAdmin}
	lookup := fakeBootstrapLookup{
		byName: map[string]*domain.User{"jgtm_xbox": admin},
		byXUID: map[string]*domain.User{ownXUID: admin},
	}
	var asked []string
	svc := NewBootstrapService(&config.AppConfig{}, &mockBootRepo{}).
		WithUserLookup(lookup).
		WithPoolSharingResolver(func(xuid string) string {
			asked = append(asked, xuid)
			if xuid == ownXUID {
				return domain.PoolSharingGranted
			}
			return domain.PoolSharingUndecided
		})

	username := "jgtm_xbox"
	other := "Chocoboflor"
	sess := &domain.SessionData{
		Username:           &username,
		CurrentPlayerSlug:  &other,
		LinkedHaloIdentity: &domain.HaloIdentity{Gamertag: "Chocoboflor", XUID: otherXUID},
	}

	if got := buildPoolSharing(t, svc, sess); got != domain.PoolSharingGranted {
		t.Fatalf("pool_sharing = %q, want granted (état du compte connecté)", got)
	}
	for _, x := range asked {
		if x != ownXUID {
			t.Fatalf("le résolveur a été interrogé pour %q, seul le xuid propre %q est attendu", x, ownXUID)
		}
	}
}
