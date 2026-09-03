package filmdec

// r9_i57_tag3_research_test.go — PORTE (a) du lot R9 : la branche `tag == 3` d'i57.
//
// LA QUESTION. R8 a trouve le PROPULSEUR (impulsion `tag == 1` d'i57/i59) et laisse le
// REPULSEUR ouvert. Le registre des reports de R8 (par. 11, ligne 3) designe comme piste la
// plus prometteuse la branche `tag == 3` d'i57 (`consumeSpartanAbilityTag3`, miroir de
// FUN_142f262d4) : c'est LE SEUL endroit du composant ou une largeur reste indeterminee,
// parce que son corps est garde par un OCTET D'ETAT RUNTIME (`dst[2]`) invisible du flux.
//
// CE QUI SE LIT SANS RIEN DEVINER. La branche s'ouvre sur `a = R(1)` :
//
//	a == 0 : la suite est ENTIEREMENT portee (porte de queue + queue-handle d'i60) -> ported
//	a == 1 : R(6) puis la branche gardee par dst[2] -> le deser rend false, RECORD ABANDONNE
//
// Le bit `a` n'a donc pas besoin d'etre lu a la main : `consumeSpartanAbilityTag3` rend false
// SI ET SEULEMENT SI a != 0, et `walkRecordTo` ne visite le composant que s'il est porte.
// **`walkRecordTo == true` sur une lecture tag==3 equivaut a `a == 0`**, et l'inverse a
// `a == 1`. Aucune copie de la marche de production n'est necessaire (regle des <= 2 copies).
// Reserve honnete : un debordement de payload (`br.BitPos() > total`) produirait le meme
// `false` ; il est compte a part (`deborde`) et reste marginal.
//
// SEUILS ECRITS AVANT LA MESURE (pre-inscription, cf. RAPPORT_R9 par. 1.1) :
//
//	concentration  >= 75 % des lectures `tag==3, a==1` a rang connu sur le rang du REPULSEUR
//	non-confusion  elles ne doivent PAS se concentrer sur le rang du grappin ni du propulseur
//	par vie        facteur >= 5 entre le rang repulseur et le meilleur rang temoin
//	oracle voisin  mediane du pic du voisin > P90 du temoin aleatoire apparie
//
// TEMOINS POSITIFS OBLIGATOIRES, publies par le meme instrument : (i) le tag 3 d'i59 doit
// retrouver le GRAPPIN (>= 75 % sur son rang), (ii) le tag 1 d'i57 doit retrouver le
// PROPULSEUR (>= 0,3 impulsion par vie de son rang, 0,000 sur les rangs temoins). Si ces deux
// controles ne passent pas, aucun negatif n'est publie : c'est l'instrument qui est en cause.
//
// GARDES : `R8_FILMS`, `R8_BOUNDS`, `R8_IDS` (memes variables que les instruments R8).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R8_FILMS=<repo>/data/cache/film_chunks \
//	  R8_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  R8_IDS=00ba2e1c,06dfe6d9 go test ./internal/analysis/filmdec/ \
//	  -run '^TestR9I57Tag3$' -count=1 -timeout 120m -v

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"
)

// r9Read est UNE lecture d'i57 avec, en plus du tag, l'issue de la marche — qui vaut le
// bit `a` sur la branche tag==3 (cf. en-tete).
type r9Read struct {
	Slot   uint32
	TSUS   uint64
	Tag    uint32
	Sub    uint64
	HasR   bool
	Ported bool
}

// r9Cell : une cellule de la table (une valeur de tag, scindee par `a` sur le tag 3).
type r9Cell struct {
	n        int
	episodes int
	ranks    map[int]int
	peaks    []float64
}

// r9Label nomme la cellule d'une lecture. Le tag 3 se scinde en deux : c'est TOUT l'objet de
// ce lot, et la scission doit se lire dans l'etiquette.
func r9Label(r r9Read) string {
	switch {
	case r.Tag == 3 && r.Ported:
		return "tag=3/a=0 (porte)"
	case r.Tag == 3:
		return "tag=3/a=1 (RUNTIME)"
	case r.HasR:
		return fmt.Sprintf("tag=%d/sub=%d", r.Tag, r.Sub)
	default:
		return fmt.Sprintf("tag=%d", r.Tag)
	}
}

