package main

// verdict_reclassement_test.go — DEUX REQUALIFICATIONS DU VERDICT, ET LA GARDE QUI LES BORNE.
//
// (A) une baisse de `coverage.deathsPaths.directScan.{matched,published}` que la hausse de la
// feuille homonyme de `walk` compense est un changement (verdict_deplacements.go) ;
// (B) les compteurs d'echec de `coverage.vehicles.destructions.*` d'un bloc absent de la base
// sont des changements, parce que `destroyed` est leur denominateur declare (verdict_metriques.go).
//
// Chaque test passe par `replaydiff.Comparer` puis `remplirBilan` : le sens des ecarts est celui
// que la production leur donne (polarite comprise), pas un sens ecrit a la main.
//
// MUTATIONS QUI LES FONT ROUGIR :
//   - retirer l'appel a `estDeplacementCompense` de `classerPourLeVerdict` : les cas compenses
//     comptent une perte ;
//   - `hausse >= baisse` -> `hausse > 0` dans `estDeplacementCompense` : le cas partiellement
//     compense ne compte plus de perte ;
//   - retirer le test `!connue` : le cas sans hausse de la marche ne compte plus de perte ;
//   - comparer une cible a une autre feuille que son homonyme : le cas croise ne compte plus de perte ;
//   - retirer la famille `coverage.vehicles.destructions.` de `rejetsDeCouverture` : le bloc absent
//     de la base compte six pertes ;
//   - lui retirer son denominateur (case vide) : meme effet, et le cas a rapport degrade est
//     compte comme une perte pour la mauvaise raison.

import (
	"testing"

	"levelup/go-api/internal/replaydiff"
)

const (
	balayageDirect = "coverage.deathsPaths.directScan."
	marcheDesMorts = "coverage.deathsPaths.walk."
	destructions   = "coverage.vehicles.destructions."
)

// ligneDeMesures confronte deux jeux de feuilles de couverture (absente = feuille non publiee) et
// range l'ecart comme le fait le gate.
func ligneDeMesures(base, head map[string]float64) ligneRapport {
	empreinte := func(m map[string]float64) replaydiff.Empreinte {
		e := replaydiff.Empreinte{Mesures: map[string]replaydiff.Mesure{}}
		for k, v := range m {
			e.Mesures["couverture/"+k] = replaydiff.Mesure{EstNum: true, Num: v}
		}
		return e
	}
	var l ligneRapport
	l.Temoin = Temoin{ID: "temoin", Famille: "famille"}
	l.remplirBilan(replaydiff.Comparer(empreinte(base), empreinte(head)))
	return l
}

// comptePertes rend les metriques classees en perte, pour des messages qui nomment l'ecart.
func comptePertes(l ligneRapport) []string {
	var out []string
	for _, d := range l.PertesDetail {
		out = append(out, d.Metrique)
	}
	return out
}

// TestDeplacementCompense_NeBloquePas — (A) cas 1 : la marche reprend ce que le balayage lisait.
// Les valeurs sont celles d'un temoin du corpus : 132 + 5 -> 137 + 0, total inchange.
func TestDeplacementCompense_NeBloquePas(t *testing.T) {
	l := ligneDeMesures(
		map[string]float64{
			balayageDirect + "matched": 5, balayageDirect + "published": 5,
			marcheDesMorts + "matched": 132, marcheDesMorts + "published": 132,
		},
		map[string]float64{
			balayageDirect + "matched": 0, balayageDirect + "published": 0,
			marcheDesMorts + "matched": 137, marcheDesMorts + "published": 137,
		})
	if l.Pertes != 0 || len(l.Filets) != 0 {
		t.Fatalf("pertes %v, filets %d : une baisse compensee par la marche n'est pas une perte",
			comptePertes(l), len(l.Filets))
	}
	if l.Changements != 2 || l.Gains != 2 {
		t.Errorf("changements=%d gains=%d, attendu 2 changements (les deux feuilles du balayage) "+
			"et 2 gains (celles de la marche)", l.Changements, l.Gains)
	}
	if l.statut() != statutOK || codeSortie([]ligneRapport{l}, true) != codeOK {
		t.Errorf("statut %q : un deplacement compense ne bloque pas", l.statut())
	}
}

// TestDeplacementCompense_FeuilleDisparue — la feuille du balayage peut ne plus etre publiee du
// tout (sens `disparu`, valeur nouvelle vide) : la regle s'applique de la meme facon.
func TestDeplacementCompense_FeuilleDisparue(t *testing.T) {
	l := ligneDeMesures(
		map[string]float64{balayageDirect + "matched": 2, marcheDesMorts + "matched": 65},
		map[string]float64{marcheDesMorts + "matched": 70})
	if l.Pertes != 0 {
		t.Fatalf("pertes %v : une feuille disparue et compensee n'est pas une perte", comptePertes(l))
	}
	if l.Changements != 1 {
		t.Errorf("changements=%d, attendu 1 (la feuille disparue)", l.Changements)
	}
}

// TestDeplacementPartiellementCompense_Bloque — (A) cas 2 : la marche ne reprend qu'une partie.
// 13 morts quittent le balayage, 10 arrivent a la marche : trois ont disparu.
func TestDeplacementPartiellementCompense_Bloque(t *testing.T) {
	l := ligneDeMesures(
		map[string]float64{balayageDirect + "matched": 13, marcheDesMorts + "matched": 124},
		map[string]float64{balayageDirect + "matched": 0, marcheDesMorts + "matched": 134})
	if got := comptePertes(l); len(got) != 1 || got[0] != balayageDirect+"matched" {
		t.Fatalf("pertes %v, attendu la seule baisse du balayage : la hausse de la marche ne la couvre pas", got)
	}
	if l.statut() != statutPerte {
		t.Errorf("statut %q, attendu %q", l.statut(), statutPerte)
	}
}

