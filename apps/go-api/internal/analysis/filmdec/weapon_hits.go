package filmdec

// weapon_hits.go — NUMERATEUR de precision par arme, depuis le film (Lot 2 du plan
// PRECISION_ARME_DISTANCE). Attribution PAR LE TIR (NOTE_ATTRIBUTION_ARME_TIR_2026-08-31) :
// chaque tir action_weapon_fire (0xD2 type 36, WeaponID + attaquant lisibles) est apparie au
// damage_aftermath (0xC0 type 0) du MEME attaquant (ref0 du tir == ref1 du degat, domaine-1)
// dans une fenetre W. Un tir apparie a >=1 degat = une TOUCHE (un tir -> une touche, pas le
// volume de degats). La distance tireur<->victime au ts du degat alimente un histogramme.
//
// Cette logique vivait comme instrument de recherche dans lot1_attrib_arme_tir_research_test.go
// (attribCollectShots / attribBuildIndex / attribM2 / attribM3) ; elle est ici PRODUCTIONISEE
// pour que la passe killcollector (Lot 3) l'appelle hors test. Les instruments de recherche
// APPELLENT desormais ce code (PairWeaponHits, ScanFilmWeaponShots, ScanFilmWeaponDamages) :
// une seule copie du pairing (CLAUDE.md regle 6).
//
// Classe capturee : degat DIRECT (ref1 = joueur, damage_aftermath). La voie explosive/projectile
// (detonation estimee) est HORS de ce Lot (Phase 2, plan §9).

import (
	"fmt"
	"sort"
)

// WeaponHitPairWindowUS est la fenetre d'appariement W = 1 s : le degat est horodate a l'IMPACT,
// pas au tir. Verdict de fiabilite = RATIO au temoin decale (l'instrument de recherche le mesure).
const WeaponHitPairWindowUS uint64 = 1_000_000

// WeaponHitDistanceEdges : bornes (metres) des tranches de distance tireur<->victime. Identiques
// aux bornes de l'instrument d'attribution et a celles documentees par la table
// match_weapon_hit_distance (migration Lot 1).
var WeaponHitDistanceEdges = []float64{2, 5, 10, 15, 25, 40}

// WeaponHitBucket rend l'index de tranche d'une distance (0..len(edges)).
func WeaponHitBucket(d float64) int {
	for i, e := range WeaponHitDistanceEdges {
		if d < e {
			return i
		}
	}
	return len(WeaponHitDistanceEdges)
}

// WeaponHitBucketCount est le nombre de tranches de l'histogramme (edges + 1).
func WeaponHitBucketCount() int { return len(WeaponHitDistanceEdges) + 1 }

// WeaponShot est un tir action_weapon_fire (0xD2 type 36) LONG horodate.
type WeaponShot struct {
	TimestampUS uint64
	// Attacker est l'index brut de l'attaquant (ref0 domaine-1), MEME espace que le responsable
	// (ref1) d'un damage_aftermath — c'est le pont d'appariement.
	Attacker uint64
	// WeaponID est l'identifiant global 64 bits de l'arme (cle metadata.weapon_labels).
	WeaponID uint64
	// FilmIndex est l'index de tireur INTERNE AU FILM, SUR SA LARGEUR REELLE (5 bits, ShooterIndex5) :
	// l'identite reste le xuid, resolu par l'appelant (Lot 3) via resolvePlayerIndices, LUI AUSSI
	// keye sur ce 5 bits. C'est le DENOMINATEUR (shared.match_weapon_shots) qui impose la largeur :
	// il key sur analysis.PlayerIndex5 (5 bits). Un 4 bits (ancien decodeFireEvent.FilmIndex) SATURE
	// a 15 au-dela de 16 joueurs (BTB) et pointerait un AUTRE joueur que le denominateur -> precision
	// fausse. Num et denom keyent desormais IDENTIQUE (Lot 3, reserve levee).
	FilmIndex int
	// HasPair indique que l'attaquant ET le WeaponID sont lisibles (tir appariable).
	HasPair bool
}

