package replay

// filmfacts_entete_test.go — L EN-TETE DU FICHIER DE FAITS, JALON J3 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25 (codec 2 : revisions par consommateur, gardes de
// l appelant, empreinte de l entree de catalogue).

import (
	"errors"
	"testing"
)

// TestFaitsDuCodec1SontRefusesSurLePrefixe : un fichier ecrit par le codec d AVANT le jalon J3 est
// PERIME, et il se dit perime sur son PREFIXE — avant que l en-tete, dont la forme a change, soit
// seulement lu. C est la regle du §4.4 du plan : toute montee du codec vient avec le refus de
// l ancien fichier.
func TestFaitsDuCodec1SontRefusesSurLePrefixe(t *testing.T) {
	blob, err := EncodeFilmFactsFile(fichierTemoin(t))
	if err != nil {
		t.Fatalf("encodage : %v", err)
	}
	// Le prefixe est `magie | uvarint(codec) | uvarint(schema) | uvarint(longueur)` ; le codec
	// tient sur un octet.
	ancien := append([]byte{}, blob...)
	ancien[len(magieFaitsDeFilm)] = 1
	if _, err := DecodeFilmFactsEntete(ancien); !errors.Is(err, ErrFilmFactsVersion) {
		t.Fatalf("un fichier du codec 1 est lu : err = %v, attendu ErrFilmFactsVersion — un "+
			"en-tete d une autre forme serait lu de travers", err)
	}
	if _, err := DecodeFilmFactsFile(ancien, goldenEntryPourTest(t)); !errors.Is(err, ErrFilmFactsVersion) {
		t.Fatalf("un fichier du codec 1 est relu en entier : err = %v", err)
	}
}

// TestUtilisable_CompareChaqueRevisionDeConsommateur : depuis le lot J3.3 la couche des faits a
// DEUX revisions (une par consommateur) ; des faits pris sous une autre valeur de L UNE ou de
// L AUTRE sont perimes.
func TestUtilisable_CompareChaqueRevisionDeConsommateur(t *testing.T) {
	entry := goldenEntryPourTest(t)
	e := enteteFrais(t, fichierTemoin(t))
	if err := e.Utilisable(entry); err != nil {
		t.Fatalf("des faits frais sont refuses : %v", err)
	}
	for nom, muter := range map[string]func(*DecoderCoverage){
		"kill-feed": func(c *DecoderCoverage) { c.KillsourceRev += "-bis" },
		"objectifs": func(c *DecoderCoverage) { c.ObjectivesRev += "-bis" },
	} {
		autre := e
		muter(&autre.Coverage)
		if err := autre.Utilisable(entry); !errors.Is(err, ErrFilmFactsRevisions) {
			t.Errorf("revision %s differente : err = %v, attendu ErrFilmFactsRevisions", nom, err)
		}
	}
}

// enteteFrais encode `f` sous les revisions COURANTES du binaire et rend son en-tete relu.
func enteteFrais(t *testing.T, f *FilmFactsFile) FilmFactsEntete {
	t.Helper()
	f.Coverage = *couvertureDuDecodeur(nil)
	blob, err := EncodeFilmFactsFile(f)
	if err != nil {
		t.Fatalf("encodage : %v", err)
	}
	e, err := DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Fatalf("en-tete : %v", err)
	}
	return e
}
