package replay

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// manches_segments_test.go — LA GARDE PAR SLOT, ET LE JOURNAL DES BORNES DE MANCHE.
//
// Extrait de `manches_compteurs_test.go` le 2026-09-07 (constat N3 de la revue MANCHES-R2 : le
// fichier passait 500 lignes). La coupe suit la RESPONSABILITE : ici vivent les tests qui portent
// sur le BLOC (slot, manche) — la garde qui interdit de jeter un segment entier (revue
// MANCHES-R1), le journal qui la nomme sans se contredire (revue MANCHES-R2) et la source unique
// du debut d'une manche. Les fixtures partagees restent dans `manches_compteurs_test.go`.

// manchesFixtureRelecteur reproduit la fixture de la revue MANCHES-R1 : huit slots, la manche 1
// OUVERTE PAR TROIS SLOTS a 40 s et par les CINQ AUTRES a 70 s. La mediane basse des debuts pose
// la borne a 70 s, et tout le bloc de manche 1 des trois slots precoces (12 enregistrements) tombe
// hors fenetre — sans la garde par slot il disparaissait, `rounds[1]` restait VIDE pour eux et le
// total d'assistances passait de 8 a 5.
func manchesFixtureRelecteur() []objectiveevents.StatRecord {
	precoces, tardifs := manchesSlots[:3], manchesSlots[3:]
	recs := manchesCorps(0, manchesDebutR0)
	recs = append(recs, modeRamp(6, 1, 70_000, 500, 1, 2, 3)...)
	recs = append(recs, manchesBloc(1, 40_000, 3, precoces)...)
	recs = append(recs, manchesBloc(1, 70_000, 3, tardifs)...)
	return recs
}

// TestSegmentEntierDUnSlotNEstJamaisJete — LE CONSTAT C1 DE LA REVUE MANCHES-R1.
//
// Une lecture vraie n'est jamais jetee : quand TOUT le bloc d'un slot pour une manche tombe hors
// de la fenetre consensuelle, il n'est contredit par rien et reste dans sa manche declaree.
//
// MUTATION : retirer l'exemption `w.kept[...]` de `RoundBounds.Excludes` (round_bounds.go) vide
// la manche 1 des trois slots precoces.
func TestSegmentEntierDUnSlotNEstJamaisJete(t *testing.T) {
	recs := manchesFixtureRelecteur()
	bornes := objectiveevents.ResolveRoundBounds(recs)
	if n := bornes.Outliers(recs); n != 0 {
		t.Errorf("%d enregistrements ecartes : un bloc entier de slot a ete jete", n)
	}
	if got := len(bornes.KeptSegments()); got != 3 {
		t.Fatalf("%d bloc(s) exempte(s), attendu 3 (les trois slots precoces)", got)
	}
	for _, s := range bornes.KeptSegments() {
		if s.Round != 1 || s.Records != 4 || s.GapMS <= 0 {
			t.Errorf("bloc exempte incoherent : %+v (manche 1, 4 enregistrements, ecart > 0 attendus)", s)
		}
	}
	// La manche 1 des trois slots precoces existe, et leur total est complet.
	for _, slot := range manchesSlots[:3] {
		pts := assistsDuSlot(recs, slot)[1]
		if len(pts) == 0 {
			t.Errorf("slot %d : manche 1 VIDE — son bloc a ete jete", slot)
			continue
		}
		if v := pts[len(pts)-1].Value; v != 3 {
			t.Errorf("slot %d : manche 1 finit a %d assistances au lieu de 3", slot, v)
		}
	}
}