// WeaponDamage est un damage_aftermath (0xC0 type 0) horodate, references d'en-tete non resolues.
type WeaponDamage struct {
	TimestampUS uint64
	// VictimIdx (ref0, blesse) et ResponsibleIdx (ref1, attaquant) sont des index bruts domaine-1 ;
	// -1 si la reference est absente. base512 + idx -> slot bipede -> position.
	VictimIdx      int
	ResponsibleIdx int
	Source         uint64
	HasSource      bool
	Negative       bool    // soin (Kscale = -1)
	MagClear       float64 // magnitude en clair (signee)
	MagRaw         uint64  // code magnitude R(5)
}

// WeaponHitStats agrege, par (index tireur film, WeaponID) : tirs appariables, touches et
// histogramme de distance. DistBuckets a toujours WeaponHitBucketCount() elements.
type WeaponHitStats struct {
	FilmIndex   int
	WeaponID    uint64
	ShotsPaired int   // tirs appariables (HasPair) de cette cle
	Hits        int   // tirs apparies a >=1 degat du meme attaquant dans W
	DistBuckets []int // histogramme des distances des touches (positions resolues)
}

// WeaponHitDistanceFunc resout la distance tireur<->victime (m) d'un degat apparie ; ok=false si
// l'une des deux positions ne se resout pas (touche comptee, distance non). Injectee : la
// resolution des positions (bornes carte, base slot) est propre a l'appelant.
type WeaponHitDistanceFunc func(d WeaponDamage) (meters float64, ok bool)

// dmgSlot : un degat indexe par ts pour la recherche du plus proche.
type dmgSlot struct {
	ts  uint64
	dmg WeaponDamage
}

// PairWeaponHits apparie tirs et degats (fenetre window, cle = attaquant du tir == responsable du
// degat) et rend les stats par (FilmIndex, WeaponID). Fonction PURE : dist peut etre nil (aucun
// bucket rempli). Un tir -> au plus une touche (le degat le plus proche dans la fenetre).
func PairWeaponHits(shots []WeaponShot, damages []WeaponDamage, window uint64, dist WeaponHitDistanceFunc) []WeaponHitStats {
	byResp := map[uint64][]dmgSlot{}
	for _, d := range damages {
		if d.ResponsibleIdx < 0 {
			continue
		}
		k := uint64(d.ResponsibleIdx)
		byResp[k] = append(byResp[k], dmgSlot{ts: d.TimestampUS, dmg: d})
	}
	for k := range byResp {
		s := byResp[k]
		sort.Slice(s, func(a, b int) bool { return s[a].ts < s[b].ts })
		byResp[k] = s
	}

	type statKey struct {
		fidx int
		wid  uint64
	}
	acc := map[statKey]*WeaponHitStats{}
	var order []statKey
	nb := WeaponHitBucketCount()
	for _, s := range shots {
		if !s.HasPair {
			continue
		}
		k := statKey{fidx: s.FilmIndex, wid: s.WeaponID}
		st := acc[k]
		if st == nil {
			st = &WeaponHitStats{FilmIndex: s.FilmIndex, WeaponID: s.WeaponID, DistBuckets: make([]int, nb)}
			acc[k] = st
			order = append(order, k)
		}
		st.ShotsPaired++
		md, ok := nearestDamage(byResp[s.Attacker], s.TimestampUS, window)
		if !ok {
			continue
		}
		st.Hits++
		if dist != nil {
			if m, okd := dist(md); okd {
				st.DistBuckets[WeaponHitBucket(m)]++
			}
		}
	}

	out := make([]WeaponHitStats, 0, len(order))
	for _, k := range order {
		out = append(out, *acc[k])
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ShotsPaired != out[j].ShotsPaired {
			return out[i].ShotsPaired > out[j].ShotsPaired
		}
		if out[i].FilmIndex != out[j].FilmIndex {
			return out[i].FilmIndex < out[j].FilmIndex
		}
		return out[i].WeaponID < out[j].WeaponID
	})
	return out
}

// nearestDamage rend le degat de la liste (triee par ts) le plus proche de T dans [T-W, T+W].
func nearestDamage(slots []dmgSlot, T, W uint64) (WeaponDamage, bool) {
	i := sort.Search(len(slots), func(i int) bool { return slots[i].ts >= T })
	best, ok := WeaponDamage{}, false
	bd := ^uint64(0)
	consider := func(j int) {
		if j < 0 || j >= len(slots) {
			return
		}
		d := T - slots[j].ts
		if slots[j].ts > T {
			d = slots[j].ts - T
		}
		if d <= W && d < bd {
			best, ok, bd = slots[j].dmg, true, d
		}
	}
	consider(i - 1)
	consider(i)
	return best, ok
}

