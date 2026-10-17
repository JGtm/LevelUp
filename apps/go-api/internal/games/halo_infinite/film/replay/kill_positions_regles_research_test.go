//go:build research

package replay

// kill_positions_regles_research_test.go — E8.1 : LA LECTURE D'OCCUPATION (i10) SEPARE-T-ELLE LES
// SILENCES « TENUS » DES SILENCES « A BORD » ?
//
// Instrument de LECTURE (faits cuits, aucun octet de film decode). Pour chaque cote non place par
// `positionOf` (meme instant, meme registre que l'enquete `kill_positions_trous_research_test.go`),
// il dit si une montee a bord LUE (`object-parent-state`, `VehicleScan.Occupancy`) du siege du
// joueur couvre l'instant (episode non ferme par la lecture suivante du meme objet ni par le retour
// du bipede au flux de positions), et si le vehicule parent a un echantillon a l'instant.
//
//	L1_DEPOT=<checkout qui porte data/cache> KP_KILLS=<tsv> \
//	  go test -tags research -count=1 -run '^TestOccupationDesSilences$' -v ./internal/games/halo_infinite/film/replay/

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

type occBilan struct {
	exemples  []string
	parClasse map[string]map[string]int // classe enquete -> verdict occupation -> n
}

func (b *occBilan) noter(cote, classe, verdict string) {
	if b.parClasse == nil {
		b.parClasse = map[string]map[string]int{}
	}
	k := cote + " " + strings.Fields(classe)[0]
	if b.parClasse[k] == nil {
		b.parClasse[k] = map[string]int{}
	}
	b.parClasse[k][verdict]++
}

// aBordLu dit si une montee a bord lue du slot couvre tUS, et si le parent a un echantillon.
func (c *contexteFilm) aBordLu(slot uint32, tUS uint64) string {
	occ := c.g.Vehicles.Occupancy
	var derniere *types.VehicleOccupancy
	for i := range occ {
		o := &occ[i]
		if o.Slot != slot || o.TimestampUS > tUS {
			continue
		}
		if derniere == nil || o.TimestampUS >= derniere.TimestampUS {
			derniere = o
		}
	}
	if derniere == nil {
		return "aucune-lecture"
	}
	if !derniere.Attached {
		return "lecture-libre"
	}
	pts := c.pistes[slot].pts
	i := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS > derniere.TimestampUS })
	if i < len(pts) && pts[i].TimestampUS <= tUS {
		return "retour-au-flux-avant"
	}
	var v []grammar.BipedPosition
	for _, p := range c.vehicules {
		if p.Slot == derniere.ParentSlot && p.HasWorld {
			v = append(v, p)
		}
	}
	if len(v) == 0 {
		return "a-bord:parent-sans-echantillon"
	}
	tr := slotTrack{pts: v}
	if _, d := tr.at(tUS); d <= killPosToleranceUS {
		return "a-bord:parent-a-120ms"
	}
	j := sort.Search(len(v), func(k int) bool { return v[k].TimestampUS > tUS })
	if j > 0 && v[j-1].TimestampUS >= derniere.TimestampUS {
		return "a-bord:parent-tenu-depuis-montee"
	}
	if j > 0 {
		return "a-bord:parent-tenu-avant-montee"
	}
	return "a-bord:parent-apres-seulement"
}

func (b *occBilan) mesurer(c *contexteFilm, kills []tsvKill) {
	for _, d := range c.g.Deaths {
		if d.XUID == 0 {
			continue
		}
		tUS := uint64(d.TimeMS*1000 + c.origine)
		r := c.diagnostiquer(d.XUID, tUS, false, d.TimeMS)
		if r.place {
			continue
		}
		v := c.verdictOccupation(d.XUID, tUS)
		b.noter("victime", r.cause, v)
		b.exemple(c, "victime", r, v, d.TimeMS)
	}
	for _, k := range kills {
		tUS := uint64(k.timeMS*1000 + c.origine)
		r := c.diagnostiquer(k.killer, tUS, true, k.timeMS)
		if r.place {
			continue
		}
		v := c.verdictOccupation(k.killer, tUS)
		b.noter("tueur", r.cause, v)
		b.exemple(c, "tueur", r, v, k.timeMS)
	}
}

func (c *contexteFilm) verdictOccupation(xuid uint64, tUS uint64) string {
	sieges := siegesDe(c.reg, c.sieges, xuid, tUS)
	best := ""
	var bestT uint64
	for _, s := range sieges {
		pts := c.pistes[s].pts
		i := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS > tUS })
		if i > 0 && (best == "" || pts[i-1].TimestampUS > bestT) {
			bestT, best = pts[i-1].TimestampUS, c.aBordLu(s, tUS)
		}
	}
	if best == "" {
		return "sans-siege-ou-sans-point"
	}
	return best
}

func (b *occBilan) rendre(t *testing.T, titre string) {
	t.Logf("==== %s", titre)
	cles := make([]string, 0, len(b.parClasse))
	for k := range b.parClasse {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	for _, k := range cles {
		t.Logf("  %-40s %v", k, b.parClasse[k])
	}
	for _, e := range b.exemples {
		t.Logf("  EX %s", e)
	}
}

func TestOccupationDesSilences(t *testing.T) {
	kills := lireKillsTSV(t)
	fix := &occBilan{}
	for _, b := range goldenBuilds() {
		g, _ := chargerGoldenBuild(t, b)
		c, ok := preparerFilm(b.Short8, b.Build, g)
		if !ok {
			continue
		}
		fix.mesurer(c, kills[b.Short8])
	}
	fix.rendre(t, "CORPUS VERSIONNE")
	depot := os.Getenv("L1_DEPOT")
	if depot == "" {
		return
	}
	loc := &occBilan{}
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(depot, "data", "titles", "halo_infinite", "reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	dir := filepath.Join(depot, "data", "cache", "film_facts", "halo_infinite")
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cache de faits : %v", err)
	}
	for _, e := range entrees {
		if !strings.HasSuffix(e.Name(), ".filmfacts.bin") {
			continue
		}
		blob, err := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // mesure
		if err != nil {
			continue
		}
		head, err := DecodeFilmFactsEntete(blob)
		if err != nil {
			continue
		}
		var entry profile.MapQuantEntry
		for _, m := range cat.Maps {
			if m.Module == head.MapModule {
				entry = m
				break
			}
		}
		ff, err := DecodeFilmFactsFile(blob, entry)
		if err != nil {
			continue
		}
		nom := strings.TrimSuffix(e.Name(), ".filmfacts.bin")
		c, ok := preparerFilm(nom, "", &ff.Facts)
		if !ok {
			continue
		}
		loc.mesurer(c, kills[nom])
	}
	loc.rendre(t, "FAITS LOCAUX")
}

func (b *occBilan) exemple(c *contexteFilm, qui string, r cote, verdict string, tMS int64) {
	if !strings.HasPrefix(r.cause, "en-mouvement") || verdict != "aucune-lecture" {
		return
	}
	b.exemples = append(b.exemples, fmt.Sprintf("%s %s t=%d ms %s avant=%d apres=%d saut=%.2f/%v",
		qui, c.nom, tMS, r.cause, r.avantMS, r.apresMS, r.saut, r.aSaut))
}