// TestSegmentEntierExemptePubliePourLeJoueur — le meme constat, vu du DOCUMENT : le total du
// joueur reste la somme des deux manches (3 + 3 = 6), pas la seule manche 0.
func TestSegmentEntierExemptePubliePourLeJoueur(t *testing.T) {
	tl, cov := buildScoreTimeline(&ScoreInput{Records: manchesFixtureRelecteur()},
		manchesMortsRelecteur(), manchesClocheRelecteur())
	if tl == nil || cov == nil || cov.Rounds != 2 {
		t.Fatalf("calque absent ou %v manche(s), attendu 2", cov)
	}
	p := playerByXUID(tl.Players, "1010")
	if p == nil {
		t.Fatalf("le joueur du slot 10 (precoce) n'est pas publie : %+v", tl.Players)
	}
	if got := lastValue(p.Assists); got != 6 {
		t.Errorf("assistances totales du slot precoce = %d, attendu 6 (3 par manche) — sa manche 1 "+
			"a ete jetee par la borne consensuelle", got)
	}
}

// manchesMortsRelecteur date les morts sur les instants EXACTS de la fixture du relecteur :
// l'index de decalage d'un slot est celui qu'il a DANS SON GROUPE (precoces ou tardifs), comme
// dans [manchesBloc].
func manchesMortsRelecteur() []Death {
	var out []Death
	for j, slot := range manchesSlots {
		xuid := uint64(1_000 + slot)
		for _, t := range manchesInstants(j, manchesDebutR0) {
			out = append(out, Death{XUID: xuid, TimeMS: int64(t)})
		}
		debut, idx := 70_000, j-3
		if j < 3 {
			debut, idx = 40_000, j
		}
		for _, t := range manchesInstants(idx, debut) {
			out = append(out, Death{XUID: xuid, TimeMS: int64(t)})
		}
	}
	return out
}

// TestEgareSeulResteEcarteMalgreLaGardeParSlot — LA MUTATION INVERSE de C1.
//
// La garde par slot ne doit pas rouvrir la porte a l'egare : un enregistrement SEUL, hors
// fenetre, dont le slot a par ailleurs un bloc DANS la fenetre pour la meme manche, reste
// contredit — donc ecarte. C'est le cas exact de `51ebbc0f`.
//
// MUTATION : exempter un bloc des qu'il a UN enregistrement hors fenetre (au lieu de TOUS) fait
// revenir les 60 assistances.
func TestEgareSeulResteEcarteMalgreLaGardeParSlot(t *testing.T) {
	recs := deuxManchesFixture(true)
	bornes := objectiveevents.ResolveRoundBounds(recs)
	if n := len(bornes.KeptSegments()); n != 0 {
		t.Errorf("%d bloc(s) exempte(s) : l'egare de `51ebbc0f` a un bloc DANS la fenetre, il est "+
			"contredit et ne doit pas etre exempte", n)
	}
	if n := bornes.Outliers(recs); n != 1 {
		t.Errorf("%d enregistrement(s) ecarte(s), attendu 1 (l'egare)", n)
	}
	if _, v := dernierPoint(assistsDuSlot(recs, 12)[0]); v != 3 {
		t.Errorf("la manche 0 du slot 12 finit a %d assistances au lieu de 3 : l'egare a compte", v)
	}
}

// manchesClocheRelecteur : la fixture du relecteur va jusqu'a 77,3 s, la grille de
// [multiRoundClock] s'arrete a 30 s. Mille frames de 100 ms couvrent tout le film.
func manchesClocheRelecteur() scoreClock {
	return scoreClock{intervalMS: 100, frames: 1_000, originMS: 0}
}

// journalDeCuisson capte les lignes que `logRoundBounds` emet pour un jeu d'enregistrements, en
// detournant le journal par defaut le temps de l'appel.
func journalDeCuisson(t *testing.T, recs []objectiveevents.StatRecord, manches int) string {
	t.Helper()
	var tampon bytes.Buffer
	precedent := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&tampon, &slog.HandlerOptions{Level: slog.LevelInfo})))
	defer slog.SetDefault(precedent)
	logRoundBounds("test", &ScoreInput{Records: recs}, &ScoreCoverage{Rounds: manches})
	return tampon.String()
}

