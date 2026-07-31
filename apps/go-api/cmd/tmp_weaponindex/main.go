// tmp_weaponindex — THROWAWAY : chercher un WEAPON INDEX dans l'event de kill (chunk_27).
// Hypothèse user : comme le player_index, le code utiliserait un weapon-index (petite table par match)
// pour ne pas répéter l'arme. "Arme du kill" (sans lien explicite au tueur) suffit (tueur déjà connu).
// Vérité-terrain = nos events FIABLES mêlée/grenade (on connaît l'arme pour ces kills) + records de dégât
// non-ambigus pour quelques gun. On corrèle chaque champ (octet/nibble/bit) du bloc kill avec l'arme connue.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"unicode/utf16"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)
const maxMatchMs = 600000
const variantSuffix = uint32(0x42c9679f)
const deserStartBit = 36
const minXUID = uint64(2e15)
const maxXUID = uint64(3e15)

var endMarker = []byte{0x00, 0x00, 0x2e, 0xe0}
var h32 = map[uint32]string{}
var grenades = map[uint32]string{0xB0171062: "Frag", 0xC0E34C44: "Plasma", 0x3B2567D4: "Shock", 0x9212E428: "Spike"}
var xuidToPi = map[uint64]int{
	2535467794760703: 0, 2535437947245250: 1, 2533274823110022: 2, 2533274980284321: 3,
	2533274815845110: 4, 2535444178793711: 5, 2533274882097883: 6, 2533274826120416: 7,
}

func build() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
}
func inflate(p string) []byte {
	raw, _ := os.ReadFile(p)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}
func bitsAt(p []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		q := bp + i
		if q>>3 >= len(p) || q < 0 {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((p[q>>3]>>uint(7-(q&7)))&1)
	}
	return v
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ---- events FIABLES (mêlée/grenade) pour la vérité-terrain arme ----
type actEvt struct {
	tms  int
	kind string
	wpn  string
	pidx int
}

func reliableActions() []actEvt {
	var evs []actEvt
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			size := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if size <= 0 || off+16+size > len(d) {
				break
			}
			pl := d[off+16 : off+16+size]
			off += 16 + size
			if ts < t0Us {
				continue
			}
			tms := int((ts - t0Us) / 1000)
			if tms < 0 || tms > maxMatchMs {
				continue
			}
			total := len(pl) * 8
			_ = typ
			// MELEE
			for bp := 0; bp+120 < total; bp++ {
				m := bitsAt(pl, bp, 11)
				if m != 0x534 && m != 0x535 {
					continue
				}
				anchor := bp + 3
				t := uint8(bitsAt(pl, anchor+76, 8))
				var woff int
				switch t {
				case 0x47:
					woff = anchor + 86
				case 0x42:
					woff = anchor + 88
				case 0x60:
					woff = anchor + 101
				default:
					continue
				}
				hi := uint32(bitsAt(pl, woff, 32))
				name, ok := h32[hi]
				if !ok {
					continue
				}
				evs = append(evs, actEvt{tms, "melee", name, int(bitsAt(pl, anchor+23, 5))})
			}
			// GRENADE
			for bp := 0; bp+110 < total; bp++ {
				if bitsAt(pl, bp, 24) != 0x4c0c00 {
					continue
				}
				if g, ok := grenades[uint32(bitsAt(pl, bp+24, 32))]; ok {
					evs = append(evs, actEvt{tms, "grenade", g + " Grenade", int(bitsAt(pl, bp+24+32+47, 5))})
				}
			}
		}
	}
	return evs
}

// ---- 519 records de dégât (famille + temps) ----
type dmg struct {
	tms int
	fam string
}

func damageRecs() []dmg {
	var out []dmg
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			size := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if size <= 0 || off+16+size > len(d) {
				break
			}
			pl := d[off+16 : off+16+size]
			off += 16 + size
			if typ != 0 || ts < t0Us || len(pl) == 0 || pl[0] != 0xd2 {
				continue
			}
			br := filmdec.NewBitReader(pl)
			br.Skip(deserStartBit)
			if !br.ReadBit() {
				br.ReadBits(2)
			}
			br.ReadBits(5)
			gid := uint32(br.ReadBits(32))
			if uint32(br.ReadBits(32)) != variantSuffix {
				continue
			}
			if fam, ok := h32[gid]; ok {
				out = append(out, dmg{int((ts - t0Us) / 1000), fam})
			}
		}
	}
	return out
}

// ---- KILL FEED chunk_27 : garder le BLOC BRUT (80 octets autour de l'event) ----
type kill struct {
	killerPi, victimPi int
	t                  int
	block              []byte // 80 octets terminant à endMarker
}