// TestDeplacementSansHausseDeLaMarche_Bloque — (A) cas 3 : le balayage baisse, la marche ne bouge
// pas. C'est la perte nue.
func TestDeplacementSansHausseDeLaMarche_Bloque(t *testing.T) {
	l := ligneDeMesures(
		map[string]float64{balayageDirect + "published": 5, marcheDesMorts + "published": 132},
		map[string]float64{balayageDirect + "published": 0, marcheDesMorts + "published": 132})
	if got := comptePertes(l); len(got) != 1 || got[0] != balayageDirect+"published" {
		t.Fatalf("pertes %v, attendu la baisse du balayage : la marche n'a rien repris", got)
	}
	if l.statut() != statutPerte {
		t.Errorf("statut %q, attendu %q", l.statut(), statutPerte)
	}
}

// TestDeplacementMarcheEnBaisse_Bloque — la cible baisse aussi : les deux voies perdent.
func TestDeplacementMarcheEnBaisse_Bloque(t *testing.T) {
	l := ligneDeMesures(
		map[string]float64{balayageDirect + "matched": 5, marcheDesMorts + "matched": 132},
		map[string]float64{balayageDirect + "matched": 0, marcheDesMorts + "matched": 120})
	if len(comptePertes(l)) != 2 {
		t.Fatalf("pertes %v, attendu les deux baisses (balayage et marche)", comptePertes(l))
	}
}

// TestDeplacementCroise_Bloque — la hausse d'une feuille ne compense jamais la baisse d'une
// autre : `published` monte a la marche, mais `matched` baisse au balayage sans reprise.
func TestDeplacementCroise_Bloque(t *testing.T) {
	l := ligneDeMesures(
		map[string]float64{balayageDirect + "matched": 5, marcheDesMorts + "published": 132},
		map[string]float64{balayageDirect + "matched": 0, marcheDesMorts + "published": 140})
	if got := comptePertes(l); len(got) != 1 || got[0] != balayageDirect+"matched" {
		t.Fatalf("pertes %v, attendu la baisse de `matched` : seule la feuille homonyme compense", got)
	}
}

// TestDestructionsBlocAbsentDeLaBase_NeBloquePas — (B) cas 1 : le bloc n'existe pas dans
// l'artefact de base. Les six compteurs d'echec « apparaissent » avec `destroyed`, dont l'absence
// vaut un denominateur nul : des changements.
func TestDestructionsBlocAbsentDeLaBase_NeBloquePas(t *testing.T) {
	l := ligneDeMesures(
		map[string]float64{},
		map[string]float64{
			destructions + "destroyed": 20, destructions + "pieces": 17,
			destructions + "killerAbsent": 16, destructions + "killerOutOfRoster": 4,
			destructions + "killerAmbiguous": 1, destructions + "causeUnresolved": 20,
			destructions + "causeAbsent": 2, destructions + "killedNoIdentity": 2,
		})
	if l.Pertes != 0 || len(l.Filets) != 0 {
		t.Fatalf("pertes %v : un bloc neuf n'est pas une regression", comptePertes(l))
	}
	if l.Changements != 8 {
		t.Errorf("changements=%d, attendu 8 (le denominateur, les pieces et les six rejets)", l.Changements)
	}
	if l.statut() != statutOK {
		t.Errorf("statut %q, attendu %q", l.statut(), statutOK)
	}
}

// TestDestructionsRapportDegrade_Bloque — (B) cas 2 : le bloc existe des deux cotes avec un
// denominateur non nul ; la regle du rapport au denominateur s'applique telle quelle.
func TestDestructionsRapportDegrade_Bloque(t *testing.T) {
	cas := []struct {
		nom        string
		base, head map[string]float64
	}{
		{"denominateur constant",
			map[string]float64{destructions + "destroyed": 20, destructions + "killerAbsent": 2},
			map[string]float64{destructions + "destroyed": 20, destructions + "killerAbsent": 8}},
		{"denominateur en hausse, rapport degrade",
			map[string]float64{destructions + "destroyed": 20, destructions + "killerAbsent": 2},
			map[string]float64{destructions + "destroyed": 40, destructions + "killerAbsent": 30}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			l := ligneDeMesures(c.base, c.head)
			if got := comptePertes(l); len(got) != 1 || got[0] != destructions+"killerAbsent" {
				t.Fatalf("pertes %v, attendu `killerAbsent` : le rapport au denominateur se degrade", got)
			}
			if l.statut() != statutPerte {
				t.Errorf("statut %q, attendu %q", l.statut(), statutPerte)
			}
		})
	}
}

// TestDestructionsRapportStable_NeBloquePas — le meme compteur qui monte avec son denominateur,
// rapport inchange ou meilleur, reste un changement.
func TestDestructionsRapportStable_NeBloquePas(t *testing.T) {
	l := ligneDeMesures(
		map[string]float64{destructions + "destroyed": 20, destructions + "killerAbsent": 2},
		map[string]float64{destructions + "destroyed": 100, destructions + "killerAbsent": 10})
	if l.Pertes != 0 {
		t.Fatalf("pertes %v : le rapport 2/20 -> 10/100 ne se degrade pas", comptePertes(l))
	}
}