// TestLeJournalDeBornesNeSeContreditPas — LE CONSTAT N1 DE LA REVUE MANCHES-R2.
//
// Sur la fixture de C1, trois blocs sont GARDES par la garde par slot — donc des bornes ONT ete
// posees. Le message de repli « AUCUNE borne de manche posee ... les compteurs restent ceux
// d'avant » y etait pourtant emis a la suite, parce qu'il se decidait sur le COMPTE D'ECARTES
// (nul, tout etant exempte) et non sur l'existence des bornes.
//
// MUTATION : revenir a `if n == 0 {` au lieu de `if !bornes.Posed() {` dans `logRoundBounds`
// (build_score.go) fait reapparaitre le message de repli sous les trois blocs gardes.
func TestLeJournalDeBornesNeSeContreditPas(t *testing.T) {
	journal := journalDeCuisson(t, manchesFixtureRelecteur(), 2)
	if n := strings.Count(journal, "bloc de manche GARDE"); n != 3 {
		t.Fatalf("%d bloc(s) garde(s) journalise(s), attendu 3 — la fixture ne prouve rien :\n%s", n, journal)
	}
	if strings.Contains(journal, "AUCUNE borne de manche posee") {
		t.Errorf("le journal affirme qu'aucune borne n'est posee JUSTE APRES avoir nomme trois blocs "+
			"gardes par une borne :\n%s", journal)
	}
	if !strings.Contains(journal, "hors de la fenetre de leur manche declaree") {
		t.Errorf("le journal ne dit pas que des bornes sont posees :\n%s", journal)
	}
}

// TestLeJournalDitQuandAucuneBorneNEstPosee — le pendant : quand il n'y a VRAIMENT aucune borne
// (etiquetage de manche qui ne suit pas l'horloge, forme de `a4083bd2`), le message de repli doit
// etre emis. Sans lui, la correction de N1 aurait supprime un signal utile.
func TestLeJournalDitQuandAucuneBorneNEstPosee(t *testing.T) {
	recs := manchesCorps(0, 30_000)
	recs = append(recs, manchesCorps(1, manchesDebutR0)...)
	journal := journalDeCuisson(t, recs, 2)
	if !strings.Contains(journal, "AUCUNE borne de manche posee") {
		t.Errorf("aucune borne n'est posable sur ce film et le journal ne le dit pas :\n%s", journal)
	}
	if strings.Contains(journal, "bloc de manche GARDE") {
		t.Errorf("aucun bloc ne peut etre garde sans borne :\n%s", journal)
	}
}

// Les instants de la fixture d'identite : la manche 1 commence VRAIMENT a 298 s pour la majorite
// des slots, mais un slot minoritaire la declare des 85 s. C'est la forme de `24dbb67d`
// (minimum 85 193 ms contre mediane 298 909 ms, 213 s d'ecart).
const (
	identiteDebutR1  = 298_000
	identiteEgarePre = 85_000
	// identiteSlotReattribue change de joueur d'une manche a l'autre : c'est ce qui rend la
	// manche resolue OBSERVABLE — se tromper de manche rend un AUTRE xuid.
	identiteSlotReattribue = 22
	identiteXUIDManche0    = "1022"
	identiteXUIDManche1    = "9022"
)

// identiteFixture rend un film a deux manches franchement separees, avec un train de score de
// mode par manche (sans lui `RealRounds` n'en reconnait aucune, et la fixture ne prouverait rien),
// un slot minoritaire qui declare la manche 1 en avance, et un slot REATTRIBUE d'une manche a
// l'autre.
func identiteFixture() ([]objectiveevents.StatRecord, []Death) {
	recs := manchesCorps(0, manchesDebutR0)
	recs = append(recs, manchesCorps(1, identiteDebutR1)...)
	// LE FAUX POSITIF : un enregistrement isole, slot 10, qui declare la manche 1 a 85 s.
	recs = append(recs, statRec(identiteEgarePre, 10, 1,
		map[int]objectiveevents.StatValue{3: {A: 0}}))

	var deaths []Death
	for j, slot := range manchesSlots {
		for _, t := range manchesInstants(j, manchesDebutR0) {
			deaths = append(deaths, Death{XUID: uint64(1_000 + slot), TimeMS: int64(t)})
		}
		xuid := uint64(1_000 + slot)
		if slot == identiteSlotReattribue {
			xuid = 9_022 // le slot change de joueur en manche 1
		}
		for _, t := range manchesInstants(j, identiteDebutR1) {
			deaths = append(deaths, Death{XUID: xuid, TimeMS: int64(t)})
		}
	}
	return recs, deaths
}

