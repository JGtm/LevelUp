package main

// rapport_fixture_test.go — LE RAPPORT DU GATE REJOUE SUR UNE PAIRE REELLE (lot 2.8.5).
//
// LA FIXTURE EST CELLE DE `internal/replaydiff/testdata/paire_bcb6d393_54_60.json`, et il n'y en
// a qu'UNE : c'est la sortie de `replay-diff -json` sur les deux artefacts CONSERVES du corpus
// gate de la cloture M1 (temoin `bcb6d393`, schema 54 -> 60, 60 gains / 13 pertes / 2
// changements). La recopier ici en ferait deux fichiers qui divergeraient au premier
// re-enregistrement (CLAUDE.md n°6, et la lecon « DDL de test recopiees = derive
// indetectable ») ; ce test lit donc L'ORIGINALE par un chemin relatif.
//
// CE QU'IL PROUVE, ET QUE LES TESTS UNITAIRES DU PAQUET NE PROUVENT PAS : que la CHAINE
// COMPLETE tient sur une vraie paire — `remplirBilan` depuis un rapport reel, puis le tableau,
// puis les deux sections de detail, puis le JSON. Les tests unitaires verifient chaque maillon
// sur un cas fabrique de deux ou trois mesures ; celui-ci verifie qu'ils s'enchainent sur 75
// ecarts repartis sur cinq axes, avec des cles a rallonge par joueur.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/replaydiff"
)

// cheminFixturePaire : l'UNIQUE fixture de paire reelle du depot, chez `replaydiff`.
func cheminFixturePaire() string {
	return filepath.Join("..", "..", "internal", "replaydiff", "testdata", "paire_bcb6d393_54_60.json")
}

// ligneDepuisLaPaireReelle construit la ligne de rapport du gate pour le temoin `bcb6d393`,
// depuis la fixture — exactement ce que `traiterTemoin` fait apres `compareTemoin`.
func ligneDepuisLaPaireReelle(t *testing.T) ligneRapport {
	t.Helper()
	raw, err := os.ReadFile(cheminFixturePaire()) //nolint:gosec // chemin relatif fige dans ce test
	if err != nil {
		t.Fatalf("fixture de paire illisible (%s) : %v", cheminFixturePaire(), err)
	}
	var rap replaydiff.Rapport
	if err := json.Unmarshal(raw, &rap); err != nil {
		t.Fatalf("fixture de paire invalide : %v", err)
	}
	l := ligneRapport{Temoin: Temoin{ID: "bcb6d393", Famille: "ctf_mono_manche"}}
	l.remplirBilan(rap)
	return l
}

// TestRapportDuGateSurUnePaireReelle — LE TABLEAU ET SES DEUX SECTIONS, sur la vraie paire.
func TestRapportDuGateSurUnePaireReelle(t *testing.T) {
	l := ligneDepuisLaPaireReelle(t)
	if l.Gains != 60 || l.Pertes != 13 || l.Changements != 2 {
		t.Fatalf("%d gains / %d pertes / %d changements, 60 / 13 / 2 attendus (les colonnes que "+
			"le corpus gate avait ecrites pour ce temoin)", l.Gains, l.Pertes, l.Changements)
	}
	if got := l.statut(); got != statutPerte {
		t.Errorf("statut = %q, %q attendu : 13 pertes priment sur 2 changements", got, statutPerte)
	}

	var b strings.Builder
	imprimerTableau(&b, []ligneRapport{l}, "base(79a3f7eb7)")
	imprimerDetailPertes(&b, []ligneRapport{l})
	imprimerDetailChangements(&b, []ligneRapport{l})
	out := b.String()
	t.Logf("rapport rendu :\n%s", out)

	for _, attendu := range []string{
		// Le tableau.
		"bcb6d393", "ctf_mono_manche", "54", "60", statutPerte,
		// La section des pertes, avec une metrique nommee de chaque sens.
		"DETAIL DES PERTES (1 temoin(s)) :",
		"coverage.bridge.livesTotal", "coverage.placements.byFamilyOrigin.repulsor/deployed",
		// LA SECTION QUI N'EXISTAIT PAS AVANT LE 2026-09-17, avec les deux changements NOMMES.
		"DETAIL DES CHANGEMENTS (1 temoin(s)) :",
		"tracks/par-xuid/2535429985869093",
		"tracks/vies-par-xuid/2535429985869093",
	} {
		if !strings.Contains(out, attendu) {
			t.Errorf("le rapport doit contenir %q :\n%s", attendu, out)
		}
	}
	// Les deux sections sont DISJOINTES : un gain n'entre dans aucune des deux.
	if strings.Contains(out, "pickups.origin/presents") {
		t.Errorf("un GAIN ne doit apparaitre dans aucune section de detail :\n%s", out)
	}
}

