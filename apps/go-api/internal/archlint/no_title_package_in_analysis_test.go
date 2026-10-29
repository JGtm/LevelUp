// no_title_package_in_analysis_test.go — `internal/analysis/` N'IMPORTE PAS `internal/games/{slug}/`.
//
// # POURQUOI CE GARDE-RAIL, ET POURQUOI MAINTENANT (2026-09-12)
//
// ADR 0012 pose la frontière : le code SPÉCIFIQUE à un titre vit sous `internal/games/{slug}/`,
// et `internal/analysis/` ne porte que des algorithmes title-agnostic (ADR 0025). La frontière
// n'avait aucun garde-rail GÉNÉRAL : seul `no_temporal_title_import_test.go` la tenait, et
// uniquement pour `internal/analysis/temporal`. Tout le reste d'`analysis/` pouvait donc
// importer un paquet de titre sans que rien ne rougisse.
//
// Ce test est posé AVANT le déplacement du décodeur de film (`grammar`, `replay`) d'
// `internal/analysis/` vers `libs/film/` (lot E du plan
// `.ai/archive/V7.5/PLAN_FORK_ET_RELEASE_2026-09-11.md`). Sans lui, le déplacement transformerait des
// imports internes à `analysis/` en franchissements de frontière INVISIBLES : c'est
// exactement la dette que ce lot doit rendre visible, pas enfouir.
//
// # CE QU'IL VÉRIFIE, ET COMMENT
//
// Il PARSE les imports (go/parser, ImportsOnly) de tous les `.go` d'`internal/analysis/`,
// tests compris — un test de recherche qui ouvre un paquet de titre franchit la frontière
// aussi sûrement qu'un fichier de production, et c'est par les tests que la porte s'est
// ouverte ici. Un grep se ferait tromper par les chemins cités en commentaire, qui abondent
// dans ces paquets.
//
// Un import est une VIOLATION quand il vise `levelup/go-api/internal/games/<dir>/...` où
// `<dir>` n'est PAS un paquet inter-titres déclaré ci-dessous (`paquetsInterTitres`). Les
// paquets inter-titres (`canonical`, `mappings`, `weapons`, `classification`) sont la lingua
// franca : les importer ne couple à AUCUN titre. Tout autre répertoire d'`internal/games/`
// est un titre (`halo_infinite`, `halo_5`, `synthetic_title_b`) — un répertoire neuf y est
// donc traité comme un titre par DÉFAUT, ce qui est le bon sens de la faute : ajouter un
// paquet inter-titres demande une ligne ici, ajouter un titre n'en demande aucune.
//
// # MUTATION QUI DOIT LE FAIRE ROUGIR
//
// Ajouter `"levelup/go-api/internal/games/halo_infinite/rankedplaylists"` à n'importe quel
// fichier d'`internal/analysis/` hors allowlist.
//
// # LE DÉCODEUR DE FILM, DIRECT ET TRANSITIF (2026-10-29, plan D2 lot 1, décision D2-G c)
//
// Le décodeur de film est une brique (ligne `film` de `briques_imports_test.go`) : `analysis/` n'en
// importe aucun paquet, directement (même parcours, tests compris, SANS allowlist) ni par la
// fermeture (`TestFermetureDeAnalysisSansDecodeurDeFilm`, `go list -deps -test`, allowlist datée
// des fuites qui ne peut que descendre). Mutation : ajouter à un fichier d'`analysis/` un import
// de `github.com/JGtm/LevelUp/libs/film/types` (direct), ou faire importer
// `games/weapons/filmshell` par un paquet que la fermeture d'`analysis` contient (transitif).
package archlint

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// prefixeGames : le préfixe des paquets par titre et inter-titres.
const prefixeGames = "levelup/go-api/internal/games/"

// paquetsInterTitres : les répertoires d'`internal/games/` qui ne sont PAS des titres. Les
// importer depuis `analysis/` est légitime : ils ne portent aucune logique d'un titre donné
// (types canoniques, chargeurs de mappings versionnés, table des armes, classifieur).
//
// Cette liste est vérifiée à chaque exécution (chaque nom doit encore exister sous
// `internal/games/`) : un paquet inter-titres supprimé ou renommé le dit ici.
var paquetsInterTitres = map[string]bool{
	"canonical":      true,
	"classification": true,
	"mappings":       true,
	"weapons":        true,
}

