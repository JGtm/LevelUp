package replay

// golden_inputs_codec_test.go — LES PRIMITIVES DU CODEC : flux binaire et sous-codecs partages.
//
// Extrait de golden_inputs_test.go le 2026-09-14 (revue R1, constat R1-7). DEPLACEMENT PUR.

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
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

// Drapeaux de presence d une position, sur un octet.

func encodeTracks(w *gwriter, tracks []filmdec.ProjectileTrack) {
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

func decodeTracks(r *greader) []filmdec.ProjectileTrack {
	n := int(r.u())
	out := make([]filmdec.ProjectileTrack, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		tr := filmdec.ProjectileTrack{Slot: uint32(r.u()), Gen: uint32(r.u())}
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
			tr.Pts = append(tr.Pts, filmdec.ProjectileSample{
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
	w.u(uint64(len(s.Creations)))
	var lastTS uint64
	for _, c := range s.Creations {
		w.u(c.TimestampUS - lastTS) // les creations sortent du balayage dans l ordre du film
		lastTS = c.TimestampUS
		w.u(uint64(c.Slot))
		w.u(uint64(c.Gen))
		w.f32(c.X)
		w.f32(c.Y)
		w.f32(c.Z)
		w.bool8(c.MPPPresent[filmdec.MPPWord32])
		w.u(c.MPPVal[filmdec.MPPWord32])
	}
	w.u(uint64(s.Stats.Slots))
	w.u(uint64(s.Stats.Anchors))
	w.u(uint64(s.Stats.Accepted))
	w.u(uint64(len(s.Keyframes.TimesUS)))
	lastTS = 0
	for _, t := range s.Keyframes.TimesUS {
		w.u(t - lastTS)
		lastTS = t
	}
	// L ORDRE DES CLES EST RENDU TOTAL : une map Go s itere au hasard, et un fixture dont les
	// octets changent a chaque regeneration n est plus un fixture.
	keys := make([]filmdec.EquipmentLifeKey, 0, len(s.Keyframes.SeenUS))
	for k := range s.Keyframes.SeenUS {
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
		seen := s.Keyframes.SeenUS[k]
		w.u(uint64(len(seen)))
		lastTS = 0
		for _, t := range seen {
			w.u(t - lastTS)
			lastTS = t
		}
	}
	encodeTracks(w, s.Tracks)
}

func decodeWorldObjectScan(r *greader) WorldObjectScan {
	s := WorldObjectScan{Scanned: r.bool8()}
	n := int(r.u())
	s.Creations = make([]filmdec.EquipmentCreation, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		c := filmdec.EquipmentCreation{TimestampUS: lastTS, Slot: uint32(r.u()), Gen: uint32(r.u())}
		c.X, c.Y, c.Z = r.f32(), r.f32(), r.f32()
		c.MPPPresent[filmdec.MPPWord32] = r.bool8()
		c.MPPVal[filmdec.MPPWord32] = r.u()
		s.Creations = append(s.Creations, c)
	}
	s.Stats.Slots, s.Stats.Anchors, s.Stats.Accepted = int(r.u()), int(r.u()), int(r.u())
	n = int(r.u())
	s.Keyframes.TimesUS = make([]uint64, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		s.Keyframes.TimesUS = append(s.Keyframes.TimesUS, lastTS)
	}
	n = int(r.u())
	s.Keyframes.SeenUS = make(map[filmdec.EquipmentLifeKey][]uint64, n)
	for k := 0; k < n && r.err == nil; k++ {
		key := filmdec.EquipmentLifeKey{Slot: uint32(r.u()), Gen: uint32(r.u())}
		np := int(r.u())
		seen := make([]uint64, 0, np)
		lastTS = 0
		for j := 0; j < np && r.err == nil; j++ {
			lastTS += r.u()
			seen = append(seen, lastTS)
		}
		s.Keyframes.SeenUS[key] = seen
	}
	s.Tracks = decodeTracks(r)
	return s
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
