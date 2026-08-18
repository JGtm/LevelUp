package objectiveevents

import "testing"

// flagfilm_test.go — LE CORPUS MESURE, GELE ET REJOUE SANS FILM.
//
// Les quinze lignes ci-dessous sont les comptes RELEVES le 2026-08-18 sur quinze films de mode
// CONNU (instrument `analysis/replay/objectifs_phase1_ctf_test.go`, sous garde `OBJ_FILM`). Les
// figer ici transforme une mesure ponctuelle en garde-rail permanent : si la regle du verdict
// bouge, ce test tombe en CI, sans qu'aucun film soit necessaire.
//
// Le mode de chaque film vient du registre du depot, jamais d'une supposition :
// `PLAN_ARMES_AU_SOL_2E_LECTURE.md` (tableau du corpus) pour `bcb6d393`, `00162144`, `000d5950`,
// `extract_test.go` et `named_test.go` pour les films de verite terrain d'`objectiveevents`,
// `PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md` (ligne « Corpus ») pour les autres.

// TestFlagFilmVerdictSurCorpusMesure — quinze films, quinze verdicts justes, zero faux positif.
func TestFlagFilmVerdictSurCorpusMesure(t *testing.T) {
	cas := []struct {
		film                    string
		mode                    string
		bursts                  int
		captures, steals, grabs int
	}{
		// CTF — les six films de drapeau du corpus.
		{"64e8adfa", ObjectiveTypeFlag, 5, 4, 17, 65}, // film TRONQUE : 4 captures pour 5 bursts
		{"530820e5", ObjectiveTypeFlag, 3, 3, 10, 23},
		{"53ce4390", ObjectiveTypeFlag, 3, 3, 13, 21},
		{"0f9550e5", ObjectiveTypeFlag, 5, 5, 4, 34},
		{"1bc77d2e", ObjectiveTypeFlag, 5, 5, 14, 46},
		{"bcb6d393", ObjectiveTypeFlag, 3, 3, 5, 14},
		// Oddball — la table drapeau y compte n'importe quoi (1 470 prises), et c'est
		// l'inegalite captures <= bursts qui l'ecarte.
		{"24dbb67d", ObjectiveTypeSkull, 2, 6, 994, 1470},
		// Bastion / zones — aucun burst.
		{"696a9d7c", ObjectiveTypeZone, 0, 16, 0, 27},
		{"7344d24f", ObjectiveTypeZone, 0, 12, 0, 24},
		// Roi de la colline — des bursts sur un film, mais aucune capture de drapeau.
		{"0a247154", ObjectiveTypeHill, 4, 0, 0, 22},
		{"01e1f945", ObjectiveTypeHill, 0, 2, 0, 13},
		{"606d9844", ObjectiveTypeHill, 0, 0, 0, 8},
		{"8076f97f", ObjectiveTypeHill, 0, 3, 55, 15},
		// Slayer — le mode sans objectif.
		{"000d5950", "", 0, 0, 0, 0},
		{"00162144", "", 2, 0, 0, 0},
	}
	retenus, ecartes := 0, 0
	for _, c := range cas {
		s := FlagFilmSignals{Bursts: c.bursts, Captures: c.captures, Steals: c.steals, Grabs: c.grabs}
		want := c.mode == ObjectiveTypeFlag
		if got := s.IsFlagFilm(); got != want {
			t.Errorf("%s (mode %q) : verdict %v, attendu %v — signaux %+v", c.film, c.mode, got, want, s)
		}
		if want {
			retenus++
		} else {
			ecartes++
		}
	}
	if retenus != 6 || ecartes != 9 {
		t.Fatalf("corpus altere : %d films CTF et %d non-CTF, attendu 6 et 9", retenus, ecartes)
	}
}

// TestFlagFilmSignalsFromCompteCeQuIlDoit — le comptage ne retient que les trois statistiques
// de la regle, et ignore tout le reste de la table (frags, assistances, retours...).
func TestFlagFilmSignalsFromCompteCeQuIlDoit(t *testing.T) {
	evs := []NamedEvent{
		{Stat: StatFlagCaptures}, {Stat: StatFlagCaptures},
		{Stat: StatFlagSteals},
		{Stat: StatFlagGrabs}, {Stat: StatFlagGrabs}, {Stat: StatFlagGrabs},
		{Stat: StatFlagReturns}, {Stat: StatFlagCarriersKilled}, {Stat: StatKills}, {Stat: StatAssists},
	}
	got := FlagFilmSignalsFrom([]int{1000, 2000, 3000}, evs)
	want := FlagFilmSignals{Bursts: 3, Captures: 2, Steals: 1, Grabs: 3}
	if got != want {
		t.Errorf("signaux %+v, attendu %+v", got, want)
	}
	if !got.IsFlagFilm() {
		t.Error("verdict negatif sur des signaux qui tiennent la regle")
	}
}

// TestFlagFilmVerdictRefuseChaqueClause — chaque clause de la regle est NECESSAIRE : la retirer
// une par une doit faire tomber le verdict.
//
// Sans ce test, une clause pourrait devenir inoperante (toujours vraie) sans que rien ne le
// dise, et le corpus gele ci-dessus continuerait de passer.
func TestFlagFilmVerdictRefuseChaqueClause(t *testing.T) {
	base := FlagFilmSignals{Bursts: 3, Captures: 3, Steals: 5, Grabs: 14}
	if !base.IsFlagFilm() {
		t.Fatalf("les signaux de reference %+v devraient tenir la regle", base)
	}
	for nom, s := range map[string]FlagFilmSignals{
		"aucun burst":                    {Bursts: 0, Captures: 3, Steals: 5, Grabs: 14},
		"aucune capture":                 {Bursts: 3, Captures: 0, Steals: 5, Grabs: 14},
		"plus de captures que de bursts": {Bursts: 2, Captures: 6, Steals: 5, Grabs: 14},
		"aucun vol":                      {Bursts: 3, Captures: 3, Steals: 0, Grabs: 14},
	} {
		if s.IsFlagFilm() {
			t.Errorf("%s : verdict POSITIF sur %+v", nom, s)
		}
	}
}