// plancherFichiersAnalysis : le nombre minimal de `.go` que le parcours doit voir sous
// `internal/analysis/`. Mesuré le 2026-09-12 : 1 337 avant le déplacement du décodeur,
// 399 après. Le plancher est posé à 300 (75 % de l'état d'après), assez serré pour qu'un
// parcours cassé échoue, assez lâche pour ne pas devenir un compteur à maintenir. Un ratchet
// qui ne scanne rien passe en silence, ce qui est pire que pas de ratchet.
//
// RE-MESURE DU 2026-09-16 (lot 2.5.d.2) : 427 fichiers avant la descente d'`objectiveevents`
// sous `film/facts/objectives`, 365 après. Le plancher tient (le lot 2.5 en retirera encore
// `weaponv3` et `source`, soit ~351 à sa clôture) ; il est ré-examiné à chaque descente.
const plancherFichiersAnalysis = 300

// franchissementsToleres : les fichiers d'`internal/analysis/` qui importent encore un paquet
// de titre, par chemin relatif à `apps/go-api/`, avec la DATE d'inscription, le paquet visé et
// la RAISON pour laquelle le portage n'est pas fait ici. C'est de la dette RENDUE VISIBLE, pas
// un blanc-seing : chaque ligne décrit le portage qui reste à faire.
//
// Une entrée qui ne correspond plus à aucune violation fait rougir ce test (une exemption qui
// survit à son site finit par en couvrir un autre).
var franchissementsToleres = map[string]string{
	// RETIRÉES LE 2026-09-16 (lot 2.5.d.2) : `internal/analysis/objectiveevents/` —
	// `assaut_footer_research_test.go` et `extract_test.go`. Le portage attendu n'a pas eu lieu
	// (la source du film n'est toujours pas un paramètre de ces tests) : c'est le PAQUET qui a
	// quitté `internal/analysis/`, descendu sous
	// `libs/film/internal/facts/objectives` — il est désormais une couche du
	// décodeur, chez lui, et ouvrir un film du cache local y est légitime. La dette décrite par
	// ces deux entrées disparaît donc avec sa cause, et non par contournement.
	// RETIRÉE LE 2026-09-16 (lot 2.4, item 2.4.2) : `libs/film/internal/source/source_test.go`
	// n'importe plus `grammar` : le test compare le marcheur canonique (`source.Paquets`) à une
	// COPIE DE RÉFÉRENCE de l'ancienne grammaire portée par le test lui-même (738 paquets).
	// RETIRÉE LE 2026-09-16 (item 2.5.f) : `internal/analysis/sessionusage/usage_outcomes.go`
	// était le SEUL franchissement de PRODUCTION de cette liste. Les quatre symboles d'usage
	// d'équipement qu'il lisait dans `games/halo_infinite/film/replay` vivent désormais dans
	// `internal/domain/equipmentusage` — le décodeur et l'agrégat de session les y lisent tous
	// les deux. Reste TROIS entrées, toutes des TESTS.
	// RETIRÉE LE 2026-09-16 (lot 2.5.e) : `internal/analysis/weapon_index_equivalence_test.go`
	// était la DERNIÈRE. Le portage annoncé a eu lieu tel qu'il était écrit — « la comparaison
	// descend côté décodeur » : le test vit désormais dans
	// `libs/film/internal/grammar/weaponscan/`, à côté du scanner dont il prouve
	// la clé de tireur, et le catalogue d'armes qu'il opposait est sorti d'`internal/analysis`
	// pour `games/weapons/filmshell`.
	//
	// LA TABLE EST VIDE, ET C'EST UN RATCHET : `internal/analysis/` n'importe plus AUCUN paquet
	// de titre, ni en production ni en test. Toute entrée neuve doit se justifier par écrit et
	// nommer son lot de retrait — ce n'est plus une liste de dette à vider, c'est une exception
	// à décider.
}

