package main

// cmd_backfill_squad_creators_test.go — idempotence du backfill : le créateur
// n'est réinséré QUE s'il manque du roster. squadHasMember porte cette décision
// (présent -> skip ; absent -> insert), donc une relance est sans effet de bord.

import (
	"testing"

	"levelup/go-api/internal/prestige"
)

func TestSquadHasMember_IdempotencyGuard(t *testing.T) {
	roster := []prestige.SquadMember{
		{Xuid: "px", UserID: "jgtm"},
		{Xuid: "xa"},
	}
	if !squadHasMember(roster, "px") {
		t.Errorf("créateur déjà membre -> squadHasMember(px) doit être true (skip, pas de doublon)")
	}
	if squadHasMember(roster, "missing") {
		t.Errorf("créateur absent -> squadHasMember doit être false (réinsertion)")
	}
	if squadHasMember(nil, "px") {
		t.Errorf("roster vide -> false")
	}
}