// ScanFilmWeaponShots decode les tirs LONGS 0xD2 (type 36) des chunks 1..n : attaquant (ref0
// dom1) et WeaponID / index tireur (decodeFireEvent, offsets fixes). Un seul decodeur par champ.
// L'ordre du film est conserve ; les tirs non lisibles sont emis avec HasPair=false (comptes dans
// le total, ecartes du pairing). Le verrou de process de decode est a l'appelant (cf.
// ScanFilmFireEvents).
func ScanFilmWeaponShots(dir string, n int) ([]WeaponShot, error) {
	var out []WeaponShot
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			return nil, fmt.Errorf("chunk_%02d illisible : %w", c, err)
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 4 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xD2 { // type 36 variante LONGUE (porte l'arme)
				continue
			}
			s := WeaponShot{TimestampUS: pk.TimestampUS}
			br := NewBitReader(pay)
			br.Skip(2)
			if br.ReadBits(7) != 36 {
				continue
			}
			att, okA := lot1RefDom1(br) // ref0 = attaquant (dom1)
			fe, okF := decodeFireEvent(pay)
			if okA && okF {
				// La cle de tireur est le 5 bits (ShooterIndex5), PAS le 4 bits FilmIndex : le pont
				// FilmIndex->xuid (resolvePlayerIndices) et le denominateur (match_weapon_shots) sont
				// keyes sur ce 5 bits. Voir WeaponShot.FilmIndex et fire_events.go:fireShooterBit.
				s.Attacker, s.WeaponID, s.FilmIndex, s.HasPair = att, fe.WeaponID, fe.ShooterIndex5, true
			}
			out = append(out, s)
		}
	}
	return out, nil
}

// ScanFilmWeaponDamages rejoue les chunks 1..n (keyframe + tick-frames comme le decodeur de
// trame) puis decode les damage_aftermath (0xC0 type 0) : refs d'en-tete domaine-1 (blesse ref0,
// responsable ref1), source et magnitude. Rend aussi la BASE d'atterrissage bipede (resolution
// slot base-512), argmax des slots lies a un bipede — servant la resolution de position pour la
// distance. Verrou de process = appelant.
func ScanFilmWeaponDamages(dir string, reg *Registry, n int) ([]WeaponDamage, int, error) {
	cfg := DefaultFrameConfig()
	hit := map[int]int{}
	var out []WeaponDamage
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			return nil, 0, fmt.Errorf("chunk_%02d illisible : %w", c, err)
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
		out = scanChunkDamages(pks, data, w, hit, out)
	}
	return out, lot1ArgmaxBase(hit), nil
}

// scanChunkDamages decode les damage_aftermath d'un chunk et accumule l'atterrissage bipede par
// base (extrait de ScanFilmWeaponDamages pour tenir le seuil de 80 lignes / fonction).
func scanChunkDamages(pks []FilmPacket, data []byte, w *World, hit map[int]int, out []WeaponDamage) []WeaponDamage {
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
			continue // type 1 (damage_section_response), pas type 0
		}
		d := WeaponDamage{TimestampUS: pk.TimestampUS, VictimIdx: -1, ResponsibleIdx: -1}
		if i0, ok := lot1RefDom1(br); ok {
			d.VictimIdx = int(i0)
		}
		if i1, ok := lot1RefDom1(br); ok {
			d.ResponsibleIdx = int(i1)
		}
		lot1RefDom(br, 7)
		r := lot1DecodeDamageAftermath(br)
		d.Source, d.HasSource, d.Negative = r.sourceID, r.hasSource, r.negatif
		d.MagClear, d.MagRaw = r.dmgClear, r.dmgRaw
		out = append(out, d)
		for _, b := range lot1chBases {
			if lot1chIsBiped(w, b, d.VictimIdx) {
				hit[b]++
			}
			if lot1chIsBiped(w, b, d.ResponsibleIdx) {
				hit[b]++
			}
		}
	}
	return out
}