// TestIdentiteParMancheSuitLeDebutConsensuel — LE GARDE-RAIL DU CORRECTIF C2 (constat N2 de la
// revue MANCHES-R2 : la delegation n'en avait aucun, et la neutraliser laissait 23 paquets verts).
//
// `roundStartsOf` prenait le MINIMUM des instants declares ; un seul faux positif suffisait donc a
// dater la manche 1 a 85 s au lieu de 298 s, et `RoundIdentity.At` resolvait la manche SUIVANTE
// sur 213 s. Le slot reattribue rend l'erreur visible : s'y tromper de manche rend l'autre joueur.
//
// MUTATION : `consensus := RoundStartsMS(recs)` -> `consensus := map[int]int{}` dans
// `slotidentity_rounds.go` (retour au minimum pur) fait resoudre la manche 1 des 85 s, et
// `At` rend `9022` la ou la manche 0 est encore en cours.
func TestIdentiteParMancheSuitLeDebutConsensuel(t *testing.T) {
	recs, deaths := identiteFixture()

	// La divergence que le correctif ferme : minimum declare contre debut consensuel.
	if debut := objectiveevents.RoundStartsMS(recs)[1]; debut != identiteDebutR1 {
		t.Fatalf("debut consensuel de la manche 1 = %d, attendu %d — la fixture ne pose pas le "+
			"probleme", debut, identiteDebutR1)
	}
	minDeclare := 1 << 30
	for _, r := range recs {
		if r.Round == 1 && r.TimeMS < minDeclare {
			minDeclare = r.TimeMS
		}
	}
	if minDeclare != identiteEgarePre {
		t.Fatalf("minimum declare de la manche 1 = %d, attendu %d", minDeclare, identiteEgarePre)
	}

	ri := objectiveevents.ResolveRoundIdentity(recs, deathInstantsOf(deaths))
	// Le slot reattribue doit etre nomme dans les DEUX manches, sinon le test ne mesure rien.
	if got := ri.AtRound(0, identiteSlotReattribue); got != identiteXUIDManche0 {
		t.Fatalf("manche 0, slot %d : %q, attendu %q", identiteSlotReattribue, got, identiteXUIDManche0)
	}
	if got := ri.AtRound(1, identiteSlotReattribue); got != identiteXUIDManche1 {
		t.Fatalf("manche 1, slot %d : %q, attendu %q", identiteSlotReattribue, got, identiteXUIDManche1)
	}

	// L'INTERVALLE LITIGIEUX : entre le minimum declare et le debut consensuel, la manche 0 est
	// encore en cours — l'identite doit donc rendre son occupant.
	for _, t0 := range []int{identiteEgarePre, 150_000, identiteDebutR1 - 1} {
		if got := ri.At(identiteSlotReattribue, t0); got != identiteXUIDManche0 {
			t.Errorf("a %d ms (manche 0 en cours, la manche 1 ne commence qu'a %d) le slot %d est "+
				"attribue a %q au lieu de %q : l'identite suit le MINIMUM declare (%d), pas le "+
				"debut consensuel", t0, identiteDebutR1, identiteSlotReattribue, got,
				identiteXUIDManche0, identiteEgarePre)
		}
	}
	// Et apres le debut consensuel, c'est bien le joueur de la manche 1.
	if got := ri.At(identiteSlotReattribue, identiteDebutR1+1_000); got != identiteXUIDManche1 {
		t.Errorf("apres le debut de la manche 1, le slot %d est attribue a %q au lieu de %q",
			identiteSlotReattribue, got, identiteXUIDManche1)
	}
}
