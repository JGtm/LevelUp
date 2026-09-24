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
// donc qu'une chose, et c'est la seule qui soit toujours une faute : une hémorragie d'entrées.
// Il se baisse à la main, dans le commit qui retire les entrées, avec la raison.
//
// Mesuré au lot 1.9.0 (2026-09-14) : 94 entrées. Plancher à 60. Mesuré de nouveau le
// 2026-09-16 (revue de jalon M1, ronde 2) : 96 entrées ; le plancher NE MONTE PAS avec elles —
// c'est un plancher de sécurité, pas un compte.
const plancherEntrees = 60

// plancherTranches : le nombre de FAMILLES du registre. Il ne descend que délibérément.
//
// IL ÉTAIT À 2 ET NE MORDAIT SUR RIEN (revue de jalon M1, ronde 2, constat F1) : la garde
// `len(tranches) < 2` laissait passer 6 -> 5, donc le retrait d'un fichier de tranche entier.
// Mesuré le 2026-09-23 : HUIT familles (la huitième, `replay/positions`, est née au lot M1 des
// retours du rejeu : les deux gardes mesurées de la publication des positions).
// Mesuré le 2026-09-17 : SEPT familles (la septième, `killsource/calibration`, est née au
// lot 3.4.1 : ce que le décodeur décide encore par balayage, faute de source lue)
// Mesuré le 2026-09-16 : SIX familles (`replay/equipement` 14, `replay/identites` 22,
// `killsource` 26, `killsource/carte` 2, `objectifs et construction` 21, `grammar` 11 — 96
// entrées). Le plancher vaut donc la valeur réelle : une famille en moins se voit.
const plancherTranches = 8

// famillesAttendues : LES SEPT FAMILLES, NOMMÉES, DANS L'ORDRE DE L'ASSEMBLAGE.
//
// POURQUOI UNE LISTE DE NOMS, ET PAS UN CHAÎNON ARITHMÉTIQUE (revue de jalon M1, ronde 2,
// constat F1). Le test refermait sa boucle sur `somme(tranches) == len(Table())` — une
// TAUTOLOGIE : [registre] VAUT `concat(Tranches())`, l'égalité est vraie par construction et
// aucune mutation ne peut la casser. Mutation jouée par le relecteur, la tranche
// `killsource/carte` retirée de [Tranches] : `fallback` ET `archlint` restaient VERTS pendant
// que deux entrées quittaient le registre en silence.
//
// Ce que `concat` ne peut PAS déduire de lui-même, c'est la liste de ce qu'il DOIT porter. Elle
// vient donc du dehors, elle est nommée, et elle se met à jour À LA MAIN dans le commit qui
// ajoute ou retire un fichier de tranche — c'est ce geste manuel qui rend le retrait délibéré.
//
// ELLE NE COMPTE AUCUNE ENTRÉE : une entrée de plus (ou de moins) dans une famille ne la fait
// pas rougir. Ce qu'elle tient est la FAMILLE, parce qu'une famille perdue est toujours une
// faute, là où une entrée retirée est le but du chantier.
var famillesAttendues = []string{
	"replay/equipement",
	"replay/identites",
	"killsource",
	"killsource/carte",
	"killsource/calibration",
	"objectifs et construction",
	"filmdec",
	"replay/positions",
}

