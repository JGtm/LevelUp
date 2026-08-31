package filmdec

// lot1_degats_blesse_research_test.go — LOT 1 : DEPARTAGER, dans damage_aftermath (0xC0
// type 0), laquelle des DEUX references d'en-tete domaine-1 est le BLESSE (touche) et
// laquelle est le RESPONSABLE (attaquant). Le record porte deux references qui resolvent
// toutes deux en slots de bipede ; le juge est la VITALITE (sante i4 + bouclier i5,
// offline_biped/CaptureDirs) du slot resolu : elle BAISSE autour de l'instant de l'evenement
// pour le BLESSE, l'ATTAQUANT ne perd rien en frappant.
//
// SEUILS ECRITS AVANT LA MESURE :
//
//	W       = 500 ms : pour chaque composante (sante, bouclier), avant = derniere mesure
//	          avec ts < T ; apres = premiere mesure avec ts >= T ; chacune dans [T-W, T+W].
//	          Le delta de vitalite = somme des deltas des composantes disponibles.
//	dropThr = 0.05 (5 % d'une barre) : delta <= -dropThr compte comme une BAISSE.
//	margin  = 0.05 : en tete-a-tete (les deux refs resolues avec vitalite), le BLESSE est la
//	          ref dont la baisse depasse l'autre d'au moins margin ; sinon l'evenement est
//	          ambigu (ecarte du vote).
//	base    : choisie par ref comme dans victime_slot — la base qui maximise les slots LIES
//	          A UN BIPEDE dans le monde (fin de chunk) resout la ref -> slot.
//
// TEMOIN (piste iii) : sur les evenements a magnitude NEGATIVE (soin, Kscale=-1), la
// vitalite du slot resolu doit MONTER (delta > 0), pas chuter — le soin se cible soi-meme.
//
// CHIFFRE RENDU : part des evenements en tete-a-tete ou ref0 (resp. ref1) est le slot dont
// la vitalite baisse le plus. La ref gagnante est le BLESSE.
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule.

import (
	"os"
	"sort"
	"testing"
)

// lot1cSample : une mesure horodatee d'une composante de vitalite (sante OU bouclier).
type lot1cSample struct {
	ts uint64
	v  float64 // fraction [0,1]
}

// lot1DmgEvt : un evenement damage_aftermath horodate, references non resolues.
type lot1DmgEvt struct {
	ts         uint64
	idx0, idx1 int // -1 si la ref est absente
	negatif    bool
}

// lot1VitTL porte, par slot, les deux timelines de composantes (sante, bouclier).
type lot1VitTL struct {
	health map[uint32][]lot1cSample
	shield map[uint32][]lot1cSample
}

// lot1BuildVitTimeline balaie les positions bipedes (QuantaOnly + CaptureDirs) et rend, par
// slot, les timelines CHRONOLOGIQUES de sante et de bouclier (chacune sur ses seuls records
// porteurs), plus les compteurs de couverture.
func lot1BuildVitTimeline(t *testing.T, dir string, chunks []int) (lot1VitTL, int, int) {
	opt := DefaultScanFilmOptions()
	opt.QuantaOnly = true
	opt.CaptureDirs = true
	opt.IsolationGapMS = 0 // garder toutes les mesures : ne pas jeter l'echantillon post-degat
	opt.Chunks = chunks
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("balayage biped impossible : %v", err)
	}
	tl := lot1VitTL{health: map[uint32][]lot1cSample{}, shield: map[uint32][]lot1cSample{}}
	nH, nS := 0, 0
	for _, p := range pos {
		if h, ok := p.HealthAt(); ok {
			tl.health[p.Slot] = append(tl.health[p.Slot], lot1cSample{ts: p.TimestampUS, v: float64(h)})
			nH++
		}
		if s, ok := p.ShieldAt(); ok {
			tl.shield[p.Slot] = append(tl.shield[p.Slot], lot1cSample{ts: p.TimestampUS, v: float64(s)})
			nS++
		}
	}
	sortTL := func(m map[uint32][]lot1cSample) {
		for slot := range m {
			ss := m[slot]
			sort.Slice(ss, func(i, j int) bool { return ss[i].ts < ss[j].ts })
			m[slot] = ss
		}
	}
	sortTL(tl.health)
	sortTL(tl.shield)
	return tl, nH, nS
}

// lot1CompDelta rend la variation d'UNE composante autour de T (apres - avant), dans W, et sa
// validite (une mesure de part et d'autre de T).
func lot1CompDelta(tl []lot1cSample, T, W uint64) (float64, bool) {
	if len(tl) == 0 {
		return 0, false
	}
	i := sort.Search(len(tl), func(i int) bool { return tl[i].ts >= T })
	var before, after *lot1cSample
	if i-1 >= 0 && tl[i-1].ts < T && T-tl[i-1].ts <= W {
		before = &tl[i-1]
	}
	if i < len(tl) && tl[i].ts >= T && tl[i].ts-T <= W {
		after = &tl[i]
	}
	if before == nil || after == nil {
		return 0, false
	}
	return after.v - before.v, true
}

