package objectiveevents

import "testing"

// Les tests de ce fichier sont adosses au cache film de dev (meme convention que
// extract_test.go : SKIP propre si le film est absent, donc verts en CI sans films).
//
// Ils encodent la verite terrain etablie le 2026-08-01 par confrontation avec une capture
// Cheat Engine et avec un releve terrain ecrit a l'oeil AVANT tout decodage
// (.ai/ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md §15).

// finalScores rend le dernier score connu de chaque slot d'equipe.
func finalScores(pts []ScorePoint) map[int]int64 {
	out := map[int]int64{}
	for _, p := range pts {
		if IsTeamSlot(p.Slot) {
			out[p.Slot] = p.Value
		}
	}
	return out
}

// scoreAt rend le score de chaque slot d'equipe a l'instant tMS.
func scoreAt(pts []ScorePoint, tMS int) map[int]int64 {
	out := map[int]int64{}
	for _, p := range pts {
		if p.TimeMS <= tMS && IsTeamSlot(p.Slot) {
			out[p.Slot] = p.Value
		}
	}
	return out
}

// TestScoreCurveStrongholdsGroundTruth — le releve terrain du 2026-07-31 sur `696a9d7c`
// (Strongholds, Nomad) donne « 3:10 : score 69-30 » et l'API donne 200-94 au final. Les
// deux doivent tomber, sinon le decodage n'est pas celui du score de mode.
func TestScoreCurveStrongholdsGroundTruth(t *testing.T) {
	bobine, ok := newDiskFilm(t, "696a9d7c")
	if !ok {
		t.Skipf("film 696a9d7c absent du cache (%s=%q)", filmCacheEnv, cacheRoot())
	}
	pts := ScoreCurve(bobine)
	if len(pts) == 0 {
		t.Fatal("aucune emission de score decodee")
	}

	final := finalScores(pts)
	if final[6] != 200 || final[8] != 94 {
		t.Errorf("score final = %d-%d, attendu 200-94 (API)", final[6], final[8])
	}

	// Releve terrain : « 3:10 — score 69 - 30 ».
	at190 := scoreAt(pts, 190_000)
	if at190[6] != 69 || at190[8] != 30 {
		t.Errorf("score a t=190 s = %d-%d, attendu 69-30 (releve terrain 3:10)",
			at190[6], at190[8])
	}

	// Releve terrain : « 1:30 — une equipe controle les trois bases, 21 pour l'autre ».
	if got := scoreAt(pts, 90_000)[8]; got != 21 {
		t.Errorf("score de l'equipe 8 a t=90 s = %d, attendu 21 (releve terrain 1:30)", got)
	}

	// Un score de mode ne recule jamais.
	last := map[int]int64{}
	for _, p := range pts {
		if v, seen := last[p.Slot]; seen && p.Value <= v {
			t.Fatalf("slot %d : score non monotone (%d apres %d) a t=%d ms",
				p.Slot, p.Value, v, p.TimeMS)
		}
		last[p.Slot] = p.Value
	}
}

// TestScoreCurveMatchesCTFCaptures — la preuve croisee la plus forte du chantier : en CTF
// le score d'equipe n'avance QUE sur une capture. Les captures (bursts a 6 tiers du
// footer) et les increments de score (composant 0 des paquets FRAME) sont decodes par des
// chemins qui n'ont rien en commun ; ils doivent tomber au meme instant.
func TestScoreCurveMatchesCTFCaptures(t *testing.T) {
	bobine, ok := newDiskFilm(t, "530820e5")
	if !ok {
		t.Skipf("film 530820e5 absent du cache (%s=%q)", filmCacheEnv, cacheRoot())
	}
	events := Extract("530820e5", "CTF:Arena", bobine, MapRoster{})
	teamPts := []ScorePoint{}
	for _, p := range ScoreCurve(bobine) {
		if IsTeamSlot(p.Slot) {
			teamPts = append(teamPts, p)
		}
	}
	if len(events) == 0 {
		t.Fatal("aucun event de capture decode")
	}
	if len(teamPts) != len(events) {
		t.Fatalf("%d increments de score d'equipe pour %d captures — les deux decodeurs "+
			"doivent compter pareil", len(teamPts), len(events))
	}
	// Tolerance : le score est emis dans le paquet qui SUIT la capture. 50 ms borne
	// largement l'ecart mesure (1 ms sur les 3 captures de ce film) sans autoriser un
	// appariement fortuit — les captures sont espacees de dizaines de secondes.
	const toleranceMS = 50
	for i, e := range events {
		if e.TimeMS == nil {
			t.Fatalf("capture %d sans horodatage", i)
		}
		d := teamPts[i].TimeMS - *e.TimeMS
		if d < 0 || d > toleranceMS {
			t.Errorf("capture %d a t=%d ms : increment de score a t=%d ms (ecart %d ms)",
				i, *e.TimeMS, teamPts[i].TimeMS, d)
		}
	}
	if got := finalScores(teamPts)[8]; got != int64(len(events)) {
		t.Errorf("score final de l'equipe = %d, attendu %d (une capture = un point)",
			got, len(events))
	}
}
