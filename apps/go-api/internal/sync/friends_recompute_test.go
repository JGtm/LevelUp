// Tests purs (sans DuckDB) pour RecomputeIsWithFriendsCore.
//
// Couverture :
//   - Early return quand friendGamertags vide → no-op gracieux
//   - WithFriendsLoader câble correctement (API smoke)
//
// Les chemins critiques (loadMatchesWithFriends + updateIsWithFriendsBatch +
// integration leases) restent couverts par les tests integration sous tag
// `integration` (DuckDB :memory:).
package sync

import (
	"context"
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
