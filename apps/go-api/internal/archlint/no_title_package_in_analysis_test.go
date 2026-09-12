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
// Ce test est posé AVANT le déplacement du décodeur de film (`filmdec`, `replay`) d'
// `internal/analysis/` vers `internal/games/halo_infinite/film/` (lot E du plan
// `.ai/PLAN_FORK_ET_RELEASE_2026-09-11.md`). Sans lui, le déplacement transformerait des
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
package archlint

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
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
const plancherFichiersAnalysis = 300

// franchissementsToleres : les fichiers d'`internal/analysis/` qui importent encore un paquet
// de titre, par chemin relatif à `apps/go-api/`, avec la DATE d'inscription, le paquet visé et
// la RAISON pour laquelle le portage n'est pas fait ici. C'est de la dette RENDUE VISIBLE, pas
// un blanc-seing : chaque ligne décrit le portage qui reste à faire.
//
// Une entrée qui ne correspond plus à aucune violation fait rougir ce test (une exemption qui
// survit à son site finit par en couvrir un autre).
var franchissementsToleres = map[string]string{
	"internal/analysis/objectiveevents/assaut_footer_research_test.go": "2026-09-12 — " +
		"`games/halo_infinite/film/filmcache` : test de RECHERCHE (pied de paquet du mode " +
		"Assaut) qui ouvre des films réels du cache local. La dépendance est au CACHE DE " +
		"FILMS d'un titre, pas à l'algorithme : le portage consiste à faire passer le film " +
		"par un paramètre (fixture ou interface de source), comme le fait déjà " +
		"`analysis/filmsource`. Hors périmètre du lot E (déplacement pur).",
	"internal/analysis/objectiveevents/extract_test.go": "2026-09-12 — " +
		"`games/halo_infinite/film/filmcache` : même motif que ci-dessus (extraction des " +
		"événements d'objectif vérifiée sur films réels). Même portage attendu : la source " +
		"du film devient un paramètre du test.",
	"internal/analysis/filmsource/source_test.go": "2026-09-12, rendu visible par le " +
		"déplacement du décodeur (commit E.2) — `games/halo_infinite/film/filmdec` : le test " +
		"EXTERNE de `filmsource` compare les deux marcheurs de paquets sur un film réel " +
		"(preuve d'équivalence de la grammaire, cf. `filmsource_leaf_test.go`). Le paquet " +
		"testé, lui, reste une feuille sans aucun import du dépôt : c'est le TEST qui " +
		"franchit. Portage attendu : la preuve d'équivalence descend avec le décodeur, sous " +
		"`games/halo_infinite/film/`.",
	"internal/analysis/sessionusage/usage_outcomes.go": "2026-09-12, rendu visible par le " +
		"déplacement du décodeur (commit E.2) — `games/halo_infinite/film/replay` : SEUL " +
		"franchissement de PRODUCTION de la liste. `sessionusage` lit les types de sortie " +
		"d'usage d'équipement produits par le décodeur. Portage attendu : ces types " +
		"remontent en `domain/` (ou `games/canonical/`), comme `domain/replaydoc` l'a déjà " +
		"fait pour le document de rejeu au lot A — après quoi `sessionusage` n'importera " +
		"plus rien d'un titre.",
	"internal/analysis/weapon_index_equivalence_test.go": "2026-09-12, rendu visible par le " +
		"déplacement du décodeur (commit E.2) — `games/halo_infinite/film/filmdec` : test " +
		"d'équivalence entre l'index d'armes d'`analysis` et celui du décodeur. Portage " +
		"attendu : l'index d'armes est title-agnostic (`games/weapons`), la comparaison " +
		"descend côté décodeur.",
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
