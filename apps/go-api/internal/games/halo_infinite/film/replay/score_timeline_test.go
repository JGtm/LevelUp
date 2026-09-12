package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// score_timeline_test.go — L'ASSEMBLAGE DE LA COURBE DE SCORE, SANS UN OCTET DE FILM.
//
// Tout ce qui est verifie ici l'est sur des enregistrements CONSTRUITS : le decodage du film a
// ses propres tests (objectiveevents), celui-ci ne teste que l'assemblage — la pose sur la
// grille de frames, la correction d'origine, les deux formes (manche / total) et les trois
// issues de l'identite des camps. Un test qui exigerait un film ne dirait pas laquelle des deux
// couches a bouge.

const (
	testInterval = 100
	testOrigin   = 1_000
	testFrames   = 100
)

// testClock est l'horloge des tests : frame 0 a 1 000 ms de film, pas de 100 ms, 100 frames.
func testClock() scoreClock {
	return scoreClock{intervalMS: testInterval, frames: testFrames, originMS: testOrigin}
}

// statRec construit un enregistrement d'entite.
func statRec(timeMS, slot, round int, comps map[int]objectiveevents.StatValue) objectiveevents.StatRecord {
	return objectiveevents.StatRecord{TimeMS: timeMS, Slot: slot, Round: round, Comps: comps}
}

// modeRamp construit une rampe de score de MODE (composant 0, valeur A) pour un slot et une
// manche. Trois emissions au moins sont necessaires pour qu'une manche soit tenue pour reelle.
func modeRamp(slot, round, startMS, stepMS int, values ...int64) []objectiveevents.StatRecord {
	out := make([]objectiveevents.StatRecord, 0, len(values))
	for i, v := range values {
		out = append(out, statRec(startMS+i*stepMS, slot, round,
			map[int]objectiveevents.StatValue{0: {A: v}}))
	}
	return out
}

// coreLine construit l'emission des trois compteurs de base d'un slot (frags et morts en
// composant 2, assistances en composant 3), plus son score personnel (composant 1, valeur B).
func coreLine(slot, round, timeMS int, kills, deaths, assists, personal int64) []objectiveevents.StatRecord {
	return []objectiveevents.StatRecord{
		statRec(timeMS, slot, round, map[int]objectiveevents.StatValue{
			1: {B: personal},
			2: {A: kills, B: deaths},
			3: {A: assists},
		}),
	}
}

// TestScoreTimelineRoundsAndTotal — LES DEUX FORMES, ET LE TOTAL EST LA SOMME DES MANCHES.
//
// C'est le defaut que la lecture par manche a ferme : le score de mode repart de zero a chaque
// manche, donc la derniere valeur brute donne la DERNIERE MANCHE pour le match. Le total doit
// valoir la somme, et rester croissant de bout en bout.
func TestScoreTimelineRoundsAndTotal(t *testing.T) {
	var recs []objectiveevents.StatRecord
	recs = append(recs, modeRamp(6, 0, 2_000, 1_000, 1, 2, 3)...)
	recs = append(recs, modeRamp(6, 1, 6_000, 1_000, 1, 2, 3)...)

	tl, cov := buildScoreTimeline(&ScoreInput{Records: recs}, nil, testClock())
	if tl == nil || len(tl.Teams) != 1 {
		t.Fatalf("attendu 1 courbe d'equipe, obtenu %+v", tl)
	}
	team := tl.Teams[0]
	if len(team.Rounds) != 2 {
		t.Fatalf("%d manche(s) publiee(s), attendu 2 : %+v", len(team.Rounds), team.Rounds)
	}
	if team.Rounds[0].Round != 0 || team.Rounds[1].Round != 1 {
		t.Errorf("manches publiees dans le desordre : %+v", team.Rounds)
	}
	// La manche 1 repart de 1 : c'est bien la valeur PROPRE a la manche.
	if got := team.Rounds[1].Points[0].V; got != 1 {
		t.Errorf("premiere valeur de la manche 1 = %d, attendu 1 (valeur propre a la manche)", got)
	}
	// Le total, lui, cumule : 3 a la fin de la manche 0, puis 4, 5, 6.
	wantTotal := []int{1, 2, 3, 4, 5, 6}
	if len(team.Total) != len(wantTotal) {
		t.Fatalf("%d point(s) de total, attendu %d : %+v", len(team.Total), len(wantTotal), team.Total)
	}
	for i, want := range wantTotal {
		if team.Total[i].V != want {
			t.Errorf("total[%d] = %d, attendu %d", i, team.Total[i].V, want)
		}
	}
	if cov.Rounds != 2 {
		t.Errorf("couverture : %d manche(s), attendu 2", cov.Rounds)
	}
	if cov.Oracle != ScoreOracleDisplayed || !cov.ModeSupported {
		t.Errorf("couverture inattendue : %+v", cov)
	}
}

