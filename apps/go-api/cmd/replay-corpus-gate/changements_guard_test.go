package main

// changements_guard_test.go — LES `Changements` NE DOIVENT PLUS RETOMBER A ZERO (2026-09-16),
// ET ILS PORTENT DESORMAIS LEUR PROPRE STATUT (2026-09-17, lot 2.8.1).
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
// Ce test tient les quatre maillons de la chaine d'un seul coup, sur le cas minimal qui compte :
// ZERO gain, ZERO perte, UN changement.
//
//	LE STATUT    le temoin doit sortir `CHANGEMENT` — depuis le 2026-09-17 un statut A LUI,
//	             plus `PERTE` : un changement se JUSTIFIE (reattribution documentee, voie de
//	             nommage qui cede) ou il se corrige, et le confondre avec une perte envoie
//	             chercher une regression la ou une valeur a seulement change de main.
//	LE DETAIL    il doit etre NOMME, pas seulement compte (D5 (1.9.9)).
//	LE TABLEAU   la colonne `chang.` doit porter le compte.
//	LE CODE      `codeSortie` doit refuser le run : distinct de la perte ne veut pas dire tolere.
//
// LES MUTATIONS QUI LE FONT ROUGIR, NOMMEES :
//   - retirer `l.Changements += b.Changements` de `remplirBilan` (report.go) : le compte
//     retombe a zero, le statut repasse a `ok`, le code a `codeOK` — trois assertions tombent ;
//   - retirer la branche `SensChangement` de `remplirBilan` : `ChangementsDetail` reste vide et
//     la section « DETAIL DES CHANGEMENTS » disparait ;
//   - retirer `case l.aUnChangement()` de `statut()` : le statut repasse a `ok`.

import (
	"strings"
	"testing"

	"levelup/go-api/internal/replaydiff"
)

// rapportUnSeulChangement fabrique le cas minimal : un axe, aucun gain, aucune perte, un
// changement NOMME. Les comptes sont ECRITS EN CLAIR — un test qui relit la valeur qu'il
// verifie ne verifie rien.
func rapportUnSeulChangement() replaydiff.Rapport {
	return replaydiff.Rapport{
		SchemaAncien:  59,
		SchemaNouveau: 59,
		Bilans: map[string]replaydiff.BilanAxe{
			"couverture": {Axe: "couverture", Gains: 0, Pertes: 0, Changements: 1, Identiques: 12},
		},
		Differences: []replaydiff.Difference{{
			Axe: "couverture", Metrique: "coverage.bridge.namedByNextLife",
			Sens: replaydiff.SensChangement, Ancien: "13", Nouveau: "8",
		}},
	}
}

// TestBilanPorteLesChangements — LE MAILLON 1 : la somme ne jette plus la troisieme categorie,
// et le detail la NOMME.
func TestBilanPorteLesChangements(t *testing.T) {
	var l ligneRapport
	l.remplirBilan(rapportUnSeulChangement())
	if l.Gains != 0 || l.Pertes != 0 {
		t.Fatalf("gains = %d et pertes = %d, 0 et 0 attendus — le cas de test a derive", l.Gains, l.Pertes)
	}
	if l.Changements != 1 {
		t.Errorf("changements = %d, 1 attendu — `remplirBilan` jette la categorie, "+
			"et un temoin dont une valeur publiee bouge ressortira `ok`", l.Changements)
	}
	if len(l.ChangementsDetail) != 1 {
		t.Fatalf("%d changement(s) nomme(s), 1 attendu — le compte sans le detail oblige le "+
			"pilote a relancer `replay-diff` a la main (D5 (1.9.9))", len(l.ChangementsDetail))
	}
	if got := l.ChangementsDetail[0].Metrique; got != "coverage.bridge.namedByNextLife" {
		t.Errorf("metrique du changement = %q, `coverage.bridge.namedByNextLife` attendue", got)
	}
}

