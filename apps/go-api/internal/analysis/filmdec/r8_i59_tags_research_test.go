package filmdec

// r8_i59_tags_research_test.go — LA PIECE QUI MANQUAIT : les TAGS d'i59 et d'i57.
//
// CE QUI A MIS SUR LA PISTE. `grapple_state.go` etablit que le GRAPPIN est le corps
// **tag == 3** du composant i59 `biped-spartan-ability-non-predicted-state` : 115 des 117
// lectures tag==3 a porteur identifie tombent sur des vies de rang « grappin ». Le tag est
// un R(2) : il a QUATRE valeurs. La production n'en a jamais exploite qu'UNE, et les trois
// autres n'ont jamais ete confrontees a une identite de capacite. Le composant s'appelle
// « spartan-ability » — pas « grapple ».
//
// LA MESURE, ET ELLE NE SUPPOSE RIEN. Pour chaque lecture d'i59 (et d'i57, son jumeau
// PREDIT), on croise le TAG avec le RANG DE CAPACITE que le porteur tient a cet instant
// (canal i48, `ScanFilmAbilityRanks` — independant), et avec l'ORACLE PHYSIQUE (pic de
// vitesse du porteur). Le tag 3 sert de CLE DE LECTURE : il doit ressortir sur le rang du
// grappin, et cette verification est le controle qui valide la table entiere.
//
// SEUILS ECRITS AVANT LA MESURE :
//
//	concentration   un tag « est » une capacite si >= 75 % de ses lectures a rang connu
//	                tombent sur UN rang. C'est le seuil que le tag 3 doit tenir pour le
//	                grappin ; s'il ne le tient pas, la table n'est pas lisible et aucun
//	                autre tag ne sera lu comme une capacite.
//	oracle          pour un tag candidat « propulseur », le pic de vitesse du porteur doit
//	                depasser le P90 du temoin aleatoire, comme le fait le grappin.
//
// GARDES : `R8_FILMS`, `R8_BOUNDS`, `R8_IDS`.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R8_FILMS=<repo>/data/cache/film_chunks \
//	  R8_BOUNDS=<worktree>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  R8_IDS=00ba2e1c go test ./internal/analysis/filmdec/ -run '^TestR8I59Tags$' \
//	  -timeout 120m -v

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"
)

// r8I57Names / r8I59Names : les deux etiquettes possibles de chaque composant (les films
// portent l'une OU l'autre, meme dualite que le dispatch de consumeByName).
var (
	r8I57Names = []string{"biped-spartan-ability-component", "biped-spartan-ability"}
	r8I59Names = []string{
		"biped-spartan-ability-non-predicted-state-component",
		"biped-spartan-ability-non-predicted-state",
	}
)

// r8TagRead est UNE lecture taguee d'i57 ou d'i59.
type r8TagRead struct {
	Slot  uint32
	TSUS  uint64
	Tag   uint32
	Inner int // valeur interne du corps (i59 tag==3), -1 sinon
	Sub   uint64
	Ref   uint64
	HasR  bool
}

