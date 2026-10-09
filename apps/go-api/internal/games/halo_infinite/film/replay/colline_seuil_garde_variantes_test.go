package replay

// colline_seuil_garde_variantes_test.go — LE SEUIL DE GARDE D UNE VARIANTE KOTH, MESURE SUR SON
// FILM SANS CUISSON : pour les films dont la carte manque aux bornes de quantification (la cuisson
// echoue « carte hors catalogue »), le compteur de garde `comp 23 A` du statborg se lit quand
// meme — il ne depend d aucune position. A chaque point marque, l UNION des tics de chaque camp
// depuis le point precedent (meme regle que `buildHoldTicks`) ; le camp qui marque doit rendre
// le seuil.
//
// SOUS GARDE D ENVIRONNEMENT, un film par processus, lecture seule, aucune base :
//
//	$env:CGO_ENABLED=0
//	$env:HOLD_FILM="C:/.../data/cache/film_chunks/e449a696"
//	$env:HOLD_LINES="xuid:camp:frags:morts:assistances;..."   (lignes de match, relevees en base)
//	$env:HOLD_SCORES="180:79"                                  (facultatif : scores finaux du registre)
//	go test -count=1 -run TestCollineSeuilGardeVariantes -v ./internal/games/halo_infinite/film/replay/

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestCollineSeuilGardeVariantes publie, point par point, l union des tics de garde par camp.
func TestCollineSeuilGardeVariantes(t *testing.T) {
	dir, raw := os.Getenv("HOLD_FILM"), os.Getenv("HOLD_LINES")
	if dir == "" || raw == "" {
		t.Skip("mesure non demandee : HOLD_FILM / HOLD_LINES vides")
	}
	lines, team := holdLignes(t, raw)
	recs := objectives.StatRecords(p2aBobine(t, dir))
	identity := objectives.SlotIdentityFrom(recs, lines, nil)
	series := objectives.SeriesTotal(recs, holdTicksComponent, false, nil)
	t.Logf("%d enregistrements, %d slots nommes sur %d lignes", len(recs), len(identity), len(lines))
	prev := 0
	for _, p := range holdPointsMarques(recs) {
		t.Logf("point slot %d -> %d a %d ms : UNION par camp %v", p.Slot, p.Value, p.TimeMS,
			e1cUnion(identity, team, series, prev, p.TimeMS))
		prev = p.TimeMS
	}
	// LA REGLE DE PRODUCTION SUR CE FILM, SANS CUISSON (E8 du plan KOTH, 2026-10-09) : le calque de
	// score assemble en mode a colline, horloge du film (origine 0, pas de 100 ms). Ce que la
	// cuisson publierait comme seuil, et d'ou il viendrait.
	// `HOLD_SCORES="s0:s1"` (scores finaux du registre) ouvre l'identite des camps par le score final.
	in := &ScoreInput{Records: recs, Lines: lines, TeamByXUID: team, HillScoring: true}
	var s0, s1 int
	if _, err := fmt.Sscanf(os.Getenv("HOLD_SCORES"), "%d:%d", &s0, &s1); err == nil {
		in.TeamScores = &[2]int{s0, s1}
	}
	c := scoreClock{intervalMS: 100, frames: 1 << 20}
	tl, cov, lu := assembleScoreTimeline(in, nil, c, nil)
	var seuil any = "absent"
	if tl != nil && tl.HoldTicksPerPoint != nil {
		seuil = *tl.HoldTicksPerPoint
	}
	t.Logf("regle de production : seuil publie %v, couverture %+v, identite des camps %s", seuil, *lu, cov.TeamIdentity)
}

// holdLignes decode `HOLD_LINES`.
func holdLignes(t *testing.T, raw string) ([]types.PlayerLine, map[string]int) {
	t.Helper()
	var lines []types.PlayerLine
	team := map[string]int{}
	for _, l := range strings.Split(raw, ";") {
		var xuid string
		var tm, k, d, a int
		if _, err := fmt.Sscanf(strings.ReplaceAll(l, ":", " "), "%s %d %d %d %d", &xuid, &tm, &k, &d, &a); err != nil {
			t.Fatalf("ligne %q illisible : %v", l, err)
		}
		lines = append(lines, types.PlayerLine{XUID: xuid, Kills: k, Deaths: d, Assists: a})
		team[xuid] = tm
	}
	return lines, team
}

// holdPointsMarques rend les montees du score de mode des slots d equipe, triees par instant.
func holdPointsMarques(recs []types.StatRecord) []types.ScorePoint {
	score := objectives.SeriesTotal(recs, objectives.ModeScoreComponent(), true, nil)
	var pts []types.ScorePoint
	for s, ss := range score {
		var prev int64
		for _, p := range ss {
			if p.Value > prev {
				prev = p.Value
				pts = append(pts, types.ScorePoint{TimeMS: p.TimeMS, Slot: s, Value: p.Value})
			}
		}
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].TimeMS < pts[j].TimeMS })
	return pts
}
