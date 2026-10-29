//go:build research

package replay

// kill_positions_silences_research_test.go — E8.1 (b) (c) : CE QUE L'OCCUPATION LUE (i10) DIT DES
// SILENCES DE POSITION A L'INSTANT D'UN FRAG.
//
// Instrument de LECTURE (faits cuits, aucun octet de film decode, rien d'ecrit), sur les couples du
// journal (KP_KILLS) comme `kill_positions_e81_mesure_research_test.go`. Pour chaque cote que
// `positionOf` ne place pas, il classe le silence par les lectures d'occupation du siege du joueur
// (`object-parent-state`, la lecture que la marche de killsource recolte) entre son dernier
// echantillon et l'instant : a bord (montee sans descente ni retour au flux), debarque (montee puis
// descente), descente seule, aucune lecture ; puis, pour les silences sans montee, si les deux derniers
// echantillons portent le meme quantum et leur vitesse ; et pour un tueur, l'ecart entre le point tenu
// et la reprise (premier echantillon du meme siege apres l'instant).
//
//	L1_DEPOT=<checkout qui porte data/cache> KP_KILLS=<tsv> REG_COLLECTEUR=1 \
//	  go test -tags research -count=1 -run '^TestSilencesEtOccupation$' -v ./internal/games/halo_infinite/film/replay/

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/grammar"
	"github.com/JGtm/LevelUp/libs/film/internal/profile"
)

type silenceBilan struct {
	classes map[string]int
	reprise map[string][]float64
}

func (b *silenceBilan) noter(cle string, reprise float64, aReprise bool) {
	b.classes[cle]++
	if aReprise {
		b.reprise[cle] = append(b.reprise[cle], reprise)
	}
}

// classerLeSilence rend la classe du silence d'un cote non place, ou "" s'il est place.
func (c *contexteFilm) classerLeSilence(xuid uint64, tUS uint64, estTueur bool) (string, float64, bool) {
	sieges := siegesDe(c.reg, c.sieges, xuid, tUS)
	if v, _ := positionOf(c.pistes, sieges, tUS); v != nil {
		return "", 0, false
	}
	if len(sieges) == 0 {
		return "sans-siege", 0, false
	}
	var dernier grammar.BipedPosition
	var avant []grammar.BipedPosition
	var slot uint32
	avecPoint, exAequo := 0, false
	for _, s := range sieges {
		pts := c.pistes[s].pts
		i := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS > tUS })
		if i == 0 {
			continue
		}
		switch {
		case avecPoint > 0 && pts[i-1].TimestampUS == dernier.TimestampUS:
			exAequo = true
		case avecPoint == 0 || pts[i-1].TimestampUS > dernier.TimestampUS:
			dernier, slot, avant, exAequo = pts[i-1], s, pts[:i], false
		}
		avecPoint++
	}
	switch {
	case avecPoint == 0:
		return "sans-echantillon-avant", 0, false
	case exAequo:
		return "deux-corps-au-dernier-point", 0, false
	}
	mortMS := (int64(tUS) - c.origine) / 1000
	for _, m := range c.mortsPar[xuid] {
		if m < mortMS && m*1000+c.origine > int64(dernier.TimestampUS) {
			return "mort-pendant-le-silence", 0, false
		}
	}
	classe := c.occupationDuSilence(slot, dernier.TimestampUS, tUS)
	if classe == "a-bord" {
		classe += "/" + strings.TrimPrefix(c.aBordLu(slot, tUS), "a-bord:") + "/" + c.natureDuParent(slot, dernier.TimestampUS, tUS)
	}
	if classe == "aucune-lecture" || classe == "descente-seule" {
		classe += "/" + allureDesDeuxDerniers(avant)
	}
	if !estTueur {
		return classe, 0, false
	}
	pts := c.pistes[slot].pts
	j := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS > tUS })
	if j < len(pts) && pts[j].HasWorld && dernier.HasWorld {
		return classe, distPos(pts[j], dernier), true
	}
	return classe, 0, false
}