// r9Scan balaye le film et rend les lectures d'i57 (tag + issue de marche) et celles d'i59
// (le temoin positif du grappin). L'appelant doit detenir LockProcessDecode.
func r9Scan(s r8MobSetup) (i57 []r9Read, i59 []r8TagRead, masked int) {
	idx57 := r8IndexOfAny(s.arch, r8I57Names)
	idx59 := r8IndexOfAny(s.arch, r8I59Names)
	var last57 r9Read
	var got57 bool
	var last59 AbilityNonPredictedState
	var got59 bool
	prev57, prev59 := spartanAbilityHook, abilityNonPredictedHook
	SetSpartanAbilityHook(func(tag, sub, ref uint64, hasRef bool) {
		last57 = r9Read{Tag: uint32(tag), Sub: sub, HasR: hasRef}
		got57 = true
	})
	SetAbilityNonPredictedHook(func(st AbilityNonPredictedState) { last59, got59 = st, true })
	defer func() {
		SetSpartanAbilityHook(prev57)
		SetAbilityNonPredictedHook(prev59)
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
					masked++
					got57 = false
					ported := walkRecordTo(pay, i0, total, ids, s.lay, s.arch, idx57)
					if got57 {
						last57.Slot, last57.TSUS, last57.Ported = slot, pk.TimestampUS, ported
						i57 = append(i57, last57)
					}
				}
				p = i0 + s.lay.TotalBits()
			}
		}
	}
	return i57, i59, masked
}

func TestR9I57Tag3(t *testing.T) {
	for _, dir := range r8FilmDirs(t) {
		r9OneFilm(t, dir)
	}
}

func r9OneFilm(t *testing.T, dir string) {
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
	i57, i59, masked := r9Scan(s)
	lives := r8Lives(speeds)
	t.Logf("%s : i57 masque=%d lu=%d | i59 lu=%d | positions=%d rangs=%d vies=%d",
		filepath.Base(dir), masked, len(i57), len(i59), len(pos), len(ranks), r9CountLives(lives))
	r9LogCells(t, i57, ranks, lives, speeds)
	r8LogRandomPeak(t, speeds)
	r9LogParVie(t, i57, ranks, lives)
	r9LogVoisins(t, i57, ranks, lives, r8PosBySlot(pos), speeds)
	r9LogTemoinI59(t, i59, ranks, lives)
}

func r9CountLives(lives map[uint32][]r8LifeSpan) int {
	n := 0
	for _, v := range lives {
		n += len(v)
	}
	return n
}

