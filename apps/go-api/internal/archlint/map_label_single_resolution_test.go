// Package archlint — map_label_single_resolution_test.go : garde-rail du LIBELLÉ D'UNE CARTE
// (CLAUDE.md règle 6, plan Tactique v2 lot L13 F1).
//
// Le libellé d'une carte a UNE règle, `libelleDeCarte` / `traductionsDeCartes`
// (`internal/platform/duckdb/map_labels.go`), lue par l'Explorateur (son historique et les options de
// son filtre), l'onglet Tactique (vignettes, lien vers l'Explorateur) et le score d'engagement. Le
// lien d'une vignette vers l'Explorateur passe ce libellé en `?maps=` : une seconde résolution
// donnerait une autre chaîne pour la même carte, et le lien ne filtrerait plus rien.
//
// Les empreintes d'une résolution, telles qu'elles s'écrivent (lignes de commentaire ignorées) :
//  1. un résolveur de noms d'asset appelé pour le type "map" ;
//  2. une requête sur `asset_translations` filtrée par `asset_type = 'map'` ;
//  3. le repli de libellé du registre `COALESCE(map_name_fr, map_name …)`.
//
// Hors du helper, seuls les fichiers de `resolutionsGelees` en portent, chacun avec sa raison : des
// surfaces hors de ce périmètre (accueil, vue match, médias, carrière, relations, escouade) et des
// lectures TECHNIQUES du nom EN (résolution du module d'une carte pour le rejeu). La liste est
// GELÉE au 2026-10-07 : elle ne grossit pas, et une entrée devenue sans empreinte doit en sortir.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// libelleCarteHelper : la source unique.
const libelleCarteHelper = "internal/platform/duckdb/map_labels.go"

// resolutionsGelees : les fichiers qui portent encore une empreinte, gelés au 2026-10-07 (plan
// Tactique v2, lot L13 F1), avec leur raison. Ne jamais y ajouter un fichier.
var resolutionsGelees = map[string]string{
	"internal/platform/duckdb/filters_repo_asset_names.go":         "pipeline de filtres : noms des titres sans nom au registre (Halo 5)",
	"internal/platform/duckdb/filters_repo_fr_cascades.go":         "pipeline de filtres : cascade FR historique par nom EN",
	"internal/platform/duckdb/home_repo_translations.go":           "accueil",
	"internal/platform/duckdb/home_repo_translations_canonical.go": "accueil (lignes canoniques)",
	"internal/platform/duckdb/match_view_repo.go":                  "vue match",
	"internal/platform/duckdb/media_repo_translations.go":          "médias",
	"internal/platform/duckdb/media_repo_filters.go":               "médias",
	"internal/platform/duckdb/media_repo_writes.go":                "médias",
	"internal/platform/duckdb/metadata_repo_assets_list.go":        "catalogue des cartes (administration des assets)",
	"internal/platform/duckdb/patterns_repo.go":                    "patterns",
	"internal/platform/duckdb/replay_map_repo.go":                  "rejeu : nom EN technique de la carte (module)",
	"internal/platform/duckdb/queries_career.go":                   "carrière ; entrée registre de l'historique, que le helper reprend",
	"internal/platform/duckdb/queries_home_citations.go":           "accueil (citations)",
	"internal/platform/duckdb/queries_relations_moments.go":        "relations",
	"internal/platform/duckdb/queries_squad.go":                    "escouade",
	"internal/api/wire/registry_replay_build.go":                   "rejeu : nom EN technique de la carte (cuisson)",
	"internal/sync/replayartifacts/backlog.go":                     "rejeu : nom EN technique de la carte (file de cuisson)",
	"cmd/diag_db_health/main.go":                                   "outil de diagnostic",
	"cmd/diag_media_assets/main.go":                                "outil de diagnostic",
	"cmd/levelup/cmd_backfill_replay.go":                           "rejeu : nom EN technique de la carte (rattrapage)",
	"cmd/mapfond-inventaire/lecture.go":                            "outil d'inventaire des fonds de carte",
}

