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
//
// INSTRUMENTS QUI OUVRENT LE JEU (plan D1, lot 2c, 2026-10-28, décision utilisateur). Les sondes de
// rétro-ingénierie des briques qui lisent `.ai/re_dump` ou les données du dépôt hôte sont
// conservées comme instruments `*_research_test.go` (jamais supprimées). Quand elles ouvrent
// aussi le jeu, elles portent `//go:build gamefiles && research` : `appelSansDeclarerGamefiles`
// l'accepte pour un fichier dont le NOM est `*_research_test.go` et dont la contrainte porte
// `gamefiles` ET `research` comme termes d'une conjonction. Ce N'EST PAS un assouplissement :
// le fichier n'entre dans aucun build automatique (il faut `-tags=gamefiles,research`, donné à
// la main), il sort de la suite `gamefiles`, et la contrainte dit ce que le nom ne dit pas.
// La règle reste « un instrument qui ouvre le jeu déclare `gamefiles` » ; un `*_research_test.go`
// sous `//go:build research` seul qui appelle l'installation reste refusé
// (`TestInstrumentQuiOuvreLeJeuDeclareGamefiles`, auto-preuve sur sources synthétiques).
// Un `*_gamefiles_test.go`, lui, garde exactement `//go:build gamefiles` : un instrument ne porte
// jamais ce suffixe.

