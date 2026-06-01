package authz

import (
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

func TestEnforced(t *testing.T) {
	cases := []struct {
		name     string
		demoMode bool
		authMode string
		want     bool
	}{
		{"demo désactive", true, AuthModePassword, false},
		{"demo désactive même xbox", true, AuthModeXbox, false},
		{"password actif", false, AuthModePassword, true},
		{"xbox actif", false, AuthModeXbox, true},
		{"none ouvert", false, AuthModeNone, false},
		{"vide ouvert", false, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Enforced(tc.demoMode, tc.authMode); got != tc.want {
				t.Fatalf("Enforced(%v, %q) = %v, want %v", tc.demoMode, tc.authMode, got, tc.want)
			}
		})
	}
}

func TestCanAccessPlayer(t *testing.T) {
	admin := &domain.User{Username: "boss", Role: domain.RoleAdmin, XUID: "111"}
	owner := &domain.User{Username: "alice", Role: domain.RoleUser, XUID: "222"}
	unlinked := &domain.User{Username: "bob", Role: domain.RoleUser, XUID: ""}

	cases := []struct {
		name        string
		enforced    bool
		user        *domain.User
		profileXUID string
		want        bool
	}{
		{"non enforced → ouvert même sans user", false, nil, "999", true},
		{"admin accède à tout", true, admin, "999", true},
		{"propriétaire accède à son xuid", true, owner, "222", true},
		{"user refusé sur xuid étranger", true, owner, "999", false},
		{"user non lié ne possède rien", true, unlinked, "999", false},
		{"user non lié refusé même si profil xuid vide", true, unlinked, "", false},
		{"nil user refusé quand enforced", true, nil, "222", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanAccessPlayer(tc.enforced, tc.user, tc.profileXUID); got != tc.want {
				t.Fatalf("CanAccessPlayer(%v, %+v, %q) = %v, want %v",
					tc.enforced, tc.user, tc.profileXUID, got, tc.want)
			}
		})
	}
}

// fakeLookup implémente UserLookup pour les tests de CurrentUser.
type fakeLookup struct {
	byName map[string]*domain.User
	byXUID map[string]*domain.User
}

func (f fakeLookup) Get(username string) (*domain.User, error) {
	if u, ok := f.byName[username]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}

func (f fakeLookup) GetByXUID(xuid string) (*domain.User, error) {
	if u, ok := f.byXUID[xuid]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}

func strptr(s string) *string { return &s }

func TestCurrentUser(t *testing.T) {
	alice := &domain.User{Username: "alice", Role: domain.RoleUser, XUID: "222"}
	xboxUser := &domain.User{Username: "spartan", Role: domain.RoleUser, XUID: "333", Gamertag: "Spartan117"}
	lookup := fakeLookup{
		byName: map[string]*domain.User{"alice": alice},
		byXUID: map[string]*domain.User{"333": xboxUser},
	}

	t.Run("nil session → nil", func(t *testing.T) {
		if u := CurrentUser(nil, lookup); u != nil {
			t.Fatalf("attendu nil, obtenu %+v", u)
		}
	})

	t.Run("nil lookup → nil", func(t *testing.T) {
		sess := &domain.SessionData{Username: strptr("alice")}
		if u := CurrentUser(sess, nil); u != nil {
			t.Fatalf("attendu nil, obtenu %+v", u)
		}
	})

	t.Run("username mode password", func(t *testing.T) {
		sess := &domain.SessionData{Username: strptr("alice")}
		u := CurrentUser(sess, lookup)
		if u == nil || u.XUID != "222" {
			t.Fatalf("attendu alice (xuid 222), obtenu %+v", u)
		}
	})

	t.Run("identité Halo liée mode xbox", func(t *testing.T) {
		sess := &domain.SessionData{LinkedHaloIdentity: &domain.HaloIdentity{XUID: "333", Gamertag: "Spartan117"}}
		u := CurrentUser(sess, lookup)
		if u == nil || u.XUID != "333" {
			t.Fatalf("attendu spartan (xuid 333), obtenu %+v", u)
		}
	})

	t.Run("identité liée sans compte persisté → user synthétisé", func(t *testing.T) {
		sess := &domain.SessionData{LinkedHaloIdentity: &domain.HaloIdentity{XUID: "444", Gamertag: "Ghost"}}
		u := CurrentUser(sess, lookup)
		if u == nil || u.XUID != "444" || u.Role != domain.RoleUser {
			t.Fatalf("attendu user synthétisé (xuid 444, rôle user), obtenu %+v", u)
		}
	})

	t.Run("session vide → nil", func(t *testing.T) {
		if u := CurrentUser(&domain.SessionData{}, lookup); u != nil {
			t.Fatalf("attendu nil, obtenu %+v", u)
		}
	})
}
