package filmdec

// zone_census_scan_test.go — LE BALAYAGE (lot C phase 0). Une seule passe sur les paquets
// delta du film sert TOUTES les mesures de l'item C.0.1 (3) et de l'item C.0.3 : records
// reconnus par archetype, annonces au masque par index de composant, et repartition de ces
// annonces DANS et HORS des fenetres d'evenement d'objectif.
//
// POURQUOI UNE SEULE PASSE. Le balayage par ancrage teste chaque position de bit du payload :
// c'est lui qui coute (la machine de l'utilisateur paie, D17). Sept bandes balayees
// separement coutaient sept fois ce prix ; une bande UNION plus une table slot -> archetype
// rend exactement la meme mesure pour un seul parcours.
//
// L'HORLOGE est celle d'`objectiveevents.StatRecords` (statborg.go) et pas une autre :
// `tMS = manifeste.start_ms[chunk] + (us_du_paquet - us_du_PREMIER_paquet_delta_du_chunk)/1000`.
// Sans cette identite de base, comparer des annonces a des evenements nommes n'aurait aucun
// sens.

import (
	"sort"
)

// zcDeltaStats agrege ce qu'une bande a rendu sur les paquets delta.
type zcDeltaStats struct {
	// records est le nombre de records d'en-tete « objet du monde » reconnus sur la bande.
	records int
	// slots est le nombre de slots de la bande effectivement peuples.
	slots map[uint32]bool
	// byIndex[i] compte les records dont le masque ANNONCE le composant i (le recensement
	// d'annonces, modele `EquipmentStateStats.MaskCensus`).
	byIndex [worldObjectMaxComponent]int
	// inWin[i] / outWin[i] : les memes annonces, selon que le paquet tombe dans une fenetre
	// d'evenement d'objectif ou non. La fenetre est celle du GATE (+/- zcWindowMS).
	inWin, outWin [worldObjectMaxComponent]int
	// inTight[i] / outTight[i] : la meme repartition sous une fenetre SERREE
	// (+/- zcTightMS). DIAGNOSTIC et rien d'autre : le gate 0 se juge sur inWin/outWin, et
	// une fenetre choisie apres la mesure ne peut pas etre un critere.
	inTight, outTight [worldObjectMaxComponent]int
	// firstIndex[i] : records dont i est le PREMIER index annonce (c'est lui qui decide ou
	// la traversee canonique s'arrete).
	firstIndex map[int]int
	// maskCount[n] : records annoncant n composants.
	maskCount map[int]int
	// outOfGrammar : records dont le masque porte un index hors de la grammaire de
	// l'archetype — le controle de purete de la bande.
	outOfGrammar int
}

func newZCDeltaStats() *zcDeltaStats {
	return &zcDeltaStats{slots: map[uint32]bool{}, firstIndex: map[int]int{}, maskCount: map[int]int{}}
}

// zcScanResult porte le resultat de la passe : les stats par archetype, celles des trois
// temoins, et les denominateurs de temps.
type zcScanResult struct {
	// byClass[c] : les stats de la classe c (un ti >= 0, ou une classe de controle < 0).
	byClass map[int]*zcDeltaStats
	// paquets delta lus, et ceux qui tombent dans une fenetre d'evenement.
	packets, packetsInWin int
	// bornes de l'horloge de match couverte par les paquets delta (ms).
	tMinMS, tMaxMS int
	// secondes couvertes par l UNION des fenetres, et hors fenetre (gate, puis diagnostic).
	secInWin, secOutWin     float64
	secInTight, secOutTight float64
	// paquets sans horodatage convertible (chunk absent du manifeste) — dit, jamais tu.
	packetsNoClock int
}

// zcClock convertit l'horodatage moteur d'un paquet en millisecondes de match, sur la MEME
// base que `objectiveevents.StatRecords`.
type zcClock struct {
	// startMS[chunk] vient du manifeste du film.
	startMS map[int]int
}

// zcWindows est l ensemble des instants d evenement d objectif, trie, avec sa demi-largeur.
// `has(t)` dit si t tombe a moins de `halfMS` d'un evenement.
type zcWindows struct {
	times  []int
	halfMS int
}

func newZCWindows(times []int, halfMS int) zcWindows {
	cp := make([]int, len(times))
	copy(cp, times)
	sort.Ints(cp)
	return zcWindows{times: cp, halfMS: halfMS}
}

