package friendstore

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *FriendStore {
	t.Helper()
	return NewFriendStore(filepath.Join(t.TempDir(), "player_friends.json"))
}

func TestGet_AbsentPlayerReturnsEmptyNotError(t *testing.T) {
	s := newTestStore(t)
	got, err := s.Get("2533274800000001")
	if err != nil {
		t.Fatalf("Get sur fichier absent = %v, want nil", err)
	}
	if got == nil {
		t.Fatal("Get = nil, want slice vide (contrat non nullable)")
	}
	if len(got) != 0 {
		t.Errorf("Get = %v, want vide", got)
	}
}

func TestGet_EmptyXUIDReturnsEmpty(t *testing.T) {
	s := newTestStore(t)
	got, err := s.Get("")
	if err != nil || len(got) != 0 {
		t.Fatalf("Get(\"\") = %v, %v ; want liste vide, nil", got, err)
	}
}

func TestSetThenGet_RoundTrip(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Set("xuid1", "Moi", []string{"Alpha", "Bravo"}); err != nil {
		t.Fatalf("Set = %v", err)
	}
	got, err := s.Get("xuid1")
	if err != nil {
		t.Fatalf("Get = %v", err)
	}
	if len(got) != 2 || got[0] != "Alpha" || got[1] != "Bravo" {
		t.Errorf("Get = %v, want [Alpha Bravo]", got)
	}
	if ts, err := s.UpdatedAt("xuid1"); err != nil || ts == "" {
		t.Errorf("UpdatedAt = %q, %v ; want horodatage non vide", ts, err)
	}
}

func TestSet_NormalisationTrimDedupCaseAndSelf(t *testing.T) {
	s := newTestStore(t)
	got, err := s.Set("xuid1", "Moi", []string{"  Alpha ", "ALPHA", "", "moi", "Bravo"})
	if err != nil {
		t.Fatalf("Set = %v", err)
	}
	if len(got) != 2 || got[0] != "Alpha" || got[1] != "Bravo" {
		t.Fatalf("Set normalisé = %v, want [Alpha Bravo]", got)
	}
	// La normalisation est persistée, pas seulement retournée.
	stored, err := s.Get("xuid1")
	if err != nil || len(stored) != 2 {
		t.Errorf("Get après Set = %v, %v ; want 2 entrées", stored, err)
	}
}

func TestSet_ReplacesWholeList(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Set("xuid1", "", []string{"Alpha", "Bravo"}); err != nil {
		t.Fatalf("Set 1 = %v", err)
	}
	if _, err := s.Set("xuid1", "", []string{"Charlie"}); err != nil {
		t.Fatalf("Set 2 = %v", err)
	}
	got, _ := s.Get("xuid1")
	if len(got) != 1 || got[0] != "Charlie" {
		t.Errorf("Get = %v, want [Charlie] (remplacement complet)", got)
	}
}

func TestSet_EmptyXUIDRejected(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Set("", "", []string{"Alpha"}); err == nil {
		t.Fatal("Set sans xuid = nil, want ErrMissingXUID")
	}
}

func TestSet_IsolatesPlayers(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Set("xuid1", "", []string{"Alpha"}); err != nil {
		t.Fatalf("Set xuid1 = %v", err)
	}
	if _, err := s.Set("xuid2", "", []string{"Bravo"}); err != nil {
		t.Fatalf("Set xuid2 = %v", err)
	}
	one, _ := s.Get("xuid1")
	two, _ := s.Get("xuid2")
	if len(one) != 1 || one[0] != "Alpha" || len(two) != 1 || two[0] != "Bravo" {
		t.Errorf("listes croisées : xuid1=%v xuid2=%v", one, two)
	}
}

func TestAll_ReturnsEveryPlayer(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Set("xuid1", "", []string{"Alpha"}); err != nil {
		t.Fatalf("Set = %v", err)
	}
	if _, err := s.Set("xuid2", "", []string{"Bravo", "Charlie"}); err != nil {
		t.Fatalf("Set = %v", err)
	}
	all, err := s.All()
	if err != nil {
		t.Fatalf("All = %v", err)
	}
	if len(all) != 2 || len(all["xuid1"]) != 1 || len(all["xuid2"]) != 2 {
		t.Errorf("All = %v, want 2 joueurs (1 et 2 amis)", all)
	}
}

func TestGet_NullGamertagsNormalizedToEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "player_friends.json")
	if err := os.WriteFile(path, []byte(`{"version":"1.0","friends":{"xuid1":{"gamertags":null}}}`), 0o600); err != nil {
		t.Fatalf("écriture fixture = %v", err)
	}
	got, err := NewFriendStore(path).Get("xuid1")
	if err != nil {
		t.Fatalf("Get = %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("Get = %v, want slice vide non nil", got)
	}
}

func TestLoad_CorruptedFileReturnsExplicitError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "player_friends.json")
	if err := os.WriteFile(path, []byte("{ ceci n'est pas du json"), 0o600); err != nil {
		t.Fatalf("écriture fixture = %v", err)
	}
	s := NewFriendStore(path)
	if _, err := s.Get("xuid1"); err == nil {
		t.Fatal("Get sur fichier corrompu = nil, want erreur explicite")
	}
	if _, err := s.All(); err == nil {
		t.Fatal("All sur fichier corrompu = nil, want erreur explicite")
	}
}