func TestAnalysisImporteAucunPaquetDeTitre(t *testing.T) {
	racineAPI := apiRootDepuisIci(t)
	racineGames := filepath.Join(racineAPI, "internal", "games")

	for nom := range paquetsInterTitres {
		if _, err := os.Stat(filepath.Join(racineGames, nom)); err != nil {
			t.Errorf("paquetsInterTitres cite %q, absent d'internal/games/ : %v — la liste "+
				"décrit une arborescence qui n'existe plus", nom, err)
		}
	}

	racineAnalysis := filepath.Join(racineAPI, "internal", "analysis")
	fset := token.NewFileSet()
	var fichiers int
	var violations []string
	vus := map[string]bool{}

	err := filepath.WalkDir(racineAnalysis, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(chemin, ".go") {
			return nil
		}
		fichiers++
		f, perr := parser.ParseFile(fset, chemin, nil, parser.ImportsOnly)
		if perr != nil {
			return perr
		}
		rel, _ := filepath.Rel(racineAPI, chemin)
		rel = filepath.ToSlash(rel)
		for _, imp := range f.Imports {
			paquet := strings.Trim(imp.Path.Value, `"`)
			if estPaquetDeLaBriqueFilm(paquet) {
				violations = append(violations, rel+"  -> "+paquet+"  (brique du décodeur de film, aucune tolérance)")
				continue
			}
			titre, ok := titreDuPaquetGames(paquet)
			if !ok {
				continue
			}
			vus[rel] = true
			if motif, tolere := franchissementsToleres[rel]; tolere {
				if strings.TrimSpace(motif) == "" {
					violations = append(violations, rel+"  (toléré SANS justification datée)")
				}
				continue
			}
			violations = append(violations, rel+"  -> "+paquet+"  (titre "+titre+")")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours d'internal/analysis : %v", err)
	}

	if fichiers < plancherFichiersAnalysis {
		t.Fatalf("%d fichiers .go parcourus sous internal/analysis, au moins %d attendus : le "+
			"parcours ne voit plus l'arborescence (déplacement de paquets, filtre cassé)",
			fichiers, plancherFichiersAnalysis)
	}
	for cle := range franchissementsToleres {
		if !vus[cle] {
			t.Errorf("franchissementsToleres cite %q, qui n'importe plus aucun paquet de titre "+
				"— retirer l'entrée", cle)
		}
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Errorf("internal/analysis/ importe un paquet de titre (ADR 0012 / ADR 0025) : les "+
			"algorithmes d'analyse doivent rester title-agnostic et recevoir les données du "+
			"titre par paramètre ou par adapter. Poser le code spécifique sous "+
			"internal/games/{slug}/ :\n  %s", strings.Join(violations, "\n  "))
	}
}

// titreDuPaquetGames rend le nom du titre quand le chemin d'import vise un paquet de titre.
func titreDuPaquetGames(paquet string) (string, bool) {
	if !strings.HasPrefix(paquet, prefixeGames) {
		return "", false
	}
	reste := paquet[len(prefixeGames):]
	premier := reste
	if i := strings.IndexByte(reste, '/'); i >= 0 {
		premier = reste[:i]
	}
	if premier == "" || paquetsInterTitres[premier] {
		return "", false
	}
	return premier, true
}

// racinesDeLaBriqueFilm rend les chemins d'import racines de la brique du décodeur de film : ceux
// de la ligne `film` de la table des briques (le module `libs/film`).
func racinesDeLaBriqueFilm() []string {
	var racines []string
	for _, b := range briques {
		if b.nom != "film" {
			continue
		}
		for _, p := range b.paquets {
			racines = append(racines, cheminImport(p))
		}
	}
	return racines
}

// estPaquetDeLaBriqueFilm dit si le chemin d'import est une racine de la brique du décodeur ou un
// paquet dessous (frontière de chemin : `filmprofile` n'est pas sous `film`).
func estPaquetDeLaBriqueFilm(paquet string) bool {
	for _, r := range racinesDeLaBriqueFilm() {
		if paquet == r || strings.HasPrefix(paquet, r+"/") {
			return true
		}
	}
	return false
}

// fuitesTransitivesAnalysis : les paquets de la brique du décodeur de film que la fermeture
// (`go list -deps -test`) d'`internal/analysis/...` contient malgré l'interdit direct, avec la
// DATE d'inscription, le chemin d'arrivée et le LOT du plan D2 qui coupe la fuite. Ne peut que
// descendre : une entrée que la fermeture ne contient plus fait rougir le test.
//
// VIDE LE 2026-10-29 (mesure du lot 1 sur `a5bae50b4`) : la fermeture d'`analysis` compte 312
// lignes (paquets et variantes de test), aucune du décodeur ; `filmshell` et `equipenergy` n'y
// sont pas non plus. Re-mesuré le 2026-10-29 à l'item 2.5 (`filmshell` sous `film/filmshell`,
// importé par `games/weapons`) : `games/weapons` n'est pas dans la fermeture d'`analysis`, donc
// ni lui ni `filmshell` n'y entrent ; la table reste vide. Si une fuite apparaît, elle s'inscrit
// ICI, datée, avec le lot qui la coupe, pas avant.
var fuitesTransitivesAnalysis = map[string]string{}

// paquetsDeFermeture extrait les chemins d'import d'une sortie `go list -deps -test -f
// {{.ImportPath}}` : les variantes de test (`p [q.test]`) et les binaires de test (`q.test`)
// ramenés au paquet qu'ils exercent.
func paquetsDeFermeture(sortie string) []string {
	vus := map[string]bool{}
	var out []string
	for _, ligne := range strings.Split(sortie, "\n") {
		p := strings.TrimSpace(ligne)
		if i := strings.IndexByte(p, ' '); i >= 0 {
			p = p[:i]
		}
		p = strings.TrimSuffix(p, ".test")
		if p == "" || vus[p] {
			continue
		}
		vus[p] = true
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// fuitesDeFermeture rend, triés, les paquets de la brique du décodeur présents dans la fermeture et
// absents de `tolerees`, et les entrées de `tolerees` que la fermeture ne contient plus.
func fuitesDeFermeture(paquets []string, tolerees map[string]string) (fuites, mortes []string) {
	dans := map[string]bool{}
	for _, p := range paquets {
		if !estPaquetDeLaBriqueFilm(p) {
			continue
		}
		dans[p] = true
		if _, ok := tolerees[p]; !ok {
			fuites = append(fuites, p)
		}
	}
	for p := range tolerees {
		if !dans[p] {
			mortes = append(mortes, p)
		}
	}
	sort.Strings(fuites)
	sort.Strings(mortes)
	return fuites, mortes
}

func contientChaine(liste []string, cible string) bool {
	for _, s := range liste {
		if s == cible {
			return true
		}
	}
	return false
}

// TestFermetureDeAnalysisSansDecodeurDeFilm : la fermeture de dépendances d'`internal/analysis/...`
// (production et tests) ne contient aucun paquet du décodeur de film, hors `fuitesTransitivesAnalysis`.
// Un import indirect (analysis -> A -> décodeur) échappe au parcours des imports directs : seul
// `go list` voit le chemin.
func TestFermetureDeAnalysisSansDecodeurDeFilm(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("go introuvable dans le PATH : %v", err)
	}
	cmd := exec.Command(goBin, "list", "-deps", "-test", "-f", "{{.ImportPath}}", "./internal/analysis/...")
	cmd.Dir = apiRootDepuisIci(t)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	brut, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -deps -test ./internal/analysis/... : %v\n%s", err, stderr.String())
	}
	paquets := paquetsDeFermeture(string(brut))
	if len(paquets) < 100 || !contientChaine(paquets, "levelup/go-api/internal/analysis") {
		t.Fatalf("la fermeture lue compte %d paquets et ne contient pas internal/analysis : la sortie de go list "+
			"n'est plus celle qu'on parse", len(paquets))
	}
	fuites, mortes := fuitesDeFermeture(paquets, fuitesTransitivesAnalysis)
	for p, motif := range fuitesTransitivesAnalysis {
		if strings.TrimSpace(motif) == "" {
			t.Errorf("fuitesTransitivesAnalysis : %q sans justification datée", p)
		}
	}
	for _, p := range mortes {
		t.Errorf("fuitesTransitivesAnalysis cite %q, que la fermeture d'internal/analysis ne contient plus — retirer l'entrée", p)
	}
	if len(fuites) > 0 {
		t.Errorf("la fermeture d'internal/analysis/... contient des paquets du décodeur de film (audit découplage, "+
			"constat G ; `go list -deps -test ./internal/analysis/...` pour retrouver le chemin) : couper l'import "+
			"qui les tire, ou inscrire la fuite datée dans fuitesTransitivesAnalysis avec le lot qui la coupe :\n  %s",
			strings.Join(fuites, "\n  "))
	}
}

// TestFermetureDeAnalysis_AutoPreuve : le contrôle pur lit les variantes de test, rougit sur un
// paquet du décodeur (tous les chemins de la brique, `libs/film` compris) absent de l'allowlist,
// laisse passer le voisin qui partage seulement le début du nom, honore l'allowlist et signale
// une entrée morte.
func TestFermetureDeAnalysis_AutoPreuve(t *testing.T) {
	sortie := "levelup/go-api/internal/analysis\n" +
		"levelup/go-api/internal/games/weapons [levelup/go-api/internal/games/weapons.test]\n" +
		"github.com/JGtm/LevelUp/libs/film/types\n" +
		"levelup/go-api/internal/games/halo_infinite/filmprofile\n" +
		"github.com/JGtm/LevelUp/libs/film/filmshell\n" +
		"github.com/JGtm/LevelUp/libs/film/replay\n" +
		"levelup/go-api/internal/analysis.test\n"
	paquets := paquetsDeFermeture(sortie)
	if !contientChaine(paquets, "levelup/go-api/internal/games/weapons") || contientChaine(paquets, "levelup/go-api/internal/analysis.test") {
		t.Fatalf("variantes de test mal ramenées : %v", paquets)
	}
	fuites, mortes := fuitesDeFermeture(paquets, nil)
	attendu := "github.com/JGtm/LevelUp/libs/film/filmshell,github.com/JGtm/LevelUp/libs/film/replay,github.com/JGtm/LevelUp/libs/film/types"
	if strings.Join(fuites, ",") != attendu || len(mortes) != 0 {
		t.Fatalf("fuites = %v, mortes = %v ; attendu %s et aucune morte", fuites, mortes, attendu)
	}
	fuites, mortes = fuitesDeFermeture(paquets, map[string]string{
		"github.com/JGtm/LevelUp/libs/film/filmshell": "2026-10-29 test",
		"github.com/JGtm/LevelUp/libs/film/types":     "2026-10-29 test",
		"github.com/JGtm/LevelUp/libs/film/replay":    "2026-10-29 test",
		"github.com/JGtm/LevelUp/libs/film/absent":    "2026-10-29 test",
	})
	if len(fuites) != 0 || len(mortes) != 1 || !strings.HasSuffix(mortes[0], "film/absent") {
		t.Fatalf("allowlist honorée et entrée morte signalée attendues ; fuites = %v, mortes = %v", fuites, mortes)
	}
	for imp, attendu := range map[string]bool{
		"github.com/JGtm/LevelUp/libs/film/replay":                true,
		"github.com/JGtm/LevelUp/libs/film/filmshell":             true,
		"github.com/JGtm/LevelUp/libs/film/equipenergy":           true,
		"github.com/JGtm/LevelUp/libs/film/decfilm":               true,
		"levelup/go-api/internal/games/halo_infinite/film":        false,
		"levelup/go-api/internal/games/halo_infinite/filmprofile": false,
		"levelup/go-api/internal/games/weapons":                   false,
	} {
		if got := estPaquetDeLaBriqueFilm(imp); got != attendu {
			t.Errorf("estPaquetDeLaBriqueFilm(%q) = %v, attendu %v", imp, got, attendu)
		}
	}
}
