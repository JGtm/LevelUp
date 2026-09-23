//go:build research

package replay

// fiche_armes_research_test.go — SONDE EN LECTURE SEULE (enquete « fiche_armes », 2026-09-23).
//
// Lit AU PLUS TROIS fichiers de faits persistes (`<short8>.filmfacts.bin`) ; n'ouvre aucun film,
// n'ecrit rien. Trois questions :
//
//	A. une image-cle « trouee » : quels slots manquent, TOUS archetypes confondus (bipede par les
//	   loadouts/inventaires, ti=42/ti=37/ti=40 par les recensements), et encadres par les
//	   images-cles voisines (vus avant ET apres, donc vivants pendant) ;
//	B. a la NAISSANCE d'un bipede, qu'emet le flux delta : i48 (equipement de reapparition) et
//	   i43..i46 (identite d'arme) — l'un sert d'etalon a l'autre, meme marcheur, meme record ;
//	C. des creations ti=42 tombent-elles a la naissance d'un bipede (armes de depart creees
//	   comme objets du monde) ?
//
// USAGE (depuis apps/go-api du worktree) :
//
//	FA_FACTS_ROOT=<repo>/data/cache/film_facts/halo_infinite FA_FILMS=81c02726,a0c36016,b1f01a33 \
//	  go test -tags research -count=1 -run TestFicheArmesFaits ./internal/games/halo_infinite/film/replay/ -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
	"levelup/go-api/internal/testutil"
)

const faStepUS = int64(100_000)

func TestFicheArmesFaits(t *testing.T) {
	racine := os.Getenv("FA_FACTS_ROOT")
	liste := os.Getenv("FA_FILMS")
	if racine == "" || liste == "" {
		t.Skip("FA_FACTS_ROOT et FA_FILMS requis")
	}
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine : %v", err)
	}
	cat, err := profile.LoadMapQuantCatalog(title.NewPathResolver(root).MapQuantBoundsPath(title.DefaultSlug))
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	ids := strings.Split(liste, ",")
	if len(ids) > 3 {
		t.Fatalf("au plus trois films")
	}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		t.Run(id, func(t *testing.T) { faUnFilm(t, cat, filepath.Join(racine, id+".filmfacts.bin")) })
	}
}

func faCharger(t *testing.T, cat *profile.MapQuantCatalog, chemin string) *FilmFacts {
	t.Helper()
	blob, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("lecture %s : %v", chemin, err)
	}
	ent, err := DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Fatalf("en-tete : %v", err)
	}
	names := make([]string, 0, len(cat.Maps))
	for n := range cat.Maps {
		names = append(names, n)
	}
	sort.Strings(names)
	var lastErr error
	for _, n := range names {
		e := cat.Maps[n]
		if e.Module != ent.MapModule {
			continue
		}
		f, err := DecodeFilmFactsFile(blob, e)
		if err != nil {
			lastErr = err
			continue
		}
		t.Logf("carte %q module %s", n, ent.MapModule)
		return &f.Facts
	}
	t.Fatalf("module %q : aucune entree ne decode (%v)", ent.MapModule, lastErr)
	return nil
}

func faUnFilm(t *testing.T, cat *profile.MapQuantCatalog, chemin string) {
	g := faCharger(t, cat, chemin)
	origin := uint64(0)
	for i, p := range g.Positions {
		if i == 0 || p.TimestampUS < origin {
			origin = p.TimestampUS
		}
	}
	fr := func(us uint64) int64 { return (int64(us) - int64(origin)) / faStepUS }
	t.Logf("origine %d us ; positions %d ; loadouts %d ; inventaires %d ; changements d'arme %d ; "+
		"ramassages %d ; changements d'equipement %d ; creations bipede %d ; creations ti42 %d",
		origin, len(g.Positions), len(g.Loadouts), len(g.Inventory), len(g.WeaponChanges),
		len(g.Pickups), len(g.EquipmentChanges), len(g.BipedCreations), len(g.Pads.Weapons.Creations))
	faImagesCles(t, g, fr)
	faNaissances(t, g, fr)
	faStats(t, g)
}

// faParInstant range des cles de vie par instant d'image-cle.
func faParInstant(seen map[types.EquipmentLifeKey][]uint64) map[uint64]map[types.EquipmentLifeKey]bool {
	out := map[uint64]map[types.EquipmentLifeKey]bool{}
	for k, ts := range seen {
		for _, u := range ts {
			if out[u] == nil {
				out[u] = map[types.EquipmentLifeKey]bool{}
			}
			out[u][k] = true
		}
	}
	return out
}

