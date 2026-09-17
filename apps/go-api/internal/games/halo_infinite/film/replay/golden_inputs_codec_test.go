package replay

// golden_inputs_codec_test.go — LES PRIMITIVES DU CODEC : flux binaire et sous-codecs partages.
//
// Extrait de golden_inputs_test.go le 2026-09-14 (revue R1, constat R1-7). DEPLACEMENT PUR.

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

const cmScale = 100

// gwriter accumule un flux binaire. Les entiers sont en varint : les deltas d horodatage et de
// position tiennent sur un a deux octets, ce qui fait tout le poids du fixture.
type gwriter struct{ b []byte }

func (w *gwriter) u(v uint64)   { w.b = binary.AppendUvarint(w.b, v) }
func (w *gwriter) i(v int64)    { w.b = binary.AppendVarint(w.b, v) }
func (w *gwriter) byte8(v byte) { w.b = append(w.b, v) }
func (w *gwriter) f32(v float32) {
	w.b = binary.LittleEndian.AppendUint32(w.b, math.Float32bits(v))
}
func (w *gwriter) str(s string) {
	w.u(uint64(len(s)))
	w.b = append(w.b, s...)
}
func (w *gwriter) bool8(v bool) {
	if v {
		w.byte8(1)
		return
	}
	w.byte8(0)
}

// greader relit le flux. Toute incoherence est une ERREUR remontee, jamais une valeur nulle
// servie en silence.
type greader struct {
	b   []byte
	off int
	err error
}

func (r *greader) u() uint64 {
	if r.err != nil {
		return 0
	}
	v, n := binary.Uvarint(r.b[r.off:])
	if n <= 0 {
		r.err = fmt.Errorf("uvarint illisible a l offset %d", r.off)
		return 0
	}
	r.off += n
	return v
}

func (r *greader) i() int64 {
	if r.err != nil {
		return 0
	}
	v, n := binary.Varint(r.b[r.off:])
	if n <= 0 {
		r.err = fmt.Errorf("varint illisible a l offset %d", r.off)
		return 0
	}
	r.off += n
	return v
}

func (r *greader) byte8() byte {
	if r.err != nil {
		return 0
	}
	if r.off >= len(r.b) {
		r.err = fmt.Errorf("fin de flux prematuree a l offset %d", r.off)
		return 0
	}
	v := r.b[r.off]
	r.off++
	return v
}

func (r *greader) f32() float32 {
	if r.err != nil {
		return 0
	}
	if r.off+4 > len(r.b) {
		r.err = fmt.Errorf("float32 tronque a l offset %d", r.off)
		return 0
	}
	v := math.Float32frombits(binary.LittleEndian.Uint32(r.b[r.off:]))
	r.off += 4
	return v
}

func (r *greader) str() string {
	n := int(r.u())
	if r.err != nil {
		return ""
	}
	if r.off+n > len(r.b) {
		r.err = fmt.Errorf("chaine tronquee a l offset %d", r.off)
		return ""
	}
	s := string(r.b[r.off : r.off+n])
	r.off += n
	return s
}

func (r *greader) bool8() bool { return r.byte8() == 1 }

