package filmdec

// geo_explosifs_helpers_test.go — collecteurs, types et geometrie de l'instrument
// geo_explosifs_research_test.go (scinde pour le seuil de 500 lignes). Voir l'en-tete de ce
// fichier-la pour le raisonnement, la verite terrain et les mesures.

import (
	"math"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis"
)

// geoActiveBase : base bipede DETECTEE du film courant (ref dom1 -> slot absolu = base+idx).
// Fixee par TestGeoExplosifs apres le sweep geoDetectBase ; 512 par defaut. Le paquet serialise
// le decodage (LockProcessDecode), donc un etat de paquet est sur, comme les hooks du paquet.
var geoActiveBase = geoBase

const (
	geoBase      = lot1chReferenceBase // base bipede par defaut (512, calibree 000d5950)
	geoFlightW   = uint64(2_000_000)   // 2 s : fenetre de vol tir lourd -> impact
	geoMatchWin  = uint64(400_000)     // 400 ms : touche fatale <-> mort (dead-state)
	geoPosTolUS  = sondePosTolUS       // 120 ms : ecart max evenement <-> echantillon position
	geoMinDtUS   = uint64(30_000)      // 30 ms : dt de vol minimal pour calibrer une vitesse
	geoDefSpeedU = 45.0                // vitesse projectile par defaut (unites monde/s) si non calibree
)

// geoShot : un tir 0xD2 t36 LONG horodate. att = attaquant (ref0 dom1 brut, MEME espace que
// ref1 d'un damage_aftermath -> slot via geoBase) ; film = FilmIndex (index participant, MEME
// espace que EnumB du dead-state) ; wid = WeaponID ; heavy/direct classent l'arme.
type geoShot struct {
	ts     uint64
	att    uint64
	film   int
	wid    uint64
	name   string
	heavy  bool
	direct bool // projectile a trajectoire DIRECTE (roquette/empaleur/ravageur...) vs traqueur/cloche
}

// geoTouch : un damage_aftermath 0xC0 t0 dont ref1 est NON-bipede (candidate explosive), avec
// sa victime ref0 resolue en slot (geoBase+idx0).
type geoTouch struct {
	ts       uint64
	victSlot uint32
	hasVict  bool
	mag      float64
	fatal    bool  // appariee a une mort (dead-state) de la meme victime
	killer   int32 // EnumB de la mort appariee (-1 si non fatale)
}

// geoKill : un dead-state de bipede mort, oracle. killer = EnumB (roster), victim = EnumA.
type geoKill struct {
	victSlot uint32
	victRost int32
	killer   int32
	ts       uint64
}

// geoAimSample : position monde + cap/elevation de visee (i21) horodates, pour un slot.
type geoAimSample struct {
	ts      uint64
	x, y, z float64
	heading float64
	pitch   float64
	hasAim  bool
}

// geoIsDirect : l'arme lourde tire-t-elle un projectile a trajectoire DIRECTE (l'alignement de
// visee est net) ? Faux pour les traqueurs (Hydra en verrouillage) et les tirs en cloche
// (Fuel Rod), ou la visee ne pointe pas la victime.
func geoIsDirect(name string) bool {
	for _, k := range []string{"SPNKr", "Skewer", "Shock", "Mangler", "Stalker", "Bulldog"} {
		if strings.Contains(name, k) {
			return true
		}
	}
	return false // Hydra (traqueur), Ravager/Fuel Rod/Rod (cloche) -> non direct
}

// geoWeaponName : nom d'arme par WeaponID (table statique) ou l'hexa.
func geoWeaponName(wid uint64) string {
	if nm, ok := analysis.WeaponIDToName[wid]; ok {
		return nm
	}
	return attribWeaponName(wid)
}

// geoCollectShots decode les tirs longs 0xD2 t36 : attaquant dom1 + FilmIndex + WeaponID + classe.
func geoCollectShots(t *testing.T, dir string, n int) []geoShot {
	t.Helper()
	var out []geoShot
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 4 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xD2 {
				continue
			}
			br := NewBitReader(pay)
			br.Skip(2)
			if br.ReadBits(7) != 36 {
				continue
			}
			att, okA := lot1RefDom1(br)
			fe, okF := decodeFireEvent(pay)
			if !okA || !okF {
				continue
			}
			name := geoWeaponName(fe.WeaponID)
			out = append(out, geoShot{
				ts: pk.TimestampUS, att: att, film: fe.FilmIndex, wid: fe.WeaponID,
				name: name, heavy: lot1IsHeavy(name), direct: geoIsDirect(name),
			})
		}
	}
	return out
}