// lot1VitDelta somme les deltas des composantes disponibles pour un slot autour de T.
func (tl lot1VitTL) delta(slot uint32, T, W uint64) (float64, bool) {
	d, ok := 0.0, false
	if hd, okh := lot1CompDelta(tl.health[slot], T, W); okh {
		d += hd
		ok = true
	}
	if sd, oks := lot1CompDelta(tl.shield[slot], T, W); oks {
		d += sd
		ok = true
	}
	return d, ok
}

// lot1WorldBaseAndEvents rejoue les chunks (keyframe + tick-frames) comme victime_slot, rend
// les evenements damage_aftermath horodates et UNE base partagee. ref0 et ref1 sont deux
// references de la MEME table domaine-1 (descripteur 0x1451f98d0) : elles partagent donc la
// base. La bande bipede etant contigue, l'argmax "atterrit sur un bipede" par ref est
// instable entre bases voisines (510/512) ; mettre ref0 et ref1 EN COMMUN stabilise le choix.
func lot1WorldBaseAndEvents(t *testing.T, dir string, reg *Registry, n int) ([]lot1DmgEvt, int, int, int) {
	cfg := DefaultFrameConfig()
	hit0, hit1 := map[int]int{}, map[int]int{}
	present0, present1 := 0, 0
	var events []lot1DmgEvt
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		pks := WalkPackets(data)
		w := NewWorld(reg)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if pay := pk.Payload(data); pay[0]&0x40 == 0 {
				br := NewBitReader(pay)
				_, _ = DecodeFrameRecords(br, w, cfg)
			}
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xC0 {
				continue
			}
			br := NewBitReader(pay)
			br.Skip(2)
			if br.ReadBits(7) != 0 {
				continue
			}
			ev := lot1DmgEvt{ts: pk.TimestampUS, idx0: -1, idx1: -1}
			if i0, ok0 := lot1RefDom1(br); ok0 {
				ev.idx0 = int(i0)
			}
			if i1, ok1 := lot1RefDom1(br); ok1 {
				ev.idx1 = int(i1)
			}
			lot1RefDom(br, 7)
			ev.negatif = lot1DecodeDamageAftermath(br).negatif
			events = append(events, ev)
			if ev.idx0 >= 0 {
				present0++
				for _, b := range lot1chBases {
					if lot1chIsBiped(w, b, ev.idx0) {
						hit0[b]++
					}
				}
			}
			if ev.idx1 >= 0 {
				present1++
				for _, b := range lot1chBases {
					if lot1chIsBiped(w, b, ev.idx1) {
						hit1[b]++
					}
				}
			}
		}
	}
	combined := map[int]int{}
	for _, b := range lot1chBases {
		combined[b] = hit0[b] + hit1[b]
	}
	return events, lot1ArgmaxBase(combined), present0, present1
}

// lot1ArgmaxBase rend la base a l'atterrissage bipede maximal (base la plus basse en cas d'egalite).
func lot1ArgmaxBase(hits map[int]int) int {
	best, bestN := lot1chBases[0], -1
	for _, b := range lot1chBases {
		if hits[b] > bestN {
			best, bestN = b, hits[b]
		}
	}
	return best
}

