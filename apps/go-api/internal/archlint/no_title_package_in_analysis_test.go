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
}

// paquetsEnDeplacement : les répertoires d'`internal/analysis/` qui QUITTENT `analysis/` et
// dont les franchissements sont donc tolérés le temps du déplacement — entrée TRANSITOIRE,
// avec sa date de pose et sa date cible de retrait.
//
//	2026-09-12, retrait cible le MÊME JOUR, au commit E.2 du plan
//	`.ai/PLAN_FORK_ET_RELEASE_2026-09-11.md` : `filmdec` et `replay` (le décodeur de film,
//	Halo-only de bout en bout) descendent sous `internal/games/halo_infinite/film/` (ADR 0012).
//	Leurs 15 tests qui ouvrent `film/filmcache`, `film/damagetag` et `film/killsource` ne
//	franchissent alors plus aucune frontière : ils sont chez eux. Ce ratchet est posé AVANT le
//	déplacement pour que le déplacement se fasse sous surveillance, pas après coup — d'où
//	cette clause, qui disparaît avec les répertoires qu'elle nomme.
//	Critère mesurable de retrait : `internal/analysis/filmdec` et `internal/analysis/replay`
//	n'existent plus (le test l'exige ci-dessous : une entrée sans répertoire fait rougir).
var paquetsEnDeplacement = map[string]string{
	"internal/analysis/filmdec": "2026-09-12, retrait au commit E.2 — descend sous " +
		"internal/games/halo_infinite/film/filmdec (ADR 0012).",
	"internal/analysis/replay": "2026-09-12, retrait au commit E.2 — descend sous " +
		"internal/games/halo_infinite/film/replay (ADR 0012).",
}

// enDeplacement dit si le fichier appartient à un répertoire en instance de déplacement.
func enDeplacement(rel string) bool {
	for prefixe := range paquetsEnDeplacement {
		if strings.HasPrefix(rel, prefixe+"/") {
			return true
		}
	}
	return false
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
	for rep, motif := range paquetsEnDeplacement {
		if strings.TrimSpace(motif) == "" {
			t.Errorf("paquetsEnDeplacement cite %q sans justification datée", rep)
		}
		if _, err := os.Stat(filepath.Join(racineAPI, filepath.FromSlash(rep))); err != nil {
			t.Errorf("paquetsEnDeplacement cite %q, qui n'existe plus : le déplacement est "+
				"fait, retirer l'entrée (c'est son critère de retrait)", rep)
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
			if enDeplacement(rel) {
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