// encodePositionSection / decodePositionSection serialisent UNE suite de positions de bipede AVEC
// sa table de slots.
//
// UN SEUL CODEC POUR LES DEUX SUITES (lot 1.0, 2026-09-14) : les positions de BIPEDE et celles
// des VEHICULES (`VehicleScan.Positions`, lues a la meme grammaire sur la bande `ti=40`) ont la
// meme forme. Un second codec aurait diverge du premier au premier champ ajoute — c est
// exactement ce qui est arrive a la sequence de balayages elle-meme.
//
// LA TABLE DES SLOTS EST DANS LA SECTION, et non dans l en-tete du blob : un slot tient sur
// 13 bits mais un film n en emploie qu une centaine, et l indirection ramene 2 octets a 1 sur
// chaque position. Chaque suite a la sienne — celle des vehicules n est pas celle des bipedes.
func encodePositionSection(w *gwriter, pos []grammar.BipedPosition) {
	slotIdx := map[uint32]int{}
	var slots []uint32
	for _, p := range pos {
		if _, ok := slotIdx[p.Slot]; !ok {
			slotIdx[p.Slot] = len(slots)
			slots = append(slots, p.Slot)
		}
	}
	w.u(uint64(len(slots)))
	for _, s := range slots {
		w.u(uint64(s))
	}
	w.u(uint64(len(pos)))
	var lastTS uint64
	lastXYZ := map[uint32][3]int64{}
	for _, p := range pos {
		w.u(p.TimestampUS - lastTS) // horodatages non decroissants dans l ordre du film
		lastTS = p.TimestampUS
		w.u(uint64(slotIdx[p.Slot]))
		var fl byte
		if p.HasWorld {
			fl |= gpHasWorld
		}
		if p.HasYaw {
			fl |= gpHasYaw
		}
		if p.HasBody {
			fl |= gpHasBody
		}
		if p.HasShield {
			fl |= gpHasShield
		}
		w.byte8(fl)
		if p.HasWorld {
			// LES QUANTA DU FILM, PAS LES FLOTTANTS DERIVES (lot 0.D.3 bis). `X/Y/Z` sont
			// le resultat de `DequantBipedAxis(Q[ax], ax, layout, bornes)` : porter `Q` et
			// redequantifier a la relecture par LE MEME chemin rend la coordonnee a
			// l identique, sans coder un flottant. Et un quantum est un ENTIER qui bouge
			// peu d une position a la suivante : le delta signe tient sur un a deux octets,
			// la ou les bits d un float32 n en tenaient aucun.
			cur := [3]int64{int64(p.Q[0]), int64(p.Q[1]), int64(p.Q[2])}
			prev := lastXYZ[p.Slot]
			for a := 0; a < 3; a++ {
				w.i(cur[a] - prev[a])
			}
			lastXYZ[p.Slot] = cur
		}
		if p.HasYaw {
			// LES DEUX ANGLES D I21, ensemble : le cap et l elevation viennent du MEME
			// composant et partagent leur validite. En serialiser un seul rendrait un
			// fixture ou toutes les visees sont a plat.
			w.u(uint64(p.YawRaw))
			w.u(uint64(p.PitchRaw))
		}
		if p.HasBody {
			w.f32(p.Body.Health)
		}
		if p.HasShield {
			w.f32(p.Shield.Shield)
			w.byte8(p.Shield.Q) // le QUANTUM : la regle du surbouclier (q > 64) le lit, pas la valeur clampee
		}
	}
}

func decodePositionSection(r *greader, lay profile.I0Layout, world profile.Vec3Range) []grammar.BipedPosition {
	nSlots := int(r.u())
	slots := make([]uint32, 0, nSlots)
	for k := 0; k < nSlots && r.err == nil; k++ {
		slots = append(slots, uint32(r.u()))
	}
	n := int(r.u())
	out := make([]grammar.BipedPosition, 0, n)
	var lastTS uint64
	lastXYZ := map[uint32][3]int64{}
	for k := 0; k < n && r.err == nil; k++ {
		var p grammar.BipedPosition
		lastTS += r.u()
		p.TimestampUS = lastTS
		si := int(r.u())
		if si >= len(slots) {
			r.err = fmt.Errorf("index de slot %d hors table (%d)", si, len(slots))
			return out
		}
		p.Slot = slots[si]
		fl := r.byte8()
		if fl&gpHasWorld != 0 {
			p.HasWorld = true
			prev := lastXYZ[p.Slot]
			var cur [3]int64
			for a := 0; a < 3; a++ {
				cur[a] = prev[a] + r.i()
			}
			lastXYZ[p.Slot] = cur
			p.Q = [3]uint32{uint32(cur[0]), uint32(cur[1]), uint32(cur[2])}
			p.X = grammar.DequantBipedAxis(p.Q[0], 0, lay, world)
			p.Y = grammar.DequantBipedAxis(p.Q[1], 1, lay, world)
			p.Z = grammar.DequantBipedAxis(p.Q[2], 2, lay, world)
		}
		if fl&gpHasYaw != 0 {
			p.HasYaw = true
			p.YawRaw = uint32(r.u())
			p.PitchRaw = uint32(r.u())
		}
		if fl&gpHasBody != 0 {
			p.HasBody = true
			p.Body.Health = r.f32()
		}
		if fl&gpHasShield != 0 {
			p.HasShield = true
			p.Shield.Shield = r.f32()
			p.Shield.Q = r.byte8()
		}
		out = append(out, p)
	}
	return out
}