// r9LogCells publie la table tag x rang x oracle, tag 3 SCINDE par le bit `a`.
func r9LogCells(t *testing.T, i57 []r9Read, ranks []AbilityRank,
	lives map[uint32][]r8LifeSpan, speeds r8SpeedIndex) {
	t.Helper()
	cells := map[string]*r9Cell{}
	lastEp := map[string]uint64{}
	for _, r := range i57 {
		k := r9Label(r)
		c := cells[k]
		if c == nil {
			c = &r9Cell{ranks: map[int]int{}}
			cells[k] = c
		}
		c.n++
		c.ranks[r8RankInLife(ranks, lives, r.Slot, r.TSUS)]++
		epk := fmt.Sprintf("%s|%d", k, r.Slot)
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
	t.Logf("  i57 biped-spartan-ability — tag 3 SCINDE par le bit `a`")
	t.Logf("    %-20s %8s %8s %8s %8s   %s",
		"cellule", "lectures", "episodes", "medPic", "p90Pic", "rangs i48 (rang x n)")
	for _, k := range keys {
		c := cells[k]
		t.Logf("    %-20s %8d %8d %8.2f %8.2f   %v", k, c.n, c.episodes,
			r8Quantile(c.peaks, 0.5), r8Quantile(c.peaks, 0.9), r8RankSummary(c.ranks))
	}
}

// r9LogParVie publie LE DENOMINATEUR : episodes par VIE, par rang porte, pour chaque cellule
// de tag. C'est la forme de tableau qui a tranche le propulseur au par. 8.8 de R8, et c'est
// elle qui tranche ici : un rang tres porte et muet, ou un rang rare et bavard.
func r9LogParVie(t *testing.T, i57 []r9Read, ranks []AbilityRank, lives map[uint32][]r8LifeSpan) {
	t.Helper()
	vies := map[int]int{}
	for slot, spans := range lives {
		for _, sp := range spans {
			vies[r8RankInLife(ranks, lives, slot, sp.to)]++
		}
	}
	for _, cell := range []string{"tag=1", "tag=3/a=0 (porte)", "tag=3/a=1 (RUNTIME)"} {
		ep := map[int]int{}
		last := map[string]uint64{}
		for _, r := range i57 {
			k := r9Label(r)
			if k != cell && !(cell == "tag=1" && r.Tag == 1) {
				continue
			}
			rk := r8RankInLife(ranks, lives, r.Slot, r.TSUS)
			ek := fmt.Sprintf("%d|%d", rk, r.Slot)
			if p, ok := last[ek]; !ok || r.TSUS-p > r8MobEpisodeGapUS {
				ep[rk]++
			}
			last[ek] = r.TSUS
		}
		t.Logf("    episodes par VIE — cellule %s", cell)
		keys := make([]int, 0, len(vies))
		for k := range vies {
			keys = append(keys, k)
		}
		sort.Ints(keys)
		for _, k := range keys {
			if vies[k] == 0 {
				continue
			}
			t.Logf("      rang %-4d vies=%-5d episodes=%-5d %.3f par vie",
				k, vies[k], ep[k], float64(ep[k])/float64(vies[k]))
		}
	}
}

// r9LogVoisins repond a la question propre au REPULSEUR : il pousse LES AUTRES. A l'instant
// d'une lecture `tag==3, a==1`, un bipede a moins de 6 m montre-t-il une bouffee ? La
// comparaison se fait contre le temoin aleatoire apparie du meme film.
func r9LogVoisins(t *testing.T, i57 []r9Read, ranks []AbilityRank,
	lives map[uint32][]r8LifeSpan, pos map[uint32][]BipedPosition, speeds r8SpeedIndex) {
	t.Helper()
	for _, cell := range []string{"tag=3/a=1 (RUNTIME)", "tag=3/a=0 (porte)"} {
		var peaks []float64
		voisi := map[int]int{}
		seuls, n := 0, 0
		for _, r := range i57 {
			if r9Label(r) != cell {
				continue
			}
			me, ok := r8NearestAt(pos[r.Slot], r.TSUS)
			if !ok {
				continue
			}
			n++
			found := 0
			for slot, list := range pos {
				if slot == r.Slot {
					continue
				}
				o, ok := r8NearestAt(list, r.TSUS)
				if !ok || r8Dist2(float64(me.X), float64(me.Y), float64(o.X), float64(o.Y)) >
					r8NeighbourRadiusFilmM {
					continue
				}
				found++
				voisi[r8RankInLife(ranks, lives, slot, r.TSUS)]++
				if p, k := speeds.peak(slot, r.TSUS, r8PeakWindowUS); k > 0 {
					peaks = append(peaks, p)
				}
			}
			if found == 0 {
				seuls++
			}
		}
		t.Logf("    voisinage %s : n=%d sansVoisin=%d voisins=%d medPic=%.2f p90=%.2f rangs %v",
			cell, n, seuls, len(peaks), r8Quantile(peaks, 0.5), r8Quantile(peaks, 0.9),
			r8RankSummary(voisi))
	}
}

// r9LogTemoinI59 publie LE TEMOIN POSITIF : le tag 3 d'i59 doit retrouver le grappin. Sans
// lui, un zero sur le repulseur ne prouverait rien.
func r9LogTemoinI59(t *testing.T, i59 []r8TagRead, ranks []AbilityRank,
	lives map[uint32][]r8LifeSpan) {
	t.Helper()
	byTag := map[uint32]map[int]int{}
	for _, r := range i59 {
		if r.Tag != 3 {
			continue
		}
		m := byTag[r.Tag]
		if m == nil {
			m = map[int]int{}
			byTag[r.Tag] = m
		}
		m[r8RankInLife(ranks, lives, r.Slot, r.TSUS)]++
	}
	for tag, m := range byTag {
		t.Logf("    TEMOIN POSITIF i59 tag=%d : rangs %v", tag, r8RankSummary(m))
	}
}
