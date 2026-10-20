package replay

// zone_states_hill_garde_test.go — LA COLLINE D UNE PERIODE SE PLACE PAR LA GARDE (regle de
// zone_states_hill_garde.go), sur des enregistrements construits : la garde l emporte sur la
// grappe pendant les montees de la jauge ; une garde qui ne couvre aucune zone ecarte la periode ;
// une garde illisible revient aux votes de la jauge par un repli compte.

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// hillGardeCase : designateur au slot 40 (une bascule a 300 : periodes [100 ; 299] et [300 ; 599]),
// proprietaire au slot 41 (camp 0 de 120 a 299, camp 1 de 320), une montee de jauge au slot 43
// de 280 a 290. Les positions sont fournies par l appelant.
func hillGardeCase(tracks []Track) (ZoneInput, zoneCtx) {
	reads := []grammar.ManagedPropertyRead{
		zoneChainedReadAt(40, 100, grammar.ManagedPropertyTagStringID, 0x78F81557),
		zoneChainedReadAt(40, 300, grammar.ManagedPropertyTagStringID, 0x8727C0FF),
		zoneReadAt(41, 120, grammar.ManagedPropertyTagU32, 0),
		zoneReadAt(41, 300, grammar.ManagedPropertyTagU32, zoneNeutralOwner),
		zoneReadAt(41, 320, grammar.ManagedPropertyTagU32, 1),
	}
	reads = append(reads, zoneGaugeSamplesAt(43, 280, 285, 290, 980)...)
	in := zoneTestInput(reads)
	in.Hill = true
	c := zoneTestCtx(nil, tracks)
	c.fb = fallback.NouveauCompteur()
	return in, c
}

// pointsIn rend une position par frame de [t0, t1] au point (x, 0, 0).
func pointsIn(t0, t1 int, x float32) []Point {
	var out []Point
	for f := t0; f <= t1; f++ {
		out = append(out, pointAt(f, x, 0, 0))
	}
	return out
}

// refOfPeriod rend la zone qui porte l intervalle couvrant la frame, ou -1.
func refOfPeriod(states []ZoneState, frame int) int {
	for _, s := range states {
		for _, sp := range s.Spans {
			if frame >= sp.T0 && frame <= sp.T1 {
				return s.ZoneRef
			}
		}
	}
	return -1
}

// TestCollineGardePrimeSurLaGrappe — pendant la montee de jauge de la premiere periode, un joueur
// du camp 1 passe dans la zone 0 ; le camp 0, proprietaire, se tient dans la zone 1 toute la
// periode. La colline est la zone 1 : la grappe pendant la montee aurait dit la zone 0.
func TestCollineGardePrimeSurLaGrappe(t *testing.T) {
	gardien := Track{XUID: "2533", Team: 0, Points: pointsIn(120, 299, 20.5)}
	passant := Track{XUID: "2535", Team: 1, Points: pointsIn(270, 299, -19.5)}
	passant.Points = append(passant.Points, pointsIn(320, 599, -19.5)...)
	in, c := hillGardeCase([]Track{gardien, passant})
	states, cov := buildZoneStates(context.Background(), in, c)
	if got := refOfPeriod(states, 200); got != 1 {
		t.Errorf("periode 1 posee sur la zone %d, attendu 1 (la ou se tient le proprietaire)", got)
	}
	if got := refOfPeriod(states, 400); got != 0 {
		t.Errorf("periode 2 posee sur la zone %d, attendu 0", got)
	}
	if n := c.fb.Compte(fallback.NomCollineVotesSansGarde); n != 0 {
		t.Errorf("repli sans garde declenche %d fois, attendu 0", n)
	}
	if cov.Unpaired != 0 {
		t.Errorf("unpaired = %d, attendu 0", cov.Unpaired)
	}
}

// TestCollineGardeHorsCatalogueEcartee — le camp proprietaire tient la colline LOIN de toute zone
// du catalogue, et ne passe dans la zone 1 qu un quart du temps : la periode n est pas publiee
// (elle ne se pose pas sur la zone 1), et elle se compte.
func TestCollineGardeHorsCatalogueEcartee(t *testing.T) {
	pts := pointsIn(120, 249, 60)
	pts = append(pts, pointsIn(250, 299, 20.5)...)
	gardien := Track{XUID: "2533", Team: 0, Points: pts}
	autre := Track{XUID: "2535", Team: 1, Points: pointsIn(320, 599, -19.5)}
	in, c := hillGardeCase([]Track{gardien, autre})
	states, cov := buildZoneStates(context.Background(), in, c)
	if got := refOfPeriod(states, 200); got != -1 {
		t.Errorf("periode 1 publiee sur la zone %d, attendu aucune", got)
	}
	if cov.Unpaired != 1 {
		t.Errorf("unpaired = %d, attendu 1", cov.Unpaired)
	}
	if got := refOfPeriod(states, 400); got != 0 {
		t.Errorf("periode 2 posee sur la zone %d, attendu 0", got)
	}
}

// TestCollineGardeIllisibleRevientALaJauge — aucune vie n a d equipe lue : la garde ne dit rien,
// la periode se place par la grappe pendant la montee de jauge, et le repli se compte.
func TestCollineGardeIllisibleRevientALaJauge(t *testing.T) {
	sansCamp := Track{XUID: "2533", Team: -1, Points: pointsIn(280, 290, -19.5)}
	in, c := hillGardeCase([]Track{sansCamp})
	states, _ := buildZoneStates(context.Background(), in, c)
	if got := refOfPeriod(states, 200); got != 0 {
		t.Errorf("periode 1 posee sur la zone %d, attendu 0 (grappe pendant la montee)", got)
	}
	if n := c.fb.Compte(fallback.NomCollineVotesSansGarde); n == 0 {
		t.Error("repli sans garde non compte")
	}
}