// faImagesCles : la table par image-cle, et les absents ENCADRES de chaque image-cle.
func faImagesCles(t *testing.T, g *FilmFacts, fr func(uint64) int64) {
	kts := append([]uint64(nil), g.Pads.Weapons.Keyframes.TimesUS...)
	sort.Slice(kts, func(i, j int) bool { return kts[i] < kts[j] })
	lo, inv := map[uint64]map[uint32]bool{}, map[uint64]map[uint32]bool{}
	for _, l := range g.Loadouts {
		if lo[l.TimestampUS] == nil {
			lo[l.TimestampUS] = map[uint32]bool{}
		}
		lo[l.TimestampUS][l.Slot] = true
	}
	for _, r := range g.Inventory {
		if inv[r.TimestampUS] == nil {
			inv[r.TimestampUS] = map[uint32]bool{}
		}
		inv[r.TimestampUS][r.Slot] = true
	}
	w42 := faParInstant(g.Pads.Weapons.Keyframes.SeenUS)
	w37 := faParInstant(g.Pads.Powerups.Keyframes.SeenUS)
	w40 := faParInstant(g.Vehicles.Keyframes.SeenUS)
	t.Logf("images-cles : %d (ti42 %d, ti37 %d, ti40 %d instants)", len(kts),
		len(g.Pads.Weapons.Keyframes.TimesUS), len(g.Pads.Powerups.Keyframes.TimesUS),
		len(g.Vehicles.Keyframes.TimesUS))
	var sb strings.Builder
	fmt.Fprintf(&sb, "\n  %6s %5s %5s %5s %5s %5s %s\n", "frame", "lo", "inv", "ti42", "ti37", "ti40", "slots[min..max] tous archetypes")
	for _, ts := range kts {
		mn, mx := uint32(1<<31), uint32(0)
		acc := func(s uint32) {
			if s < mn {
				mn = s
			}
			if s > mx {
				mx = s
			}
		}
		for s := range inv[ts] {
			acc(s)
		}
		for k := range w42[ts] {
			acc(k.Slot)
		}
		for k := range w37[ts] {
			acc(k.Slot)
		}
		for k := range w40[ts] {
			acc(k.Slot)
		}
		fmt.Fprintf(&sb, "  %6d %5d %5d %5d %5d %5d [%d..%d]\n", fr(ts), len(lo[ts]), len(inv[ts]),
			len(w42[ts]), len(w37[ts]), len(w40[ts]), mn, mx)
	}
	t.Log(sb.String())
	for i := 1; i+1 < len(kts); i++ {
		faAbsentsEncadres(t, fr, kts[i-1], kts[i], kts[i+1], inv, w42, w37, w40)
	}
}

// faAbsentsEncadres imprime, pour l'image-cle `cur`, les slots vus a `prev` ET a `next` mais pas
// a `cur`, par archetype — et les slots PRESENTS a `cur`, pour situer l'intervalle perdu.
func faAbsentsEncadres(t *testing.T, fr func(uint64) int64, prev, cur, next uint64,
	inv map[uint64]map[uint32]bool, w42, w37, w40 map[uint64]map[types.EquipmentLifeKey]bool,
) {
	type tag struct {
		slot uint32
		ti   string
	}
	var absents, presents []tag
	for s := range inv[prev] {
		if inv[next][s] && !inv[cur][s] {
			absents = append(absents, tag{s, "35"})
		}
	}
	for s := range inv[cur] {
		presents = append(presents, tag{s, "35"})
	}
	for _, x := range []struct {
		m  map[uint64]map[types.EquipmentLifeKey]bool
		ti string
	}{{w42, "42"}, {w37, "37"}, {w40, "40"}} {
		for k := range x.m[prev] {
			if x.m[next][k] && !x.m[cur][k] {
				absents = append(absents, tag{k.Slot, x.ti})
			}
		}
		for k := range x.m[cur] {
			presents = append(presents, tag{k.Slot, x.ti})
		}
	}
	if len(absents) == 0 {
		return
	}
	sort.Slice(absents, func(i, j int) bool { return absents[i].slot < absents[j].slot })
	sort.Slice(presents, func(i, j int) bool { return presents[i].slot < presents[j].slot })
	fmtTags := func(v []tag, max int) string {
		var b strings.Builder
		for i, x := range v {
			if i == max {
				fmt.Fprintf(&b, " ...(+%d)", len(v)-max)
				break
			}
			fmt.Fprintf(&b, " %d/%s", x.slot, x.ti)
		}
		return b.String()
	}
	// les presents SOUS le plus petit absent et AU-DESSUS du plus grand : les bornes du trou
	lowA, highA := absents[0].slot, absents[len(absents)-1].slot
	var below, inside, above int
	for _, p := range presents {
		switch {
		case p.slot < lowA:
			below++
		case p.slot > highA:
			above++
		default:
			inside++
		}
	}
	t.Logf("IMAGE-CLE TROUEE frame %d : %d absents encadres [%d..%d] ; presents sous %d, dans %d, au-dessus %d\n"+
		"    absents :%s\n    presents :%s", fr(cur), len(absents), lowA, highA, below, inside, above,
		fmtTags(absents, 80), fmtTags(presents, 80))
}

