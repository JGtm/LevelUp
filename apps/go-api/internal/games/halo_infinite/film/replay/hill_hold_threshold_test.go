package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// hill_hold_threshold_test.go — LE SEUIL DE GARDE LU AUX POINTS DU MATCH, sans un octet de film :
// les series de garde et les courbes de score sont construites a la main, en images.

// htMatch decrit un match par ses intervalles : pour chaque point, le camp qui marque et la garde
// prise par chaque camp depuis le point precedent. Un tic par image, a partir de l'image qui suit
// le point precedent ; le dernier tic du camp qui marque tombe sur l'image du point.
type htPoint struct {
	team  int
	gains [2]int
}

// htBuild rend les series de garde par camp et les courbes de score correspondantes.
func htBuild(points []htPoint) ([]TeamHold, []TeamScore) {
	var hold [2][]ScoreTick
	var score [2][]ScoreTick
	cumul, marques, debut := [2]int{}, [2]int{}, 0
	for _, p := range points {
		fin := debut
		for team := 0; team < 2; team++ {
			for i := 1; i <= p.gains[team]; i++ {
				cumul[team]++
				hold[team] = append(hold[team], ScoreTick{T: debut + i, V: cumul[team]})
				if debut+i > fin {
					fin = debut + i
				}
			}
		}
		marques[p.team]++
		score[p.team] = append(score[p.team], ScoreTick{T: fin, V: marques[p.team]})
		debut = fin
	}
	var holds []TeamHold
	var teams []TeamScore
	for team := 0; team < 2; team++ {
		id := team
		if len(hold[team]) > 0 {
			holds = append(holds, TeamHold{TeamID: &id, Ticks: hold[team]})
		}
		teams = append(teams, TeamScore{TeamID: &id, Total: score[team]})
	}
	return holds, teams
}

// TestSeuilGardeMesureDansLeFilm — LE SEUIL EST LA GARDE DU CAMP QUI MARQUE, la valeur la plus
// frequente sur les points du match : 35, 35, 34, 35 (le 34 est une seconde interrompue) rend 35,
// sans aucune table.
func TestSeuilGardeMesureDansLeFilm(t *testing.T) {
	holds, teams := htBuild([]htPoint{
		{team: 1, gains: [2]int{17, 35}},
		{team: 1, gains: [2]int{28, 35}},
		{team: 0, gains: [2]int{34, 27}},
		{team: 1, gains: [2]int{0, 35}},
	})
	fb := fallback.NouveauCompteur()
	got, cov := resolveHoldThreshold(holds, teams, 0, fb)
	if got != 35 || cov.source != holdThresholdFromFilm || cov.film != 35 || cov.points != 4 {
		t.Fatalf("seuil %d, lecture %+v ; attendu 35 mesure dans le film sur 4 points", got, cov)
	}
	if n := fb.Compte(fallback.NomSeuilGardeTableDeVariante); n != 0 {
		t.Errorf("repli de table compte %d fois alors que le film porte le seuil", n)
	}
}

// TestSeuilGardeEgaliteVaAuPlusHaut — UN TIC PERDU, JAMAIS UN TIC DE TROP : a egalite de
// frequence, la valeur la plus haute est le seuil (40, 39 rend 40).
func TestSeuilGardeEgaliteVaAuPlusHaut(t *testing.T) {
	holds, teams := htBuild([]htPoint{
		{team: 0, gains: [2]int{39, 5}},
		{team: 1, gains: [2]int{11, 40}},
	})
	if got, _ := resolveHoldThreshold(holds, teams, 0, nil); got != 40 {
		t.Errorf("seuil %d, attendu 40", got)
	}
}

// TestSeuilGardePointFusionneNeGagnePas — UN POINT ABSENT DE LA COURBE FUSIONNE DEUX INTERVALLES
// (73 tics) : le maximum le prendrait pour le seuil, la valeur la plus frequente l'ecarte.
func TestSeuilGardePointFusionneNeGagnePas(t *testing.T) {
	holds, teams := htBuild([]htPoint{
		{team: 0, gains: [2]int{73, 44}},
		{team: 0, gains: [2]int{40, 27}},
		{team: 0, gains: [2]int{40, 38}},
	})
	if got, _ := resolveHoldThreshold(holds, teams, 0, nil); got != 40 {
		t.Errorf("seuil %d, attendu 40 (le 73 est un intervalle fusionne)", got)
	}
}

// TestSeuilGardeIntervalleIncoherentEcarte — UN AUTRE CAMP A GARDE PLUS QUE CELUI QUI MARQUE : il
// manque un point a la courbe, l'intervalle n'est pas une mesure et se compte en `Rejected`.
func TestSeuilGardeIntervalleIncoherentEcarte(t *testing.T) {
	holds, teams := htBuild([]htPoint{
		{team: 1, gains: [2]int{20, 40}},
		{team: 1, gains: [2]int{77, 66}},
	})
	got, cov := resolveHoldThreshold(holds, teams, 0, nil)
	if got != 40 || cov.points != 1 || cov.rejected != 1 {
		t.Errorf("seuil %d, lecture %+v ; attendu 40 sur 1 point, 1 ecarte", got, cov)
	}
}

