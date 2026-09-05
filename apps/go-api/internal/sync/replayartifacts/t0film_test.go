package replayartifacts

// t0film_test.go — LA LECTURE DU CHAMP, SANS BASE.
//
// Ce qui est éprouvé ici : la chaîne `lireArtefacts` -> `rapportsT0` (derivations.go) ne rend
// AUCUN rapport dans les quatre situations qui veulent toutes la même chose — ne rien écrire —
// et un rapport dans la seule qui l'autorise. Le PIÈGE visé est celui du champ lui-même : un
// `t0FilmMs` absent (schéma antérieur, ou refus du détecteur) et un `t0FilmMs` à zéro doivent se
// distinguer, et seul un POINTEUR le permet.
//
// LE TEST PASSE PAR LA CHAÎNE RÉELLE (2026-09-06) : la lecture du champ vivait dans
// `lireT0FilmArtefact`, qui ouvrait le fichier pour son seul compte. Elle vit désormais dans
// [Deriver], qui lit UNE fois pour toutes les dérivations — éprouver une fonction qui n'existe
// plus aurait laissé le vrai chemin sans test.
//
// Le report en base, lui, exige une vraie table `match_registry` : il est éprouvé dans
// t0film_integration_test.go.

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func ecrireFichier(t *testing.T, nom, contenu string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), nom)
	if err := os.WriteFile(path, []byte(contenu), 0o644); err != nil {
		t.Fatalf("write %s: %v", nom, err)
	}
	return path
}

// t0DuFichier rejoue la chaîne de production sur UN artefact et rend le t0 reporté, ou nil.
func t0DuFichier(t *testing.T, path string) *int64 {
	t.Helper()
	lus := lireArtefacts(context.Background(), Deps{Gamertag: "t"},
		[]ArtefactRange{{MatchID: "m1", Path: path}})
	rapports := rapportsT0(lus)
	if len(rapports) == 0 {
		return nil
	}
	if len(rapports) != 1 {
		t.Fatalf("%d rapports pour un seul artefact", len(rapports))
	}
	return &rapports[0].t0FilmMs
}

func TestRapportsT0DepuisUnArtefact(t *testing.T) {
	zero := int64(0)
	mesure := int64(26304)
	cas := map[string]struct {
		contenu string
		attendu *int64
	}{
		"coup d envoi mesure":        {`{"schemaVersion":36,"t0FilmMs":26304}`, &mesure},
		"coup d envoi a zero":        {`{"schemaVersion":36,"t0FilmMs":0}`, &zero},
		"champ absent (refus)":       {`{"schemaVersion":36,"matchId":"abc"}`, nil},
		"schema anterieur au champ":  {`{"schemaVersion":35,"matchId":"abc"}`, nil},
		"artefact illisible":         {`{"schemaVersion":`, nil},
		"artefact vide":              {``, nil},
		"champ explicitement a null": {`{"t0FilmMs":null}`, nil},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			got := t0DuFichier(t, ecrireFichier(t, "artefact.json", c.contenu))
			switch {
			case c.attendu == nil && got != nil:
				t.Fatalf("attendu nil (rien à écrire), got %d", *got)
			case c.attendu != nil && got == nil:
				t.Fatalf("attendu %d, got nil", *c.attendu)
			case c.attendu != nil && *got != *c.attendu:
				t.Fatalf("t0FilmMs = %d, attendu %d", *got, *c.attendu)
			}
		})
	}
}

// TestRapportsT0_FichierAbsent : un artefact que `StoreArtifact` n'a pas écrit ne doit rien
// reporter — pas paniquer, pas écrire zéro.
func TestRapportsT0_FichierAbsent(t *testing.T) {
	if got := t0DuFichier(t, filepath.Join(t.TempDir(), "jamais-ecrit.json")); got != nil {
		t.Fatalf("fichier absent : attendu nil, got %d", *got)
	}
}

// TestReporterT0Film_SansWriter : un chemin de sync sans writer câblé ne panique pas et ne
// prétend pas avoir reporté. La dégradation est explicite (WARN + compteur d'échecs) ; ce test
// couvre le fait qu'elle n'appelle rien d'autre.
func TestReporterT0Film_SansWriter(t *testing.T) {
	reporterT0Film(context.Background(), Deps{Gamertag: "t"},
		[]rapportT0Film{{matchID: "m1", t0FilmMs: 26304}})
	// Rien à assert au-delà de l'absence de panique : sans writer, la fonction ne peut avoir
	// touché aucune base. Le cas nominal est en test d'intégration.
}

// TestDeriverSansArtefactLisibleNeFaitRien : le point d'entrée unique ne panique pas et
// n'écrit rien quand aucun artefact du lot n'est lisible — la garde qui protège les deux
// rangeurs (cuisson locale et dépôt d'ouvrier) d'un fichier disparu entre le rangement et la
// dérivation.
func TestDeriverSansArtefactLisibleNeFaitRien(t *testing.T) {
	Deriver(context.Background(), DerivationsDeps{Gamertag: "t", TitleSlug: "halo_infinite"},
		[]ArtefactRange{{MatchID: "m1", Path: filepath.Join(t.TempDir(), "absent.json")}})
	// Aucune assertion possible au-delà de l'absence de panique et d'écriture : sans writer
	// câblé ET sans document lisible, la fonction sort avant toute projection.
}
