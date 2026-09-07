package main

import (
	"errors"
	"testing"
)

// TestExportFactsAvecContinueApresUnEchec — CORPUS-R1 C4 : un id inconnu du registre (ou tout
// autre echec d'export) ne doit JAMAIS empecher l'export des AUTRES ids du meme lot — avant ce
// correctif, une invocation groupee unique sortait en erreur fatale au premier id en echec,
// avant meme la moindre cuisson.
func TestExportFactsAvecContinueApresUnEchec(t *testing.T) {
	var appeles []string
	exporter := func(id string) error {
		appeles = append(appeles, id)
		if id == "deadbeef" {
			return errors.New("id inconnu du registre de la base partagee")
		}
		return nil
	}

	exportFactsAvec(exporter, []string{"bon1a2b3c", "deadbeef", "bon4d5e6f"})

	if len(appeles) != 3 {
		t.Fatalf("attendu 3 tentatives (un echec ne doit pas interrompre les suivantes), obtenu %d : %v",
			len(appeles), appeles)
	}
	if appeles[2] != "bon4d5e6f" {
		t.Fatalf("le dernier id du lot n'a pas ete tente apres l'echec du precedent : %v", appeles)
	}
}

// TestExportFactsAvecVideNeFaitRienEtNePanicPas — un manifeste vide est un cas legitime
// (utilise par les tests du manifeste reduit) : aucune tentative, aucune panique.
func TestExportFactsAvecVideNeFaitRienEtNePanicPas(t *testing.T) {
	appele := false
	exportFactsAvec(func(string) error { appele = true; return nil }, nil)
	if appele {
		t.Fatal("un lot vide n'a rien a exporter")
	}
}
