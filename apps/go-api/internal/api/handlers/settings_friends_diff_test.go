// settings_friends_diff_test.go : tests purs pour la détection de diff
// friend_gamertags (§4 plan Squad/Sessions). Indépendant de DuckDB.
package handlers

import "testing"

func TestFriendGamertagsChanged_LengthDiff(t *testing.T) {
	if !friendGamertagsChanged([]string{"a"}, []string{"a", "b"}) {
		t.Fatal("ajout d'un ami doit retourner true")
	}
	if !friendGamertagsChanged([]string{"a", "b"}, []string{"a"}) {
		t.Fatal("retrait d'un ami doit retourner true")
	}
}

func TestFriendGamertagsChanged_SameSetReturnsFalse(t *testing.T) {
	if friendGamertagsChanged([]string{"Alice", "Bob"}, []string{"Bob", "Alice"}) {
		t.Fatal("ordre inverse mais même set → pas de changement")
	}
}

func TestFriendGamertagsChanged_CaseInsensitive(t *testing.T) {
	if friendGamertagsChanged([]string{"Alice"}, []string{"alice"}) {
		t.Fatal("différence de casse seule → pas de changement")
	}
}

func TestFriendGamertagsChanged_TrimWhitespace(t *testing.T) {
	if friendGamertagsChanged([]string{"  Alice  "}, []string{"Alice"}) {
		t.Fatal("différence de whitespace seule → pas de changement")
	}
}

func TestFriendGamertagsChanged_DifferentMembers(t *testing.T) {
	if !friendGamertagsChanged([]string{"Alice"}, []string{"Bob"}) {
		t.Fatal("membre différent → changement")
	}
}

func TestFriendGamertagsChanged_BothEmptyReturnsFalse(t *testing.T) {
	if friendGamertagsChanged(nil, nil) {
		t.Fatal("nil + nil → pas de changement")
	}
	if friendGamertagsChanged([]string{}, []string{}) {
		t.Fatal("vide + vide → pas de changement")
	}
	if friendGamertagsChanged(nil, []string{}) {
		t.Fatal("nil + vide → pas de changement")
	}
}