// TestScoreTimelineSubtractsOrigin — L'ORIGINE EST RETRANCHEE, ET C'EST LE POINT.
//
// Les enregistrements sont dates depuis le premier paquet du FILM ; la grille de frames compte
// depuis le premier paquet de POSITION. Publier le quotient brut decalerait toute la courbe de
// l'ecart entre les deux zeros (3,6 s a 50,8 s selon le match).
func TestScoreTimelineSubtractsOrigin(t *testing.T) {
	recs := modeRamp(6, 0, 2_000, 1_000, 1, 2, 3)
	tl, _ := buildScoreTimeline(&ScoreInput{Records: recs}, nil, testClock())
	if tl == nil || len(tl.Teams) != 1 {
		t.Fatalf("aucune courbe d'equipe : %+v", tl)
	}
	// 2 000 ms de film, origine 1 000 ms, pas 100 ms -> frame 10 (et non 20).
	for i, want := range []int{10, 20, 30} {
		if got := tl.Teams[0].Total[i].T; got != want {
			t.Errorf("total[%d].t = %d, attendu %d (origine non retranchee ?)", i, got, want)
		}
	}
}

// TestScoreTimelineDropsEmissionsBeforeFrameZero — CE QUI PRECEDE LA FRAME 0 N'EST PAS POSABLE.
//
// La mise en place du match emet avant le premier paquet de position. Ces emissions n'ont pas
// de frame ou aller ; les ecraser sur la frame 0 inventerait un score au coup d'envoi.
func TestScoreTimelineDropsEmissionsBeforeFrameZero(t *testing.T) {
	recs := modeRamp(6, 0, 500, 1_000, 1, 2, 3) // 500, 1 500, 2 500 ms de film
	tl, _ := buildScoreTimeline(&ScoreInput{Records: recs}, nil, testClock())
	if tl == nil || len(tl.Teams) != 1 {
		t.Fatalf("aucune courbe d'equipe : %+v", tl)
	}
	if got := len(tl.Teams[0].Total); got != 2 {
		t.Fatalf("%d point(s) publie(s), attendu 2 (l'emission a 500 ms precede la frame 0)", got)
	}
	if tl.Teams[0].Total[0].T != 5 {
		t.Errorf("premier point a la frame %d, attendu 5", tl.Teams[0].Total[0].T)
	}
}

// TestScoreTimelineTeamIdentityByFinalScore — PREUVE (a) : le score final designe le camp, et
// l'ordre des slots n'est PAS l'ordre des camps.
func TestScoreTimelineTeamIdentityByFinalScore(t *testing.T) {
	var recs []objectiveevents.StatRecord
	recs = append(recs, modeRamp(6, 0, 2_000, 1_000, 1, 2, 3)...) // final 3
	recs = append(recs, modeRamp(8, 0, 2_500, 1_000, 1, 2)...)    // final 2
	scores := [2]int{2, 3}                                        // team_0 = 2, team_1 = 3

	tl, cov := buildScoreTimeline(&ScoreInput{Records: recs, TeamScores: &scores}, nil, testClock())
	if cov.TeamIdentity != ScoreIdentityFinal {
		t.Fatalf("identite = %q, attendu %q", cov.TeamIdentity, ScoreIdentityFinal)
	}
	if len(tl.Teams) != 2 {
		t.Fatalf("%d courbe(s) d'equipe, attendu 2", len(tl.Teams))
	}
	// Slot 6 finit a 3 = team_1 ; slot 8 finit a 2 = team_0. L'ordre des slots est inverse.
	if tl.Teams[0].TeamID == nil || *tl.Teams[0].TeamID != 1 {
		t.Errorf("slot 6 : camp %v, attendu 1", tl.Teams[0].TeamID)
	}
	if tl.Teams[1].TeamID == nil || *tl.Teams[1].TeamID != 0 {
		t.Errorf("slot 8 : camp %v, attendu 0", tl.Teams[1].TeamID)
	}
}

