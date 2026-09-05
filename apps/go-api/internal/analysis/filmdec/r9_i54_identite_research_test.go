package filmdec

// r9_i54_identite_research_test.go — i54 REJUGE PAR L'IDENTITE
// (par. 5 du RAPPORT_R9_REPULSEUR_2026-09-03).
//
// POURQUOI ROUVRIR i54, QUE R8 A FERME. R8 (par. 7.2) a lu la charge utile d'i54
// `biped-mobility-action` — six champs terminaux, dont un `B7` a sept valeurs — et l'a
// declare muet. Mais il l'a juge sur UN SEUL oracle : **la bouffee de vitesse du porteur**.
// « Mediane du pic 3,00 m/s contre 3,18 au temoin aleatoire » ; conclusion ecrite : « i54
// n'est pas le canal de l'usage du PROPULSEUR ». Elle est juste, et elle ne dit rien du
// repulseur — parce que **le repulseur ne projette pas celui qui s'en sert**. Un canal juge
// par un oracle de vitesse ne peut pas trancher une capacite qui n'en produit pas.
//
// CE LOT LE REJUGE PAR L'IDENTITE, l'oracle qui a tranche le propulseur : chaque valeur de
// `B7` (et des autres champs terminaux) est croisee avec le RANG DE CAPACITE que le porteur
// tient DANS LA MEME VIE et ANTERIEUREMENT (canal i48, independant), avec le denominateur par
// VIE — la forme de tableau du par. 8.8 de R8.
//
// SEUILS ECRITS AVANT LA MESURE :
//
//	concentration  une valeur de champ « est » le repulseur si >= 75 % de ses episodes a rang
//	               connu tombent sur le rang du repulseur. Meme seuil que celui que le tag 3
//	               d'i59 tient pour le grappin (90-100 % mesures au par. 2 de ce rapport).
//	par vie        facteur >= 5 entre le rang repulseur et le meilleur rang temoin, sur les
//	               episodes par vie.
//	oracle voisin  a ces instants, un bipede a <= 6 m montre un pic median au-dessus du P90
//	               du temoin aleatoire du meme film.
//
// TEMOIN POSITIF : le tableau publie TOUTES les cellules, et l'instrument publie en regard le
// tableau par vie des impulsions i57 tag==1 — dont on sait qu'il donne 0,4 a 0,5 par vie de
// propulseur et 0,000 partout ailleurs. Si une cellule d'i54 se comportait comme celle-la sur
// le rang du repulseur, elle serait le canal cherche.
//
// GARDES : `R8_FILMS`, `R8_BOUNDS`, `R8_IDS`.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R8_FILMS=<repo>/data/cache/film_chunks \
//	  R8_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  R8_IDS=00ba2e1c,06dfe6d9 go test ./internal/analysis/filmdec/ \
//	  -run '^TestR9I54Identite$' -count=1 -timeout 180m -v

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"
)

// r9MobCell croise une valeur de champ d'i54 avec les rangs et l'oracle.
type r9MobCell struct {
	n        int
	episodes int
	ranks    map[int]int
	peaks    []float64
	vois     []float64
}

func TestR9I54Identite(t *testing.T) {
	for _, dir := range r8FilmDirs(t) {
		r9I54OneFilm(t, dir)
	}
}

func r9I54OneFilm(t *testing.T, dir string) {
	t.Helper()
	entry := r8MapEntry(t, dir)
	wr := entry.Range()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	SetWorldObjectPrecisionFromLayout(entry.Layout())
	defer func() { WorldObjectPrecision = saved }()

	s := r8MobResolve(t, dir)
	opt := DefaultScanFilmOptions()
	opt.WorldRange = &wr
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("positions illisibles : %v", err)
	}
	speeds := r8BuildSpeeds(pos)
	lives := r8Lives(speeds)
	ranks, _, err := ScanFilmAbilityRanks(dir)
	if err != nil {
		t.Logf("rangs de capacite illisibles : %v", err)
	}
	evs, records, with54 := r8ScanMobility(t, s)
	t.Logf("%s : records=%d avec_i54=%d emissions=%d | rangs=%d vies=%d",
		filepath.Base(dir), records, with54, len(evs), len(ranks), r9CountLives(lives))
	ctx := r9MobCtx{ranks: ranks, lives: lives, speeds: speeds, pos: r8PosBySlot(pos)}
	ctx.table(t, "flag1/flag2", evs, func(e r8MobEvent) (string, bool) {
		return fmt.Sprintf("f1=%v f2=%v", e.Flag1, e.Flag2), true
	})
	ctx.table(t, "B7 (corps lu)", evs, func(e r8MobEvent) (string, bool) {
		return fmt.Sprintf("B7=%d", e.B7), e.Body
	})
	ctx.table(t, "B1/B2/BLast (corps lu)", evs, func(e r8MobEvent) (string, bool) {
		return fmt.Sprintf("B1=%d B2=%d BL=%d", e.B1, e.B2, e.BLast), e.Body
	})
	r8LogRandomPeak(t, speeds)
	ctx.parVie(t, evs)
}

