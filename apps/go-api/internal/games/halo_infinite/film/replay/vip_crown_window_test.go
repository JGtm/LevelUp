package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// vip_crown_window_test.go — LA FENETRE DU REJEU NE JETTE PAS LES PREMIERS VIP.
//
// Les premiers VIP de chaque camp sont designes au lancement de la partie, avant le premier
// paquet de position (la frame 0 du rejeu). Une periode ouverte avant la frame 0 et fermee
// apres commence a la frame 0 ; seule une periode deja close avant la frame 0 est hors fenetre.

func TestVipCrownPeriodeOuverteAvantLaFrame0(t *testing.T) {
	// Axe : frame 0 a 1 500 ms d'horloge du match, 100 ms par frame, 200 frames.
	ctx := matchClock{origin: 1_500_000, step: 100_000, frames: 200}
	identity := objectives.FlatRoundIdentity(map[int]string{22: "111", 23: "222"})
	events := []objectives.NamedEvent{
		// Close avant la frame 0 (mort a 1 000 ms) : hors fenetre.
		{Slot: 23, TimeMS: 500, Stat: objectives.StatVipSelected},
		// Ouverte avant la frame 0, morte a 2 000 ms : frames 0 -> 5.
		{Slot: 22, TimeMS: 1200, Stat: objectives.StatVipSelected},
		// Ouverte dans la fenetre : inchangee.
		{Slot: 22, TimeMS: 3000, Stat: objectives.StatVipSelected},
	}
	deaths := []types.Death{
		{XUID: 222, TimeMS: 1000},
		{XUID: 111, TimeMS: 2000},
		{XUID: 111, TimeMS: 4000},
	}
	scan := VipCrownScan{Scanned: true, Events: events, Identity: identity, Deaths: deaths}

	got, cov := buildVipCrown(scan, ctx)

	want := []VipPeriod{
		{XUID: "111", T0: 0, T1: 5, Closed: true},
		{XUID: "111", T0: 15, T1: 25, Closed: true},
	}
	if len(got) != len(want) {
		t.Fatalf("periodes = %+v, attendu %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("periode %d = %+v, attendu %+v", i, got[i], want[i])
		}
	}
	if cov.Selections != 3 || cov.Periods != 2 || cov.OutOfWindow != 1 {
		t.Errorf("couverture = %+v, attendu 3 selections, 2 periodes, 1 hors fenetre", *cov)
	}
}