func TestRegistrePorteToutesSesFamilles(t *testing.T) {
	if n := len(Table()); n < plancherEntrees {
		t.Fatalf("le registre ne porte que %d entrees (plancher %d) — une tranche entiere a-t-elle "+
			"quitte `concat` ? Si le retrait est voulu, baisser le plancher DANS le commit qui retire",
			n, plancherEntrees)
	}
	tranches := Tranches()
	if len(tranches) < plancherTranches {
		t.Fatalf("Tranches() ne rend que %d familles (plancher %d) — un fichier de tranche a quitte "+
			"l assemblage. Si le retrait est voulu, baisser `plancherTranches` ET retirer le nom de "+
			"`famillesAttendues`, DANS le commit qui retire", len(tranches), plancherTranches)
	}
	rangs := map[string]int{}
	for i, fam := range tranches {
		if _, deja := rangs[fam.Nom]; deja {
			t.Errorf("la famille %q est declaree deux fois dans Tranches()", fam.Nom)
		}
		rangs[fam.Nom] = i
		if len(fam.Replis) == 0 {
			t.Errorf("la famille %q est vide — elle a ete videe sans que le plancher bouge", fam.Nom)
		}
		if i < len(famillesAttendues) && fam.Nom != famillesAttendues[i] {
			t.Errorf("famille de rang %d : %q, attendue %q — l ordre de Tranches() a change",
				i, fam.Nom, famillesAttendues[i])
		}
	}
	// LE CONTROLE QUI MORD : chaque famille ATTENDUE est encore la. C est la direction que le
	// chainon arithmetique ne tenait pas — lui partait de `Tranches()` pour y revenir.
	for _, nom := range famillesAttendues {
		if _, ok := rangs[nom]; !ok {
			t.Errorf("la famille %q a QUITTE Tranches() : ses entrees ne sont plus au registre, et "+
				"rien d autre ne le dirait. Retrait volontaire ? le retirer AUSSI de "+
				"`famillesAttendues` et baisser `plancherTranches`, dans le meme commit", nom)
		}
	}
}

// ratchetDevantLaLecture : le nombre de replis qui décident DEVANT une lecture disponible.
//
// C'EST LA VIOLATION DE D14 (b), ET CE NOMBRE NE MONTE JAMAIS. Chaque unité est un fait que le
// film ÉCRIT et qu'une heuristique tranche sans le consulter. Il descend au fil des lots 1.9.x ;
// le baisser est le geste qui clôt une conversion.
//
// Mesuré au lot 1.9.0 (2026-09-14) : 7. BAISSÉ À 6 au lot 1.9.2 (2026-09-15) :
// `repli_i0_porte_et_region_par_defaut` passe en `apres_lecture` — les deux chemins de
// `sync/killcollector` imposent désormais le découpage d'i0 du CATALOGUE de carte, comme le
// chemin de cuisson, et l'auto-détection n'entre plus que là où le catalogue se tait.
//
// BAISSÉ À 5 AU LOT 1.9.13 (2026-09-15) : `repli_vie_coupee_au_trou_de_replication` passe en
// `apres_lecture`. Où finit une vie SE LIT désormais — une mort écrite du joueur, un record de
// création de bipède, une frontière de manche —, et le seuil de trou (`lifeGapUS`) n'entre plus
// que pour un joueur dont le film n'écrit AUCUNE mort. Mesure qui l'autorise : sur les 212
// coupures que le seuil décidait sur les 8 builds, 208 n'étaient justifiées PAR RIEN ; elles sont
// devenues des lacunes, les 4 restantes portent toutes une mort écrite, et le repli ne se
// déclenche sur aucun des 8 builds.
// BAISSÉ À 5 AU LOT 1.9.10 (2026-09-16) : `repli_fin_de_vie_vehicule_par_recensement` est
// SUPPRIMÉ, pas rétrogradé — la fin de vie d'un véhicule se lit au composant
// `object-dead-state` de `ti=40`, et la borne « dernier recensement + 20 s » a disparu du code
// avec son entrée du registre.
// FUSION (2026-09-16) : les deux baisses (1.9.13 et 1.9.10) partaient toutes deux de 6 ; réunies,
// le registre ne porte plus que QUATRE entrées `devant_la_lecture` — le ratchet suit.
// BAISSÉ À 5 AU LOT 1.9.11 (2026-09-16) : `repli_manches_contigues_decretees` est devenu
// `repli_manche_zero_decretee` et passe en `film_muet` / `apres_lecture`. L'entrée couvrait
// DEUX mécanismes ; seul le plancher (« aucune manche admise, la manche 0 est décrétée ») est un
// repli, et il se déclenche bien sur un silence. La règle d'ORDRE, elle, se déclenche sur un
// DÉSACCORD avec une lecture : elle N'EST PAS un repli au sens de D14 (b) et le lot la publie
// comme une CONTRADICTION comptée (`coverage.score.roundsContradicted`).
// FUSION (2026-09-17) : trois baisses partant de 6 (1.9.13, 1.9.10, 1.9.11) ; le registre ne porte
// plus que TROIS entrées `devant_la_lecture` — le ratchet suit.
const ratchetDevantLaLecture = 3

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
//	go test ./internal/games/halo_infinite/film/internal/facts/fallback/ -run RapportDuRegistre -v
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
