package main

// couverture_verdict_test.go — LE VERDICT DES TEMOINS PRESENTS PRIME SUR LA COUVERTURE
// (revue M2 anticipee, 2026-09-16).
//
// # LE DEFAUT QUE CES TESTS FERMENT
//
// `finaliser` appelait `verifierCouverture` AVANT de calculer `codeSortie`, et rendait
// `codeCouvertureIncomplete` des qu'un temoin etait ABSENT. Un temoin absent est un ALEA —
// l'export des faits qui tombe sur la base tenue en ecriture (D2 (cloture M1)), un cache de
// film partiel : il survient sans rapport avec le diff sous revue. Il MASQUAIT donc le verdict
// de tous les autres temoins. Le gate imprimait « couverture incomplete », l'appelant relancait
// les temoins manquants, et la PERTE mesuree sur les temoins presents ne sortait jamais en
// code 1 — le silence exact que ce gate existe pour interdire, deplace d'un cran.
//
// Ces tests tiennent la priorite dans les deux sens, et la troisieme sortie du fanion JSON.
// `--allow-missing` reste inchange : il eteint la verification de couverture, pas le verdict.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// ligneAbsente : un temoin que le gate n'a pas pu comparer (l'alea).
func ligneAbsente(id string) ligneRapport {
	return ligneRapport{
		Temoin:      Temoin{ID: id, Famille: "ctf_mono_manche"},
		Absent:      true,
		AbsentCause: "faits de reference absents (base tenue en ecriture)",
	}
}

// lignePerdante : un temoin compare qui a PERDU — le fait a rapporter.
func lignePerdante(id string) ligneRapport {
	return ligneRapport{
		Temoin: Temoin{ID: id, Famille: "slayer"},
		Pertes: 3,
	}
}

// ligneAZero : un temoin compare sans perte ni changement.
func ligneAZero(id string) ligneRapport {
	return ligneRapport{Temoin: Temoin{ID: id, Famille: "slayer"}}
}

// optionsGate : le mode ou les pertes bloquent (reference = base, le mode par defaut du gate).
func optionsGate(sortieJSON string) executerOptions {
	return executerOptions{Reference: referenceBase, SortieJSON: sortieJSON}
}

// TestPerteSortEnUnMemeAvecUnTemoinAbsent — LE FAIT DE CE LOT. Un temoin absent ne fait plus
// taire la perte des autres : le code est 1, et la perte est dans le rapport.
func TestPerteSortEnUnMemeAvecUnTemoinAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gate.json")
	lignes := []ligneRapport{ligneAbsente("aaaaaaaa"), lignePerdante("bbbbbbbb")}

	code, err := finaliser(lignes, "base(abc1234)", optionsGate(path))
	if code != codePerte {
		t.Fatalf("code de sortie = %d, %d (codePerte) attendu : un temoin ABSENT ne doit pas "+
			"masquer la perte mesuree sur les temoins PRESENTS", code, codePerte)
	}
	if err != nil {
		t.Errorf("erreur rendue = %v, nil attendue : le verdict d'une perte se lit dans le "+
			"tableau et le code, l'avertissement de couverture est journalise a part", err)
	}

	rap := relireRapport(t, path)
	if !rap.CouvertureIncomplete {
		t.Errorf("couverture_incomplete = false, true attendu : le JSON doit porter LES DEUX " +
			"informations, sans quoi un lecteur automatique croit le corpus complet")
	}
	if len(rap.Temoins) != 2 {
		t.Fatalf("%d temoin(s) dans le JSON, 2 attendus", len(rap.Temoins))
	}
	if rap.Temoins[1].Pertes != 3 {
		t.Errorf("le temoin perdant porte %d perte(s) dans le JSON, 3 attendues — la perte doit "+
			"etre RAPPORTEE, pas seulement comptee dans le code de sortie", rap.Temoins[1].Pertes)
	}
}

// TestCouvertureIncompleteSortEnQuatreQuandLesPresentsSontAZero — l'autre sens : quand il n'y a
// rien d'autre a dire, « il en manque » reste le verdict, avec son erreur nommant lesquels.
func TestCouvertureIncompleteSortEnQuatreQuandLesPresentsSontAZero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gate.json")
	lignes := []ligneRapport{ligneAbsente("aaaaaaaa"), ligneAZero("bbbbbbbb")}

	code, err := finaliser(lignes, "base(abc1234)", optionsGate(path))
	if code != codeCouvertureIncomplete {
		t.Fatalf("code de sortie = %d, %d (codeCouvertureIncomplete) attendu : les temoins "+
			"presents sont a zero, il ne reste que la couverture a dire", code,
			codeCouvertureIncomplete)
	}
	if err == nil {
		t.Fatal("erreur nil : le code 4 doit etre accompagne de l'erreur qui NOMME les temoins " +
			"absents et leur cause — c'est elle qui dit lesquels relancer")
	}
	if rap := relireRapport(t, path); !rap.CouvertureIncomplete {
		t.Errorf("couverture_incomplete = false, true attendu")
	}
}

// TestAllowMissingResteInchange — `--allow-missing` eteint la verification de couverture : le
// meme corpus sort alors sur le seul verdict des temoins presents, et le fanion JSON retombe.
func TestAllowMissingResteInchange(t *testing.T) {
	cas := []struct {
		nom     string
		lignes  []ligneRapport
		attendu int
	}{
		{"absent + a zero", []ligneRapport{ligneAbsente("a"), ligneAZero("b")}, codeOK},
		{"absent + perdant", []ligneRapport{ligneAbsente("a"), lignePerdante("b")}, codePerte},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "gate.json")
			o := optionsGate(path)
			o.AllowMissing = true
			code, err := finaliser(c.lignes, "base(abc1234)", o)
			if code != c.attendu {
				t.Errorf("code = %d, %d attendu avec --allow-missing", code, c.attendu)
			}
			if err != nil {
				t.Errorf("erreur = %v, nil attendue avec --allow-missing", err)
			}
			if rap := relireRapport(t, path); rap.CouvertureIncomplete {
				t.Errorf("couverture_incomplete = true : --allow-missing rend la couverture " +
					"acceptee, le fanion doit retomber")
			}
		})
	}
}

// relireRapport relit le rapport JSON depose par `finaliser`.
func relireRapport(t *testing.T, path string) rapportJSON {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // chemin construit par le test
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Fatalf("aucun rapport JSON ecrit (%s) : il doit exister MEME quand la couverture "+
				"est incomplete — c'est lui qui dit lesquels relancer", path)
		}
		t.Fatalf("relecture du rapport : %v", err)
	}
	var rap rapportJSON
	if err := json.Unmarshal(raw, &rap); err != nil {
		t.Fatalf("le rapport JSON n'est pas relisible : %v", err)
	}
	return rap
}
