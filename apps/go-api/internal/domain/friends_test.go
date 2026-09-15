package domain

import (
	"strings"
	"testing"
)

func TestNormalizeFriendGamertags_TrimDedupAndSelf(t *testing.T) {
	got := NormalizeFriendGamertags([]string{" Alpha ", "ALPHA", "", "  ", "Moi", "Bravo"}, "moi")
	if len(got) != 2 || got[0] != "Alpha" || got[1] != "Bravo" {
		t.Errorf("= %v, want [Alpha Bravo]", got)
	}
}

func TestNormalizeFriendGamertags_NilInputReturnsEmptySlice(t *testing.T) {
	got := NormalizeFriendGamertags(nil, "")
	if got == nil {
		t.Fatal("= nil, want slice vide (contrat non nullable)")
	}
	if len(got) != 0 {
		t.Errorf("= %v, want vide", got)
	}
}

func TestNormalizeFriendGamertags_EmptyOwnGamertagKeepsAll(t *testing.T) {
	got := NormalizeFriendGamertags([]string{"Alpha", "Bravo"}, "")
	if len(got) != 2 {
		t.Errorf("= %v, want les 2 entrées", got)
	}
}

func TestValidateFriendGamertags_Bounds(t *testing.T) {
	if msg := ValidateFriendGamertags([]string{"Alpha"}); msg != "" {
		t.Errorf("liste valide refusée : %q", msg)
	}
	tooMany := make([]string, MaxFriendGamertags+1)
	for i := range tooMany {
		tooMany[i] = "Ami"
	}
	if msg := ValidateFriendGamertags(tooMany); msg != "too_many" {
		t.Errorf("= %q, want too_many", msg)
	}
	long := strings.Repeat("x", MaxFriendGamertagLen+1)
	if msg := ValidateFriendGamertags([]string{long}); msg != "gamertag_too_long" {
		t.Errorf("= %q, want gamertag_too_long", msg)
	}
	atLimit := strings.Repeat("x", MaxFriendGamertagLen)
	if msg := ValidateFriendGamertags([]string{atLimit}); msg != "" {
		t.Errorf("gamertag à la borne refusé : %q", msg)
	}
}