// geoRawDmg : un damage_aftermath brut, avec la resolution bipede de ses deux refs sous CHAQUE
// base candidate (la base bipede est propre au film, comme la sonde de precision la calibre).
type geoRawDmg struct {
	ts         uint64
	idx0, idx1 int
	mag        float64
	bip0, bip1 []bool // aligne sur lot1chBases
}

// geoCollectDamageKills fait UNE passe par chunk : bind keyframes, rejoue la trame (qui capture
// les dead-states = oracle), puis decode les damage_aftermath 0xC0 contre le monde de fin de
// chunk en resolvant sous TOUTES les bases candidates. Rend les degats bruts et les morts.
func geoCollectDamageKills(t *testing.T, dir string, reg *Registry, n int) ([]geoRawDmg, []geoKill) {
	t.Helper()
	cfg := DefaultFrameConfig()
	var raws []geoRawDmg
	var kills []geoKill
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
			pay := pk.Payload(data)
			if pay[0]&0x40 == 0 {
				br := NewBitReader(pay)
				recs, _ := DecodeFrameRecords(br, w, cfg)
				kills = geoHarvestKills(recs, pk.TimestampUS, kills)
			}
		}
		raws = geoRawDamageInChunk(pks, data, w, raws)
	}
	return raws, kills
}

// geoHarvestKills recolte les dead-states de bipede mort (victime slot + roster EnumA + tueur EnumB).
func geoHarvestKills(recs []FrameRecord, ts uint64, kills []geoKill) []geoKill {
	for i := range recs {
		rec := &recs[i]
		if rec.TypeIndex != BipedTypeIndex || rec.Trace.Dead == nil {
			continue
		}
		d := rec.Trace.Dead
		if !d.Mort || d.EnumB < 0 {
			continue
		}
		kills = append(kills, geoKill{victSlot: rec.Slot, victRost: d.EnumA, killer: d.EnumB, ts: ts})
	}
	return kills
}

// geoRawDamageInChunk decode les 0xC0 t0 d'un chunk et resout ref0/ref1 en bipede sous CHAQUE
// base candidate (la selection de base est faite globalement, apres la collecte).
func geoRawDamageInChunk(pks []FilmPacket, data []byte, w *World, out []geoRawDmg) []geoRawDmg {
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
		i0, ok0 := lot1RefDom1(br)
		i1, ok1 := lot1RefDom1(br)
		lot1RefDom(br, 7)
		r := lot1DecodeDamageAftermath(br)
		if !ok1 {
			continue
		}
		e := geoRawDmg{ts: pk.TimestampUS, idx0: -1, idx1: int(i1), mag: r.dmgClear,
			bip0: make([]bool, len(lot1chBases)), bip1: make([]bool, len(lot1chBases))}
		if ok0 {
			e.idx0 = int(i0)
		}
		for bi, b := range lot1chBases {
			e.bip0[bi] = geoResolveBip(w, b, e.idx0)
			e.bip1[bi] = geoResolveBip(w, b, e.idx1)
		}
		out = append(out, e)
	}
	return out
}

// geoDetectBase rend la base bipede du film : celle qui resout le PLUS de refs de degat en
// bipede (comme sondeBaseSweep). geoBase (512) en repli si le sweep est vide.
func geoDetectBase(raws []geoRawDmg) int {
	bestBase, bestHits := geoBase, -1
	for bi, b := range lot1chBases {
		if b < 400 {
			continue // seule la bande bipede est pertinente
		}
		hits := 0
		for _, e := range raws {
			if e.bip0[bi] || e.bip1[bi] {
				hits++
			}
		}
		if hits > bestHits {
			bestBase, bestHits = b, hits
		}
	}
	return bestBase
}

// geoBuildTouches selectionne, sous la base retenue, les degats a ref1 NON-bipede (candidats
// explosifs) et resout leur victime ref0 en slot absolu (base+idx0).
func geoBuildTouches(raws []geoRawDmg, base int) []geoTouch {
	bi := 0
	for i, b := range lot1chBases {
		if b == base {
			bi = i
		}
	}
	var out []geoTouch
	for _, e := range raws {
		if e.bip1[bi] {
			continue // tir direct (ref1 bipede) : hors perimetre explosif
		}
		tc := geoTouch{ts: e.ts, mag: e.mag, killer: -1}
		if e.idx0 >= 0 && e.bip0[bi] {
			tc.victSlot = uint32(base + e.idx0)
			tc.hasVict = true
		}
		out = append(out, tc)
	}
	return out
}

