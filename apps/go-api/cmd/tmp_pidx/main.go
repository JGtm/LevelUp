// tmp_pidx — THROWAWAY : localise le player-index dans le record dense de keyframe.
// Hypothèse (structure déterministe) : à un offset FIXE rel + largeur w par rapport à
// l'arme-tête d'un record, le champ vaut le player-index. Validation SANS fire-events :
// les armes mono-propriétaire (Hydra, Bulldog, Skewer, Ravager...) doivent mapper sur
// UNE valeur stable. Cross-check : le record BR75-fin-de-partie doit valoir 2 (=pi JGtm).
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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
func extractType2(d []byte) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == 2 {
			return d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return nil
}
func bitAt(d []byte, p int) uint64 {
	if p < 0 || p>>3 >= len(d) {
		return 0
	}
	return uint64((d[p>>3] >> uint(7-(p&7))) & 1)
}
func rb(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v = (v << 1) | bitAt(d, bp+i)
	}
	return v
}

type record struct {
	anchor  int
	weapons []string
}
type keyframe struct {
	chunk   string
	timeS   float64
	payload []byte
	records []record
}

func extractRecords(payload []byte) []record {
	h2n := map[uint32]string{}
	for id, n := range analysis.WeaponIDToName {
		h2n[uint32(id>>32)] = n
	}
	type lit struct {
		pos  int
		name string
	}
	var lits []lit
	tot := len(payload) * 8
	for bp := 0; bp+32 <= tot; bp++ {
		if n, ok := h2n[uint32(rb(payload, bp, 32))]; ok {
			lits = append(lits, lit{bp, n})
		}
	}
	sort.Slice(lits, func(i, j int) bool { return lits[i].pos < lits[j].pos })
	// groupe : gap <= 500 = même record ; > 500 = nouveau record. Région loadout (< 350000).
	var recs []record
	var cur record
	last := -1 << 20
	for _, l := range lits {
		if l.pos > 350000 {
			break
		}
		if l.pos-last > 500 {
			if len(cur.weapons) > 0 {
				recs = append(recs, cur)
			}
			cur = record{anchor: l.pos}
		}
		cur.weapons = append(cur.weapons, l.name)
		last = l.pos
	}
	if len(cur.weapons) > 0 {
		recs = append(recs, cur)
	}
	return recs
}

func main() {
	var kfs []keyframe
	for n := 2; n <= 26; n++ {
		name := fmt.Sprintf("chunk_%02d.bin", n)
		p := extractType2(inflate(cache + "/" + name))
		if p == nil {
			continue
		}
		kfs = append(kfs, keyframe{name, float64(n-2) * 20, p, extractRecords(p)})
	}
	fmt.Printf("%d keyframes. records/keyframe : ", len(kfs))
	for _, k := range kfs {
		fmt.Printf("%d ", len(k.records))
	}
	fmt.Println()

	// armes mono-propriétaire au spawn (validation identité, SANS fire-events)
	soloWeapons := map[string]bool{
		"MLRS-2 Hydra": true, "CQS48 Bulldog": true, "Skewer": true,
		"Ravager": true, "Cindershot": true, "S7 Sniper": true, "Stalker Rifle": true,
	}

	type cand struct {
		rel, w   int
		score    float64
		consist  float64
		distinct int // nb de valeurs modales distinctes parmi les armes solo
		gdist    int // nb de valeurs F distinctes globalement (doit être ~8)
		nWpn     int
	}
	var cands []cand
	for w := 3; w <= 8; w++ {
		for rel := -2800; rel <= 2600; rel++ {
			wf := map[string][]uint64{}
			globalF := map[uint64]bool{}
			for _, k := range kfs {
				for _, r := range k.records {
					f := rb(k.payload, r.anchor+rel, w)
					globalF[f] = true
					for _, wn := range r.weapons {
						wf[wn] = append(wf[wn], f)
					}
				}
			}
			// le champ doit VARIER globalement (index 0..7) : au moins 5 valeurs
			if len(globalF) < 5 {
				continue
			}
			var sumFrac float64
			var cnt int
			modalVals := map[uint64]bool{}
			for wn := range soloWeapons {
				fs := wf[wn]
				if len(fs) < 4 {
					continue
				}
				mode := map[uint64]int{}
				bestV, best := uint64(0), 0
				for _, f := range fs {
					mode[f]++
					if mode[f] > best {
						best = mode[f]
						bestV = f
					}
				}
				sumFrac += float64(best) / float64(len(fs))
				modalVals[bestV] = true
				cnt++
			}
			if cnt < 4 {
				continue
			}
			consist := sumFrac / float64(cnt)
			distinct := float64(len(modalVals)) / float64(cnt) // 1.0 = chaque arme solo a SA valeur
			cands = append(cands, cand{rel, w, consist * distinct, consist, len(modalVals), len(globalF), cnt})
		}
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	fmt.Println("\n=== Top 15 (rel,w) par consistance des armes mono-propriétaire ===")
	for i, c := range cands {
		if i >= 15 {
			break
		}
		fmt.Printf("  rel=%-6d w=%-2d score=%.3f (sur %d armes)\n", c.rel, c.w, c.score, c.nWpn)
	}
	if len(cands) == 0 {
		fmt.Println("  (aucun)")
		return
	}

	// Détail du meilleur candidat
	best := cands[0]
	fmt.Printf("\n=== DÉTAIL meilleur : rel=%d w=%d ===\n", best.rel, best.w)
	// arme -> distribution F
	wf := map[string]map[uint64]int{}
	for _, k := range kfs {
		for _, r := range k.records {
			f := rb(k.payload, r.anchor+best.rel, best.w)
			for _, wn := range r.weapons {
				if wf[wn] == nil {
					wf[wn] = map[uint64]int{}
				}
				wf[wn][f]++
			}
		}
	}
	var names []string
	for n := range wf {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Printf("  %-22s : %v\n", n, wf[n])
	}

	// Table par keyframe : record = (F: armes), trié par F. Cherche F=2 (JGtm) tenant BR75.
	fmt.Println("\n=== Par keyframe : (F=player-index : armes) ===")
	for _, k := range kfs {
		type fr struct {
			f   uint64
			wps []string
		}
		var frs []fr
		for _, r := range k.records {
			frs = append(frs, fr{rb(k.payload, r.anchor+best.rel, best.w), r.weapons})
		}
		sort.Slice(frs, func(i, j int) bool { return frs[i].f < frs[j].f })
		fmt.Printf("  %-12s ~%4.0fs : ", k.chunk, k.timeS)
		for _, f := range frs {
			fmt.Printf("[%d:%v] ", f.f, f.wps)
		}
		fmt.Println()
	}
}
