package main

// GARDE-RAIL DES RÉGLAGES PAR CARTE.
//
// Un réglage par carte n'est défendable que parce qu'il porte sa RAISON et la DATE de son
// gate. Sans ces deux champs, dans six mois, personne ne sait si le réglage tient encore, ni
// sur quelles images il a été décidé — et on retombe exactement sur ce que la doctrine
// « aucun réglage par carte » cherchait à éviter, avec la dette en plus.
//
// Ce test lit l'asset PUBLIÉ, celui que la cuisson consomme.

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"levelup/go-api/internal/himap"
)

const cheminReglagesPublies = "../../../../data/titles/halo_infinite/reference/map_fond_reglages.json"

// raisonMinimale : une raison d'une ligne n'explique rien. Le seuil est bas exprès — il
// attrape le « ok user » et laisse passer une justification honnête.
const raisonMinimale = 80

var formatDateGate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func TestReglagesFondJustifies(t *testing.T) {
	chemin := filepath.FromSlash(cheminReglagesPublies)
	if _, err := os.Stat(chemin); err != nil {
		t.Skipf("aucun réglage publié (%v) — c'est le cas nominal tant que rien n'a été jugé", err)
	}
	reg, err := chargeReglages(chemin)
	if err != nil {
		t.Fatalf("réglages publiés illisibles : %v", err)
	}
	if len(reg.Cartes) == 0 {
		t.Fatal("fichier de réglages présent mais vide : le supprimer plutôt que le garder")
	}
	for cle, c := range reg.Cartes {
		if len(c.Raison) < raisonMinimale {
			t.Errorf("réglage %q : raison de %d caractères (minimum %d).\n"+
				"Un réglage par carte sans justification écrite est un réglage orphelin.",
				cle, len(c.Raison), raisonMinimale)
		}
		if !formatDateGate.MatchString(c.GateLe) {
			t.Errorf("réglage %q : gateLe = %q, attendu AAAA-MM-JJ — la date du gate qui l'a décidé",
				cle, c.GateLe)
		}
		if c.Style == "" && c.Echelle <= 0 {
			t.Errorf("réglage %q : ne déclare NI habillage NI échelle — entrée sans effet, à retirer", cle)
		}
		if c.Style != "" && !himap.StyleFondValide(himap.StyleFond(c.Style)) {
			t.Errorf("réglage %q : habillage inconnu %q", cle, c.Style)
		}
	}
}

// Un habillage inconnu doit FAIRE ÉCHOUER le chargement, pas être ignoré : une carte cuite
// dans l'habillage par défaut alors que la donnée en demandait un autre passerait le gate
// sous une fausse identité.
func TestChargeReglagesRefuseUnHabillageInconnu(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "r.json")
	blob := `{"schemaVersion":1,"cartes":{"x":{"style":"aquarelle","raison":"...","gateLe":"2026-08-26"}}}`
	if err := os.WriteFile(p, []byte(blob), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := chargeReglages(p); err == nil {
		t.Fatal("un habillage inconnu a été accepté")
	}
}

// Fichier absent = cas nominal, pas une erreur.
func TestChargeReglagesAbsentEstNominal(t *testing.T) {
	r, err := chargeReglages(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("un fichier absent doit être nominal, or : %v", err)
	}
	if len(r.Cartes) != 0 {
		t.Fatalf("réglages non vides sur fichier absent : %v", r.Cartes)
	}
}
