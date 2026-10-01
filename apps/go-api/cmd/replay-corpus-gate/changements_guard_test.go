package main

// changements_guard_test.go — LES `Changements` NE DOIVENT PLUS RETOMBER A ZERO (2026-09-16),
// ET DEPUIS LE BANC DE VERITE (2026-09-30) ILS INFORMENT SANS BLOQUER.
//
// # CE QUE CE FICHIER GARDE, ET POURQUOI IL EXISTE
//
// `replaydiff.BilanAxe` compte TROIS categories : les gains, les pertes, et les CHANGEMENTS —
// une valeur publiee qui BOUGE sans etre ni l'un ni l'autre. Jusqu'au 2026-09-16,
// `bilanDepuisRapport` sommait `b.Gains` et `b.Pertes` et s'arretait la : la troisieme categorie
// etait JETEE, du tableau comme du JSON.
//
// Du 2026-09-17 au 2026-09-30 un changement portait son propre statut, `CHANGEMENT`, BLOQUANT.
// Depuis le banc de verite (verite.go, decision D-5), c'est le banc qui rend le verdict : un
// changement ne dit pas si la donnee est plus JUSTE, il ne bloque donc plus — mais il doit rester
// COMPTE et NOMME, pour qu'un operateur le voie. Ce test tient les maillons qui restent :
//
//	LE DETAIL    il doit etre NOMME, pas seulement compte (D5 (1.9.9)).
//	LE TABLEAU   la colonne `chang.` doit porter le compte.
//	LE VERDICT   un changement SEUL rend `ok` et le code 0 : le banc decide.
//
// LES MUTATIONS QUI LE FONT ROUGIR, NOMMEES :
//   - retirer `l.Changements++` de `remplirBilan` (report.go) : le compte retombe a zero ;
//   - retirer la branche `SensChangement` de `remplirBilan` : `ChangementsDetail` reste vide et
//     la section « DETAIL DES CHANGEMENTS » disparait ;
//   - rendre un changement bloquant dans `statutVerite` : le statut sort autre chose que `ok`.

import (
	"strings"
	"testing"

	"levelup/go-api/internal/replaydiff"
	"levelup/go-api/internal/replayverite"
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

// TestUnChangementSeulInformeSansBloquer — LE MAILLON 2 : un changement seul, avec un banc `ok`,
// rend `ok` et le code 0 ; il reste compte (maillons 1 et 3).
func TestUnChangementSeulInformeSansBloquer(t *testing.T) {
	var l ligneRapport
	l.Temoin = Temoin{ID: "x", Famille: "f"}
	l.Verite = comparaisonDe(replayverite.StatutOK)
	l.remplirBilan(rapportUnSeulChangement())
	if l.Changements != 1 {
		t.Fatalf("changements = %d, 1 attendu", l.Changements)
	}
	if got := l.statut(); got != statutOK {
		t.Errorf("statut = %q, %q attendu — un changement seul informe, le banc decide", got, statutOK)
	}
	if l.estBloquant() {
		t.Errorf("un changement seul ne doit plus bloquer (banc de verite, D-5)")
	}
	if got := codeSortie([]ligneRapport{l}, true); got != codeOK {
		t.Errorf("code de sortie = %d, %d attendu", got, codeOK)
	}
}

// TestUnFiletEtUnChangementSortentPERTE — un filet (perte non couverte par le banc) bloque, quels
// que soient les changements qui l'accompagnent.
func TestUnFiletEtUnChangementSortentPERTE(t *testing.T) {
	l := ligneRapport{Temoin: Temoin{ID: "x"}, Pertes: 1, Changements: 4,
		Filets: []replaydiff.Difference{perte("armes", "shots/n", "2", "1")}}
	if got := l.statut(); got != statutPerte {
		t.Errorf("statut = %q, %q attendu : un filet bloque", got, statutPerte)
	}
}

// TestTableauAfficheLesChangements — LE MAILLON 3 : la colonne existe, porte le compte, et le
// statut imprime est `ok` (un changement informe, le banc de verite decide).
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
	if !strings.HasSuffix(strings.TrimSpace(sortie), statutOK) {
		t.Errorf("le statut n'est pas `%s` alors que le temoin ne porte que 7 changements :\n%s",
			statutOK, sortie)
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
