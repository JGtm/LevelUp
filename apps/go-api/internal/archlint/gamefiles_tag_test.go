package archlint

// gamefiles_tag_test.go — LE GARDE-RAIL DU TAG `gamefiles`, SUR TOUT LE MODULE.
//
// PROMU DEPUIS `internal/himap/corpus_tag_test.go` LE 2026-09-05 (registre d'audit, constat
// I1). L'original ne regardait qu'UN paquet, par `filepath.Glob("*_test.go")` dans son propre
// répertoire : deux tests hors de `himap` ouvraient l'installation du jeu sans tag, et rien ne
// les voyait — `cmd/mapstruct-build/equivalence_gamefiles_test.go` (nommé « gamefiles » mais
// SANS la ligne de tag, donc compilé par défaut) et `cmd/mapfond-build/reglages_test.go`
// (`TestModuleGeometrieExisteDansLInstallation`, depuis isolé sous son propre fichier tagué).
// Ce fichier-ci balaie `internal/` ET `cmd/`, et remplace l'original.
//
// CE QUE LE TAG PAIE. Le 2026-09-05, `go test ./internal/himap/` ne terminait plus sur un poste
// où Halo Infinite est installé : le paquet mélangeait deux populations de tests sans rien pour
// les séparer. Mesure du jour, sur `feat/v75` nu :
//
//	TestBalayageCoquille seul ...... 1 246 s (20 min 47 s), 26 cartes cuites, VERT
//	  (203 s depuis le 2026-09-05 : le lecteur de modules PROJETTE au lieu de copier)
//	  la plus chère (behemoth) ......... 86 s
//	  aquarius_map ..................... 59 s
//
// Ce n'était donc ni une boucle infinie, ni une carte manquante, ni un blocage sur une
// ressource : c'est un balayage de recherche qui coûte vingt minutes — et ce n'est qu'UN des
// 61 fichiers `*_gamefiles_test.go` du module. Les timeouts de 2, 10 et 15 min tombaient tous à
// l'intérieur de sa durée légitime.
//
// L'ASYMÉTRIE QUI L'A CACHÉ, et qui vaut pour tout le module. En CI, le jeu n'est pas installé :
// `DeployRoot()` échoue, chaque test gamefiles prend son `t.Skip` et le paquet est vert en une
// seconde. Le coût n'existe que sur un poste de développement — le seul endroit où personne ne
// regarde un tableau de bord. Un tag posé à la main sur 61 fichiers se re-perd au premier
// fichier ajouté sans lui, d'où ces deux tests, qui tournent dans le build PAR DÉFAUT (donc en
// CI) et lisent les fichiers sur disque plutôt que de dépendre de leur compilation.

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// tagGamefiles : la ligne exacte attendue en tête de chaque fichier du corpus.
const tagGamefiles = "//go:build gamefiles"

// corpusGamefilesPlancher : le corpus mesuré le 2026-09-05 (60 dans `internal/himap`, 1 dans
// `cmd/mapstruct-build`). Un balayage qui rend moins que ça ne garde plus rien — le glob s'est
// cassé, ou le corpus a été renommé sans que personne ne le dise.
const corpusGamefilesPlancher = 61

// appelsInstallation : les portes d'entrée vers l'installation de Halo Infinite. Un test qui en
// appelle une lit le disque du jeu, donc coûte des minutes dès que le jeu est là. Les trois
// vivent dans `internal/himap` ; hors de ce paquet elles s'écrivent `himap.DeployRoot(` etc.,
// que ces motifs attrapent aussi.
var appelsInstallation = []string{"DeployRoot(", "LevelsDir(", "ChercheModuleInstalle("}

