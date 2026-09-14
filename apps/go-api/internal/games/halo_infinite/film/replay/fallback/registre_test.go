package fallback

// registre_test.go — LES INVARIANTS DU REGISTRE, ET SON RAPPORT LISIBLE.
//
// CE QUE CE FICHIER TIENT, ET CE QU'IL LAISSE À `archlint` : ici, tout ce qui se vérifie SANS
// ouvrir le dépôt (forme des entrées, unicité des noms, ratchets de population). Le garde-rail
// `internal/archlint/no_unregistered_fallback_test.go` tient ce qui exige de lire les fichiers
// du décodeur : l'existence des ancres, et la convention de nommage des replis du code.

import (
	"sort"
	"strings"
	"testing"
)

// TestRegistreEstStructurellementValide : chaque entrée porte ses champs obligatoires, un nom
// conforme et unique, une condition et un ordre du domaine fermé, au moins un site complet.
func TestRegistreEstStructurellementValide(t *testing.T) {
	for _, pb := range VerifierRegistre() {
		t.Errorf("registre des replis : %s", pb)
	}
}

// plancherEntrees : le registre ne PERD pas d'entrées par accident.
//
// POURQUOI UN PLANCHER ET PAS UN COMPTE EXACT. Un compte exact ferait rougir tout lot de
// conversion — or la CONVERSION est le but : le registre doit MAIGRIR. Le plancher n'attrape
// donc qu'une chose, et c'est la seule qui soit toujours une faute : une tranche entière
// disparue d'un `concat` mal relu. Il se baisse à la main, dans le commit qui retire les
// entrées, avec la raison.
//
// Mesuré au lot 1.9.0 (2026-09-14) : 94 entrées. Plancher à 60.
const plancherEntrees = 60

func TestRegistrePorteToutesSesFamilles(t *testing.T) {
	if n := len(Table()); n < plancherEntrees {
		t.Fatalf("le registre ne porte que %d entrees (plancher %d) — une tranche entiere a-t-elle "+
			"quitte `concat` ? Si le retrait est voulu, baisser le plancher DANS le commit qui retire",
			n, plancherEntrees)
	}
	for _, fam := range []struct {
		nom  string
		part []Repli
	}{
		{"replay/equipement", registreReplayEquipement},
		{"replay/identites", registreReplayIdentites},
		{"killsource", registreKillsource},
		{"objectifs et construction", registreObjectifsEtConstruction},
		{"filmdec", registreFilmdec},
	} {
		if len(fam.part) == 0 {
			t.Errorf("la famille %q est vide — elle a ete videe sans que le plancher bouge", fam.nom)
		}
	}
}

// ratchetDevantLaLecture : le nombre de replis qui décident DEVANT une lecture disponible.
//
// C'EST LA VIOLATION DE D14 (b), ET CE NOMBRE NE MONTE JAMAIS. Chaque unité est un fait que le
// film ÉCRIT et qu'une heuristique tranche sans le consulter. Il descend au fil des lots 1.9.x ;
// le baisser est le geste qui clôt une conversion.
//
// Mesuré au lot 1.9.0 (2026-09-14) : 7.
const ratchetDevantLaLecture = 7

func TestReplisDevantLaLectureNeMontentPas(t *testing.T) {
	n := NbDevantLaLecture()
	if n > ratchetDevantLaLecture {
		var noms []string
		for _, r := range Table() {
			if r.Ordre == OrdreDevantLaLecture {
				noms = append(noms, string(r.Nom))
			}
		}
		t.Fatalf("%d replis decident DEVANT une lecture disponible (ratchet %d) : %v.\n"+
			"D14 (b) : lire d'abord, se replier ensuite. Un repli neuf dans cet ordre n'est pas "+
			"un repli, c'est une heuristique qui remplace la grammaire (D13)", n, ratchetDevantLaLecture, noms)
	}
	if n < ratchetDevantLaLecture {
		t.Logf("le ratchet peut descendre a %d (il vaut %d) — le baisser dans ce commit", n, ratchetDevantLaLecture)
	}
}

