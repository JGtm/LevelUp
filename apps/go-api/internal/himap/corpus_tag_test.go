package himap

// corpus_tag_test.go — LE GARDE-RAIL DU TAG `gamefiles`.
//
// SON NOM NE FINIT PAS PAR `_gamefiles_test.go`, ET C'EST LE POINT : ce fichier doit tourner
// dans le build PAR DEFAUT pour surveiller les autres. Se nommer comme eux le ferait
// s'attraper lui-meme (constate en l'ecrivant).
//
// CE QU'IL PAIE. Le 2026-09-05, `go test ./internal/himap/` ne terminait plus sur un poste ou
// Halo Infinite est installe : le paquet melange deux populations de tests sans rien pour les
// separer. Mesure du jour, sur `feat/v75` nu :
//
//	TestBalayageCoquille seul ...... 1 246 s (20 min 47 s), 26 cartes cuites, VERT
//	  la plus chere (behemoth) ......... 86 s
//	  aquarius_map ..................... 59 s
//
// Ce n'etait donc ni une boucle infinie, ni une carte manquante, ni un blocage sur une
// ressource : c'est un balayage de recherche qui coute vingt minutes — et ce n'est qu'UN des
// 59 fichiers `*_gamefiles_test.go` du paquet. Les timeouts de 2, 10 et 15 min tombaient tous
// a l'interieur de sa duree legitime.
//
// L'ASYMETRIE QUI L'A CACHE. En CI, le jeu n'est pas installe : `DeployRoot()` echoue, chaque
// test gamefiles prend son `t.Skip` et le paquet est vert en une seconde. Le cout n'existe que
// sur un poste de developpement — le seul endroit ou personne ne regarde un tableau de bord.
//
// LA REGLE, ET POURQUOI ELLE A BESOIN D'UN GARDE-RAIL. Tout test qui lit l'installation du jeu
// vit derriere `//go:build gamefiles`. Un tag pose a la main sur 59 fichiers se re-perd au
// premier fichier ajoute sans lui — d'ou ce test, qui tourne dans le build PAR DEFAUT (donc en
// CI) et lit les fichiers sur disque plutot que de dependre de leur compilation.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tagGamefiles : la ligne exacte attendue en tete de chaque fichier du corpus.
const tagGamefiles = "//go:build gamefiles"

// appelsInstallation : les portes d'entree vers l'installation de Halo Infinite. Un test qui en
// appelle une lit le disque du jeu, donc coute des minutes des que le jeu est la.
var appelsInstallation = []string{"DeployRoot(", "LevelsDir(", "ChercheModuleInstalle("}

// horsCorpusAutorises : tests non tagues qui citent une porte d'entree SANS lire l'installation
// reelle. Allowlist a une entree, justifiee — l'agrandir demande la meme demonstration.
//
//   - deploy_root_test.go (2026-09-05) : il appelle bien `DeployRoot()` et `LevelsDir()`, mais
//     toujours sous `t.Setenv(DeployRootEnv, t.TempDir())` — il teste la RESOLUTION du chemin,
//     jamais le contenu du jeu. Duree mesuree : < 0,01 s.
//   - corpus_tag_test.go (2026-09-05) : ce fichier-ci. Il ne les APPELLE pas, il les CITE dans
//     `appelsInstallation` — meme auto-reference que le garde-rail voisin
//     `TestAucuneAutreCopieDuCheminDuJeu` (deploy_root_test.go), qui s'allowliste lui aussi
//     parce qu'il cite les motifs qu'il interdit.
var horsCorpusAutorises = map[string]bool{
	"deploy_root_test.go": true,
	"corpus_tag_test.go":  true,
}

// TestCorpusGamefilesEstTague — chaque `*_gamefiles_test.go` porte le tag.
//
// Mutation qui doit le faire rougir : ajouter un `*_gamefiles_test.go` sans sa premiere ligne.
func TestCorpusGamefilesEstTague(t *testing.T) {
	fichiers, err := filepath.Glob("*_gamefiles_test.go")
	if err != nil {
		t.Fatalf("parcours du corpus : %v", err)
	}
	if len(fichiers) == 0 {
		t.Fatal("aucun *_gamefiles_test.go trouve — le garde-rail ne garde plus rien " +
			"(reference du 2026-09-05 : 59 fichiers)")
	}
	for _, f := range fichiers {
		buf, err := os.ReadFile(f) //nolint:gosec // chemin de test, lecture seule
		if err != nil {
			t.Fatalf("%s : %v", f, err)
		}
		if !strings.HasPrefix(string(buf), tagGamefiles+"\n") {
			t.Errorf("%s ne commence pas par %q — sans ce tag il tourne dans le build par "+
				"defaut et y ajoute des minutes des que le jeu est installe", f, tagGamefiles)
		}
	}
}

// TestAucunTestNonTagueNeLitLInstallation — l'autre sens de la meme regle.
//
// Taguer le corpus ne sert a rien si un test ORDINAIRE se remet a ouvrir l'installation : le
// paquet par defaut redeviendrait interminable sans qu'aucun fichier ne s'appelle
// « gamefiles ».
//
// Mutation qui doit le faire rougir : appeler `DeployRoot()` depuis un `*_test.go` non tague
// hors allowlist.
func TestAucunTestNonTagueNeLitLInstallation(t *testing.T) {
	fichiers, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatalf("parcours des tests : %v", err)
	}
	for _, f := range fichiers {
		if strings.HasSuffix(f, "_gamefiles_test.go") || horsCorpusAutorises[f] {
			continue
		}
		buf, err := os.ReadFile(f) //nolint:gosec // chemin de test, lecture seule
		if err != nil {
			t.Fatalf("%s : %v", f, err)
		}
		texte := string(buf)
		if strings.HasPrefix(texte, tagGamefiles+"\n") {
			continue // tague sans porter le suffixe : la regle est respectee au fond
		}
		for _, appel := range appelsInstallation {
			if strings.Contains(texte, appel) {
				t.Errorf("%s appelle %s hors du tag %q — renommer en *_gamefiles_test.go, "+
					"ou justifier une entree d'allowlist comme deploy_root_test.go",
					f, appel, tagGamefiles)
			}
		}
	}
}