// r8ScanTags balaye le film et rend toutes les lectures d'i57 et d'i59, tag compris.
// L'appelant doit detenir LockProcessDecode : les hooks sont des globaux de paquet.
func r8ScanTags(s r8MobSetup) (i57, i59 []r8TagRead) {
	idx57 := r8IndexOfAny(s.arch, r8I57Names)
	idx59 := r8IndexOfAny(s.arch, r8I59Names)
	var last59 AbilityNonPredictedState
	var got59 bool
	var last57 r8TagRead
	var got57 bool
	prev59, prev57 := abilityNonPredictedHook, spartanAbilityHook
	SetAbilityNonPredictedHook(func(st AbilityNonPredictedState) { last59, got59 = st, true })
	SetSpartanAbilityHook(func(tag, sub, ref uint64, hasRef bool) {
		last57 = r8TagRead{Tag: uint32(tag), Sub: sub, Ref: ref, HasR: hasRef, Inner: -1}
		got57 = true
	})
	defer func() {
		SetAbilityNonPredictedHook(prev59)
		SetSpartanAbilityHook(prev57)
	}()
	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + s.lay.TotalBits()
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(s.dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			for p := 0; p+minRecord <= total; {
				i0, slot, ids, ok := matchBipedHeader(pay, p, total, s.slots, true, s.lay)
				if !ok {
					p++
					continue
				}
				if idx59 >= 0 && maskHas(ids, idx59) {
					got59 = false
					walkRecordTo(pay, i0, total, ids, s.lay, s.arch, idx59)
					if got59 {
						i59 = append(i59, r8TagRead{Slot: slot, TSUS: pk.TimestampUS,
							Tag: last59.Tag, Inner: last59.Inner})
					}
				}
				if idx57 >= 0 && maskHas(ids, idx57) {
					got57 = false
					walkRecordTo(pay, i0, total, ids, s.lay, s.arch, idx57)
					if got57 {
						last57.Slot, last57.TSUS = slot, pk.TimestampUS
						i57 = append(i57, last57)
					}
				}
				p = i0 + s.lay.TotalBits()
			}
		}
	}
	return i57, i59
}

// r8LifeSpan est une vie de bipede : les positions d'un meme slot sans trou majeur.
type r8LifeSpan struct{ from, to uint64 }

// r8LifeGapUS : au-dela de 5 s sans position, le slot a change de VIE (meme seuil que
// `lifeGapUS` de replay/lives.go). LE SLOT MIGRE AUX REAPPARITIONS : sans ce decoupage, le
// rang de capacite lu « le dernier avant l'instant » peut venir de la vie PRECEDENTE du
// meme slot — et attribuer a un propulseur le rang du champ de reparation d'avant.
const r8LifeGapUS = 5_000_000

func r8Lives(speeds r8SpeedIndex) map[uint32][]r8LifeSpan {
	out := map[uint32][]r8LifeSpan{}
	for slot, segs := range speeds {
		for _, s := range segs {
			v := out[slot]
			if n := len(v); n > 0 && s.t0-v[n-1].to <= r8LifeGapUS {
				v[n-1].to = s.t1
				out[slot] = v
				continue
			}
			out[slot] = append(v, r8LifeSpan{from: s.t0, to: s.t1})
		}
	}
	return out
}

// r8RankInLife rend le rang de capacite lu pour ce slot DANS LA MEME VIE que `at`, ou -1.
func r8RankInLife(
	ranks []AbilityRank, lives map[uint32][]r8LifeSpan, slot uint32, at uint64,
) int {
	var span r8LifeSpan
	found := false
	for _, l := range lives[slot] {
		if at+r8LifeGapUS >= l.from && at <= l.to+r8LifeGapUS {
			span, found = l, true
			break
		}
	}
	if !found {
		return -1
	}
	// DEUX CONTRAINTES, ET LES DEUX SONT NECESSAIRES : la lecture doit appartenir a la MEME
	// VIE (le slot migre aux reapparitions) ET PRECEDER l'instant (un joueur peut ramasser
	// un second equipement dans la meme vie — la lecture suivante ne dit rien de ce qu'il
	// portait avant). Oublier la seconde credite l'impulsion a l'equipement RAMASSE APRES :
	// sur `00ba2e1c` cela deplace 4 des 8 lectures du rang 5 vers le rang 4.
	best, bestT := -1, uint64(0)
	for _, r := range ranks {
		if r.Slot != slot || r.TimestampUS < span.from || r.TimestampUS > at {
			continue
		}
		if best < 0 || r.TimestampUS >= bestT {
			best, bestT = int(r.Rank), r.TimestampUS
		}
	}
	return best
}