var (
	// resolveurMap : empreinte 1.
	resolveurMap = regexp.MustCompile(`\b(?:ResolveAssetNamesBulk|ResolveAssetName|resolveAssetNames|resolveAssetName|resolveAssetNameEN|loadAssetTranslationNames|loadAssetFRTranslations|resolveAssetNamesBulkBestEffort)\([^)]*"map"`)
	// typeMap et tableTraductions : empreinte 2 (les deux dans le même fichier).
	typeMap          = regexp.MustCompile(`asset_type\s*=\s*'map'`)
	tableTraductions = regexp.MustCompile(`\basset_translations\b`)
	// replisRegistre : empreinte 3.
	replisRegistre = regexp.MustCompile(`(?i)COALESCE\(\s*(?:\w+\.)?map_name_fr\s*,\s*(?:\w+\.)?map_name\b`)
)

// commentaireDeFin : un commentaire Go en fin de ligne (précédé d'un blanc, ce qu'une URL dans une
// chaîne n'est pas).
var commentaireDeFin = regexp.MustCompile(`\s//.*$`)

// sansCommentaires retire les commentaires Go (`//`), de ligne et de fin de ligne.
func sansCommentaires(source string) string {
	lignes := strings.Split(source, "\n")
	out := lignes[:0]
	for _, l := range lignes {
		if !strings.HasPrefix(strings.TrimSpace(l), "//") {
			out = append(out, commentaireDeFin.ReplaceAllString(l, ""))
		}
	}
	return strings.Join(out, "\n")
}

// empreintesDeLibelleCarte rend les empreintes de résolution du libellé de carte d'un texte.
func empreintesDeLibelleCarte(source string) []string {
	source = sansCommentaires(source)
	var out []string
	if resolveurMap.MatchString(source) {
		out = append(out, "résolveur de noms d'asset appelé pour les cartes")
	}
	if typeMap.MatchString(source) && tableTraductions.MatchString(source) {
		out = append(out, "requête asset_translations des cartes")
	}
	if replisRegistre.MatchString(source) {
		out = append(out, "repli de libellé du registre (map_name_fr, map_name)")
	}
	return out
}

func TestLibelleCarte_ReconnaitLesResolutions(t *testing.T) {
	for _, r := range []string{
		// la résolution propre au score d'engagement, puis à l'onglet (avant L13)
		"\tconst q = `\n\t\tSELECT name FROM asset_translations\n\t\tWHERE asset_type = 'map' AND asset_id = ?`",
		// l'historique avant L13
		"\tmapNames, _ := metaRepo.ResolveAssetNamesBulk(ctx, \"map\", mapIDs, langs)",
		// la liste des cartes du filtre avant L13
		"\t    COALESCE(r.map_name_fr, r.map_name, '') AS label,",
	} {
		if len(empreintesDeLibelleCarte(r)) == 0 {
			t.Errorf("le garde-rail ne reconnaît pas une résolution :\n%s", r)
		}
	}
	for _, sain := range []string{
		"       COALESCE(MAX(mr.map_name_fr), '') AS map_name_fr,",
		"\t\t\tCOALESCE(mr.map_name_fr, ''),",
		"\tnoms, err := repo.ResolveAssetNamesBulk(ctx, \"playlist\", ids, langs)",
		"\t// COALESCE(map_name_fr, map_name) : ce que le registre portait",
		"\tMapNameFR *string // COALESCE(map_name_fr, map_name), enrichi",
		"\tq := `SELECT name FROM asset_translations WHERE asset_type = 'playlist'`",
	} {
		if e := empreintesDeLibelleCarte(sain); len(e) != 0 {
			t.Errorf("faux positif sur %q : %v", sain, e)
		}
	}
}

func TestLibelleCarte_UneSeuleResolution(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	var violations []string
	porteurs := map[string]bool{}
	for _, sub := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, sub), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if rel == libelleCarteHelper {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			empreintes := empreintesDeLibelleCarte(string(data))
			if len(empreintes) == 0 {
				return nil
			}
			porteurs[rel] = true
			if _, gele := resolutionsGelees[rel]; !gele {
				violations = append(violations, rel+" : "+strings.Join(empreintes, ", "))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s/ : %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("résolution du libellé de carte hors de %s — lire libelleDeCarte / traductionsDeCartes :\n  %s",
			libelleCarteHelper, strings.Join(violations, "\n  "))
	}
	var mortes []string
	for f := range resolutionsGelees {
		if !porteurs[f] {
			mortes = append(mortes, f)
		}
	}
	sort.Strings(mortes)
	if len(mortes) > 0 {
		t.Errorf("entrées gelées sans empreinte — à retirer de resolutionsGelees :\n  %s", strings.Join(mortes, "\n  "))
	}
}