// TestScoreTimelineTargetScore — LA CIBLE DE VICTOIRE, ET SA GARDE.
//
// La cible ne vient jamais du film : c'est la table mesuree de la variante, fournie par
// l'appelant. Trois issues : publiee telle quelle quand les finals la respectent, ABSENTE
// quand un final la depasse (table perimee -> se taire), absente quand elle est inconnue.
func TestScoreTimelineTargetScore(t *testing.T) {
	recs := func() []objectiveevents.StatRecord {
		var out []objectiveevents.StatRecord
		out = append(out, modeRamp(6, 0, 2_000, 1_000, 1, 2, 3)...) // final 3
		out = append(out, modeRamp(8, 0, 2_500, 1_000, 1, 2)...)    // final 2
		return out
	}

	tl, _ := buildScoreTimeline(&ScoreInput{Records: recs(), TargetScore: 3}, nil, testClock())
	if tl.TargetScore == nil || *tl.TargetScore != 3 {
		t.Fatalf("cible = %v, attendu 3 (finals 3 et 2, cible respectee)", tl.TargetScore)
	}

	tl, _ = buildScoreTimeline(&ScoreInput{Records: recs(), TargetScore: 2}, nil, testClock())
	if tl.TargetScore != nil {
		t.Fatalf("cible = %d publiee alors qu'un final (3) la depasse : la table est perimee, "+
			"le calque doit se taire", *tl.TargetScore)
	}

	tl, _ = buildScoreTimeline(&ScoreInput{Records: recs()}, nil, testClock())
	if tl.TargetScore != nil {
		t.Fatalf("cible = %d publiee sans cible en entree", *tl.TargetScore)
	}
}

// TestScoreTimelineTeamIdentityByFrags — PREUVE (b) : a EGALITE de scores, c'est la somme des
// frags de chaque camp qui designe le slot d'equipe.
func TestScoreTimelineTeamIdentityByFrags(t *testing.T) {
	in := fragsIdentityInput()
	tl, cov := buildScoreTimeline(in, nil, testClock())
	if cov.TeamIdentity != ScoreIdentityFrags {
		t.Fatalf("identite = %q, attendu %q", cov.TeamIdentity, ScoreIdentityFrags)
	}
	if len(tl.Teams) != 2 {
		t.Fatalf("%d courbe(s) d'equipe, attendu 2", len(tl.Teams))
	}
	// Slot 6 porte 5 frags = camp 0 (3 + 2) ; slot 8 en porte 7 = camp 1 (4 + 3).
	if tl.Teams[0].TeamID == nil || *tl.Teams[0].TeamID != 0 {
		t.Errorf("slot 6 : camp %v, attendu 0", tl.Teams[0].TeamID)
	}
	if tl.Teams[1].TeamID == nil || *tl.Teams[1].TeamID != 1 {
		t.Errorf("slot 8 : camp %v, attendu 1", tl.Teams[1].TeamID)
	}
}

// TestScoreTimelineTeamIdentityUnresolved — PREUVE (c) : sans score du registre ET sans camp
// par joueur, les courbes sortent SANS `teamId`. Jamais devinees.
func TestScoreTimelineTeamIdentityUnresolved(t *testing.T) {
	in := fragsIdentityInput()
	in.TeamByXUID = nil // le pont des camps disparait ; les scores etaient deja a egalite

	tl, cov := buildScoreTimeline(in, nil, testClock())
	if cov.TeamIdentity != ScoreIdentityUnresolved {
		t.Fatalf("identite = %q, attendu %q", cov.TeamIdentity, ScoreIdentityUnresolved)
	}
	for i, team := range tl.Teams {
		if team.TeamID != nil {
			t.Errorf("courbe %d : camp %d publie alors que l'identite n'est pas resolue", i, *team.TeamID)
		}
	}
}

// TestScoreTimelinePublishesOnlyPairedPlayers — un slot que le triplet n'apparie pas n'est PAS
// publie : attribuer les compteurs d'un joueur a un autre serait invisible a l'ecran.
func TestScoreTimelinePublishesOnlyPairedPlayers(t *testing.T) {
	in := fragsIdentityInput()
	// La ligne du joueur du slot 16 disparait : son triplet ne designe plus aucune ligne.
	in.Lines = in.Lines[:3]

	tl, _ := buildScoreTimeline(in, nil, testClock())
	if len(tl.Players) != 3 {
		t.Fatalf("%d joueur(s) publie(s), attendu 3 : %+v", len(tl.Players), tl.Players)
	}
	for _, p := range tl.Players {
		if p.XUID == "x16" {
			t.Errorf("le slot non apparie est publie sous %q", p.XUID)
		}
	}
}