func TestLot1DegatsBlesse(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	chunks := make([]int, 0, n)
	for c := 1; c <= n; c++ {
		chunks = append(chunks, c)
	}

	const (
		W       = uint64(500_000) // 500 ms
		dropThr = 0.05
		margin  = 0.05
	)

	tl, nH, nS := lot1BuildVitTimeline(t, dir, chunks)
	events, worldBase, present0, present1 := lot1WorldBaseAndEvents(t, dir, reg, n)
	t.Logf("== film : %d evenements damage_aftermath · sante %d mesures / bouclier %d mesures ==", len(events), nH, nS)
	t.Logf("ref0 presente %d · ref1 presente %d · base 'monde' (victime_slot) = %d", present0, present1, worldBase)

	// CALIBRATION DE BASE : la bande bipede etant contigue, choisir la base par "atterrit sur
	// un bipede lie" (victime_slot) est instable entre bases voisines. On retient plutot la
	// base qui resout le PLUS d'evenements a des slots PORTEURS DE VITALITE (echantillon max) —
	// les bipedes vivants sont un ensemble creux, donc ce critere discrimine, la ou un simple
	// "dans la bande" ne le fait pas. Le taux de baisse d'une base a petit echantillon (9 sur 9
	// a 100 %) est du bruit : on l'ecarte en pesant par le nombre de mesures.
	const minValid = 8
	bestBase, bestCover := worldBase, -1
	for _, b := range lot1chBases {
		ta := lot1TallyAt(events, tl, b, W, dropThr, margin)
		cover := ta.ref0Valid + ta.ref1Valid
		if cover >= minValid {
			t.Logf("  base=%d : couverture %d · ref0 baisse %d/%d (%.0f %%) · ref1 baisse %d/%d (%.0f %%)",
				b, cover, ta.ref0Drop, ta.ref0Valid, lot1Pct(ta.ref0Drop, ta.ref0Valid),
				ta.ref1Drop, ta.ref1Valid, lot1Pct(ta.ref1Drop, ta.ref1Valid))
		}
		if cover > bestCover {
			bestCover, bestBase = cover, b
		}
	}

	ta := lot1TallyAt(events, tl, bestBase, W, dropThr, margin)
	r0 := lot1Pct(ta.ref0Drop, ta.ref0Valid)
	r1 := lot1Pct(ta.ref1Drop, ta.ref1Valid)
	t.Logf("BASE CALIBREE (vitalite) = %d", bestBase)
	t.Logf("COUVERTURE : ref0 resolue+vitalite %d · ref1 %d · tete-a-tete (les deux) %d",
		ta.ref0Valid, ta.ref1Valid, ta.bothValid)
	t.Logf("BAISSE ABSOLUE : ref0 chute (delta<=-%.2f) %d/%d (%.1f %%) · ref1 %d/%d (%.1f %%)",
		dropThr, ta.ref0Drop, ta.ref0Valid, r0, ta.ref1Drop, ta.ref1Valid, r1)
	t.Logf("TETE-A-TETE (qui baisse le PLUS) : ref0 %d (%.1f %%) · ref1 %d (%.1f %%) · ambigu %d (%.1f %%)",
		ta.ref0Wins, lot1Pct(ta.ref0Wins, ta.bothValid), ta.ref1Wins, lot1Pct(ta.ref1Wins, ta.bothValid),
		ta.ambigu, lot1Pct(ta.ambigu, ta.bothValid))
	t.Logf("TEMOIN SOIN (magnitude negative) : slot resolu monte %d/%d (%.1f %%)",
		ta.negRise, ta.negRes, lot1Pct(ta.negRise, ta.negRes))
	blesse := "ref0"
	if r1 > r0 {
		blesse = "ref1"
	}
	// Tranche : le blesse franchement au-dessus de l'autre (>= 2x, ou marge >= 25 pts) ET au
	// moins minValid mesures par ref.
	hi, lo := r0, r1
	if r1 > r0 {
		hi, lo = r1, r0
	}
	decisif := ta.ref0Valid >= minValid && ta.ref1Valid >= minValid && hi >= 50 && (hi >= 2*lo || hi-lo >= 25)
	t.Logf("CONCLUSION : BLESSE (touche) = %s (baisse %.1f %% contre %.1f %%) · attaquant = l'autre · verdict tranche : %s",
		blesse, hi, lo, lot1Verdict(decisif))
}

// lot1Tally agrege les compteurs de vitalite a une base donnee.
type lot1Tally struct {
	ref0Valid, ref0Drop        int
	ref1Valid, ref1Drop        int
	bothValid                  int
	ref0Wins, ref1Wins, ambigu int
	negRes, negRise            int
}

// lot1TallyAt calcule, pour une base, la baisse de vitalite par ref et le tete-a-tete.
func lot1TallyAt(events []lot1DmgEvt, tl lot1VitTL, base int, W uint64, dropThr, margin float64) lot1Tally {
	var ta lot1Tally
	for _, e := range events {
		var d0, d1 float64
		var hd0, hd1 bool
		if e.idx0 >= 0 {
			d0, hd0 = tl.delta(uint32(base+e.idx0), e.ts, W)
		}
		if e.idx1 >= 0 {
			d1, hd1 = tl.delta(uint32(base+e.idx1), e.ts, W)
		}
		if hd0 {
			ta.ref0Valid++
			if d0 <= -dropThr {
				ta.ref0Drop++
			}
		}
		if hd1 {
			ta.ref1Valid++
			if d1 <= -dropThr {
				ta.ref1Drop++
			}
		}
		if hd0 && hd1 {
			ta.bothValid++
			switch {
			case d0 <= d1-margin:
				ta.ref0Wins++
			case d1 <= d0-margin:
				ta.ref1Wins++
			default:
				ta.ambigu++
			}
		}
		if e.negatif {
			switch {
			case hd0:
				ta.negRes++
				if d0 > dropThr {
					ta.negRise++
				}
			case hd1:
				ta.negRes++
				if d1 > dropThr {
					ta.negRise++
				}
			}
		}
	}
	return ta
}
