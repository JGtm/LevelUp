// Package ops — seed_demo_synthetic_sources_test.go : LE GARDE-RAIL DES SOURCES DE DÉGÂT
// DU CORPUS DE DÉMO.
//
// POURQUOI CE FICHIER EXISTE. Les trois `source_tag` du corpus sont des entiers écrits en
// dur (`demoSourceTagBR75`, …). Un entier ne se relit pas : si la table embarquée du titre
// change — une entrée retirée, un statut passé à AMBIGU, une clé de registre renommée — le
// seeder continuerait d'écrire les mêmes valeurs, le classificateur n'en résoudrait plus
// aucune, et l'encart « arme favorite » de l'Accueil disparaîtrait de la démo SANS QU'UN
// SEUL TEST NE TOMBE. C'est exactement le mode de panne muet que la règle du dépôt vise :
// un nombre magique se double d'un garde-rail, sinon il pourrit.
//
// CE QUE LE TEST FAIT PASSER PAR LE VRAI CHEMIN : `halo_infinite.KillSourceRegistry`, le
// classificateur que la lecture (`favoriteWeaponFromSource`) reçoit par injection. Pas une
// copie, pas une table de test — le même objet.
package ops

import (
	"testing"

	haloinfinite "levelup/go-api/internal/games/halo_infinite"
)

// TestDemoSourceTagsResolventAuRegistre : les trois tags du corpus rendent les trois clés
// de registre attendues, par le classificateur de production.
func TestDemoSourceTagsResolventAuRegistre(t *testing.T) {
	registre := haloinfinite.NewKillSourceRegistry()
	for _, cas := range []struct {
		nom     string
		tag     uint32
		cleVoue string
	}{
		{"BR75", demoSourceTagBR75, "hinf_br75"},
		{"MA40 AR", demoSourceTagMA40, "hinf_ma40_ar"},
		{"Bandit", demoSourceTagBandit, "hinf_bandit"},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			cle, ok := registre.KillSourceRegistryKey(cas.tag)
			if !ok {
				t.Fatalf("tag 0x%08x (%s) ne resout AUCUNE cle de registre — la table embarquee "+
					"du titre a change : relever un tag courant dans "+
					"games/halo_infinite/film/damagetag/data/labels.tsv et corriger la constante",
					cas.tag, cas.nom)
			}
			if cle != cas.cleVoue {
				t.Fatalf("tag 0x%08x (%s) resout %q, attendu %q", cas.tag, cas.nom, cle, cas.cleVoue)
			}
		})
	}
}

// TestDemoSourceTagsSontDistincts : trois armes, trois tags. Deux constantes égales
// rendraient la repartition de demoSourceTagPour muette (l'arme favorite resterait la
// bonne, mais le corpus n'aurait plus qu'une seule arme et le top armes serait vide).
func TestDemoSourceTagsSontDistincts(t *testing.T) {
	vus := map[uint32]bool{}
	for _, tag := range []uint32{demoSourceTagBR75, demoSourceTagMA40, demoSourceTagBandit} {
		if vus[tag] {
			t.Fatalf("tag 0x%08x en double dans la palette de demo", tag)
		}
		vus[tag] = true
	}
}

// TestDemoSourceTagPourDesigneUneGagnanteNette : sur le corpus complet, une arme sort
// DEVANT les autres, et une part des morts reste non attribuee.
//
// POURQUOI VERIFIER LA MARGE ET PAS SEULEMENT LE VAINQUEUR : `weaponKillsFromSourceForPlayer`
// trie par frags puis par cle de registre. A egalite, l'arme affichee serait decidee par
// l'ordre alphabetique des cles — un detail d'implementation deviendrait le contenu de la
// demo. La marge doit venir de la repartition, pas du departage.
func TestDemoSourceTagPourDesigneUneGagnanteNette(t *testing.T) {
	// Le corpus reel compte 60 matchs ; on balaie large pour couvrir toutes les phases des
	// dix seaux quel que soit le nombre de morts par match.
	const matchs, mortsParMatch = 60, 20

	compte := map[uint32]int{}
	nonAttribuees := 0
	for idxMatch := 0; idxMatch < matchs; idxMatch++ {
		for idxKill := 0; idxKill < mortsParMatch; idxKill++ {
			tag, mesuree := demoSourceTagPour(idxMatch, idxKill)
			if !mesuree {
				nonAttribuees++
				continue
			}
			compte[tag]++
		}
	}
	if nonAttribuees == 0 {
		t.Error("aucune mort non attribuee : la demo montrerait une couverture que la " +
			"production n'a jamais (la portee est PAR LIGNE)")
	}
	if len(compte) != 3 {
		t.Fatalf("%d armes distinctes dans le corpus, attendu 3", len(compte))
	}
	gagnante, meilleur, second := uint32(0), 0, 0
	for tag, n := range compte {
		switch {
		case n > meilleur:
			gagnante, second, meilleur = tag, meilleur, n
		case n > second:
			second = n
		}
	}
	if gagnante != demoSourceTagBR75 {
		t.Errorf("arme favorite = 0x%08x, attendu le BR75 (0x%08x)", gagnante, demoSourceTagBR75)
	}
	if meilleur <= second {
		t.Errorf("egalite en tete (%d vs %d) : l'arme affichee serait decidee par le "+
			"departage sur la cle de registre, pas par la repartition", meilleur, second)
	}
}