// TestScoreTimelinePlayerCounters — les quatre compteurs d'un joueur sont publies, chacun sous
// ses deux formes, et leur total vaut la ligne de match.
func TestScoreTimelinePlayerCounters(t *testing.T) {
	tl, _ := buildScoreTimeline(fragsIdentityInput(), nil, testClock())
	var p *PlayerScore
	for i := range tl.Players {
		if tl.Players[i].XUID == "x10" {
			p = &tl.Players[i]
		}
	}
	if p == nil {
		t.Fatal("le joueur x10 n'est pas publie")
	}
	// Les quatre valeurs sont EPINGLEES, et elles sont distinctes deux a deux : intervertir
	// deux compteurs fait rougir ce test, ce qu'un simple « non nul » ne ferait pas.
	for _, cas := range []struct {
		nom  string
		got  int
		want int
	}{
		{"frags", lastValue(p.Kills), 3},
		{"morts", lastValue(p.Deaths), 1},
		{"assistances", lastValue(p.Assists), 2},
		{"score personnel", lastValue(p.Score), 300},
	} {
		if cas.got != cas.want {
			t.Errorf("%s de x10 = %d, attendu %d", cas.nom, cas.got, cas.want)
		}
	}
}

// TestScoreTimelinePropagatesTruncation — un score TRONQUE se dit. Le publier en silence serait
// un mensonge : la courbe s'arrete avant la fin du match.
func TestScoreTimelinePropagatesTruncation(t *testing.T) {
	recs := modeRamp(6, 0, 2_000, 1_000, 1, 2, 3)
	_, cov := buildScoreTimeline(&ScoreInput{Records: recs, Truncated: true}, nil, testClock())
	if !cov.Truncated {
		t.Error("la troncature n'est pas propagee a la couverture")
	}
}

// TestScoreTimelineWithoutInput — sans entree, ni calque NI couverture : l'absence de
// couverture dit « rien n'a ete fourni a lire », ce qu'aucun compteur a zero ne dirait.
func TestScoreTimelineWithoutInput(t *testing.T) {
	tl, cov := buildScoreTimeline(nil, nil, testClock())
	if tl != nil || cov != nil {
		t.Errorf("sans entree : calque %v, couverture %v ; attendu nil et nil", tl, cov)
	}
}

// TestScoreTimelineEmptyFilmKeepsCoverage — un film sans enregistrement rend une couverture
// PRESENTE et vide de courbes : c'est ce qui le distingue du cas precedent.
func TestScoreTimelineEmptyFilmKeepsCoverage(t *testing.T) {
	tl, cov := buildScoreTimeline(&ScoreInput{}, nil, testClock())
	if tl != nil {
		t.Errorf("calque publie sur un film vide : %+v", tl)
	}
	if cov == nil || cov.ModeSupported || cov.Points != 0 {
		t.Errorf("couverture inattendue sur un film vide : %+v", cov)
	}
}

// fragsIdentityInput construit un match a deux camps ou les scores du registre sont EGAUX
// (la preuve (a) ne s'applique pas) et ou les frags departagent : camp 0 = 3 + 2, camp 1 = 4 + 3.
func fragsIdentityInput() *ScoreInput {
	var recs []objectiveevents.StatRecord
	recs = append(recs, modeRamp(6, 0, 2_000, 1_000, 1, 2, 3)...)
	recs = append(recs, modeRamp(8, 0, 2_500, 1_000, 1, 2, 3)...)
	// Les slots d'equipe repliquent le total de frags de leur camp (composant 2, valeur A).
	recs = append(recs, statRec(5_000, 6, 0, map[int]objectiveevents.StatValue{2: {A: 5}}))
	recs = append(recs, statRec(5_000, 8, 0, map[int]objectiveevents.StatValue{2: {A: 7}}))
	recs = append(recs, coreLine(10, 0, 3_000, 3, 1, 2, 300)...)
	recs = append(recs, coreLine(12, 0, 3_100, 2, 2, 3, 250)...)
	recs = append(recs, coreLine(14, 0, 3_200, 4, 3, 6, 400)...)
	recs = append(recs, coreLine(16, 0, 3_300, 3, 4, 5, 350)...)

	scores := [2]int{3, 3} // egalite : la preuve (a) ne peut pas trancher
	return &ScoreInput{
		Records: recs,
		Lines: []objectiveevents.PlayerLine{
			{XUID: "x10", Kills: 3, Deaths: 1, Assists: 2},
			{XUID: "x12", Kills: 2, Deaths: 2, Assists: 3},
			{XUID: "x14", Kills: 4, Deaths: 3, Assists: 6},
			{XUID: "x16", Kills: 3, Deaths: 4, Assists: 5},
		},
		TeamByXUID: map[string]int{"x10": 0, "x12": 0, "x14": 1, "x16": 1},
		TeamScores: &scores,
	}
}