func readByteAtBit(d []byte, bit int) byte {
	if bit < 0 || bit+8 > len(d)*8 {
		return 0
	}
	bi := bit / 8
	o := uint(bit % 8)
	if o == 0 {
		return d[bi]
	}
	return d[bi]<<o | d[bi+1]>>(8-o)
}
func readBytes(d []byte, bit, n int) []byte {
	if bit < 0 || bit+n*8 > len(d)*8 {
		return nil
	}
	o := make([]byte, n)
	for i := 0; i < n; i++ {
		o[i] = readByteAtBit(d, bit+i*8)
	}
	return o
}
func readU64LE(d []byte, bit int) uint64 {
	b := readBytes(d, bit, 8)
	if b == nil {
		return 0
	}
	var x uint64
	for i := 0; i < 8; i++ {
		x |= uint64(b[i]) << (uint(i) * 8)
	}
	return x
}
func findMarker(d []byte, s, e int) int {
	tb := len(d) * 8
	if e > tb {
		e = tb
	}
	for bit := s; bit <= e-32; bit++ {
		if readByteAtBit(d, bit) == 0 && readByteAtBit(d, bit+8) == 0 &&
			readByteAtBit(d, bit+16) == 0x2e && readByteAtBit(d, bit+24) == 0xe0 {
			return bit
		}
	}
	return -1
}
func utf16le(b []byte) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[i*2 : i*2+2])
	}
	for i, c := range u {
		if c == 0 {
			u = u[:i]
			break
		}
	}
	return string(utf16.Decode(u))
}

const blockBytes = 80