// horsCorpusAutorises : tests NON tagués qui citent une porte d'entrée SANS lire l'installation
// réelle. Chemins relatifs à `apps/go-api`, en slash. Allowlist à deux entrées, datées —
// l'agrandir demande la même démonstration (le test ne touche pas au jeu, et sa durée le
// prouve).
//
//   - internal/himap/deploy_root_test.go (2026-09-05) : il appelle bien `DeployRoot()` et
//     `LevelsDir()`, mais toujours sous `t.Setenv(DeployRootEnv, t.TempDir())` — il teste la
//     RÉSOLUTION du chemin, jamais le contenu du jeu. Durée mesurée : < 0,01 s.
//   - internal/archlint/gamefiles_tag_test.go (2026-09-05) : ce fichier-ci. Il ne les APPELLE
//     pas, il les CITE dans `appelsInstallation` — même auto-référence que les autres ratchets
//     de ce paquet, qui s'allowlistent parce qu'ils portent les motifs qu'ils interdisent.
var horsCorpusAutorises = map[string]bool{
	"internal/himap/deploy_root_test.go":      true,
	"internal/archlint/gamefiles_tag_test.go": true,
}

// racinesBalayees : les deux arbres de code du module.
var racinesBalayees = []string{"internal", "cmd"}

// TestCorpusGamefilesEstTague — chaque `*_gamefiles_test.go` du MODULE porte le tag.
//
// Mutation qui doit le faire rougir : retirer la première ligne d'un `*_gamefiles_test.go`
// (vérifié le 2026-09-05 sur `cmd/mapstruct-build/equivalence_gamefiles_test.go`).
func TestCorpusGamefilesEstTague(t *testing.T) {
	vus := 0
	balayerTests(t, func(rel, texte string) {
		if !strings.HasSuffix(rel, "_gamefiles_test.go") {
			return
		}
		vus++
		if !strings.HasPrefix(texte, tagGamefiles+"\n") {
			t.Errorf("%s ne commence pas par %q — sans ce tag il tourne dans le build par "+
				"défaut et y ajoute des minutes dès que le jeu est installé", rel, tagGamefiles)
		}
	})
	if vus < corpusGamefilesPlancher {
		t.Errorf("%d fichier(s) *_gamefiles_test.go balayé(s), plancher %d (mesure du 2026-09-05) "+
			"— le garde-rail ne garde plus le corpus entier", vus, corpusGamefilesPlancher)
	}
}

// TestAucunTestNonTagueNeLitLInstallation — l'autre sens de la même règle, sur tout le module.
//
// Taguer le corpus ne sert à rien si un test ORDINAIRE se remet à ouvrir l'installation : le
// build par défaut redeviendrait interminable sans qu'aucun fichier ne s'appelle « gamefiles ».
//
// Mutation qui doit le faire rougir : appeler `himap.DeployRoot()` depuis un `*_test.go` non
// tagué hors allowlist (vérifié le 2026-09-05 en remettant
// `TestModuleGeometrieExisteDansLInstallation` dans `cmd/mapfond-build/reglages_test.go`).
func TestAucunTestNonTagueNeLitLInstallation(t *testing.T) {
	balayerTests(t, func(rel, texte string) {
		if strings.HasSuffix(rel, "_gamefiles_test.go") || horsCorpusAutorises[rel] {
			return
		}
		if strings.HasPrefix(texte, tagGamefiles+"\n") {
			return // tagué sans porter le suffixe : la règle est respectée au fond
		}
		for _, appel := range appelsInstallation {
			if strings.Contains(texte, appel) {
				t.Errorf("%s appelle %s hors du tag %q — déplacer le test dans un "+
					"*_gamefiles_test.go tagué, ou justifier une entrée d'allowlist comme "+
					"internal/himap/deploy_root_test.go", rel, appel, tagGamefiles)
			}
		}
	})
}

// balayerTests appelle `visiter` pour chaque `*_test.go` de `internal/` et `cmd/`, avec son
// chemin relatif à `apps/go-api` (en slash) et son contenu.
func balayerTests(t *testing.T, visiter func(rel, texte string)) {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(ici))) // .../apps/go-api
	for _, sous := range racinesBalayees {
		racine := filepath.Join(goAPIRoot, sous)
		err := filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(chemin, "_test.go") {
				return nil
			}
			buf, rerr := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
			if rerr != nil {
				return rerr
			}
			rel, _ := filepath.Rel(goAPIRoot, chemin)
			visiter(filepath.ToSlash(rel), string(buf))
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s : %v", sous, err)
		}
	}
}