// lastValue rend la derniere valeur du total d'une serie, ou 0.
func lastValue(s ScoreSeries) int {
	if len(s.Total) == 0 {
		return 0
	}
	return s.Total[len(s.Total)-1].V
}

// TestScoreTicksKeepOnlyChanges — LE CONTRAT DU CHAMP EST « AUX CHANGEMENTS SEULEMENT », et il
// etait viole (correctif de revue R1).
//
// Le composant 2 porte les frags en valeur A et les morts en valeur B : il est reemis des que
// l'UNE des deux bouge, si bien que la serie des morts repetait sa valeur a chaque frag. Une
// emission qui ne change rien n'est pas un point.
func TestScoreTicksKeepOnlyChanges(t *testing.T) {
	// Valeurs : 1, 1, 1, 2, 2, 3 — trois paliers, donc trois points, aux instants ou la
	// valeur a ete ATTEINTE (la premiere emission du palier).
	pts := []objectiveevents.ScorePoint{
		{TimeMS: 2_000, Value: 1}, {TimeMS: 3_000, Value: 1}, {TimeMS: 4_000, Value: 1},
		{TimeMS: 5_000, Value: 2}, {TimeMS: 6_000, Value: 2}, {TimeMS: 7_000, Value: 3},
	}
	got := scoreTicksOf(pts, testClock())
	want := []ScoreTick{{T: 10, V: 1}, {T: 40, V: 2}, {T: 60, V: 3}}
	if len(got) != len(want) {
		t.Fatalf("%d point(s) publie(s), attendu %d : %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("point %d = %+v, attendu %+v", i, got[i], want[i])
		}
	}
}

// TestScoreTicksSameFrameDifferentValues — la deduplication ne se fait PAS sur `T` seul : deux
// valeurs differentes dans la meme frame sont un vrai changement, et c'est l'etat de FIN de
// frame que le client dessine.
func TestScoreTicksSameFrameDifferentValues(t *testing.T) {
	pts := []objectiveevents.ScorePoint{
		{TimeMS: 2_000, Value: 1}, {TimeMS: 2_040, Value: 2}, {TimeMS: 2_090, Value: 3},
	}
	got := scoreTicksOf(pts, testClock())
	if len(got) != 1 || got[0] != (ScoreTick{T: 10, V: 3}) {
		t.Fatalf("obtenu %+v, attendu un seul point {T:10 V:3} (l'etat de fin de frame)", got)
	}
}

// TestScoreTimelinePlayerCountersHaveNoRepeats — le meme controle, mais SUR LE CALQUE ASSEMBLE :
// aucune serie publiee ne doit porter deux points consecutifs de meme valeur.
func TestScoreTimelinePlayerCountersHaveNoRepeats(t *testing.T) {
	tl, cov := buildScoreTimeline(repeatedCountersInput(), nil, testClock())
	if tl == nil || len(tl.Players) == 0 {
		t.Fatal("aucun joueur publie")
	}
	compte := 0
	for _, p := range tl.Players {
		for nom, s := range map[string]ScoreSeries{
			"score": p.Score, "kills": p.Kills, "deaths": p.Deaths, "assists": p.Assists,
		} {
			compte += countSeriesPoints(s)
			for i := 1; i < len(s.Total); i++ {
				if s.Total[i].V == s.Total[i-1].V {
					t.Errorf("%s de %s : point %d repete la valeur %d", nom, p.XUID, i, s.Total[i].V)
				}
			}
		}
	}
	for _, team := range tl.Teams {
		compte += countSeriesPoints(ScoreSeries{Rounds: team.Rounds, Total: team.Total})
	}
	// La couverture publie le compte REEL des points, pas une estimation.
	if cov.Points != compte {
		t.Errorf("coverage.score.points = %d, compte reel %d", cov.Points, compte)
	}
	if cov.Points == 0 {
		t.Error("coverage.score.points nul alors que le calque publie des courbes")
	}
}

