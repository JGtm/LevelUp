package handlers

// prestige_squads_buildmembers_test.go — non-régression buildSquadMembers :
// le créateur DOIT être injecté comme membre (clé du fix "escouade legacy sans
// créateur" côté frontend player-agnostic), dédupliqué, gamertag résolu.

import "testing"

func TestBuildSquadMembers_CreatorIncludedFirstAndDeduped(t *testing.T) {
	body := createSquadBody{
		Name:      "Big Bsses",
		CreatedBy: "jgtm",
		Members: []squadMemberInput{
			{XUID: "xa", Gamertag: "AllyA"},
			{XUID: "px"}, // le créateur figure AUSSI dans members -> ne doit pas doublonner
			{XUID: "xb"}, // sans gamertag body -> fallback annuaire
		},
	}
	slugByXUID := map[string]string{"px": "jgtm"}
	gamertagByXUID := map[string]string{"px": "JGtm", "xb": "AllyB"}

	members := buildSquadMembers(body, "px", slugByXUID, gamertagByXUID)

	if len(members) != 3 {
		t.Fatalf("want 3 membres (px, xa, xb dédupliqués), got %d (%+v)", len(members), members)
	}
	// Créateur en tête, tagué user_id + gamertag résolu.
	if members[0].Xuid != "px" || members[0].UserID != "jgtm" || members[0].Gamertag != "JGtm" {
		t.Errorf("créateur mal formé: %+v", members[0])
	}
	// Pas de doublon du créateur.
	count := 0
	for _, m := range members {
		if m.Xuid == "px" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("créateur dédupliqué attendu (1 occurrence), got %d", count)
	}
	// Gamertag fallback annuaire pour xb (absent du body).
	for _, m := range members {
		if m.Xuid == "xb" && m.Gamertag != "AllyB" {
			t.Errorf("xb gamertag fallback annuaire attendu 'AllyB', got %q", m.Gamertag)
		}
	}
}