// has dit si l'instant t (ms) tombe dans la fenetre +/- halfMS d'au moins un evenement.
func (w zcWindows) has(t int) bool {
	if len(w.times) == 0 {
		return false
	}
	i := sort.SearchInts(w.times, t)
	if i < len(w.times) && w.times[i]-t <= w.halfMS {
		return true
	}
	if i > 0 && t-w.times[i-1] <= w.halfMS {
		return true
	}
	return false
}

// coveredSeconds rend la duree, en secondes, de l'UNION des fenetres, bornee a [lo, hi].
// C'est le denominateur de la densite « dans la fenetre » : sans lui, comparer 40 annonces
// a 4 000 ne dirait rien.
func (w zcWindows) coveredSeconds(lo, hi int) float64 {
	if len(w.times) == 0 || hi <= lo {
		return 0
	}
	total := 0
	curLo, curHi := 0, 0
	open := false
	for _, t := range w.times {
		a, b := t-w.halfMS, t+w.halfMS
		if a < lo {
			a = lo
		}
		if b > hi {
			b = hi
		}
		if b <= a {
			continue
		}
		if !open {
			curLo, curHi, open = a, b, true
			continue
		}
		if a > curHi {
			total += curHi - curLo
			curLo, curHi = a, b
			continue
		}
		if b > curHi {
			curHi = b
		}
	}
	if open {
		total += curHi - curLo
	}
	return float64(total) / 1000
}

// zcScanDelta balaye TOUS les paquets delta du film en une passe et agrege les annonces au
// masque par archetype, avec leur repartition dans / hors fenetre.
//
// `grammarLen[ti]` est le nombre de composants de l'archetype dans le registre du FILM : il
// sert au controle de purete (un record de ti=10 ne peut pas annoncer i40).
func zcScanDelta(c zcCensus, b zcBands, clk zcClock, win zcWindows, grammarLen map[int]int) zcScanResult {
	res := zcScanResult{byClass: map[int]*zcDeltaStats{}, tMinMS: -1, tMaxMS: -1}
	for _, ti := range zcTargetTIs {
		res.byClass[ti] = newZCDeltaStats()
	}
	for _, cl := range []int{zcClassInconnu, zcClassOccupe, zcClassVide} {
		res.byClass[cl] = newZCDeltaStats()
	}
	tight := newZCWindows(win.times, zcTightMS)
	ctx := zcScanCtx{bands: b, clock: clk, win: win, tight: tight, grammar: grammarLen}
	for ch := 1; ch <= c.chunks; ch++ {
		data, err := ReadFilmChunk(c.dir, ch)
		if err != nil {
			continue
		}
		zcScanChunk(data, ch, &res, ctx)
	}
	if res.tMinMS >= 0 && res.tMaxMS > res.tMinMS {
		res.secInWin = win.coveredSeconds(res.tMinMS, res.tMaxMS)
		res.secOutWin = float64(res.tMaxMS-res.tMinMS)/1000 - res.secInWin
		res.secInTight = tight.coveredSeconds(res.tMinMS, res.tMaxMS)
		res.secOutTight = float64(res.tMaxMS-res.tMinMS)/1000 - res.secInTight
	}
	return res
}

// zcTightMS est la demi-fenetre du DIAGNOSTIC serre. Elle ne juge rien : elle dit seulement
// si un canal se concentre PLUS pres de l'evenement que la fenetre du gate ne le montre —
// autrement dit si un negatif sous +/- 3 s vient du canal ou de la largeur de la fenetre.
const zcTightMS = 1000

// zcScanCtx groupe ce qu'une passe de chunk doit connaitre (regle des 5 parametres).
type zcScanCtx struct {
	bands   zcBands
	clock   zcClock
	win     zcWindows
	tight   zcWindows
	grammar map[int]int
}