// faNaissances : ce que le film emet a la naissance de chaque bipede.
func faNaissances(t *testing.T, g *FilmFacts, fr func(uint64) int64) {
	creas := append([]grammar.BipedCreation(nil), g.BipedCreations...)
	sort.Slice(creas, func(i, j int) bool { return creas[i].TimestampUS < creas[j].TimestampUS })
	// borne de vie : la creation suivante du meme slot
	nextSame := map[int]uint64{}
	lastIdx := map[uint32]int{}
	for i := len(creas) - 1; i >= 0; i-- {
		c := creas[i]
		if j, ok := lastIdx[c.Slot]; ok {
			nextSame[i] = creas[j].TimestampUS
		} else {
			nextSame[i] = ^uint64(0)
		}
		lastIdx[c.Slot] = i
	}
	const us1s = 1_000_000
	var n, heldIn1s, restIn1s, takenIn1s, anyHeld, eqSpawnIn1s, eqAnyIn1s, ti42In2s, loWithin int
	var sb strings.Builder
	for i, c := range creas {
		lo, hi := c.TimestampUS, nextSame[i]
		n++
		// premier changement d'arme en main de la vie
		var fh *types.HeldWeaponChange
		for k := range g.WeaponChanges {
			w := &g.WeaponChanges[k]
			if w.Slot == c.Slot && w.TimestampUS >= lo && w.TimestampUS < hi {
				if fh == nil || w.TimestampUS < fh.TimestampUS {
					fh = w
				}
			}
		}
		var fe *types.EquipmentChange
		for k := range g.EquipmentChanges {
			e := &g.EquipmentChanges[k]
			if e.Slot == c.Slot && e.TimestampUS >= lo && e.TimestampUS < hi {
				if fe == nil || e.TimestampUS < fe.TimestampUS {
					fe = e
				}
			}
		}
		var fl *types.KeyframeLoadout
		for k := range g.Loadouts {
			l := &g.Loadouts[k]
			if l.Slot == c.Slot && l.TimestampUS >= lo && l.TimestampUS < hi {
				if fl == nil || l.TimestampUS < fl.TimestampUS {
					fl = l
				}
			}
		}
		n42 := 0
		for _, w := range g.Pads.Weapons.Creations {
			if w.TimestampUS+us1s >= lo && w.TimestampUS <= lo+2*us1s {
				n42++
			}
		}
		if n42 > 0 {
			ti42In2s++
		}
		if fh != nil {
			anyHeld++
			if fh.TimestampUS-lo <= us1s {
				heldIn1s++
				if fh.Kind == types.HeldWeaponRestated {
					restIn1s++
				} else {
					takenIn1s++
				}
			}
		}
		if fe != nil && fe.TimestampUS-lo <= us1s {
			eqAnyIn1s++
			if fe.Kind == types.EquipmentSpawned {
				eqSpawnIn1s++
			}
		}
		if fl != nil {
			loWithin++
		}
		d := func(us uint64) string { return fmt.Sprintf("%+.1fs", float64(int64(us)-int64(lo))/1e6) }
		held, eq, lod := "-", "-", "-"
		if fh != nil {
			held = fmt.Sprintf("%s %s idx%d %08x (prev %08x)", d(fh.TimestampUS), fh.Kind, fh.SlotIndex, fh.Family, fh.Previous)
		}
		if fe != nil {
			eq = fmt.Sprintf("%s %s r%d", d(fe.TimestampUS), fe.Kind, fe.Rank)
		}
		if fl != nil {
			lod = d(fl.TimestampUS)
		}
		fmt.Fprintf(&sb, "  naissance slot %d gen %d idx %d(%v) frame %d | 1re arme : %s | 1er equip : %s | 1re image-cle lue : %s | ti42 a +-: %d\n",
			c.Slot, c.Generation, c.ParticipantIndex, c.HasIndex, fr(c.TimestampUS), held, eq, lod, n42)
	}
	if len(creas) <= 60 {
		t.Log("\n" + sb.String())
	}
	t.Logf("NAISSANCES %d : 1er changement d'arme <=1 s %d (restated %d, autre %d) ; vies avec un changement d'arme %d ; "+
		"1er equipement <=1 s %d (dont spawned %d) ; creation ti42 dans [-1 s, +2 s] %d ; vies avec loadout d'image-cle %d",
		n, heldIn1s, restIn1s, takenIn1s, anyHeld, eqAnyIn1s, eqSpawnIn1s, ti42In2s, loWithin)
	// les changements d'arme bruts, restated compris
	var rb strings.Builder
	for _, w := range g.WeaponChanges {
		fmt.Fprintf(&rb, "  frame %d slot %d idx %d %s %08x prev %08x\n", fr(w.TimestampUS), w.Slot, w.SlotIndex, w.Kind, w.Family, w.Previous)
	}
	if len(g.WeaponChanges) <= 40 {
		t.Log("\nCHANGEMENTS D'ARME BRUTS\n" + rb.String())
	}
}

func faStats(t *testing.T, g *FilmFacts) {
	t.Logf("ti42 stats %+v", g.Pads.Weapons.Stats)
	t.Logf("ti37 stats %+v", g.Pads.Powerups.Stats)
	t.Logf("ramassages stats %+v", g.PickupStats)
	t.Logf("equipement stats %+v", g.EquipmentChangeStats)
	ref, n := 0, 0
	for _, c := range g.Pads.Weapons.Creations {
		n++
		if c.HasRef {
			ref++
		}
	}
	t.Logf("creations ti42 : %d, dont porte de ref ouverte %d", n, ref)
}
