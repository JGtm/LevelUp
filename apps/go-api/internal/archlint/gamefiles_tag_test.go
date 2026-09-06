package archlint

// gamefiles_tag_test.go — LE GARDE-RAIL DU TAG `gamefiles`, SUR TOUT LE MODULE.
//
// PROMU DEPUIS `internal/himap/corpus_tag_test.go` LE 2026-09-05 (registre d'audit, constat
// I1). L'original ne regardait qu'UN paquet, par `filepath.Glob("*_test.go")` dans son propre
// répertoire : deux tests hors de `himap` ouvraient l'installation du jeu sans tag, et rien ne
// les voyait — `cmd/mapstruct-build/equivalence_gamefiles_test.go` (nommé « gamefiles » mais
// SANS la ligne de tag, donc compilé par défaut) et `cmd/mapfond-build/reglages_test.go`
// (`TestModuleGeometrieExisteDansLInstallation`, depuis isolé sous son propre fichier tagué).
// Ce fichier-ci balaie LA RACINE DU MODULE, et remplace l'original. La première version de ce
// balayage nommait deux racines (`internal`, `cmd`) dans une constante alors que l'en-tête
// promettait déjà « tout le module » : la revue F-R1-2 du 2026-09-06 a montré qu'un `_test.go`
// non tagué sous `tests/` passait vert. La liste des racines est désormais DÉRIVÉE du système de
// fichiers (cf. `balayerTests`), pas écrite.
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

// corpusGamefilesPlancher : le corpus mesuré le 2026-09-06, APRÈS cet item (60 dans
// `internal/himap`, 1 dans `cmd/mapstruct-build`, 1 dans `cmd/mapfond-build`). Un balayage qui
// rend moins que ça ne garde plus rien — le balayage s'est cassé, ou le corpus a été renommé
// sans que personne ne le dise.
const corpusGamefilesPlancher = 62

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

// dossiersInvisiblesAuGo : les répertoires que l'outil Go lui-même ignore quand il charge des
// paquets — un `_test.go` posé là n'est jamais compilé, donc jamais exécuté, donc hors sujet.
// C'est le SEUL filtre du balayage : tout le reste du module est parcouru.
//
// LA LISTE DES RACINES N'EST PAS ÉCRITE, ELLE EST DÉRIVÉE (revue F-R1-2 du 2026-09-06). La
// version précédente balayait la constante `{"internal", "cmd"}` alors que l'en-tête promettait
// « tout le module » : un `_test.go` non tagué sous `contracttest/`, `pkg/`, `scripts/` ou
// `tests/` qui ouvrait l'installation restait invisible — l'angle mort EXACT de l'ancien ratchet
// de `himap` (un `filepath.Glob` d'un seul répertoire) que ce fichier prétend fermer. Le
// balayage part désormais de la racine du module : une racine neuve est couverte le jour où
// elle apparaît, sans que personne ait à y penser.
var dossiersInvisiblesAuGo = map[string]bool{
	"testdata":     true, // ignoré par le chargeur de paquets Go
	"vendor":       true,
	"node_modules": true,
}

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

// balayerTests appelle `visiter` pour CHAQUE `*_test.go` du module, où qu'il soit, avec son
// chemin relatif à `apps/go-api` (en slash) et son contenu. Seuls sont sautés les répertoires
// que l'outil Go n'ouvre pas lui-même (cf. `dossiersInvisiblesAuGo`, plus les `.foo` et `_foo`
// que le chargeur de paquets ignore par convention).
func balayerTests(t *testing.T, visiter func(rel, texte string)) {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(ici))) // .../apps/go-api
	err := filepath.WalkDir(goAPIRoot, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			nom := d.Name()
			if chemin != goAPIRoot && (dossiersInvisiblesAuGo[nom] ||
				strings.HasPrefix(nom, ".") || strings.HasPrefix(nom, "_")) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(chemin, "_test.go") {
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
		t.Fatalf("parcours du module (%s) : %v", goAPIRoot, err)
	}
}
