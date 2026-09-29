// Package archlint — no_local_radar_range_lookup_test.go : garde-rail de la resolution
// « variante -> portee du radar » (plan `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`, lot V2b,
// CLAUDE.md regle 6).
//
// La table `[radar_range_m]` de regulation.toml se lisait par variante en deux copies
// (`service/tactical_service_isolement.go` `rayonsParMatch`,
// `service/teammates/teammates_squad_isolement.go` `rayonParMatchDuScope`) plus une copie de
// test (le temoin V2.7 du placement des vies, `v2Portee`) ; l'ecriture au sync en demandait une
// troisieme. La SOURCE UNIQUE est `mappings.PorteeDuRadar`
// (`internal/games/mappings/portee_du_radar.go`) : la lecture et l'ecriture doivent rendre la
// MEME portee pour la MEME variante, sans quoi une ligne ecrite au sync se lirait perimee.
//
// Les deux empreintes, telles que les copies etaient ecrites :
//  1. une table dont le nom dit la portee (radar, portee, rayon) indexee par un nom NETTOYE
//     (`s.radar[strings.TrimSpace(`, `radar[strings.TrimSpace(`) ;
//  2. une table tiree de `.RadarRangeMap()` et indexee par un nom nettoye dans le MEME fichier
//     (la copie du temoin : `table := reg.RadarRangeMap()` puis `table[strings.TrimSpace(v)]`).
//
// Le test du motif (`TestNoLocalRadarRangeLookup_ReconnaitLesCopies`) prouve que les empreintes
// attrapent les anciennes copies : un garde-rail qui ne voit rien ne garde rien. Tests compris
// (la troisieme copie en etait un). Pas d'allowlist hors du helper et de ce fichier.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// porteeIndexeeRE : empreinte 1.
var porteeIndexeeRE = regexp.MustCompile(`(?i)\w*(radar|portee|rayon)\w*\s*\[\s*strings\.TrimSpace\(`)

// tableDesPorteesRE et indexNettoyeRE : empreinte 2, les deux dans le meme fichier.
var (
	tableDesPorteesRE = regexp.MustCompile(`\.RadarRangeMap\(\)`)
	indexNettoyeRE    = regexp.MustCompile(`\[\s*strings\.TrimSpace\(`)
)

// porteeHelper : le seul fichier qui resout une variante dans la table des portees.
const porteeHelper = "internal/games/mappings/portee_du_radar.go"

// porteeGarde : ce fichier, qui porte les anciennes copies comme litteraux de son auto-test.
const porteeGarde = "internal/archlint/no_local_radar_range_lookup_test.go"

// empreintesDePortee rend les empreintes de resolution locale presentes dans un texte source.
func empreintesDePortee(source string) []string {
	var out []string
	if porteeIndexeeRE.MatchString(source) {
		out = append(out, "table des portees indexee par un nom nettoye")
	}
	if tableDesPorteesRE.MatchString(source) && indexNettoyeRE.MatchString(source) {
		out = append(out, "RadarRangeMap() resolue a la main")
	}
	return out
}

func TestNoLocalRadarRangeLookup_ReconnaitLesCopies(t *testing.T) {
	copies := []string{
		// service/tactical_service_isolement.go, rayonsParMatch (avant le lot V2b)
		"\t\tmetres, ok := s.radar[strings.TrimSpace(m.GameVariantName)]\n\t\tif !ok || metres <= 0 {",
		// service/teammates/teammates_squad_isolement.go, rayonParMatchDuScope (avant le lot V2b)
		"\t\tmetres, ok := radar[strings.TrimSpace(m.GameVariantName)]\n\t\tif !ok || metres <= 0 {",
		// sync/killcollector/placement_des_vies_integration_test.go, v2Portee (avant le lot V2b)
		"\ttable := reg.RadarRangeMap()\n\treturn func(v string) (float64, bool) {\n" +
			"\t\tm, ok := table[strings.TrimSpace(v)]\n\t\treturn float64(m), ok && m > 0\n\t}",
	}
	for _, c := range copies {
		if len(empreintesDePortee(c)) == 0 {
			t.Errorf("le garde-rail ne reconnait pas une ancienne copie :\n%s", c)
		}
	}
	// Et il ne confond pas la table des portees avec le reste du reglement : la lecture du
	// temps reglementaire nettoie aussi sa cle, sans aucun rapport avec le radar.
	if e := empreintesDePortee("v, ok := s.seconds[strings.TrimSpace(gameVariantName)]"); len(e) != 0 {
		t.Errorf("faux positif sur le temps reglementaire : %v", e)
	}
}

func TestNoLocalRadarRangeLookup(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	var violations []string
	for _, sub := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, sub), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if rel == porteeHelper || rel == porteeGarde {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for _, e := range empreintesDePortee(string(data)) {
				violations = append(violations, rel+" : "+e)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s/ : %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("resolution locale de la portee du radar interdite — appeler mappings.PorteeDuRadar "+
			"(%s) :\n  %s", porteeHelper, strings.Join(violations, "\n  "))
	}
}
