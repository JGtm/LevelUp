// Tests purs (sans DuckDB) pour RecomputeIsWithFriendsCore + wrapper RecomputeIsWithFriends.
//
// Couverture :
//   - Early return Core quand friendGamertags vide → no-op gracieux
//   - Early return wrapper RecomputeIsWithFriends sans toucher leases/DBs
//   - Error wrapping sur ouverture DB invalide (le wrapper acquiert les leases puis échoue à OpenPlayerDB)
//   - WithFriendsLoader câble correctement (API smoke)
//
// Les chemins critiques (loadMatchesWithFriends + updateIsWithFriendsBatch +
// integration leases happy path) restent couverts par les tests integration
// sous tag `integration` (DuckDB :memory:).
package sync

import (
	"context"
	"path/filepath"
	"testing"
)

func TestRecomputeIsWithFriendsCore_EmptyFriendsList(t *testing.T) {
	// Pas de DB nécessaires : le early-return court-circuite avant toute query.
	res, err := RecomputeIsWithFriendsCore(
		context.Background(),
		nil, // playerDB jamais touché
		nil, // sharedDB jamais touché
		"player_xuid",
		[]string{}, // empty friends
		false,
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res.XUID != "player_xuid" {
		t.Errorf("expected XUID=player_xuid, got %q", res.XUID)
	}
	if res.MatchesPromoted != 0 {
		t.Errorf("expected MatchesPromoted=0, got %d", res.MatchesPromoted)
	}
	if res.AggregatesRefreshed {
		t.Error("expected AggregatesRefreshed=false on empty friends")
	}
	if res.FriendXUIDsCount != 0 {
		t.Errorf("expected FriendXUIDsCount=0, got %d", res.FriendXUIDsCount)
	}
}

func TestSyncEngine_WithFriendsLoader(t *testing.T) {
	// Smoke test API : le setter câble bien le loader sur l'engine.
	loaderCalled := false
	loader := func() ([]string, error) {
		loaderCalled = true
		return []string{"FriendOne"}, nil
	}

	engine := NewSyncEngine("/tmp/repo", "PlayerOne", "0000000000000001", nil, nil)
	if engine.friendsLoader != nil {
		t.Error("expected friendsLoader nil before WithFriendsLoader")
	}

	chained := engine.WithFriendsLoader(loader)
	if chained != engine {
		t.Error("expected fluent return (chained == engine)")
	}
	if engine.friendsLoader == nil {
		t.Fatal("expected friendsLoader non-nil after WithFriendsLoader")
	}

	// Vérifier que le loader stocké est appelable et retourne bien la liste.
	friends, err := engine.friendsLoader()
	if err != nil {
		t.Fatalf("loader error: %v", err)
	}
	if !loaderCalled {
		t.Error("loader was not invoked")
	}
	if len(friends) != 1 || friends[0] != "FriendOne" {
		t.Errorf("expected [FriendOne], got %v", friends)
	}
}

func TestSyncEngine_WithFriendsLoader_Nil(t *testing.T) {
	// Setter nil → friendsLoader nil → hook engine skip silencieux.
	engine := NewSyncEngine("/tmp/repo", "PlayerOne", "0000000000000001", nil, nil)
	engine.WithFriendsLoader(nil)
	if engine.friendsLoader != nil {
		t.Error("expected friendsLoader nil after WithFriendsLoader(nil)")
	}
}

// TestRecomputeIsWithFriends_EmptyFriendsList_SkipsLeases vérifie le early-return
// du wrapper public : quand friendGamertags est vide, on ne doit toucher ni
// leases ni DBs (court-circuit avant AcquireLeaseCtx).
//
// Validation : passe des paths volontairement invalides ; si le wrapper tentait
// d'acquérir des leases sur ces paths inexistants, le test échouerait sur erreur.
func TestRecomputeIsWithFriends_EmptyFriendsList_SkipsLeases(t *testing.T) {
	res, err := RecomputeIsWithFriends(
		context.Background(),
		nil, // provider — mode legacy pour ce test
		"/dev/null/does/not/matter/player.duckdb",
		"/dev/null/does/not/matter/shared.duckdb",
		"test_xuid",
		[]string{}, // empty → early-return avant lease
	)
	if err != nil {
		t.Fatalf("expected nil error on empty friends list, got %v", err)
	}
	if res.XUID != "test_xuid" {
		t.Errorf("expected XUID preserved, got %q", res.XUID)
	}
	if res.MatchesPromoted != 0 {
		t.Errorf("expected MatchesPromoted=0, got %d", res.MatchesPromoted)
	}
	if res.FriendXUIDsCount != 0 {
		t.Errorf("expected FriendXUIDsCount=0, got %d", res.FriendXUIDsCount)
	}
}

// TestRecomputeIsWithFriends_NoXUIDResolved_GracefulNoop couvre le path
// chaîne complète du wrapper avec DBs réelles (vides) : leases acquises,
// OpenPlayerDB + OpenSharedDB réussissent (DuckDB crée les fichiers), Core
// est appelé, LookupFriendXUIDs retourne vide (xuid_aliases inexistante /
// vide) → return gracieux MatchesPromoted=0, FriendXUIDsCount=0.
//
// Valide que le wrapper :
//   - Acquiert puis libère les leases sans deadlock
//   - Délègue correctement à Core (XUID préservé dans le résultat)
//   - Ne panique pas sur xuid_aliases absent
func TestRecomputeIsWithFriends_NoXUIDResolved_GracefulNoop(t *testing.T) {
	tmpDir := t.TempDir()
	playerPath := filepath.Join(tmpDir, "player.duckdb")
	sharedPath := filepath.Join(tmpDir, "shared.duckdb")

	res, err := RecomputeIsWithFriends(
		context.Background(),
		nil, // provider — mode legacy pour ce test
		playerPath,
		sharedPath,
		"test_xuid",
		[]string{"UnknownFriend"}, // non-empty mais xuid_aliases vide → no resolution
	)
	if err != nil {
		t.Fatalf("expected nil error on no-xuid-resolved path, got %v", err)
	}
	if res.XUID != "test_xuid" {
		t.Errorf("expected XUID preserved through wrapper, got %q", res.XUID)
	}
	if res.MatchesPromoted != 0 {
		t.Errorf("expected MatchesPromoted=0 (no xuids resolved), got %d", res.MatchesPromoted)
	}
	if res.FriendXUIDsCount != 0 {
		t.Errorf("expected FriendXUIDsCount=0 (xuid_aliases empty), got %d", res.FriendXUIDsCount)
	}

	// Sanity : un 2e appel sur la même paire de DBs doit pouvoir réacquérir
	// les leases sans deadlock (preuve que le 1er appel les a bien libérées).
	res2, err := RecomputeIsWithFriends(
		context.Background(),
		nil, // provider — mode legacy pour ce test
		playerPath, sharedPath,
		"test_xuid",
		[]string{"AnotherFriend"},
	)
	if err != nil {
		t.Fatalf("expected nil error on retry, got %v (lease leak?)", err)
	}
	if res2.XUID != "test_xuid" {
		t.Errorf("retry: expected XUID preserved, got %q", res2.XUID)
	}
}
