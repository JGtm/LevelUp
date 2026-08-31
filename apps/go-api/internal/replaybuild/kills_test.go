package replaybuild

import (
	"testing"

	"levelup/go-api/internal/analysis/replay"
)

// kills_test.go — la résolution d'identité hors ligne (gamertag/xuid: -> xuid), sur données
// 100 % synthétiques (aucun film) : PLAN_RETOURS_UTILISATEUR_2026-08-29 §LOT F.1.

func TestResolveKillIdentity_Gamertag(t *testing.T) {
	byGamertag := map[string]uint64{"Chocoboflor": 2533274822053343}
	xuid, ok := resolveKillIdentity("Chocoboflor", byGamertag)
	if !ok || xuid != 2533274822053343 {
		t.Fatalf("xuid=%d ok=%v, attendu 2533274822053343/true", xuid, ok)
	}
}

func TestResolveKillIdentity_ReplixuidPrefixe(t *testing.T) {
	// "xuid:<N>" est la forme de repli du décodeur (killsource.XUIDNamePrefix) quand le
	// film ne porte aucun gamertag pour ce joueur : le nombre EST déjà le xuid, aucune
	// table à consulter.
	xuid, ok := resolveKillIdentity("xuid:2535469190789936", nil)
	if !ok || xuid != 2535469190789936 {
		t.Fatalf("xuid=%d ok=%v, attendu 2535469190789936/true", xuid, ok)
	}
}

func TestResolveKillIdentity_GamertagInconnu(t *testing.T) {
	xuid, ok := resolveKillIdentity("Inconnu", map[string]uint64{"Autre": 1})
	if ok || xuid != 0 {
		t.Fatalf("xuid=%d ok=%v, attendu 0/false (gamertag hors roster)", xuid, ok)
	}
}

func TestResolveKillIdentity_ReplixuidNonDecimalRefuse(t *testing.T) {
	// Un repli mal formé ne doit jamais paniquer ni renvoyer une fausse identité.
	xuid, ok := resolveKillIdentity("xuid:pas-un-nombre", nil)
	if ok || xuid != 0 {
		t.Fatalf("xuid=%d ok=%v, attendu 0/false (repli non décimal)", xuid, ok)
	}
}

func TestGamertagXUIDIndex_PremierGagneEnCasDeDoublon(t *testing.T) {
	deaths := []replay.Death{
		{XUID: 111, Gamertag: "Joueur"},
		{XUID: 222, Gamertag: "Joueur"}, // même nom, second xuid : ne doit rien écraser
		{XUID: 333, Gamertag: ""},       // sans gamertag : absent de l'index
	}
	idx := gamertagXUIDIndex(deaths)
	if len(idx) != 1 {
		t.Fatalf("index = %d entrées, attendu 1", len(idx))
	}
	if idx["Joueur"] != 111 {
		t.Errorf("xuid de \"Joueur\" = %d, attendu 111 (le premier)", idx["Joueur"])
	}
}

func TestGamertagXUIDIndex_ListeVide(t *testing.T) {
	if idx := gamertagXUIDIndex(nil); len(idx) != 0 {
		t.Fatalf("index = %+v, attendu vide", idx)
	}
}