// occupationDuSilence classe les lectures i10 du siege entre (dernier - 1 s) et tUS.
func (c *contexteFilm) occupationDuSilence(slot uint32, dernierUS, tUS uint64) string {
	monte, descend, monteeA := false, false, uint64(0)
	for _, o := range c.g.Vehicles.Occupancy {
		if o.Slot != slot || o.TimestampUS > tUS || o.TimestampUS+1_000_000 < dernierUS {
			continue
		}
		if o.Attached {
			monte, descend, monteeA = true, false, o.TimestampUS
		} else {
			descend = true
		}
	}
	switch {
	case monte && !descend:
		pts := c.pistes[slot].pts
		i := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS > monteeA+1_000_000 })
		if i < len(pts) && pts[i].TimestampUS <= tUS {
			return "montee-puis-retour-au-flux"
		}
		return "a-bord"
	case monte && descend:
		return "debarque"
	case descend:
		return "descente-seule"
	}
	return "aucune-lecture"
}

// allureDesDeuxDerniers : meme quantum, sinon vitesse sous ou au-dessus de 0,5 m/s, sinon un seul point.
func allureDesDeuxDerniers(avant []grammar.BipedPosition) string {
	if len(avant) < 2 {
		return "un-point"
	}
	a, b := avant[len(avant)-2], avant[len(avant)-1]
	if a.Q == b.Q {
		return "meme-quantum"
	}
	if b.TimestampUS > a.TimestampUS && distPos(a, b)/(float64(b.TimestampUS-a.TimestampUS)/1e6) < 0.5 {
		return "lent"
	}
	return "en-mouvement"
}

func (b *silenceBilan) mesurer(c *contexteFilm, kills []tsvKill) {
	vus := map[[2]uint64]bool{}
	for _, k := range kills {
		tUS := uint64(k.timeMS*1000 + c.origine) //nolint:gosec // mesure
		if cl, d, ok := c.classerLeSilence(k.killer, tUS, true); cl != "" {
			b.noter("tueur "+cl, d, ok)
		}
		cle := [2]uint64{k.victim, tUS}
		if vus[cle] {
			continue
		}
		vus[cle] = true
		if cl, _, _ := c.classerLeSilence(k.victim, tUS, false); cl != "" {
			b.noter("victime "+cl, 0, false)
		}
	}
}

func (b *silenceBilan) rendre(t *testing.T, titre string) {
	t.Logf("==== %s", titre)
	cles := make([]string, 0, len(b.classes))
	for k := range b.classes {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	for _, k := range cles {
		t.Logf("  %-55s %5d  reprise %s", k, b.classes[k], quantilesF(b.reprise[k]))
	}
}

func TestSilencesEtOccupation(t *testing.T) {
	kills := lireKillsTSV(t)
	nouveau := func() *silenceBilan {
		return &silenceBilan{classes: map[string]int{}, reprise: map[string][]float64{}}
	}
	fix := nouveau()
	for _, b := range goldenBuilds() {
		g, _ := chargerGoldenBuild(t, b)
		if c, ok := preparerFilm(b.Short8, b.Build, g); ok {
			fix.mesurer(c, kills[b.Short8])
		}
	}
	fix.rendre(t, "CORPUS VERSIONNE")
	depot := os.Getenv("L1_DEPOT")
	if depot == "" {
		return
	}
	loc := nouveau()
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
		if c, ok := preparerFilm(nom, "", &ff.Facts); ok {
			loc.mesurer(c, kills[nom])
		}
	}
	loc.rendre(t, "FAITS LOCAUX")
}

// natureDuParent : le parent de la derniere montee de la fenetre est-il un slot de la bande `ti=40`
// des images-cles (vehicule), un slot de bipede (pistes des joueurs), ou ni l'un ni l'autre ?
func (c *contexteFilm) natureDuParent(slot uint32, dernierUS, tUS uint64) string {
	var parent uint32
	for _, o := range c.g.Vehicles.Occupancy {
		if o.Slot == slot && o.Attached && o.TimestampUS <= tUS && o.TimestampUS+1_000_000 >= dernierUS {
			parent = o.ParentSlot
		}
	}
	switch {
	case c.g.Vehicles.Keyframes.Band[parent]:
		return "parent-vehicule"
	case len(c.pistes[parent].pts) > 0:
		return "parent-bipede"
	}
	return "parent-autre"
}