func killFeed() []kill {
	data := inflate(cache + "/chunk_27.bin")
	totalBits := len(data) * 8
	type ev struct {
		xuid     uint64
		typeHint int
		t        int
		block    []byte
	}
	var kills, deaths []ev
	seen := map[int]bool{}
	for ms := 8; ms <= totalBits-8; ms++ {
		if readByteAtBit(data, ms) != 0xc0 {
			continue
		}
		xe := ms - 8
		if xe < 64 {
			continue
		}
		if pfx := readByteAtBit(data, xe); pfx != 0x2d && pfx != 0x25 {
			continue
		}
		xs := xe - 64
		if seen[xs] {
			continue
		}
		xuid := readU64LE(data, xs)
		if xuid <= minXUID || xuid >= maxXUID {
			continue
		}
		end := findMarker(data, xs, xs+20000)
		if end < 0 {
			continue
		}
		st := end - blockBytes*8
		if st < 0 {
			continue
		}
		blk := readBytes(data, st, blockBytes)
		if blk == nil {
			continue
		}
		th := int(blk[blockBytes-13]) // ancien eb[47] sur bloc 60 => offset 60-13=47 ; ici 80-13=67
		seen[xs] = true
		e := ev{xuid, th, int(binary.BigEndian.Uint32(blk[blockBytes-12 : blockBytes-8])), blk}
		switch th {
		case 50:
			kills = append(kills, e)
		case 20:
			deaths = append(deaths, e)
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	usedD := make([]bool, len(deaths))
	var out []kill
	for _, k := range kills {
		best, bd := -1, 400
		for i, d := range deaths {
			if usedD[i] || d.xuid == k.xuid {
				continue
			}
			if dt := abs(k.t - d.t); dt < bd {
				bd, best = dt, i
			}
		}
		if best < 0 {
			continue
		}
		usedD[best] = true
		kp, ok := xuidToPi[k.xuid]
		if !ok {
			kp = -1
		}
		vp, ok2 := xuidToPi[deaths[best].xuid]
		if !ok2 {
			vp = -1
		}
		out = append(out, kill{kp, vp, k.t, k.block})
	}
	return out
}

func main() {
	build()
	acts := reliableActions()
	dmgs := damageRecs()
	kills := killFeed()
	fmt.Printf("=== %d kills (bloc %do) ; %d events fiables ; %d records dégât ===\n", len(kills), blockBytes, len(acts), len(dmgs))

	// label arme connue par kill : melee/grenade fiable (auteur==tueur, |dt|<=2500) sinon gun-unambigu
	type lk struct {
		k     kill
		label string // famille connue, "" si inconnu
		src   string
	}
	var labeled []lk
	famByMeleeGren := 0
	for _, k := range kills {
		label, src := "", ""
		// melee/grenade du tueur le plus proche
		bd := 2500
		for _, a := range acts {
			if a.pidx != k.killerPi {
				continue
			}
			if d := abs(a.tms - k.t); d <= bd {
				bd, label, src = d, famName(a.wpn), a.kind
			}
		}
		if label != "" {
			famByMeleeGren++
		} else {
			// gun non-ambigu : une seule famille dans ±300ms
			fams := map[string]bool{}
			for _, dr := range dmgs {
				if abs(dr.tms-k.t) <= 300 {
					fams[dr.fam] = true
				}
			}
			if len(fams) == 1 {
				for f := range fams {
					label, src = f, "gun1"
				}
			}
		}
		labeled = append(labeled, lk{k, label, src})
	}
	nLab := 0
	for _, l := range labeled {
		if l.label != "" {
			nLab++
		}
	}
	fmt.Printf("  labels arme : %d kills (mêlée/grenade fiable=%d, gun-non-ambigu=%d)\n", nLab, famByMeleeGren, nLab-famByMeleeGren)

	// CORRÉLATION : pour chaque offset bit o (0..blockBytes*8-8) et largeur w∈{4,5,6,8}, le champ
	// prédit-il la famille ? Score = pureté (pour chaque valeur du champ, fraction du label majoritaire),
	// pondérée. On cherche un champ à faible cardinalité (table d'armes) fortement corrélé au label.
	fmt.Println("\n=== champs du bloc kill corrélés à l'arme connue (top) ===")
	type res struct {
		o, w   int
		card   int
		purity float64
		cover  int
	}
	var results []res
	maxBit := blockBytes * 8
	for w := 4; w <= 8; w++ {
		for o := 0; o+w <= maxBit; o++ {
			valLabel := map[uint64]map[string]int{}
			tot := 0
			for _, l := range labeled {
				if l.label == "" {
					continue
				}
				v := bitsAt(l.k.block, o, w)
				if valLabel[v] == nil {
					valLabel[v] = map[string]int{}
				}
				valLabel[v][l.label]++
				tot++
			}
			if tot < 10 {
				continue
			}
			// pureté pondérée + cardinalité
			pureSum := 0
			for _, lm := range valLabel {
				mx, s := 0, 0
				for _, c := range lm {
					s += c
					if c > mx {
						mx = c
					}
				}
				pureSum += mx
				_ = s
			}
			purity := float64(pureSum) / float64(tot)
			card := len(valLabel)
			// champ intéressant : pureté élevée, cardinalité plausible (2..25 = table d'armes), couvre tot
			// weapon-index = faible cardinalité (~#armes), PAS ~#kills (=timestamp/unique).
			// exclure la zone timeMS (octets blockBytes-12..-8 = bits 544..576).
			if o >= (blockBytes-12)*8-8 && o <= (blockBytes-8)*8 {
				continue
			}
			if purity >= 0.70 && card >= 2 && card <= 16 {
				results = append(results, res{o, w, card, purity, tot})
			}
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].purity != results[j].purity {
			return results[i].purity > results[j].purity
		}
		return results[i].cover > results[j].cover
	})
	if len(results) == 0 {
		fmt.Println("  AUCUN champ du bloc kill ne prédit l'arme (pureté>=0.75). Pas de weapon-index dans le bloc kill.")
	}
	for i, r := range results {
		if i >= 15 {
			fmt.Printf("  ... (%d champs)\n", len(results))
			break
		}
		fmt.Printf("  bit o=%d w=%d : pureté=%.0f%% card=%d cover=%d (octet~%d)\n",
			r.o, r.w, 100*r.purity, r.card, r.cover, r.o/8)
	}

	// baseline : combien de labels distincts ? (la pureté doit battre le label majoritaire trivial)
	labCount := map[string]int{}
	for _, l := range labeled {
		if l.label != "" {
			labCount[l.label]++
		}
	}
	mx := 0
	for _, c := range labCount {
		if c > mx {
			mx = c
		}
	}
	fmt.Printf("\n  (baseline : %d familles distinctes labellisées ; label majoritaire = %.0f%% => pureté doit dépasser)\n",
		len(labCount), 100*float64(mx)/float64(nLab))

	// dump value->family du meilleur candidat (pour vérifier que c'est un vrai weapon-index)
	if len(results) > 0 {
		r := results[0]
		fmt.Printf("\n=== mapping valeur->arme du meilleur champ (o=%d w=%d) ===\n", r.o, r.w)
		vmap := map[uint64]map[string]int{}
		for _, l := range labeled {
			if l.label == "" {
				continue
			}
			v := bitsAt(l.k.block, r.o, r.w)
			if vmap[v] == nil {
				vmap[v] = map[string]int{}
			}
			vmap[v][l.label]++
		}
		var vs []uint64
		for v := range vmap {
			vs = append(vs, v)
		}
		sort.Slice(vs, func(i, j int) bool { return vs[i] < vs[j] })
		for _, v := range vs {
			fmt.Printf("  val=%3d : %v\n", v, vmap[v])
		}
	}
	_ = utf16le
}

func famName(w string) string {
	return w // déjà une famille
}