// repeatedCountersInput construit un joueur dont le composant 2 est reemis a chaque frag :
// les morts y REPETENT leur valeur, exactement comme dans un film reel.
func repeatedCountersInput() *ScoreInput {
	var recs []objectiveevents.StatRecord
	recs = append(recs, modeRamp(6, 0, 2_000, 1_000, 1, 2, 3)...)
	for i, kills := range []int64{1, 2, 3} {
		recs = append(recs, statRec(3_000+i*500, 10, 0, map[int]objectiveevents.StatValue{
			1: {B: int64(100 * (i + 1))},
			2: {A: kills, B: 2}, // les morts ne bougent pas : 2, 2, 2
			3: {A: 1},
		}))
	}
	return &ScoreInput{
		Records: recs,
		Lines:   []objectiveevents.PlayerLine{{XUID: "x10", Kills: 3, Deaths: 2, Assists: 1}},
	}
}

// TestScoreTimelineIdentityWhenFilmDiffersFromRegistry — LE CAS STRONGHOLDS / KOTH, ET C'EST
// CELUI QUI JUSTIFIE LA PREUVE (b) (revue R1).
//
// Le film porte le score AFFICHE ; l'API en porte un autre dans deux modes (des ticks en
// Strongholds, des secondes de colline en KOTH). Les finales du film ne concordent alors avec
// AUCUN des deux scores du registre : la preuve (a) ne peut pas trancher — et elle ne doit
// surtout pas trancher au plus proche — tandis que la somme des frags, elle, departage.
func TestScoreTimelineIdentityWhenFilmDiffersFromRegistry(t *testing.T) {
	var recs []objectiveevents.StatRecord
	// Le film dit 200 et 126 ; le registre dit 193 et 112 (des TICKS, pas des points).
	recs = append(recs, modeRamp(6, 0, 2_000, 500, 198, 199, 200)...)
	recs = append(recs, modeRamp(8, 0, 2_100, 500, 124, 125, 126)...)
	// Les frags departagent : camp 0 = 5 (3 + 2), camp 1 = 7 (4 + 3).
	recs = append(recs, statRec(5_000, 6, 0, map[int]objectiveevents.StatValue{2: {A: 5}}))
	recs = append(recs, statRec(5_000, 8, 0, map[int]objectiveevents.StatValue{2: {A: 7}}))
	recs = append(recs, coreLine(10, 0, 3_000, 3, 1, 2, 300)...)
	recs = append(recs, coreLine(12, 0, 3_100, 2, 2, 3, 250)...)
	recs = append(recs, coreLine(14, 0, 3_200, 4, 3, 6, 400)...)
	recs = append(recs, coreLine(16, 0, 3_300, 3, 4, 5, 350)...)

	scores := [2]int{193, 112} // DIFFERENTS l'un de l'autre : (a) s'applique... et echoue
	in := &ScoreInput{
		Records: recs,
		Lines: []objectiveevents.PlayerLine{
			{XUID: "x10", Kills: 3, Deaths: 1, Assists: 2},
			{XUID: "x12", Kills: 2, Deaths: 2, Assists: 3},
			{XUID: "x14", Kills: 4, Deaths: 3, Assists: 6},
			{XUID: "x16", Kills: 3, Deaths: 4, Assists: 5},
		},
		TeamByXUID: map[string]int{"x10": 0, "x12": 0, "x14": 1, "x16": 1},
		TeamScores: &scores,
	}

	tl, cov := buildScoreTimeline(in, nil, testClock())
	if cov.TeamIdentity != ScoreIdentityFrags {
		t.Fatalf("identite = %q, attendu %q : (a) doit DECLINER quand les finales du film ne "+
			"valent aucun des scores du registre, et (b) doit resoudre",
			cov.TeamIdentity, ScoreIdentityFrags)
	}
	if len(tl.Teams) != 2 {
		t.Fatalf("%d courbe(s) d'equipe, attendu 2", len(tl.Teams))
	}
	if tl.Teams[0].TeamID == nil || *tl.Teams[0].TeamID != 0 {
		t.Errorf("slot 6 (5 frags) : camp %v, attendu 0", tl.Teams[0].TeamID)
	}
	if tl.Teams[1].TeamID == nil || *tl.Teams[1].TeamID != 1 {
		t.Errorf("slot 8 (7 frags) : camp %v, attendu 1", tl.Teams[1].TeamID)
	}
	// La courbe publiee reste celle du FILM — l'oracle est le score affiche, pas celui de l'API.
	if got := tl.Teams[0].Total[len(tl.Teams[0].Total)-1].V; got != 200 {
		t.Errorf("finale du camp 0 = %d, attendu 200 (la valeur du FILM, pas les 193 de l'API)", got)
	}
}