// geoResolveBip rend vrai si (base+idx) est un bipede lie dans le monde w.
func geoResolveBip(w *World, base, idx int) bool {
	if idx < 0 {
		return false
	}
	slot := base + idx
	if slot < 0 || slot >= 8192 {
		return false
	}
	ti, ok := w.ArchetypeForSlot(uint32(slot))
	return ok && ti == BipedTypeIndex
}

// geoTracks decode positions monde ET visee (i21) par slot, tries par ts. Filtres teleport/
// isolation coupes pour MAXIMISER la couverture (on mesure la resolvabilite, pas une trajectoire).
func geoTracks(t *testing.T, dir string, wr *Vec3Range, n int) map[uint32][]geoAimSample {
	t.Helper()
	opt := DefaultScanFilmOptions()
	opt.MaxSpeedMPS = 0
	opt.IsolationGapMS = 0
	opt.CaptureDirs = true
	opt.WorldRange = wr
	opt.Chunks = make([]int, 0, n)
	for c := 1; c <= n; c++ {
		opt.Chunks = append(opt.Chunks, c)
	}
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("balayage biped impossible : %v", err)
	}
	tr := map[uint32][]geoAimSample{}
	for _, p := range pos {
		s := geoAimSample{ts: p.TimestampUS, x: float64(p.X), y: float64(p.Y), z: float64(p.Z)}
		if h, ok := p.AimHeadingDeg(); ok {
			if pi, ok2 := p.AimPitchDeg(); ok2 {
				s.heading, s.pitch, s.hasAim = float64(h), float64(pi), true
			}
		}
		tr[p.Slot] = append(tr[p.Slot], s)
	}
	for s := range tr {
		ss := tr[s]
		sort.Slice(ss, func(i, j int) bool { return ss[i].ts < ss[j].ts })
		tr[s] = ss
	}
	return tr
}

// geoLookup rend l'echantillon du slot le plus proche de T dans [T-tol, T+tol], et sa validite.
func geoLookup(track []geoAimSample, T, tol uint64) (geoAimSample, bool) {
	if len(track) == 0 {
		return geoAimSample{}, false
	}
	i := sort.Search(len(track), func(i int) bool { return track[i].ts >= T })
	best, ok := geoAimSample{}, false
	var bd uint64 = math.MaxUint64
	pick := func(s geoAimSample) {
		d := T - s.ts
		if s.ts > T {
			d = s.ts - T
		}
		if d <= tol && d < bd {
			best, ok, bd = s, true, d
		}
	}
	if i-1 >= 0 {
		pick(track[i-1])
	}
	if i < len(track) {
		pick(track[i])
	}
	return best, ok
}

// geoAimVec3 rend le vecteur unitaire de visee monde depuis (cap, elevation) en degres.
// Convention MESUREE (offline_aim) : heading = atan2(Y, X), pitch positif = vers le haut.
func geoAimVec3(headingDeg, pitchDeg float64) [3]float64 {
	h := headingDeg * math.Pi / 180
	p := pitchDeg * math.Pi / 180
	cp := math.Cos(p)
	return [3]float64{cp * math.Cos(h), cp * math.Sin(h), math.Sin(p)}
}

// geoDist rend la distance euclidienne 3D entre deux echantillons.
func geoDist(a, b geoAimSample) float64 {
	dx, dy, dz := b.x-a.x, b.y-a.y, b.z-a.z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// geoAngleToVictim rend l'angle (deg) entre la visee du tireur et la direction tireur->victime.
func geoAngleToVictim(shooter geoAimSample, victim geoAimSample) (float64, bool) {
	if !shooter.hasAim {
		return 0, false
	}
	dx, dy, dz := victim.x-shooter.x, victim.y-shooter.y, victim.z-shooter.z
	dn := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if dn < 1e-6 {
		return 0, false
	}
	a := geoAimVec3(shooter.heading, shooter.pitch)
	dot := (a[0]*dx + a[1]*dy + a[2]*dz) / dn
	if dot > 1 {
		dot = 1
	} else if dot < -1 {
		dot = -1
	}
	return math.Acos(dot) * 180 / math.Pi, true
}

// geoMedian rend la mediane d'un echantillon (0 si vide).
func geoMedian(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	return s[len(s)/2]
}
