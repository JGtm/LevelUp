//go:build research

package main

// bis2_regions_research_test.go — MESURES BIS 2 DE LA CAMPAGNE DE GRAMMAIRE (2026-10-01), T4-C3 :
// LES BORNES DE TOUTES LES PLAGES DE COMPRESSION D UNE CARTE, lues dans les `.module` installes.
//
// Le catalogue `map_quant_bounds.json` ne porte que la plage JOUEE (Live Fire : la 1 sur 4). Le jeu
// lit pourtant, pour un index de plage LU, la ligne de CETTE plage (`FUN_14076e524` :
// `DAT_1445ccbe0 + (idx*0x20 + NIVEAU)*0xc`). Pour mesurer ce que la lecture par index changerait,
// il faut les bornes des autres plages : `himap.RegionsBSPExternes`, le chemin que
// `cmd/mapquant-build` emprunte pour Live Fire. LECTURE SEULE des modules ; rien n est ecrit hors
// de la sortie demandee. Le chemin du module vient de l environnement (aucun appel aux portes
// d entree de l installation) ; sans lui, le test est saute.
//
//	BIS2_MODULE=<deploy>/ds/levels/multi/sgh_interlock/sgh_interlock-rtx-new.module \
//	BIS2_GLOBALS=<deploy>/ds/globals \
//	  go test -tags=research -count=1 -v -run '^TestBis2RegionsDeLaCarte$' \
//	  ./internal/games/halo_infinite/film/research/cmd_fermeture/

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/himap"
)

// TestBis2RegionsDeLaCarte journalise les bornes et les largeurs (niveau 0x10) de chaque plage.
func TestBis2RegionsDeLaCarte(t *testing.T) {
	module, globals := os.Getenv("BIS2_MODULE"), os.Getenv("BIS2_GLOBALS")
	if module == "" || globals == "" {
		t.Skip("BIS2_MODULE et BIS2_GLOBALS requis")
	}
	porteurs, err := filepath.Glob(filepath.Join(globals, "*.module"))
	if err != nil || len(porteurs) == 0 {
		t.Fatalf("aucun module porteur sous %s (%v)", globals, err)
	}
	regions, err := himap.RegionsBSPExternes(module, porteurs)
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range regions {
		b := r.BSP.Bounds
		t.Logf("BIS2_REGION\t%d\t%08x\t%t\t%.6f\t%.6f\t%.6f\t%.6f\t%.6f\t%.6f\t%v\t%s", i, r.BSP.GlobalID,
			b.Valid(), b.Min[0], b.Min[1], b.Min[2], b.Max[0], b.Max[1], b.Max[2], b.AxisWidths(),
			filepath.Base(r.Module))
	}
}

// TestBis2CandidatsSbsp journalise, pour chaque module de BIS2_MODULES (chemins separes par `|`),
// les tags sbsp candidats (`himap.BSPQuantification`) : leur nombre borne le nombre de plages de
// compression que la carte peut declarer (une carte a UN seul sbsp n a qu une plage, donc un index
// de plage lu different de 0 y est impossible chez l ecrivain).
func TestBis2CandidatsSbsp(t *testing.T) {
	brut := os.Getenv("BIS2_MODULES")
	if brut == "" {
		t.Skip("BIS2_MODULES requis")
	}
	for _, m := range strings.Split(brut, "|") {
		q, cands, err := himap.BSPQuantification(m)
		if err != nil {
			t.Logf("BIS2_SBSP\t%s\terreur\t%v", filepath.Base(m), err)
			continue
		}
		t.Logf("BIS2_SBSP\t%s\t%d\t%08x\t%v", filepath.Base(m), len(cands), q.GlobalID, q.Bounds.AxisWidths())
		for _, c := range cands {
			b := c.Bounds
			t.Logf("BIS2_CAND\t%s\t%08x\t%t\t%.6f,%.6f,%.6f,%.6f,%.6f,%.6f\t%v", filepath.Base(m), c.GlobalID,
				c.GlobalID == q.GlobalID, b.Min[0], b.Min[1], b.Min[2], b.Max[0], b.Max[1], b.Max[2], b.AxisWidths())
		}
	}
}