func r8IndexOfAny(arch Archetype, names []string) int {
	for _, n := range names {
		if ids := arch.indicesOf(n); len(ids) > 0 {
			return ids[0]
		}
	}
	return -1
}

// r8TagCell croise un tag avec les rangs de capacite et l'oracle de vitesse.
type r8TagCell struct {
	n        int
	ranks    map[int]int
	peaks    []float64
	episodes int
}

func TestR8I59Tags(t *testing.T) {
	for _, dir := range r8FilmDirs(t) {
		r8I59TagsOneFilm(t, dir)
	}
}

func r8I59TagsOneFilm(t *testing.T, dir string) {
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
	ranks, _, err := ScanFilmAbilityRanks(dir)
	if err != nil {
		t.Logf("rangs de capacite illisibles : %v", err)
	}
	i57, i59 := r8ScanTags(s)
	t.Logf("%s : lectures i57=%d i59=%d | positions=%d rangs=%d",
		filepath.Base(dir), len(i57), len(i59), len(pos), len(ranks))
	lives := r8Lives(speeds)
	r8LogTagTable(t, "i59 biped-spartan-ability-non-predicted-state", i59, ranks, lives, speeds)
	r8LogTagTable(t, "i57 biped-spartan-ability", i57, ranks, lives, speeds)
	r8LogRandomPeak(t, speeds)
	r8LogTag1Dump(t, i57, ranks, lives, speeds)
	r8LogTag1Neighbours(t, i57, ranks, lives, r8PosBySlot(pos))
	r8LogImpulsionsParVie(t, i57, ranks, lives)
}

// r8LogTag1Dump detaille les lectures i57 tag==1, une par ligne : c'est la population qui
// porte une charge utile, et elle est assez petite pour se lire en entier. Bornee a 60
// lignes — un instrument ne deverse pas un film dans un journal.
func r8LogTag1Dump(t *testing.T, i57 []r8TagRead, ranks []AbilityRank,
	lives map[uint32][]r8LifeSpan, speeds r8SpeedIndex) {
	t.Helper()
	n := 0
	for _, r := range i57 {
		if r.Tag != 1 {
			continue
		}
		if n >= 60 {
			t.Logf("    ... (suite tue : borne a 60 lignes)")
			return
		}
		n++
		p, _ := speeds.peak(r.Slot, r.TSUS, r8PeakWindowUS)
		t.Logf("    tag1 slot=%-5d t=%8.2fs rang=%-3d sub=%d ref=0x%06x pic=%.2f",
			r.Slot, float64(r.TSUS)/1e6, r8RankInLife(ranks, lives, r.Slot, r.TSUS), r.Sub, r.Ref, p)
	}
}

