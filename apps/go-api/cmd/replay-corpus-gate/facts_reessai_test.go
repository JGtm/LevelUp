package main

// facts_reessai_test.go — LE REESSAI BORNE DE L'EXPORT DES FAITS, ET SON MARQUEUR
// (D2 (cloture M1), 2026-09-17).
//
// LE FAIT MESURE : au gate du lot 2.1, deux temoins sur quatorze (`c75f33b8`, `111fa685`) sont
// sortis ABSENT parce que le serveur local tenait la base partagee a la seconde ou leur export
// est passe ; les douze autres, quelques secondes avant ou apres, ont reussi. Le gate a rendu
// « 2/14 absent(s) » pour un alea de quelques secondes.
//
// LES MUTATIONS QUI FONT ROUGIR CE FICHIER, NOMMEES :
//   - remplacer `estBaseTenue(dernier)` par `true` dans `exporterAvecReessai` : un id inconnu
//     du registre serait retente trois fois pour rien (TestReessaiNeRetenteJamaisUnEchecPermanent) ;
//   - remplacer `reessaisExport` par 1 : plus aucun reessai (TestReessaiRetenteJusquAuSucces) ;
//   - reformuler le texte de `duckdb.ErrBaseTenueEnEcriture` sans miroiter le nouveau marqueur
//     ici : archlint.TestMarqueurDuGateEgaleLaSentinelleBaseTenue (paquet `archlint`).

import (
	"context"
	"errors"
	"testing"
	"time"
)

// dormeurMuet : un dormeur qui compte les attentes sans jamais dormir — un test qui attend
// vraiment six secondes n'est pas un test, c'est une pause.
func dormeurMuet(compte *int) dormeur {
	return func(context.Context, time.Duration) error { *compte++; return nil }
}

// errBaseTenue fabrique l'erreur TRANSITOIRE telle que le sous-processus la rend, marqueur
// compris.
func errBaseTenue() error {
	return errors.New("export des faits (levelup replay-facts-export c75f33b8) : exit status 1\n" +
		"stderr:\nopen shared RO : ... utilise par un autre processus (" + marqueurBaseTenue + " ? l'arreter le temps de l'export)")
}

// TestReessaiRetenteJusquAuSucces — LE COMPORTEMENT DEMANDE : la base tenue a la premiere
// seconde ne perd pas le temoin ; la deuxieme tentative passe.
func TestReessaiRetenteJusquAuSucces(t *testing.T) {
	essais, attentes := 0, 0
	err := exporterAvecReessai(context.Background(), func() error {
		essais++
		if essais == 1 {
			return errBaseTenue()
		}
		return nil
	}, dormeurMuet(&attentes))
	if err != nil {
		t.Fatalf("le deuxieme essai reussit : attendu nil, obtenu %v", err)
	}
	if essais != 2 {
		t.Errorf("%d essai(s), 2 attendus — le reessai n'a pas eu lieu", essais)
	}
	if attentes != 1 {
		t.Errorf("%d attente(s), 1 attendue entre les deux essais", attentes)
	}
}

// TestReessaiEstBORNE — trois tentatives, pas plus, et deux attentes seulement (jamais une
// apres le dernier essai : un gate de 25 min n'a pas deux secondes a perdre pour rien).
func TestReessaiEstBORNE(t *testing.T) {
	essais, attentes := 0, 0
	err := exporterAvecReessai(context.Background(), func() error {
		essais++
		return errBaseTenue()
	}, dormeurMuet(&attentes))
	if err == nil {
		t.Fatal("une base tenue jusqu'au bout doit rendre l'erreur, pas un succes silencieux")
	}
	if essais != reessaisExport || reessaisExport != 3 {
		t.Errorf("%d essai(s) pour reessaisExport=%d, 3 attendus", essais, reessaisExport)
	}
	if attentes != reessaisExport-1 {
		t.Errorf("%d attente(s), %d attendues (jamais apres le dernier essai)", attentes, reessaisExport-1)
	}
	if delaiEntreReessais != 2*time.Second {
		t.Errorf("delaiEntreReessais = %v, 2 s attendues (D2 (cloture M1) : 3 x 2 s)", delaiEntreReessais)
	}
}

// TestReessaiNeRetenteJamaisUnEchecPermanent — un id inconnu du registre ou des faits vides ne
// changeront pas d'avis en deux secondes : les retenter ne fait qu'allonger le gate.
func TestReessaiNeRetenteJamaisUnEchecPermanent(t *testing.T) {
	essais, attentes := 0, 0
	err := exporterAvecReessai(context.Background(), func() error {
		essais++
		return errors.New("export des faits (levelup replay-facts-export deadbeef) : deadbeef : match absent du registre")
	}, dormeurMuet(&attentes))
	if err == nil {
		t.Fatal("un echec permanent doit remonter")
	}
	if essais != 1 {
		t.Errorf("%d essai(s), 1 attendu : un echec permanent ne se retente pas", essais)
	}
	if attentes != 0 {
		t.Errorf("%d attente(s), 0 attendue pour un echec permanent", attentes)
	}
}

// TestReessaiSArreteSurContexteAnnule — Ctrl-C pendant l'attente : on rend la main tout de
// suite, sans consommer les essais restants.
func TestReessaiSArreteSurContexteAnnule(t *testing.T) {
	ctx, annuler := context.WithCancel(context.Background())
	annuler()
	essais := 0
	err := exporterAvecReessai(ctx, func() error {
		essais++
		return errBaseTenue()
	}, dormirContexte)
	if err == nil {
		t.Fatal("un contexte annule doit remonter une erreur")
	}
	if essais != 1 {
		t.Errorf("%d essai(s), 1 attendu : l'annulation interrompt le reessai", essais)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("l'erreur doit porter l'annulation : %v", err)
	}
}