// zcScanChunk balaye les paquets delta d'UN chunk. L'ancrage du premier paquet delta du
// chunk sert d'origine a l'horloge, exactement comme `StatRecords`.
func zcScanChunk(data []byte, ch int, res *zcScanResult, ctx zcScanCtx) {
	var base uint64
	haveBase := false
	startMS, hasStart := ctx.clock.startMS[ch]
	for _, pk := range WalkPackets(data) {
		if pk.Type != PacketTypeDelta {
			continue
		}
		if !haveBase {
			base, haveBase = pk.TimestampUS, true
		}
		res.packets++
		tMS, okT := 0, false
		if hasStart {
			tMS, okT = startMS+int((pk.TimestampUS-base)/1000), true
			if res.tMinMS < 0 || tMS < res.tMinMS {
				res.tMinMS = tMS
			}
			if tMS > res.tMaxMS {
				res.tMaxMS = tMS
			}
		} else {
			res.packetsNoClock++
		}
		w := zcPacketWindow{okT: okT, inWin: okT && ctx.win.has(tMS), inTight: okT && ctx.tight.has(tMS)}
		if w.inWin {
			res.packetsInWin++
		}
		zcScanPayload(pk.Payload(data), res, ctx, w)
	}
}

// zcScanPayload balaye UN payload delta. Il n'y a pas de raccourci possible : le balayage
// par ancrage teste chaque position de bit. Une seule reconnaissance par position suffit
// pour les quatre bandes, puisqu'une position ne rend qu'une valeur de slot.
func zcScanPayload(pay []byte, res *zcScanResult, ctx zcScanCtx, w zcPacketWindow) {
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, ctx.bands.all)
		if !ok {
			continue
		}
		cl := ctx.bands.class[rec.Slot]
		zcAccumulate(res.byClass[cl], rec, ctx.grammar[cl], w)
		p = rec.After
	}
}

// zcPacketWindow dit ou UN paquet tombe sur la frise : horloge exploitable, fenetre du gate,
// fenetre serree du diagnostic.
type zcPacketWindow struct {
	okT, inWin, inTight bool
}

// zcAccumulate range UN record reconnu. `grammar` = nombre de composants de l'archetype
// (0 = pas de controle de purete, cas des bandes de controle).
func zcAccumulate(s *zcDeltaStats, rec WorldObjectRecord, grammar int, w zcPacketWindow) {
	if s == nil {
		return
	}
	s.records++
	s.slots[rec.Slot] = true
	s.maskCount[len(rec.Idx)]++
	s.firstIndex[rec.Idx[0]]++
	impur := false
	for _, i := range rec.Idx {
		if i < 0 || i >= worldObjectMaxComponent {
			continue
		}
		s.byIndex[i]++
		if w.okT {
			if w.inWin {
				s.inWin[i]++
			} else {
				s.outWin[i]++
			}
			if w.inTight {
				s.inTight[i]++
			} else {
				s.outTight[i]++
			}
		}
		if grammar > 0 && i >= grammar {
			impur = true
		}
	}
	if impur {
		s.outOfGrammar++
	}
}

// zcRecordsPerSlot rend le debit par slot peuple — la grandeur qui rend les quatre bandes
// comparables malgre des cardinalites differentes.
func zcRecordsPerSlot(s *zcDeltaStats) float64 {
	if s == nil || len(s.slots) == 0 {
		return 0
	}
	return float64(s.records) / float64(len(s.slots))
}

// zcNoiseFloor estime le PLANCHER DE BRUIT d'une bande : la MEDIANE des annonces sur les 64
// index de composant possibles.
//
// POURQUOI CETTE GRANDEUR, et pourquoi elle est indispensable ici. L'en-tete « objet du
// monde » ne contraint que 21 bits, dont la bande de slots ne fournit qu'une poignee : sur un
// balayage de plusieurs centaines de millions de positions de bit, il tombe juste par HASARD
// un grand nombre de fois, et un tirage au hasard repartit les index de composant a peu pres
// uniformement sur 0..63. La mediane des 64 index mesure donc directement ce fond, DANS la
// bande elle-meme et sans hypothese exterieure. Un composant reellement annonce s'en detache
// d'un facteur ; un composant qui ne se detache pas n'est pas mesure — il est indistinguable
// du bruit, et c'est ce qu'il faut ecrire.
func zcNoiseFloor(s *zcDeltaStats) float64 {
	if s == nil {
		return 0
	}
	v := make([]int, 0, worldObjectMaxComponent)
	for i := 0; i < worldObjectMaxComponent; i++ {
		v = append(v, s.byIndex[i])
	}
	sort.Ints(v)
	mid := len(v) / 2
	return (float64(v[mid-1]) + float64(v[mid])) / 2
}

// zcExcess rend le facteur d'un composant au-dessus du plancher de bruit de sa bande.
func zcExcess(count int, floor float64) float64 {
	if floor <= 0 {
		return 0
	}
	return float64(count) / floor
}
