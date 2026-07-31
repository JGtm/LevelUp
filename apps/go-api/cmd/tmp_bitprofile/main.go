// tmp_bitprofile — lit le DECOUPAGE des champs de i0 DIRECTEMENT dans le film, sans
// supposer aucune largeur.
//
// Principe : pour un champ quantifie de largeur W dont la valeur bouge peu d'une frame a
// l'autre, le TAUX DE BASCULE par position de bit vaut ~50% sur le LSB et DOUBLE en
// descendant du MSB vers le LSB. Le profil sur une suite de champs contigus est donc une
// DENT DE SCIE : montee geometrique vers le LSB, RESET brutal au MSB du champ suivant.
// Les frontieres de champ se lisent directement sur le profil.
//
// On n'utilise que l'en-tete du record (deja valide sur les deux cartes) ; a partir de
// l'offset i0 on ne suppose RIEN.
package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const base = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`

type sample struct {
	slot  uint32
	chunk int
	pkt   int
	bits  []byte // bits [i0 .. i0+win)
}

func readBitsAt(b []byte, pos, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		p := pos + i
		if p>>3 >= len(b) {
			return v << uint(n-i)
		}
		v = v<<1 | uint32(b[p>>3]>>(7-uint(p&7))&1)
	}
	return v
}

func main() {
	film := flag.String("film", "000d5950", "id du film")
	win := flag.Int("win", 60, "fenetre de bits profilee a partir de i0")
	prefix := flag.String("prefix", "0000", "motif de bits exige a partir de i0 (0/1/x)")
	tag1 := flag.Bool("tag1", true, "exiger tag==1")
	maxDT := flag.Int("maxdt", 3, "ecart max de paquets pour une paire temporelle")
	quiet := flag.Bool("quiet", false, "n'afficher que la ligne de synthese")
	flag.Parse()

	dir := filepath.Join(base, *film)
	n := filmdec.CountFilmChunks(dir)
	if n == 0 {
		fmt.Printf("SYNTHESE %s : aucun chunk\n", *film)
		return
	}
	slots := slotBand(dir, n)
	var samples []sample
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.Type != filmdec.PacketTypeDelta {
				continue
			}
			samples = append(samples, scan(pk.Payload(data), slots, *tag1, *prefix, *win, c, pk.Index)...)
		}
	}
	if len(samples) == 0 {
		fmt.Printf("SYNTHESE %s prefix=%s : 0 record\n", *film, *prefix)
		return
	}
	flips, ones, pairs := profile(samples, *win, *maxDT)
	bnd := boundaries(flips, pairs)
	if !*quiet {
		fmt.Printf("film %s : %d chunks, %d slots biped, prefix=%q\n", *film, n, len(slots), *prefix)
		fmt.Printf("records i0 candidats : %d ; paires temporelles (dpkt<=%d) : %d\n\n", len(samples), *maxDT, pairs)
		fmt.Println("bit  n(1)        P(1)     nFlip      flip%   |bar")
		for k := 0; k < *win; k++ {
			fp := 100 * float64(flips[k]) / float64(pairs)
			p1 := 100 * float64(ones[k]) / float64(len(samples))
			mark := ""
			for _, b := range bnd {
				if b == k {
					mark = " <-- frontiere"
				}
			}
			fmt.Printf("%3d  %-10d %6.2f%%  %-10d %6.2f%%  %s%s\n", k, ones[k], p1, flips[k], fp,
				strings.Repeat("#", int(fp/2+0.5)), mark)
		}
		fmt.Println()
	}
	fmt.Printf("SYNTHESE %s prefix=%s frontieres=%v", *film, *prefix, bnd)
	if len(bnd) >= 3 {
		fmt.Printf(" | w2=%d w3=%d fin_i0=%d | gate=5 => %d/%d/%d",
			bnd[1]-bnd[0], bnd[2]-bnd[1], bnd[2], bnd[0]-5, bnd[1]-bnd[0], bnd[2]-bnd[1])
	}
	fmt.Printf(" [n=%d pairs=%d]\n", len(samples), pairs)
}

// profile calcule, par position de bit, le nombre de 1 et le nombre de bascules entre
// deux records temporellement consecutifs du MEME slot.
func profile(samples []sample, win, maxDT int) (flips, ones []int, pairs int) {
	flips = make([]int, win)
	ones = make([]int, win)
	bySlot := map[uint32][]sample{}
	for _, s := range samples {
		bySlot[s.slot] = append(bySlot[s.slot], s)
		for k := 0; k < win; k++ {
			if s.bits[k] == 1 {
				ones[k]++
			}
		}
	}
	for _, ss := range bySlot {
		sort.Slice(ss, func(i, j int) bool {
			if ss[i].chunk != ss[j].chunk {
				return ss[i].chunk < ss[j].chunk
			}
			return ss[i].pkt < ss[j].pkt
		})
		for i := 1; i < len(ss); i++ {
			a, b := ss[i-1], ss[i]
			if a.chunk != b.chunk || b.pkt-a.pkt > maxDT || b.pkt == a.pkt {
				continue
			}
			pairs++
			for k := 0; k < win; k++ {
				if a.bits[k] != b.bits[k] {
					flips[k]++
				}
			}
		}
	}
	return flips, ones, pairs
}

// boundaries detecte les MSB de champ : une position k est une frontiere si le taux de
// bascule s'effondre (facteur >= 8) par rapport a k-1 qui etait sature (>= 10%).
// C'est la lecture directe de la dent de scie, sans aucun a priori de largeur.
func boundaries(flips []int, pairs int) []int {
	if pairs == 0 {
		return nil
	}
	var out []int
	for k := 1; k < len(flips); k++ {
		prev := float64(flips[k-1]) / float64(pairs)
		cur := float64(flips[k]) / float64(pairs)
		if prev >= 0.10 && cur*8 < prev {
			out = append(out, k)
		}
	}
	return out
}

func scan(pay []byte, slots map[uint32]bool, tag1 bool, prefix string, win, c, pkt int) []sample {
	total := len(pay) * 8
	var out []sample
	for p := 0; p+21+12+win <= total; {
		if readBitsAt(pay, p, 1) != 1 {
			p++
			continue
		}
		slot := readBitsAt(pay, p+1, 13)
		if !slots[slot] {
			p++
			continue
		}
		if tag1 && readBitsAt(pay, p+14, 2) != 1 {
			p++
			continue
		}
		if readBitsAt(pay, p+16, 2) != 0 {
			p++
			continue
		}
		mc := int(readBitsAt(pay, p+18, 3))
		if mc < 2 || mc > 7 {
			p++
			continue
		}
		i0 := p + 21 + 6*mc
		if i0+win > total {
			p++
			continue
		}
		ok := true
		prev := -1
		for k := 0; k < mc; k++ {
			idx := int(readBitsAt(pay, p+21+6*k, 6))
			if (k == 0 && idx != 0) || idx <= prev {
				ok = false
				break
			}
			prev = idx
		}
		if !ok {
			p++
			continue
		}
		for k := 0; k < len(prefix) && ok; k++ {
			if prefix[k] == 'x' {
				continue
			}
			if byte('0'+readBitsAt(pay, i0+k, 1)) != prefix[k] {
				ok = false
			}
		}
		if !ok {
			p++
			continue
		}
		bits := make([]byte, win)
		for k := 0; k < win; k++ {
			bits[k] = byte(readBitsAt(pay, i0+k, 1))
		}
		out = append(out, sample{slot: slot, chunk: c, pkt: pkt, bits: bits})
		p = i0 + 40
	}
	return out
}

func slotBand(dir string, n int) map[uint32]bool {
	seen := map[uint32]bool{}
	for c := 1; c <= n+1; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			for _, r := range filmdec.WalkKeyframeWorld(pk.Payload(data)) {
				if r.TI == filmdec.BipedTypeIndex && r.Slot >= 0 {
					seen[uint32(r.Slot)] = true
				}
			}
			break
		}
	}
	if len(seen) == 0 {
		return seen
	}
	var keys []uint32
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	out := map[uint32]bool{}
	for k := keys[0]; k <= keys[len(keys)-1]; k++ {
		out[k] = true
	}
	return out
}