// r9MobCtx porte le contexte de jugement (regle des <= 5 parametres).
type r9MobCtx struct {
	ranks  []AbilityRank
	lives  map[uint32][]r8LifeSpan
	speeds r8SpeedIndex
	pos    map[uint32][]BipedPosition
}

// table croise une CLE de cellule avec les rangs, l'oracle du porteur et celui du voisin.
func (c r9MobCtx) table(
	t *testing.T, titre string, evs []r8MobEvent, key func(r8MobEvent) (string, bool),
) {
	t.Helper()
	cells := map[string]*r9MobCell{}
	last := map[string]uint64{}
	for _, e := range evs {
		k, ok := key(e)
		if !ok {
			continue
		}
		cl := cells[k]
		if cl == nil {
			cl = &r9MobCell{ranks: map[int]int{}}
			cells[k] = cl
		}
		cl.n++
		cl.ranks[r8RankInLife(c.ranks, c.lives, e.Slot, e.TSUS)]++
		ek := fmt.Sprintf("%s|%d", k, e.Slot)
		if prev, ok := last[ek]; !ok || e.TSUS-prev > r8MobEpisodeGapUS {
			cl.episodes++
			if p, n := c.speeds.peak(e.Slot, e.TSUS, r8PeakWindowUS); n > 0 {
				cl.peaks = append(cl.peaks, p)
			}
			if vp, ok := r9NeighbourPeak(c.pos, c.speeds, e.Slot, e.TSUS); ok {
				cl.vois = append(cl.vois, vp)
			}
		}
		last[ek] = e.TSUS
	}
	keys := make([]string, 0, len(cells))
	for k := range cells {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("  i54 — %s", titre)
	t.Logf("    %-22s %8s %8s %8s %8s %8s   %s", "cellule", "lectures", "episodes",
		"medPic", "medVois", "p90Vois", "rangs i48")
	for _, k := range keys {
		cl := cells[k]
		t.Logf("    %-22s %8d %8d %8.2f %8.2f %8.2f   %v", k, cl.n, cl.episodes,
			r8Quantile(cl.peaks, 0.5), r8Quantile(cl.vois, 0.5), r8Quantile(cl.vois, 0.9),
			r8RankSummary(cl.ranks))
	}
}

// parVie publie LE DENOMINATEUR : episodes d'i54 par VIE, par rang porte, pour chaque valeur
// de `B7`. C'est la forme qui a tranche le propulseur ; c'est elle qui tranche ici.
func (c r9MobCtx) parVie(t *testing.T, evs []r8MobEvent) {
	t.Helper()
	vies := map[int]int{}
	for slot, spans := range c.lives {
		for _, sp := range spans {
			vies[r8RankInLife(c.ranks, c.lives, slot, sp.to)]++
		}
	}
	parB7 := map[uint32]map[int]int{}
	last := map[string]uint64{}
	for _, e := range evs {
		if !e.Body {
			continue
		}
		rk := r8RankInLife(c.ranks, c.lives, e.Slot, e.TSUS)
		ek := fmt.Sprintf("%d|%d|%d", e.B7, rk, e.Slot)
		if prev, ok := last[ek]; !ok || e.TSUS-prev > r8MobEpisodeGapUS {
			m := parB7[e.B7]
			if m == nil {
				m = map[int]int{}
				parB7[e.B7] = m
			}
			m[rk]++
		}
		last[ek] = e.TSUS
	}
	b7s := make([]uint32, 0, len(parB7))
	for b := range parB7 {
		b7s = append(b7s, b)
	}
	sort.Slice(b7s, func(i, j int) bool { return b7s[i] < b7s[j] })
	rks := make([]int, 0, len(vies))
	for k := range vies {
		rks = append(rks, k)
	}
	sort.Ints(rks)
	t.Logf("  i54 — episodes par VIE, par valeur de B7")
	for _, b := range b7s {
		var parts []string
		for _, k := range rks {
			if vies[k] == 0 {
				continue
			}
			parts = append(parts, fmt.Sprintf("r%d=%.3f(%d/%d)", k,
				float64(parB7[b][k])/float64(vies[k]), parB7[b][k], vies[k]))
		}
		t.Logf("    B7=%-3d %v", b, parts)
	}
}