func encodeTracks(w *gwriter, tracks []types.ProjectileTrack) {
	w.u(uint64(len(tracks)))
	for _, tr := range tracks {
		w.u(uint64(tr.Slot))
		w.u(uint64(tr.Gen))
		w.u(uint64(len(tr.Pts)))
		var pts uint64
		var prev [3]int64
		for _, s := range tr.Pts {
			w.u(s.TimestampUS - pts)
			pts = s.TimestampUS
			cur := [3]int64{cmOf(s.X), cmOf(s.Y), cmOf(s.Z)}
			for a := 0; a < 3; a++ {
				w.i(cur[a] - prev[a])
			}
			prev = cur
			w.bool8(s.AtRest)
		}
	}
}

func decodeTracks(r *greader) []types.ProjectileTrack {
	n := int(r.u())
	out := make([]types.ProjectileTrack, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		tr := types.ProjectileTrack{Slot: uint32(r.u()), Gen: uint32(r.u())}
		np := int(r.u())
		var ts uint64
		var prev [3]int64
		for j := 0; j < np && r.err == nil; j++ {
			ts += r.u()
			var cur [3]int64
			for a := 0; a < 3; a++ {
				cur[a] = prev[a] + r.i()
			}
			prev = cur
			tr.Pts = append(tr.Pts, types.ProjectileSample{
				TimestampUS: ts, X: fromCM(cur[0]), Y: fromCM(cur[1]), Z: fromCM(cur[2]),
				AtRest: r.bool8(),
			})
		}
		out = append(out, tr)
	}
	return out
}

// encodeWorldObjectScan / decodeWorldObjectScan serialisent ce que le film rend sur UN archetype
// d objet du monde : les records de CREATION (position i0, instant, identite MPP), le
// RECENSEMENT des images-cles qui borne les disparitions, et les pistes de position qui disent
// si l objet a bouge.
//
// UN SEUL CODEC POUR LES DEUX VOIES (armes `ti=42`, power-ups `ti=37`) : elles ont la meme
// forme, et un second codec aurait diverge du premier au premier champ ajoute.
//
// LA BANDE DE SLOTS N EST PAS SERIALISEE, et c est deliberé : l assemblage ne la lit pas (elle
// sert au seul balayage, qui a deja eu lieu). Le fixture porte ce que l assemblage CONSOMME,
// pas ce que le decodage a traverse.
func encodeWorldObjectScan(w *gwriter, s WorldObjectScan) {
	w.bool8(s.Scanned)
	encodeCreations(w, s.Creations)
	w.u(uint64(s.Stats.Slots))
	w.u(uint64(s.Stats.Anchors))
	w.u(uint64(s.Stats.Accepted))
	encodeKeyframes(w, s.Keyframes)
	encodeTracks(w, s.Tracks)
}

func decodeWorldObjectScan(r *greader) WorldObjectScan {
	s := WorldObjectScan{Scanned: r.bool8()}
	s.Creations = decodeCreations(r)
	s.Stats.Slots, s.Stats.Anchors, s.Stats.Accepted = int(r.u()), int(r.u()), int(r.u())
	s.Keyframes = decodeKeyframes(r)
	s.Tracks = decodeTracks(r)
	return s
}

// encodeCreations / decodeCreations serialisent les records de CREATION d un archetype d objet du
// monde : position i0, instant, identite MPP.
//
// SORTI DE `encodeWorldObjectScan` AU LOT 1.0 : les VEHICULES (`VehicleScan.Creations`) portent
// exactement les memes records, et un second codec aurait diverge du premier.
//
// L IDENTITE TIENT DANS `MPPWord32`, ET ELLE SEULE : c est le mot inconditionnel du bloc MPP —
// le GlobalID du tag `eqip` pour ti=37, l identite du chassis pour ti=40 (cf.
// filmdec/vehicle_creation.go). Les trois autres champs du bloc ne sont lus par aucun assemblage.
func encodeCreations(w *gwriter, creations []types.EquipmentCreation) {
	w.u(uint64(len(creations)))
	var lastTS uint64
	for _, c := range creations {
		w.u(c.TimestampUS - lastTS) // les creations sortent du balayage dans l ordre du film
		lastTS = c.TimestampUS
		w.u(uint64(c.Slot))
		w.u(uint64(c.Gen))
		w.f32(c.X)
		w.f32(c.Y)
		w.f32(c.Z)
		w.bool8(c.MPPPresent[grammar.MPPWord32])
		w.u(c.MPPVal[grammar.MPPWord32])
	}
}

