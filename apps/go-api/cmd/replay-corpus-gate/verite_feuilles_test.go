package main

// verite_feuilles_test.go — LA COUVERTURE DU BANC SE DECLARE A LA FEUILLE (revue finale P1-d,
// 2026-10-02).
//
// Des blocs `coverage.*` entiers etaient declares couverts par une mesure qui n'en lit que
// quelques feuilles (`coverage.teams.` par O-T1, qui ne lit que accord/contradiction/silence) :
// une perte sur une feuille NON LUE passait `ok` sans que le banc l'ait vue.

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/replaydiff"
	"levelup/go-api/internal/replayverite"
)

// TestFilets_FeuilleNonLueParLaMesure : les trois cas du relecteur posent le filet, la mesure du
// bloc etant presente des deux cotes ; une feuille lue reste couverte.
func TestFilets_FeuilleNonLueParLaMesure(t *testing.T) {
	ok := comparaisonDe(replayverite.StatutOK, replayverite.ScoreEquipes, replayverite.PreuveFermeture,
		replayverite.PreuveContradic)
	cas := []struct {
		nom    string
		d      replaydiff.Difference
		filets int
	}{
		{"tracksNamed en baisse (O-T1 ne la lit pas)", perte("couverture", "coverage.teams.tracksNamed", "16", "12"), 1},
		{"slides en hausse (P-2 ne la lit pas)", perte("couverture", "coverage.keyframes.slides", "0", "3"), 1},
		{"shots en baisse (P-1 ne la lit pas)", perte("couverture", "coverage.continuousFire.shots", "900", "850"), 1},
		{"silence lue par O-T1", perte("couverture", "coverage.teams.silence", "0", "2"), 0},
		{"refutations lues par P-2", perte("couverture", "coverage.keyframes.refutations", "0", "2"), 0},
		{"closed lue par P-1", perte("couverture", "coverage.continuousFire.closed", "9", "8"), 0},
	}
	for _, k := range cas {
		if l := ligneJugee(ok, k.d); len(l.Filets) != k.filets {
			t.Errorf("%s : %d filet(s) %v, veut %d", k.nom, len(l.Filets), l.Filets, k.filets)
		}
	}
}

// docDeCouverture : un artefact minimal dont chaque compteur de couverture lu par le banc vaut 1.
func docDeCouverture() map[string]any {
	return map[string]any{
		"matchId": "m", "frameIntervalMs": 100, "frameCount": 10,
		"coverage": map[string]any{
			"teams":          map[string]any{"accord": 1, "contradiction": 1, "silence": 1, "tracksNamed": 1},
			"continuousFire": map[string]any{"packets": 1, "closed": 1, "shots": 1},
			"keyframes":      map[string]any{"refutations": 1, "contradictoryProofs": 1, "slides": 1},
			"verdict":        map[string]any{"x": "nominal"},
		},
	}
}

// mesureDe note `doc` et rend l'entree du bulletin pour `mesure` (score ou preuve).
func mesureDe(t *testing.T, doc map[string]any, mesure string) any {
	t.Helper()
	blob, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	d, err := replayverite.LireDocument(blob)
	if err != nil {
		t.Fatal(err)
	}
	b := replayverite.Noter(d, domain.MatchFacts{}, nil)
	if s, ok := b.Scores[mesure]; ok {
		return s
	}
	if p, ok := b.Preuves[mesure]; ok {
		return p
	}
	t.Fatalf("mesure %q absente du bulletin", mesure)
	return nil
}

// poserFeuille ecrit `v` au chemin pointe `coverage.<bloc>.<feuille>` de `doc`.
func poserFeuille(t *testing.T, doc map[string]any, chemin string, v any) {
	t.Helper()
	parties := strings.Split(chemin, ".")
	noeud := doc
	for _, p := range parties[:len(parties)-1] {
		suivant, ok := noeud[p].(map[string]any)
		if !ok {
			t.Fatalf("%s : %q n'est pas un objet de l'artefact de test", chemin, p)
		}
		noeud = suivant
	}
	noeud[parties[len(parties)-1]] = v
}

// TestBlocsCouverts_ChaqueFeuilleEstLueParSaMesure — LA TABLE CONFRONTEE AU CODE DU BANC : changer
// une feuille declaree dans l'artefact doit changer la mesure qui la couvre. Une feuille declaree
// que la mesure ne lit pas (`coverage.teams.tracksNamed` sous O-T1, `coverage.continuousFire.packets`
// sous P-1) rougit : elle ferait taire le filet sur une perte que le banc ne voit pas.
func TestBlocsCouverts_ChaqueFeuilleEstLueParSaMesure(t *testing.T) {
	for _, b := range blocsCouverts {
		for _, f := range b.feuilles {
			avant := mesureDe(t, docDeCouverture(), b.mesure)
			doc := docDeCouverture()
			poserFeuille(t, doc, f, 7)
			if apres := mesureDe(t, doc, b.mesure); reflect.DeepEqual(avant, apres) {
				t.Errorf("%s declare %s couverte, mais ne la lit pas (mesure inchangee : %+v)", b.mesure, f, apres)
			}
		}
		if b.famille != "" {
			cle := b.famille + "x"
			avant := mesureDe(t, docDeCouverture(), b.mesure+" x")
			doc := docDeCouverture()
			poserFeuille(t, doc, cle, "aucune donnee")
			if apres := mesureDe(t, doc, b.mesure+" x"); reflect.DeepEqual(avant, apres) {
				t.Errorf("%s declare la famille %s couverte, mais ne lit pas %s", b.mesure, b.famille, cle)
			}
		}
		if b.derivees != nil {
			for _, m := range []string{"coverage.teams.tracksNamed", "coverage.bridge.livesNamed", "coverage.tracks.published"} {
				if b.derivees(m) {
					t.Errorf("%s : une metrique derivee couvre la feuille de couverture %s (a declarer a la feuille)", b.mesure, m)
				}
			}
		}
	}
}

// TestCalqueDisparu_BaisseSansZeroContreZero — revue finale P2 (2026-10-02) : sous O-V2, des pistes
// en BAISSE (`tracks/n` 80 -> 40) sont des fins de vie que le banc juge — pas de filet ; des pistes
// qui tombent A ZERO sont un calque disparu — filet, meme sous O-V2.
// Mutation vue rouge : `estCalqueDisparu` rend vrai pour toute baisse d'un `<cle>/n` de premier niveau.
func TestCalqueDisparu_BaisseSansZeroContreZero(t *testing.T) {
	ok := comparaisonDe(replayverite.StatutOK, replayverite.ScoreFinsDeVie)
	if l := ligneJugee(ok, perte("pistes", "tracks/n", "80", "40")); len(l.Filets) != 0 {
		t.Errorf("tracks/n 80 -> 40 sous O-V2 : filets %v, veut aucun", l.Filets)
	}
	if l := ligneJugee(ok, perte("pistes", "tracks/n", "80", "0")); len(l.Filets) != 1 {
		t.Errorf("tracks/n 80 -> 0 sous O-V2 : filets %v, veut un (calque disparu)", l.Filets)
	}
}