// TestRapportJSONDuGateSurUnePaireReelle — LE JSON, sur la vraie paire : `statut`,
// `pertesDetail` et `changementsDetail`, avec leurs comptes exacts.
func TestRapportJSONDuGateSurUnePaireReelle(t *testing.T) {
	l := ligneDepuisLaPaireReelle(t)
	path := filepath.Join(t.TempDir(), "gate.json")
	if err := ecrireRapportJSON(path, []ligneRapport{l}, false); err != nil {
		t.Fatalf("ecriture du rapport JSON : %v", err)
	}
	raw, err := os.ReadFile(path) //nolint:gosec // chemin construit par le test
	if err != nil {
		t.Fatalf("relecture : %v", err)
	}
	// La FORME rendue, affichee par `go test -v` : c'est elle que lit une CI ou un agregateur,
	// et c'est elle qui a change le 2026-09-17 (`statut`, `changementsDetail`).
	t.Logf("tete du rapport JSON :\n%s", tete(string(raw), 14))

	var rap rapportJSON
	if err := json.Unmarshal(raw, &rap); err != nil {
		t.Fatalf("le rapport JSON n'est pas relisible : %v", err)
	}
	if rap.CouvertureIncomplete {
		t.Errorf("couverture_incomplete = true, false attendu : aucun temoin absent ici")
	}
	relu := rap.Temoins
	if len(relu) != 1 {
		t.Fatalf("%d ligne(s), 1 attendue", len(relu))
	}
	lj := relu[0]
	if lj.Statut != statutPerte {
		t.Errorf("statut JSON = %q, %q attendu", lj.Statut, statutPerte)
	}
	if lj.Gains != 60 || lj.Pertes != 13 || lj.Changements != 2 {
		t.Errorf("comptes JSON = %d / %d / %d, 60 / 13 / 2 attendus", lj.Gains, lj.Pertes, lj.Changements)
	}
	if len(lj.PertesDetail) != 13 {
		t.Errorf("%d entree(s) de pertesDetail, 13 attendues", len(lj.PertesDetail))
	}
	// LE FAIT DU LOT : le JSON du gate NOMMAIT « 2 changements » sans jamais dire lesquels.
	if len(lj.ChangementsDetail) != 2 {
		t.Fatalf("%d entree(s) de changementsDetail, 2 attendues — le JSON compte les "+
			"changements sans les nommer (D5 (1.9.9))", len(lj.ChangementsDetail))
	}
	for _, d := range lj.ChangementsDetail {
		if d.Sens != replaydiff.SensChangement {
			t.Errorf("changementsDetail porte un sens %q — seul `changement` est attendu", d.Sens)
		}
		if d.Ancien != "6" || d.Nouveau != "5" {
			t.Errorf("changement %q : %s -> %s, 6 -> 5 attendu (reattribution d'une vie, "+
				"lots 1.9.13 / 1.9.14)", d.Metrique, d.Ancien, d.Nouveau)
		}
	}
	if !strings.Contains(string(raw), `"changementsDetail"`) {
		t.Error("la clef `changementsDetail` est absente du JSON ecrit")
	}
}

// tete rend les `n` premieres lignes d'une sortie — de quoi montrer une forme sans deverser 13
// pertes nommees dans le journal du test.
func tete(s string, n int) string {
	lignes := strings.Split(s, "\n")
	if len(lignes) <= n {
		return s
	}
	return strings.Join(lignes[:n], "\n") + "\n  ..."
}