// TestCompteurBrancheEstDeclareSansAmbiguite : un compteur non branché nomme le lot qui le
// câblera. Sans cela, un zéro publié dans `coverage.fallbacks` se lirait « jamais déclenché »
// et ferait supprimer un repli actif (D14 d).
func TestCompteurBrancheEstDeclareSansAmbiguite(t *testing.T) {
	branches := 0
	for _, r := range Table() {
		if r.CompteurBranche {
			branches++
		}
	}
	if branches == 0 {
		t.Error("aucun repli n'a de compteur branche : la couverture ne publierait jamais rien")
	}
	t.Logf("compteurs branches : %d sur %d entrees", branches, len(Table()))
}

// TestCompteurEstSurEtNilSafe : le contrat du compteur — nil ne compte rien et ne panique pas,
// les déclenchements s'additionnent, le rapport est trié et sans zéro.
func TestCompteurEstSurEtNilSafe(t *testing.T) {
	var absent *Compteur
	absent.Declenche("repli_identite_piste_meilleur_recouvrement")
	absent.DeclencheN("repli_identite_piste_meilleur_recouvrement", 5)
	if got := absent.Compte("repli_identite_piste_meilleur_recouvrement"); got != 0 {
		t.Errorf("compteur nil : compte = %d, attendu 0", got)
	}
	if r := absent.Rapport(); r != nil {
		t.Errorf("compteur nil : rapport = %v, attendu nil", r)
	}

	c := NouveauCompteur()
	c.Declenche("repli_plafond_grenade_par_defaut")
	c.DeclencheN("repli_plafond_grenade_par_defaut", 2)
	c.Declenche("repli_identite_piste_meilleur_recouvrement")
	c.DeclencheN("repli_position_lacher_prend_la_prise", 0) // ignoré : k <= 0
	rap := c.Rapport()
	if len(rap) != 2 {
		t.Fatalf("rapport = %v, attendu 2 lignes (le k=0 ne doit pas en creer une)", rap)
	}
	if rap[0].Nom != "repli_identite_piste_meilleur_recouvrement" || rap[1].Declenchements != 3 {
		t.Errorf("rapport mal trie ou mal cumule : %v", rap)
	}
}

// TestRapportDuRegistre n'est PAS une assertion : c'est la SORTIE HUMAINE du registre, exigée
// par l'item 1.9.0 (« les déclenchements se lisent dans l'artefact ET dans une sortie humaine »).
//
//	go test ./internal/games/halo_infinite/film/replay/fallback/ -run RapportDuRegistre -v
//
// Elle liste, par paquet, chaque repli avec sa condition, son ordre, l'état de son compteur et
// sa cible de retrait — de quoi préparer une revue de jalon sans ouvrir un fichier.
func TestRapportDuRegistre(t *testing.T) {
	parPaquet := map[string][]Repli{}
	for _, r := range Table() {
		parPaquet[r.Paquet()] = append(parPaquet[r.Paquet()], r)
	}
	paquets := make([]string, 0, len(parPaquet))
	for p := range parPaquet {
		paquets = append(paquets, p)
	}
	sort.Strings(paquets)

	t.Logf("REGISTRE DES REPLIS — %d entrees, %d paquets", len(Table()), len(paquets))
	for _, p := range paquets {
		t.Logf("")
		t.Logf("=== %s (%d)", p, len(parPaquet[p]))
		for _, r := range parPaquet[p] {
			t.Logf("  %-52s %-18s %-18s %s", r.Nom, r.Condition, r.Ordre, etatDuCompteur(r))
			t.Logf("      fait    : %s", r.Fait)
			t.Logf("      retrait : %s | %s", r.CibleRetrait, r.CritereRetrait)
			for _, s := range r.Sites {
				t.Logf("      site    : %s [%s]", s.Fichier, abrege(s.Ancre))
			}
		}
	}
}

// etatDuCompteur : la colonne « compte-t-on ce repli ? » du rapport.
func etatDuCompteur(r Repli) string {
	if r.CompteurBranche {
		return "compteur: BRANCHE"
	}
	return "compteur: a cabler (" + r.CibleComptage + ")"
}

// abrege raccourcit une ancre pour le rapport, sans jamais la couper au milieu d'un octet
// multi-octets (les ancres portent des accents).
func abrege(s string) string {
	const max = 56
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return strings.TrimSpace(string(r[:max])) + "..."
}