func decodeCreations(r *greader) []types.EquipmentCreation {
	n := int(r.u())
	out := make([]types.EquipmentCreation, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		c := types.EquipmentCreation{TimestampUS: lastTS, Slot: uint32(r.u()), Gen: uint32(r.u())}
		c.X, c.Y, c.Z = r.f32(), r.f32(), r.f32()
		c.MPPPresent[grammar.MPPWord32] = r.bool8()
		c.MPPVal[grammar.MPPWord32] = r.u()
		out = append(out, c)
	}
	return out
}

// encodeKeyframes / decodeKeyframes serialisent le RECENSEMENT d images-cles qui borne les
// disparitions d un archetype : l axe des images-cles du film, et les instants ou chaque vie
// d objet y est vue.
//
// LA BANDE DE SLOTS N EST PAS SERIALISEE, et c est delibere : l assemblage ne la lit pas (elle
// sert au seul balayage, qui a deja eu lieu). Le fixture porte ce que l assemblage CONSOMME, pas
// ce que le decodage a traverse.
func encodeKeyframes(w *gwriter, kf grammar.WorldObjectKeyframes) {
	w.u(uint64(len(kf.TimesUS)))
	var lastTS uint64
	for _, t := range kf.TimesUS {
		w.u(t - lastTS)
		lastTS = t
	}
	// L ORDRE DES CLES EST RENDU TOTAL : une map Go s itere au hasard, et un fixture dont les
	// octets changent a chaque regeneration n est plus un fixture.
	keys := make([]types.EquipmentLifeKey, 0, len(kf.SeenUS))
	for k := range kf.SeenUS {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Slot != keys[j].Slot {
			return keys[i].Slot < keys[j].Slot
		}
		return keys[i].Gen < keys[j].Gen
	})
	w.u(uint64(len(keys)))
	for _, k := range keys {
		w.u(uint64(k.Slot))
		w.u(uint64(k.Gen))
		seen := kf.SeenUS[k]
		w.u(uint64(len(seen)))
		lastTS = 0
		for _, t := range seen {
			w.u(t - lastTS)
			lastTS = t
		}
	}
}

func decodeKeyframes(r *greader) grammar.WorldObjectKeyframes {
	var kf grammar.WorldObjectKeyframes
	n := int(r.u())
	kf.TimesUS = make([]uint64, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		kf.TimesUS = append(kf.TimesUS, lastTS)
	}
	n = int(r.u())
	kf.SeenUS = make(map[types.EquipmentLifeKey][]uint64, n)
	for k := 0; k < n && r.err == nil; k++ {
		key := types.EquipmentLifeKey{Slot: uint32(r.u()), Gen: uint32(r.u())}
		np := int(r.u())
		seen := make([]uint64, 0, np)
		lastTS = 0
		for j := 0; j < np && r.err == nil; j++ {
			lastTS += r.u()
			seen = append(seen, lastTS)
		}
		kf.SeenUS[key] = seen
	}
	return kf
}

// encodeAmmo serialise un emplacement de munitions. LES TROIS CAS SONT DISTINCTS (chargeur,
// jauge, rien) : un drapeau par pointeur, jamais un zero qui vaudrait absence.
func encodeAmmo(w *gwriter, a SlotAmmo) {
	w.bool8(a.Mag != nil)
	if a.Mag != nil {
		w.u(uint64(*a.Mag))
	}
	w.bool8(a.Res != nil)
	if a.Res != nil {
		w.u(uint64(*a.Res))
	}
	w.bool8(a.Gauge != nil)
	if a.Gauge != nil {
		w.b = binary.LittleEndian.AppendUint64(w.b, math.Float64bits(*a.Gauge))
	}
	w.u(uint64(a.Overheat))
	w.u(uint64(a.Flags))
}

func decodeAmmo(r *greader) SlotAmmo {
	var a SlotAmmo
	if r.bool8() {
		v := uint32(r.u())
		a.Mag = &v
	}
	if r.bool8() {
		v := uint32(r.u())
		a.Res = &v
	}
	if r.bool8() {
		if r.off+8 > len(r.b) {
			r.err = fmt.Errorf("jauge tronquee a l offset %d", r.off)
			return a
		}
		v := math.Float64frombits(binary.LittleEndian.Uint64(r.b[r.off:]))
		r.off += 8
		a.Gauge = &v
	}
	a.Overheat = uint32(r.u())
	a.Flags = uint32(r.u())
	return a
}

// decodeGoldenInputs relit le fixture.

func cmOf(v float32) int64 { return int64(math.Round(float64(v) * cmScale)) }

func fromCM(v int64) float32 { return float32(float64(v) / cmScale) }