// TestUnChangementSeulVautChangement — LE MAILLON 2 : le statut PROPRE et le code de sortie.
func TestUnChangementSeulVautChangement(t *testing.T) {
	l := ligneRapport{Temoin: Temoin{ID: "x", Famille: "f"}, Gains: 0, Pertes: 0, Changements: 1}
	if l.aUnePerte() {
		t.Errorf("un temoin a 0 perte ne doit PAS compter comme une perte — `PERTE` et " +
			"`CHANGEMENT` sont deux verdicts distincts depuis le 2026-09-17")
	}
	if got := l.statut(); got != statutChangement {
		t.Errorf("statut = %q, %q attendu — un changement sans perte a son propre verdict",
			got, statutChangement)
	}
	if !l.estBloquant() {
		t.Errorf("un temoin a 1 changement doit rester BLOQUANT : distinct de la perte ne veut " +
			"pas dire tolere en silence")
	}
	if got := codeSortie([]ligneRapport{l}, true); got != codePerte {
		t.Errorf("code de sortie = %d, %d attendu : le gate accepte un mouvement de valeur "+
			"publiee en silence", got, codePerte)
	}
}

// TestUnePerteEtUnChangementSortentPERTE — LA REGLE DE PRIORITE : `PERTE` prime. Un temoin qui
// porte les deux est un temoin en perte, et c'est la perte qu'on instruit.
func TestUnePerteEtUnChangementSortentPERTE(t *testing.T) {
	l := ligneRapport{Temoin: Temoin{ID: "x"}, Pertes: 1, Changements: 4}
	if got := l.statut(); got != statutPerte {
		t.Errorf("statut = %q, %q attendu : une perte prime toujours sur un changement", got, statutPerte)
	}
}

// TestTableauAfficheLesChangements — LE MAILLON 3 : la colonne existe, porte le compte, et le
// statut imprime est `CHANGEMENT`.
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
	if !strings.Contains(sortie, statutChangement) {
		t.Errorf("le statut n'est pas `%s` alors que le temoin porte 7 changements et 0 perte :\n%s",
			statutChangement, sortie)
	}
}

// TestDetailDesChangementsNommeChaqueEcart — LE MAILLON 4 : la section imprimee, symetrique de
// celle des pertes. C'est elle qui manquait a la cloture M1 (plan §5 : les 7 changements ont du
// etre nommes a la main avec `replay-diff` sur les artefacts conserves).
func TestDetailDesChangementsNommeChaqueEcart(t *testing.T) {
	var b strings.Builder
	imprimerDetailChangements(&b, []ligneRapport{
		{
			Temoin: Temoin{ID: "084a804d", Famille: "ctf_multi_manche"}, Changements: 1,
			ChangementsDetail: []replaydiff.Difference{{
				Axe: "couverture", Metrique: "coverage.bridge.namedByNextLife",
				Sens: replaydiff.SensChangement, Ancien: "0", Nouveau: "2",
			}},
		},
		{Temoin: Temoin{ID: "bcb6d393", Famille: "ctf_mono_manche"}, Pertes: 3},
	})
	out := b.String()
	for _, attendu := range []string{
		"DETAIL DES CHANGEMENTS", "084a804d", "couverture",
		"coverage.bridge.namedByNextLife", "0", "2",
	} {
		if !strings.Contains(out, attendu) {
			t.Errorf("la section doit contenir %q :\n%s", attendu, out)
		}
	}
	if strings.Contains(out, "bcb6d393") {
		t.Errorf("un temoin sans changement ne doit pas apparaitre dans cette section :\n%s", out)
	}
}

// TestDetailDesChangementsVideNEcritRien — aucun changement : pas de section vide qui
// laisserait croire a un rapport tronque (meme regle que pour les pertes).
func TestDetailDesChangementsVideNEcritRien(t *testing.T) {
	var b strings.Builder
	imprimerDetailChangements(&b, []ligneRapport{{Temoin: Temoin{ID: "x"}, Pertes: 4}})
	if b.String() != "" {
		t.Fatalf("aucun changement : sortie attendue vide, obtenu %q", b.String())
	}
}
