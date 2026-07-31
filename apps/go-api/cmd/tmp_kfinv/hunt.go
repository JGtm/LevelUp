package main

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

// hunt — passe de CHASSE : pour la 1re image-clé, imprime la fenêtre entre la fin d'i22 et
// le premier identifiant d'arme, plus tous les candidats munitions [0][8][1] + R(11), et la
// fenêtre après la dernière arme (candidats compteur d'utilisations).

type recView struct {
	slot     int
	from, to int
	abil     int // position bit ABSOLUE du début de l'ancre capacité
	idx3     uint32
	i22      int // position bit ABSOLUE du motif i22 retenu
	gren     [4]uint32
	wpos     []int
	wfam     []uint32
}

// firstKeyframeViews rend les vues de records biped de l'image-clé n° kfWanted (1-based).
func firstKeyframeViews(dir string, kfWanted int, grenMax uint32) ([]recView, []byte, uint64) {
	known := map[uint32]string{}
	for f, n := range weaponv3.KnownWeaponHigh32 {
		known[f] = n
	}
	n := filmdec.CountFilmChunks(dir)
	kf := 0
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			kf++
			if kf != kfWanted {
				continue
			}
			pay := p.Payload(chunk)
			var out []recView
			for _, s := range spans(pay) {
				if s.ti != bipedTI {
					continue
				}
				v := recView{slot: s.slot, from: s.from, to: s.to, abil: -1, i22: -1}
				if ah := abilityIn(pay, s.from, s.to); len(ah) == 1 {
					v.abil = ah[0].anchorBit
					v.idx3 = ah[0].idx3
				}
				if v.abil >= 0 {
					for _, g := range grenIn(pay, v.abil, s.to, grenMax) {
						v.i22, v.gren = g.bit, g.c
						break
					}
				}
				v.wpos, v.wfam = famsIn(pay, s.from, s.to, known)
				out = append(out, v)
			}
			return out, pay, p.TimestampUS
		}
	}
	return nil, nil, 0
}

// ammoCand : candidat de lecture i30/i31 = [0][mag:8][1] puis R(11) réserve.
type ammoCand struct {
	at       int
	mag, res uint32
}

func ammoCands(pay []byte, from, to int) []ammoCand {
	var out []ammoCand
	for b := from; b+21 <= to; b++ {
		if bitAt(pay, b) != 0 || bitAt(pay, b+9) != 1 {
			continue
		}
		out = append(out, ammoCand{at: b, mag: bits(pay, b+1, 8), res: bits(pay, b+10, 11)})
	}
	return out
}

// useCand : candidat compteur d'utilisations = R(3) masque + 7 bits par bit armé.
type useCand struct {
	at   int
	mask uint32
	v    []uint32
}

func useCands(pay []byte, from, to int) []useCand {
	var out []useCand
	for b := from; b+3 <= to; b++ {
		m := bits(pay, b, 3)
		if m == 0 {
			continue
		}
		nb := 0
		for i := 0; i < 3; i++ {
			if m&(1<<uint(i)) != 0 {
				nb++
			}
		}
		if b+3+7*nb > to {
			continue
		}
		c := useCand{at: b, mask: m}
		for i := 0; i < nb; i++ {
			c.v = append(c.v, bits(pay, b+3+7*i, 7))
		}
		out = append(out, c)
	}
	return out
}

func bin(pay []byte, from, n int) string {
	s := make([]byte, 0, n+n/8)
	for i := 0; i < n; i++ {
		if i > 0 && i%8 == 0 {
			s = append(s, ' ')
		}
		s = append(s, byte('0'+bitAt(pay, from+i)))
	}
	return string(s)
}

func runHunt(dir string, kfWanted int, grenMax uint32, what string) {
	views, pay, ts := firstKeyframeViews(dir, kfWanted, grenMax)
	if views == nil {
		fmt.Println("image-clé introuvable")
		return
	}
	fmt.Printf("image-clé %d, t=%d, %d records biped\n\n", kfWanted, ts, len(views))
	sort.Slice(views, func(i, j int) bool { return views[i].slot < views[j].slot })
	for _, v := range views {
		if v.abil < 0 || v.i22 < 0 || len(v.wpos) == 0 {
			fmt.Printf("slot %d : incomplet (abil=%d i22=%d armes=%d)\n", v.slot, v.abil, v.i22, len(v.wpos))
			continue
		}
		w0 := v.wpos[0]
		fmt.Printf("--- slot %d  abil@+%d idx=%d  i22@+%d %v  arme0@+%d\n",
			v.slot, v.abil-v.from, v.idx3, v.i22-v.from, v.gren, w0-v.from)
		switch what {
		case "ammo":
			lo, hi := v.i22+35, w0
			fmt.Printf("    fenêtre i22fin..arme0 = %d bits\n", hi-lo)
			fmt.Printf("    E-160..E : %s\n", bin(pay, w0-160, 160))
			for _, c := range ammoCands(pay, lo, hi) {
				fmt.Printf("      cand @+%-4d (E-%-4d) mag=%-4d res=%d\n", c.at-lo, w0-c.at, c.mag, c.res)
			}
		case "uses":
			last := v.wpos[len(v.wpos)-1]
			lo, hi := last+64, v.to
			if hi-lo > 900 {
				hi = lo + 900
			}
			fmt.Printf("    fenêtre dernière arme+64 .. fin = %d bits\n", hi-lo)
			fmt.Printf("    %s\n", bin(pay, lo, min(hi-lo, 700)))
		case "solve":
			end0 := w0 - 1
			lo := end0 - 300
			if lo < v.from {
				lo = v.from
			}
			sols := solveAmmoBlock(pay, end0, lo)
			fmt.Printf("    %d solutions\n", len(sols))
			for _, s := range sols {
				st, sel, _, _ := parseAmmoBlock(pay, s, end0+1)
				fmt.Printf("      s=E-%-4d sel=%-2d", end0-s+1, sel)
				for k := 0; k < 2; k++ {
					fmt.Printf(" | s%d ", k)
					if st[k].mag != nil {
						fmt.Printf("mag=%d ", *st[k].mag)
					}
					if st[k].gauge != nil {
						fmt.Printf("jauge=%.3f ", *st[k].gauge)
					}
					if st[k].res != nil {
						fmt.Printf("res=%d", *st[k].res)
					}
				}
				fmt.Println()
			}
		case "pre":
			lo, hi := v.abil, v.i22
			fmt.Printf("    fenêtre abil..i22 = %d bits\n", hi-lo)
			fmt.Printf("    %s\n", bin(pay, lo, min(hi-lo, 400)))
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
