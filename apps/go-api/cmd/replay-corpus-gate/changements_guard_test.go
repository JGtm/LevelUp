package main

// changements_guard_test.go — LES `Changements` NE DOIVENT PLUS RETOMBER A ZERO (2026-09-16).
//
// # CE QUE CE FICHIER GARDE, ET POURQUOI IL EXISTE
//
// `replaydiff.BilanAxe` compte TROIS categories : les gains, les pertes, et les CHANGEMENTS —
// une valeur publiee qui BOUGE sans etre ni l'un ni l'autre. Jusqu'au 2026-09-16,
// `bilanDepuisRapport` sommait `b.Gains` et `b.Pertes` et s'arretait la : la troisieme categorie
// n'etait pas seulement absente de l'affichage, elle etait JETEE. Un temoin dont une valeur
// publiee bougeait sortait `ok`, au tableau comme au JSON, et le lot suivant heritait d'un
// changement que personne n'avait classe.
//
// Ce test tient les trois maillons de la chaine d'un seul coup, sur le cas minimal qui compte :
// ZERO gain, ZERO perte, UN changement.
//
//	LE STATUT    le temoin doit sortir `PERTE`, parce qu'un changement se classe comme une
//	             perte : il se justifie (divergence prouvee chez l'ecrivain) ou il se corrige.
//	LE TABLEAU   la colonne `chang.` doit porter le compte.
//	LE CODE      `codeSortie` doit refuser le run.
//
// LA MUTATION QUI LE FAIT ROUGIR, NOMMEE : retirer `changements += b.Changements` de
// `bilanDepuisRapport` (report.go). Le compte retombe a zero, le statut repasse a `ok`, et les
// trois assertions ci-dessous tombent ensemble.

import (
	"strings"
	"testing"

	"levelup/go-api/internal/replaydiff"
)

// rapportUnSeulChangement fabrique le cas minimal : un axe, aucun gain, aucune perte, un
// changement. Les comptes sont ECRITS EN CLAIR — un test qui relit la valeur qu'il verifie ne
// verifie rien.
func rapportUnSeulChangement() replaydiff.Rapport {
	return replaydiff.Rapport{
		SchemaAncien:  59,
		SchemaNouveau: 59,
		Bilans: map[string]replaydiff.BilanAxe{
			"couverture": {Axe: "couverture", Gains: 0, Pertes: 0, Changements: 1, Identiques: 12},
		},
	}
}

// TestBilanPorteLesChangements — LE MAILLON 1 : la somme ne jette plus la troisieme categorie.
func TestBilanPorteLesChangements(t *testing.T) {
	_, _, gains, pertes, changements, _ := bilanDepuisRapport(rapportUnSeulChangement())
	if gains != 0 || pertes != 0 {
		t.Fatalf("gains = %d et pertes = %d, 0 et 0 attendus — le cas de test a derive", gains, pertes)
	}
	if changements != 1 {
		t.Errorf("changements = %d, 1 attendu — `bilanDepuisRapport` jette la categorie, "+
			"et un temoin dont une valeur publiee bouge ressortira `ok`", changements)
	}
}

// TestUnChangementSeulVautPerte — LE MAILLON 2 : le statut et le code de sortie.
func TestUnChangementSeulVautPerte(t *testing.T) {
	l := ligneRapport{Temoin: Temoin{ID: "x", Famille: "f"}, Gains: 0, Pertes: 0, Changements: 1}
	if !l.aUnePerte() {
		t.Errorf("un temoin a 0 gain, 0 perte et 1 changement ne compte pas comme une perte — " +
			"il sortirait `ok` et le changement ne serait jamais classe")
	}
	if got := codeSortie([]ligneRapport{l}, true); got == 0 {
		t.Errorf("code de sortie = 0 alors qu'un changement n'est pas classe ; le gate accepte " +
			"un mouvement de valeur publiee en silence")
	}
}

// TestTableauAfficheLesChangements — LE MAILLON 3 : la colonne existe et porte le compte.
func TestTableauAfficheLesChangements(t *testing.T) {
	var b strings.Builder
	imprimerTableau(&b, []ligneRapport{
		{Temoin: Temoin{ID: "x", Famille: "f"}, SchemaReference: 59, SchemaHEAD: 59, Changements: 7},
	}, "base")
	sortie := b.String()
	if !strings.Contains(sortie, "chang.") {
		t.Errorf("l'en-tete du tableau ne porte pas la colonne `chang.` :\n%s", sortie)
	}
	if !strings.Contains(sortie, "7") {
		t.Errorf("le compte de changements (7) n'apparait pas dans la ligne :\n%s", sortie)
	}
	if !strings.Contains(sortie, "PERTE") {
		t.Errorf("le statut n'est pas `PERTE` alors que le temoin porte 7 changements :\n%s", sortie)
	}
}