// TestSeuilGardeLeFilmPrimeSurLaTable — DESACCORD : le seuil publie est celui du film, la table
// ne sert pas, et la contradiction se compte au registre (donc dans `coverage.fallbacks[]`).
func TestSeuilGardeLeFilmPrimeSurLaTable(t *testing.T) {
	holds, teams := htBuild([]htPoint{
		{team: 0, gains: [2]int{35, 3}},
		{team: 0, gains: [2]int{35, 10}},
	})
	fb := fallback.NouveauCompteur()
	got, cov := resolveHoldThreshold(holds, teams, 40, fb)
	if got != 35 || cov.source != holdThresholdFromFilm || !cov.tableDisagrees || cov.table != 40 {
		t.Errorf("seuil %d, lecture %+v ; attendu 35 du film, desaccord avec la table 40", got, cov)
	}
	if n := fb.Compte(fallback.NomSeuilGardeTableContredite); n != 1 {
		t.Errorf("contradiction film / table comptee %d fois, attendu 1", n)
	}
	if fb.Compte(fallback.NomSeuilGardeTableDeVariante) != 0 {
		t.Error("le repli de table ne doit pas se compter quand le film donne le seuil")
	}
	accordFb := fallback.NouveauCompteur()
	if _, accord := resolveHoldThreshold(holds, teams, 35, accordFb); accord.tableDisagrees ||
		accordFb.Compte(fallback.NomSeuilGardeTableContredite) != 0 {
		t.Error("film et table d'accord : aucune contradiction a compter")
	}
}

// TestSeuilGardeSansPointRepliTable — MATCH SANS POINT : la table sert de repli, NOMME et COMPTE ;
// sans entree, aucun seuil.
func TestSeuilGardeSansPointRepliTable(t *testing.T) {
	id0, id1 := 0, 1
	holds := []TeamHold{
		{TeamID: &id0, Ticks: []ScoreTick{{T: 5, V: 1}, {T: 6, V: 2}}},
		{TeamID: &id1, Ticks: []ScoreTick{{T: 8, V: 1}}},
	}
	teams := []TeamScore{{TeamID: &id0}, {TeamID: &id1}}
	fb := fallback.NouveauCompteur()
	got, cov := resolveHoldThreshold(holds, teams, 40, fb)
	if got != 40 || cov.source != holdThresholdFromTable || cov.film != 0 {
		t.Errorf("seuil %d, lecture %+v ; attendu 40 de la table", got, cov)
	}
	if n := fb.Compte(fallback.NomSeuilGardeTableDeVariante); n != 1 {
		t.Errorf("repli de table compte %d fois, attendu 1", n)
	}
	if got, cov := resolveHoldThreshold(holds, teams, 0, nil); got != 0 || cov.source != "" {
		t.Errorf("sans point ni table : seuil %d, source %q ; attendu aucun", got, cov.source)
	}
}

// TestSeuilGardeUnTicParPointPasDeBarre — SCORE A LA SECONDE : chaque tic est un point, le seuil
// mesure vaut 1 et rien n'est publie (une barre pleine a chaque seconde ne dirait rien).
func TestSeuilGardeUnTicParPointPasDeBarre(t *testing.T) {
	var points []htPoint
	for i := 0; i < 30; i++ {
		points = append(points, htPoint{team: i % 2, gains: [2]int{1 - i%2, i % 2}})
	}
	holds, teams := htBuild(points)
	got, cov := resolveHoldThreshold(holds, teams, 0, nil)
	if got != 0 || !cov.oneTickPerPoint || cov.source != "" {
		t.Errorf("seuil %d, lecture %+v ; attendu aucun seuil, un tic par point", got, cov)
	}
	tl := &ScoreTimeline{Teams: teams}
	if attachHillHold(tl, &ScoreInput{}, nil, nil, testClock(), nil); tl.HoldTicks != nil || tl.HoldTicksPerPoint != nil {
		t.Error("sans seuil, ni serie ni denominateur ne se publient")
	}
}

// TestSeuilGardeHorsModeAColline — BASES, TOTAL CONTROL ET LE RESTE : sans mode a colline, ni la
// serie, ni le seuil ne sont publies, ni la table lue, meme si la variante a une entree de table.
func TestSeuilGardeHorsModeAColline(t *testing.T) {
	recs := modeRamp(6, 0, 2_000, 1_000, 1, 2, 3)
	recs = append(recs, types.StatRecord{TimeMS: 2_500, Slot: 10, Comps: map[int]types.StatValue{23: {A: 1}}})
	fb := fallback.NouveauCompteur()
	tl, _, lu := assembleScoreTimeline(&ScoreInput{Records: recs, HoldTicksPerPointTable: 35}, nil, testClock(), fb)
	if tl == nil || tl.HoldTicks != nil || tl.HoldTicksPerPoint != nil || lu != nil {
		t.Errorf("hors mode a colline : aucune garde attendue, obtenu %+v / %+v", tl, lu)
	}
	if fb.Compte(fallback.NomSeuilGardeTableDeVariante) != 0 {
		t.Error("hors mode a colline : la table ne doit pas servir de repli")
	}
	if _, _, lu = assembleScoreTimeline(&ScoreInput{Records: recs, HillScoring: true}, nil, testClock(), nil); lu == nil {
		t.Error("mode a colline : la lecture du seuil doit etre rendue pour le journal")
	}
}