// r8LogTagTable croise tag x rang porte x pic de vitesse.
func r8LogTagTable(
	t *testing.T, titre string, reads []r8TagRead, ranks []AbilityRank,
	lives map[uint32][]r8LifeSpan, speeds r8SpeedIndex,
) {
	t.Helper()
	cells := map[string]*r8TagCell{}
	lastEp := map[string]uint64{}
	for _, r := range reads {
		key := fmt.Sprintf("tag=%d", r.Tag)
		switch {
		case r.Inner >= 0:
			key = fmt.Sprintf("tag=%d/inner=%d", r.Tag, r.Inner)
		case r.HasR:
			// i57 tag==1 est la SEULE branche qui paie une charge utile : R(2) interne
			// (`Sub`) puis R(24) (`Ref`). Si une capacite se distingue d'une autre a
			// l'interieur du tag, c'est la qu'elle le fait — la cellule le montre.
			key = fmt.Sprintf("tag=%d/sub=%d", r.Tag, r.Sub)
		}
		c := cells[key]
		if c == nil {
			c = &r8TagCell{ranks: map[int]int{}}
			cells[key] = c
		}
		c.n++
		c.ranks[r8RankInLife(ranks, lives, r.Slot, r.TSUS)]++
		epk := fmt.Sprintf("%s|%d", key, r.Slot)
		if prev, ok := lastEp[epk]; !ok || r.TSUS-prev > r8MobEpisodeGapUS {
			c.episodes++
			if p, n := speeds.peak(r.Slot, r.TSUS, r8PeakWindowUS); n > 0 {
				c.peaks = append(c.peaks, p)
			}
		}
		lastEp[epk] = r.TSUS
	}
	keys := make([]string, 0, len(cells))
	for k := range cells {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("  %s", titre)
	t.Logf("    %-18s %8s %8s %8s %8s   %s",
		"cellule", "lectures", "episodes", "medPic", "p90Pic", "rangs i48 (rang x n)")
	for _, k := range keys {
		c := cells[k]
		t.Logf("    %-18s %8d %8d %8.2f %8.2f   %v", k, c.n, c.episodes,
			r8Quantile(c.peaks, 0.5), r8Quantile(c.peaks, 0.9), r8RankSummary(c.ranks))
	}
}

// r8NeighbourRadiusFilmM : rayon de voisinage, en metres, pour la question du REPULSEUR.
// 6,0 m — au-dessus de la portee annoncee de l'effet, meme valeur que l'oracle de la
// mesure 3, pour que les deux se comparent.
const r8NeighbourRadiusFilmM = 6.0

// r8PosBySlot indexe les positions par slot, triees par instant.
func r8PosBySlot(pos []BipedPosition) map[uint32][]BipedPosition {
	out := map[uint32][]BipedPosition{}
	for _, p := range pos {
		if p.HasWorld {
			out[p.Slot] = append(out[p.Slot], p)
		}
	}
	for _, l := range out {
		sort.Slice(l, func(i, j int) bool { return l[i].TimestampUS < l[j].TimestampUS })
	}
	return out
}

// r8NearestAt rend la position du slot la plus proche en temps de `at`, si elle est a
// moins de 100 ms — une position vieillie n'est plus une position.
func r8NearestAt(list []BipedPosition, at uint64) (BipedPosition, bool) {
	i := sort.Search(len(list), func(k int) bool { return list[k].TimestampUS >= at })
	best, ok := BipedPosition{}, false
	for _, k := range []int{i - 1, i} {
		if k < 0 || k >= len(list) {
			continue
		}
		if d := r8AbsDiff(list[k].TimestampUS, at); d <= 100_000 {
			if !ok || d < r8AbsDiff(best.TimestampUS, at) {
				best, ok = list[k], true
			}
		}
	}
	return best, ok
}

// r8LogTag1Neighbours repond a LA question du repulseur : quand un bipede recoit une
// impulsion `tag==1`, quels rangs de capacite portent les bipedes qui l'entourent ?
//
// POURQUOI CETTE QUESTION ET PAS UNE AUTRE. Le propulseur pousse CELUI QUI S'EN SERT — son
// impulsion se lit sur son propre rang. Le repulseur, lui, pousse LES AUTRES : si le film
// enregistre la poussee comme une impulsion sur la VICTIME, c'est le rang du VOISIN qui
// porte l'identite, pas celui du bipede qui recoit. Les deux lectures sont publiees cote a
// cote pour que le croisement se voie.
func r8LogTag1Neighbours(
	t *testing.T, i57 []r8TagRead, ranks []AbilityRank,
	lives map[uint32][]r8LifeSpan, pos map[uint32][]BipedPosition,
) {
	t.Helper()
	type cell struct {
		n     int
		own   map[int]int
		voisi map[int]int
		alone int
	}
	cells := map[uint64]*cell{}
	for _, r := range i57 {
		if r.Tag != 1 {
			continue
		}
		me, ok := r8NearestAt(pos[r.Slot], r.TSUS)
		if !ok {
			continue
		}
		c := cells[r.Sub]
		if c == nil {
			c = &cell{own: map[int]int{}, voisi: map[int]int{}}
			cells[r.Sub] = c
		}
		c.n++
		c.own[r8RankInLife(ranks, lives, r.Slot, r.TSUS)]++
		found := 0
		for slot, list := range pos {
			if slot == r.Slot {
				continue
			}
			o, ok := r8NearestAt(list, r.TSUS)
			if !ok {
				continue
			}
			if r8Dist2(float64(me.X), float64(me.Y), float64(o.X), float64(o.Y)) >
				r8NeighbourRadiusFilmM {
				continue
			}
			found++
			c.voisi[r8RankInLife(ranks, lives, slot, r.TSUS)]++
		}
		if found == 0 {
			c.alone++
		}
	}
	subs := make([]uint64, 0, len(cells))
	for s := range cells {
		subs = append(subs, s)
	}
	sort.Slice(subs, func(i, j int) bool { return subs[i] < subs[j] })
	t.Logf("    voisinage des impulsions tag==1 (rayon %.1f m)", r8NeighbourRadiusFilmM)
	for _, s := range subs {
		c := cells[s]
		t.Logf("    sub=%d n=%-4d sansVoisin=%-4d rangs PORTEUR %v | rangs VOISINS %v",
			s, c.n, c.alone, r8RankSummary(c.own), r8RankSummary(c.voisi))
	}
}

// r8LogImpulsionsParVie publie LE DENOMINATEUR QUI MANQUE : combien de VIES portent chaque
// rang de capacite, et combien d'impulsions `tag==1` elles emettent.
//
// SANS LUI, « 8 lectures sur le rang 5 » ne se juge pas. Un rang tres porte qui n'emet
// jamais et un rang rare qui emet a chaque vie donnent le meme histogramme brut, et ce sont
// deux faits opposes. C'est exactement la question du REPULSEUR : ses porteurs sont-ils
// nombreux et muets, ou absents du film ?
func r8LogImpulsionsParVie(
	t *testing.T, i57 []r8TagRead, ranks []AbilityRank, lives map[uint32][]r8LifeSpan,
) {
	t.Helper()
	vies := map[int]int{}
	for slot, spans := range lives {
		for _, sp := range spans {
			vies[r8RankInLife(ranks, lives, slot, sp.to)]++
		}
	}
	imp := map[int]int{}
	last := map[string]uint64{}
	for _, r := range i57 {
		if r.Tag != 1 {
			continue
		}
		rk := r8RankInLife(ranks, lives, r.Slot, r.TSUS)
		k := fmt.Sprintf("%d|%d", rk, r.Slot)
		if p, ok := last[k]; !ok || r.TSUS-p > r8MobEpisodeGapUS {
			imp[rk]++
		}
		last[k] = r.TSUS
	}
	keys := make([]int, 0, len(vies))
	for k := range vies {
		keys = append(keys, k)
	}
	for k := range imp {
		if _, ok := vies[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Ints(keys)
	t.Logf("    impulsions tag==1 par VIE, par rang de capacite")
	for _, k := range keys {
		if vies[k] == 0 && imp[k] == 0 {
			continue
		}
		taux := 0.0
		if vies[k] > 0 {
			taux = float64(imp[k]) / float64(vies[k])
		}
		t.Logf("      rang %-4d vies=%-5d impulsions=%-5d %.3f par vie", k, vies[k], imp[k], taux)
	}
}

// r8LogRandomPeak publie le temoin aleatoire du meme film — le denominateur de l'oracle.
func r8LogRandomPeak(t *testing.T, speeds r8SpeedIndex) {
	t.Helper()
	var b r8Bucket
	r8RandomFilmWitness(speeds, &b)
	t.Logf("    %-18s %8d %8s %8.2f %8.2f", "TEMOIN aleatoire", len(b.peaks), "-",
		r8Quantile(b.peaks, 0.5), r8Quantile(b.peaks, 0.9))
}
