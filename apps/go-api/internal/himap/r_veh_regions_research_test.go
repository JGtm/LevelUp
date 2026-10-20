//go:build research

package himap

// r_veh_regions_research_test.go — CAMPAGNE DE GRAMMAIRE, RECHERCHE R-L6 (2026-10-02) : l ORDRE
// des plages de compression des cartes du corpus, lu dans les modules installes, en comptant
// AUSSI les tags sbsp que le `levl` reference HORS du module de la carte (dans les modules
// globaux), comme Live Fire. Lecture seule ; aucun fichier ecrit hors de la sortie du test.
//
// Pour chaque module de RVEH_MODULES (chemins separes par `|`) :
//
//   - les tags sbsp LOCAUX (bornes, largeurs au niveau 0x10) ;
//   - les tags sbsp des modules porteurs (RVEH_GLOBALS/*.module) que le levl reference ;
//   - l ORDRE des regions sur l union (premier bloc de donnees du levl qui les reference toutes,
//     `ordreRegionsBSP` — la regle de `BSPQuantification`), et l ordre sur les locaux seuls ;
//   - chaque bloc de donnees du levl qui reference au moins un de ces tags : index du bloc et
//     GlobalID dans leur ordre d apparition (la piece brute).
//
//	RVEH_MODULES='<deploy>/ds/levels/multi/ctf_illusion/ctf_illusion-rtx-new.module|...' \
//	RVEH_GLOBALS=<deploy>/ds/globals \
//	  go test -tags=research -count=1 -v -run '^TestRVehRegionsDesCartes$' ./internal/himap/

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/himodule"
)

// TestRVehRegionsDesCartes journalise l ordre des regions de chaque carte (lignes RVEH_*).
func TestRVehRegionsDesCartes(t *testing.T) {
	brut, globals := os.Getenv("RVEH_MODULES"), os.Getenv("RVEH_GLOBALS")
	if brut == "" || globals == "" {
		t.Skip("RVEH_MODULES et RVEH_GLOBALS requis")
	}
	porteurs, err := filepath.Glob(filepath.Join(globals, "*.module"))
	if err != nil || len(porteurs) == 0 {
		t.Fatalf("aucun module porteur sous %s (%v)", globals, err)
	}
	parGID, err := sbspDesPorteurs(porteurs)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("RVEH_PORTEURS\t%d modules\t%d tags sbsp", len(porteurs), len(parGID))
	for _, chemin := range strings.Split(brut, "|") {
		rvehRegionsDUneCarte(t, chemin, parGID)
	}
}

// rvehRegionsDUneCarte : l ordre des regions d un module, locaux et externes.
func rvehRegionsDUneCarte(t *testing.T, chemin string, parGID map[uint32]sbspPorte) {
	nom := filepath.Base(chemin)
	m, err := himodule.Open(chemin)
	if err != nil {
		t.Logf("RVEH_ERREUR\t%s\t%v", nom, err)
		return
	}
	defer func() { _ = m.Close() }()
	locaux, err := bspBoundsFrom(m, chemin)
	if err != nil && len(locaux) == 0 {
		t.Logf("RVEH_LOCAUX\t%s\t0\t%v", nom, err)
	}
	tous := map[uint32]BSP{}
	origine := map[uint32]string{}
	for _, b := range locaux {
		tous[b.GlobalID], origine[b.GlobalID] = b, "local"
	}
	// Les externes : les tags des porteurs que le levl reference.
	for _, b := range sbspReferences(m, parGID) {
		if _, deja := tous[b.GlobalID]; !deja {
			tous[b.GlobalID], origine[b.GlobalID] = b, "externe:"+filepath.Base(parGID[b.GlobalID].module)
		}
	}
	for gid, b := range tous {
		bb := b.Bounds
		t.Logf("RVEH_TAG\t%s\t%08x\t%s\t%.3f,%.3f,%.3f,%.3f,%.3f,%.3f\t%v", nom, gid, origine[gid], bb.Min[0], bb.Min[1],
			bb.Min[2], bb.Max[0], bb.Max[1], bb.Max[2], bb.AxisWidths())
	}
	liste := make([]BSP, 0, len(tous))
	for _, b := range tous {
		liste = append(liste, b)
	}
	if ordre, err := ordreRegionsBSP(m, liste); err == nil {
		t.Logf("RVEH_ORDRE_UNION\t%s\t%s", nom, rvehGIDs(ordre))
	} else {
		t.Logf("RVEH_ORDRE_UNION\t%s\terreur\t%v", nom, err)
	}
	if len(locaux) > 1 {
		if ordre, err := ordreRegionsBSP(m, locaux); err == nil {
			t.Logf("RVEH_ORDRE_LOCAUX\t%s\t%s", nom, rvehGIDs(ordre))
		} else {
			t.Logf("RVEH_ORDRE_LOCAUX\t%s\terreur\t%v", nom, err)
		}
	}
	rvehBlocsDuLevl(t, m, nom, tous)
}

// rvehBlocsDuLevl journalise chaque bloc de donnees d un levl qui reference un tag de `tous`.
func rvehBlocsDuLevl(t *testing.T, m *himodule.Module, nom string, tous map[uint32]BSP) {
	for _, f := range m.Files("levl") {
		tag, err := m.Extract(f)
		if err != nil {
			continue
		}
		ti, err := meilleurTagInfo(tag)
		if err != nil {
			continue
		}
		for b := 0; b < ti.dataBlocks; b++ {
			abs, size := ti.blockAbs(b)
			if abs < 0 || size <= 0 || abs+size > len(ti.tag) {
				continue
			}
			var vus []string
			for p := abs; p+4 <= abs+size; p += 4 {
				gid := uint32(u32(ti.tag, p)) //nolint:gosec // mot de 32 bits
				if _, ok := tous[gid]; ok {
					vus = append(vus, fmt.Sprintf("%08x@+%d", gid, p-abs))
				}
			}
			if len(vus) > 0 {
				t.Logf("RVEH_BLOC\t%s\tlevl#%d\tbloc %d\ttaille %d\t%s", nom, f.Index, b, size, strings.Join(vus, ","))
			}
		}
	}
}

// rvehGIDs rend une liste de GlobalID en clair.
func rvehGIDs(gids []uint32) string {
	parts := make([]string, len(gids))
	for i, g := range gids {
		parts[i] = fmt.Sprintf("%d:%08x", i, g)
	}
	return strings.Join(parts, ",")
}