import (
	"os"
	"path"
	"path/filepath"
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
// vivent dans `libs/himap/himap` ; hors de ce paquet elles s'écrivent `himap.DeployRoot(` etc.,
// que ces motifs attrapent aussi.
var appelsInstallation = []string{"DeployRoot(", "LevelsDir(", "ChercheModuleInstalle("}

// horsCorpusAutorises : tests NON tagués qui citent une porte d'entrée SANS lire l'installation
// réelle. Chemins relatifs à la racine du dépôt, en slash. Allowlist à deux entrées, datées —
// l'agrandir demande la même démonstration (le test ne touche pas au jeu, et sa durée le
// prouve).
//
//   - libs/himap/himap/deploy_root_test.go (2026-09-05) : il appelle bien `DeployRoot()` et
//     `LevelsDir()`, mais toujours sous `t.Setenv(DeployRootEnv, t.TempDir())` — il teste la
//     RÉSOLUTION du chemin, jamais le contenu du jeu. Durée mesurée : < 0,01 s.
//   - apps/go-api/internal/archlint/gamefiles_tag_test.go (2026-09-05) : ce fichier-ci. Il ne les APPELLE
//     pas, il les CITE dans `appelsInstallation` — même auto-référence que les autres ratchets
//     de ce paquet, qui s'allowlistent parce qu'ils portent les motifs qu'ils interdisent.
var horsCorpusAutorises = map[string]bool{
	"libs/himap/himap/deploy_root_test.go":                true,
	"apps/go-api/internal/archlint/gamefiles_tag_test.go": true,
}

// dossiersInvisiblesAuGo : les répertoires que le CHARGEUR DE PAQUETS Go n'ouvre jamais — un
// `_test.go` posé là n'est ni compilé ni exécuté par `go test ./...`, donc hors sujet pour un
// ratchet qui protège le build par défaut. C'est le SEUL filtre du balayage : tout le reste du
// module est parcouru.
//
// `node_modules` N'EN FAIT PAS PARTIE, ET C'EST MESURÉ (revue F-R2-b du 2026-09-06). Il figurait
// ici « par évidence » ; la sonde a montré que `go list ./...` liste bien
// `levelup/go-api/node_modules/zzrevue` et que `go test ./node_modules/...` l'EXÉCUTE. Le sauter
// rouvrait donc, à la marge, le trou que la correction précédente venait de fermer. Retiré : un
// `node_modules` sous un module Go serait de toute façon aberrant, et s'il apparaissait un jour
// le ratchet doit le voir comme le reste.
//
// LA LISTE DES RACINES N'EST PAS ÉCRITE, ELLE EST DÉRIVÉE (revue F-R1-2 du 2026-09-06). La
// version précédente balayait la constante `{"internal", "cmd"}` alors que l'en-tête promettait
// « tout le module » : un `_test.go` non tagué sous `contracttest/`, `pkg/`, `scripts/` ou
// `tests/` qui ouvrait l'installation restait invisible — l'angle mort EXACT de l'ancien ratchet
// de `himap` (un `filepath.Glob` d'un seul répertoire) que ce fichier prétend fermer. Le
// balayage part désormais de la racine du module : une racine neuve est couverte le jour où
// elle apparaît, sans que personne ait à y penser.
var dossiersInvisiblesAuGo = map[string]bool{
	"testdata": true, // hors de `./...` : le chargeur de paquets Go ne l'ouvre jamais
	"vendor":   true, // idem en mode module (et `-mod=vendor` compile depuis le module, pas depuis ici)
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

// appelSansDeclarerGamefiles rend la porte d'entrée vers l'installation que le fichier appelle
// sans déclarer le tag `gamefiles`, ou "" si la règle est respectée. Trois façons de la
// respecter : le suffixe `_gamefiles_test.go` (le tag exact est contrôlé ailleurs), une
// allowlist datée, ou la première ligne `//go:build gamefiles` seule.
//
// LE CAS DES INSTRUMENTS (plan D1, lot 2c, 2026-10-28, décision utilisateur) : un
// `*_research_test.go` qui OUVRE le jeu déclare `gamefiles` comme terme de conjonction
// (`//go:build gamefiles && research`, dans n'importe quel ordre). Ce n'est PAS un
// assouplissement : le fichier n'entre dans AUCUN build automatique (il faut les deux tags, donné
// à la main), et la contrainte dit ce que le nom ne dit pas, « le jeu est requis ». La règle
// reste donc « un instrument qui ouvre le jeu déclare `gamefiles` » : un `*_research_test.go`
// sous `//go:build research` seul qui appelle l'installation reste refusé, comme une disjonction
// (`gamefiles || research`) ou une négation (`research && !gamefiles`).
func appelSansDeclarerGamefiles(rel, texte string) string {
	if strings.HasSuffix(rel, "_gamefiles_test.go") || horsCorpusAutorises[rel] {
		return ""
	}
	if strings.HasPrefix(texte, tagGamefiles+"\n") {
		return "" // tagué sans porter le suffixe : la règle est respectée au fond
	}
	if strings.HasSuffix(rel, "_research_test.go") && contrainteAvecTerme(texte, "gamefiles") &&
		contrainteResearch(texte) {
		return "" // instrument qui ouvre le jeu : `gamefiles && research`
	}
	for _, appel := range appelsInstallation {
		if strings.Contains(texte, appel) {
			return appel
		}
	}
	return ""
}

// TestAucunTestNonTagueNeLitLInstallation — l'autre sens de la même règle, sur tout le module.
//
// Taguer le corpus ne sert à rien si un test ORDINAIRE se remet à ouvrir l'installation : le
// build par défaut redeviendrait interminable sans qu'aucun fichier ne s'appelle « gamefiles ».
//
// Mutation qui doit le faire rougir : appeler `himap.DeployRoot()` depuis un `*_test.go` non
// tagué hors allowlist (vérifié le 2026-09-05 en remettant
// `TestModuleGeometrieExisteDansLInstallation` dans `cmd/mapfond-build/reglages_test.go`), ou
// retirer `gamefiles &&` de la première ligne d'un `*_research_test.go` qui appelle
// `DeployRoot(` (vérifié le 2026-10-28 sur `libs/himap/himap/cloture_research_test.go`).
func TestAucunTestNonTagueNeLitLInstallation(t *testing.T) {
	balayerTests(t, func(rel, texte string) {
		if appel := appelSansDeclarerGamefiles(rel, texte); appel != "" {
			t.Errorf("%s appelle %s sans déclarer le tag %q — déplacer le test dans un "+
				"*_gamefiles_test.go tagué, déclarer `gamefiles && research` si c'est un instrument "+
				"de recherche (*_research_test.go), ou justifier une entrée d'allowlist comme "+
				"libs/himap/himap/deploy_root_test.go", rel, appel, tagGamefiles)
		}
	})
}

// TestInstrumentQuiOuvreLeJeuDeclareGamefiles — AUTO-PREUVE de la règle des instruments (lot 2c
// du plan D1, 2026-10-28) : la règle est rejouée sur des sources synthétiques, pour qu'elle
// rougisse sans qu'il faille muter un fichier du dépôt.
func TestInstrumentQuiOuvreLeJeuDeclareGamefiles(t *testing.T) {
	const corps = "\n\npackage x\n\nfunc f() { _, _ = DeployRoot() }\n"
	cas := []struct {
		nom, rel, tete string
		rouge          bool
	}{
		{"instrument avec gamefiles && research", "a/x_research_test.go", "//go:build gamefiles && research", false},
		{"instrument, termes dans l'autre ordre", "a/x_research_test.go", "//go:build research && gamefiles", false},
		{"instrument sous research seul", "a/x_research_test.go", "//go:build research", true},
		{"instrument sous disjonction", "a/x_research_test.go", "//go:build gamefiles || research", true},
		{"instrument avec gamefiles nié", "a/x_research_test.go", "//go:build research && !gamefiles", true},
		{"instrument sans aucune contrainte", "a/x_research_test.go", "// pas de contrainte", true},
		{"test ordinaire sous gamefiles && research", "a/x_test.go", "//go:build gamefiles && research", true},
		{"corpus gamefiles ordinaire", "a/x_gamefiles_test.go", "//go:build gamefiles", false},
		{"test ordinaire sous gamefiles seul", "a/x_test.go", "//go:build gamefiles", false},
	}
	for _, c := range cas {
		got := appelSansDeclarerGamefiles(c.rel, c.tete+corps) != ""
		if got != c.rouge {
			t.Errorf("%s : refusé = %v, attendu %v", c.nom, got, c.rouge)
		}
	}
}

// balayerTests appelle `visiter` pour CHAQUE `*_test.go` de l'espace de travail (`apps/go-api` et
// `libs/**`, ADR 0041), où qu'il soit, avec son chemin relatif à la RACINE DU DÉPÔT (en slash) et son
// contenu. Seuls sont sautés les répertoires que l'outil Go n'ouvre pas lui-même (cf.
// `dossiersInvisiblesAuGo`, plus les `.foo` et `_foo` que le chargeur de paquets ignore par
// convention) : le parcours est `workspaceGoSources`.
func balayerTests(t *testing.T, visiter func(rel, texte string)) {
	t.Helper()
	balayerGoTexte(t, func(rel, texte string) {
		if strings.HasSuffix(rel, "_test.go") {
			visiter(rel, texte)
		}
	})
}

// cibleMakefileGamefiles : le nom de la cible du `Makefile` qui joue le corpus.
const cibleMakefileGamefiles = "go-api-test-gamefiles:"

// TestCibleMakefileGamefilesCouvreLeCorpus — la cible `make go-api-test-gamefiles` nomme TOUS
// les paquets qui portent un `*_gamefiles_test.go`.
//
// POURQUOI (découverte D1 (3.1.2), 2026-09-16) : `TestCorpusGamefilesEstTague` garantit que ces
// tests portent le tag, donc qu'ils ne coûtent rien au build par défaut. Personne ne garantissait
// qu'ils TOURNENT : la cible ne citait que `./internal/himap/`, et les trois paquets `cmd/` tagués
// (le plus ancien depuis le 2026-09-05) n'étaient joués par AUCUNE commande du dépôt. Un tag posé
// sans entrée dans la cible, c'est un test qui existe et ne s'exécute jamais — pire qu'absent,
// puisqu'il donne l'illusion d'un gate.
//
// Mutation qui doit le faire rougir : retirer `./cmd/film-profiles-build/` de la recette.
func TestCibleMakefileGamefilesCouvreLeCorpus(t *testing.T) {
	paquets := map[string]bool{}
	balayerTests(t, func(rel, _ string) {
		if strings.HasSuffix(rel, "_gamefiles_test.go") {
			paquets[path.Dir(rel)] = true
		}
	})
	if len(paquets) == 0 {
		t.Fatal("aucun *_gamefiles_test.go balayé — le balayage s'est cassé")
	}

	recette := recetteMakefile(t, cibleMakefileGamefiles)
	for paquet := range paquets {
		cite := cheminDansLeModule(paquet)
		if !strings.Contains(recette, "./"+cite+"/") {
			t.Errorf("le paquet %s porte des *_gamefiles_test.go mais la cible %s ne le nomme pas "+
				"— ces tests ne tournent nulle part ; ajouter `./%s/` à la recette (dans la ligne "+
				"`cd <module>` de son module)", paquet, cibleMakefileGamefiles, cite)
		}
	}
}

// cheminDansLeModule rend le chemin d'un paquet de l'espace de travail relatif à la racine de SON
// module (`apps/go-api/internal/gamefilescorpus` -> `internal/gamefilescorpus`) : la forme dans laquelle une recette
// `cd <module> && go test ./<paquet>/` le nomme.
func cheminDansLeModule(paquet string) string {
	for _, racine := range racinesEspaceDeTravail {
		if reste, ok := strings.CutPrefix(paquet, racine+"/"); ok {
			return reste
		}
	}
	return paquet
}

// recetteMakefile rend le corps (lignes indentées par une tabulation, continuations comprises) de
// la cible `nomCible` du `Makefile` de la racine du dépôt.
func recetteMakefile(t *testing.T, nomCible string) string {
	t.Helper()
	chemin := filepath.Join(workspaceRoot(t), "Makefile")
	buf, err := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
	if err != nil {
		t.Fatalf("lecture du Makefile (%s) : %v", chemin, err)
	}
	var corps []string
	dedans := false
	for _, ligne := range strings.Split(string(buf), "\n") {
		switch {
		case strings.HasPrefix(ligne, nomCible):
			dedans = true
		case dedans && strings.HasPrefix(ligne, "\t"):
			corps = append(corps, ligne)
		case dedans:
			dedans = false
		}
	}
	if len(corps) == 0 {
		t.Fatalf("cible %s introuvable (ou recette vide) dans %s", nomCible, chemin)
	}
	return strings.Join(corps, "\n")
}
