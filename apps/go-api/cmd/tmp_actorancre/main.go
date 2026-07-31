// cmd/tmp_actorancre — i19 DU BIPED, PAR ANCRAGE. Pas par parcours.
//
// L'ERREUR QUE CE PROGRAMME CORRIGE. La premiere tentative (cmd/tmp_actorctl) lisait i19 en
// DEROULANT la chaine de composants du flux delta. C'est la methode que ce depot a deja
// mesuree et refusee — `keyframe_loadout.go` l'ecrit : « On ne tente PAS de derouler la chaine
// de composants jusqu'a i43 : cette voie a ete mesuree et REFUSEE ». Resultat previsible :
// 3,3 % d'atteinte et du bruit. L'instrument etait aveugle, pas le composant.
//
// LA METHODE DU DEPOT EST L'ANCRAGE : on cherche un MOTIF reconnaissable dans l'emprise d'un
// record, on ne marche pas jusqu'a lui. C'est ainsi que les familles d'arme ont ete lues
// (911 occurrences pour 0,52 attendue par hasard), et c'est possible ici parce que Ghidra
// donne la FORME exacte du composant.
//
// LA FORME, lue dans FUN_1408f0778 (deser de unit-actor-control-component) :
//
//	R(3) sel        ; si sel == 1 -> le composant s'arrete la
//	R(1) present    ; si pose -> FUN_141015740 = R(32), un handle complet
//
// LE MOTIF CHERCHE est donc : [3 bits sel != 1][1 bit a 1][32 bits de handle VALIDE].
//
// CE QUI REND L'ANCRAGE SELECTIF, et c'est le point : le handle doit designer une entite
// JOUEUR. Sur ce film elles occupent les slots 52..83 (recensement de cmd/tmp_playerrep :
// 32 entites, 832 records = 32 x 26 images-cles). Un handle valide porte en outre une
// generation dans {1,2,3} — la generation 0 est le handle nul.
//
//	P(hasard) = (32/2^30 slots) x (3/4 generations) x (7/8 sel) x (1/2 present) ~ 9,8e-9
//
// Sur les ~515 000 bits des records de biped d'une image-cle, l'esperance de faux positifs est
// de 0,005. Une touche par record de biped ne s'obtient donc pas par hasard.
//
// LE CONTROLE QUI PEUT ECHOUER, ecrit avant la mesure : les 8 bipeds joueurs d'une image-cle
// doivent designer 8 entites joueur DISTINCTES. Une collision refute le lien.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	bipedTI  = 35
	playerTI = 5
)

type recSpan struct {
	slot, ti, gen int
	from, to      int
}

func spans(pay []byte) []recSpan {
	recs := filmdec.WalkKeyframeWorld(pay)
	if len(recs) == 0 {
		return nil
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
	out := make([]recSpan, 0, len(recs))
	total := len(pay) * 8
	for i, r := range recs {
		to := total
		if i+1 < len(recs) {
			to = recs[i+1].Bit
		}
		out = append(out, recSpan{slot: r.Slot, ti: r.TI, gen: r.Gen, from: r.Bit, to: to})
	}
	return out
}

func bitAt(buf []byte, p int) uint32 {
	if idx := p >> 3; idx >= 0 && idx < len(buf) {
		return uint32(buf[idx]>>(7-uint(p&7))) & 1
	}
	return 0
}

func bits(buf []byte, p, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		v = v<<1 | bitAt(buf, p+i)
	}
	return v
}

// hit est une touche du motif d'i19 dans l'emprise d'un record.
type hit struct {
	off  int // decalage depuis le debut du record
	sel  uint32
	slot int
	gen  int
}

// anchorI19 cherche le motif [sel(3)!=1][present(1)=1][handle(32) valide] dans [from,to).
func anchorI19(pay []byte, from, to int, playerSlots map[int]bool) []hit {
	var out []hit
	for p := from; p+36 <= to; p++ {
		sel := bits(pay, p, 3)
		if sel == 1 {
			continue // le composant s'arrete la : pas de handle
		}
		if bitAt(pay, p+3) != 1 {
			continue // handle absent
		}
		h := bits(pay, p+4, 32)
		gen := int(h >> 30)
		if gen == 0 {
			continue // handle nul
		}
		slot := int(h & 0x3fffffff)
		if !playerSlots[slot] {
			continue
		}
		out = append(out, hit{off: p - from, sel: sel, slot: slot, gen: gen})
	}
	return out
}

func main() {
	repo := flag.String("repo", `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`, "racine des films")
	match := flag.String("match", "000d5950", "match")
	flag.Parse()
	dir := filepath.Join(*repo, "data", "cache", "film_chunks", *match)

	n := filmdec.CountFilmChunks(dir)
	type kfView struct {
		idx int
		ts  uint64
		pay []byte
		sp  []recSpan
	}
	var kfs []kfView
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			pay := p.Payload(chunk)
			kfs = append(kfs, kfView{idx: len(kfs) + 1, ts: p.TimestampUS, pay: pay, sp: spans(pay)})
		}
	}
	if len(kfs) == 0 {
		fmt.Fprintln(os.Stderr, "aucune image-cle")
		os.Exit(1)
	}
	fmt.Printf("=== i19 DU BIPED PAR ANCRAGE — %d images-cles\n\n", len(kfs))

	var nBiped, nWithHit int
	hitsPer := map[int]int{}
	offHist := map[int]int{}
	collisions, kfFull := 0, 0
	var detail []string

	for _, kf := range kfs {
		// Catalogue des entites JOUEUR de CETTE image-cle : c'est lui qui fait la selectivite.
		playerSlots := map[int]bool{}
		for _, s := range kf.sp {
			if s.ti == playerTI {
				playerSlots[s.slot] = true
			}
		}
		used := map[int][]int{} // slot joueur -> bipeds qui le designent
		nb := 0
		for _, s := range kf.sp {
			if s.ti != bipedTI {
				continue
			}
			nBiped++
			nb++
			hs := anchorI19(kf.pay, s.from, s.to, playerSlots)
			hitsPer[len(hs)]++
			if len(hs) == 0 {
				continue
			}
			nWithHit++
			for _, h := range hs {
				offHist[h.off]++
				used[h.slot] = append(used[h.slot], s.slot)
			}
			if len(detail) < 24 {
				detail = append(detail, fmt.Sprintf("  kf%-3d biped %-5d -> %d touche(s) : %v",
					kf.idx, s.slot, len(hs), fmtHits(hs)))
			}
		}
		dup := 0
		for _, owners := range used {
			if len(owners) > 1 {
				dup++
			}
		}
		collisions += dup
		if len(used) >= 8 && dup == 0 {
			kfFull++
		}
	}

	fmt.Printf("records de biped examines : %d\n", nBiped)
	fmt.Printf("records portant au moins une touche : %d (%.1f %%)\n",
		nWithHit, 100*float64(nWithHit)/float64(max(nBiped, 1)))
	fmt.Printf("touches par record : %s\n\n", fmtMap(hitsPer))
	for _, d := range detail {
		fmt.Println(d)
	}
	fmt.Printf("\n--- LE CONTROLE QUI PEUT ECHOUER ---\n")
	fmt.Printf("collisions (deux bipeds designant la MEME entite joueur) : %d\n", collisions)
	fmt.Printf("images-cles ou 8 entites joueur distinctes sont designees : %d / %d\n", kfFull, len(kfs))
	fmt.Printf("\ndecalages les plus frequents (bits depuis le debut du record) :\n")
	fmt.Printf("  %s\n", topN(offHist, 10))
	fmt.Printf("\nESPERANCE PAR HASARD : ~9,8e-9 par position, soit ~0,005 faux positif par image-cle.\n")
}

func fmtHits(hs []hit) string {
	s := ""
	for i, h := range hs {
		if i >= 4 {
			s += "…"
			break
		}
		s += fmt.Sprintf("[+%d slot %d gen %d sel %d] ", h.off, h.slot, h.gen, h.sel)
	}
	return s
}

func fmtMap(m map[int]int) string {
	var ks []int
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	s := ""
	for _, k := range ks {
		s += fmt.Sprintf("%d:%d ", k, m[k])
	}
	return s
}

func topN(m map[int]int, n int) string {
	type kv struct{ k, v int }
	var l []kv
	for k, v := range m {
		l = append(l, kv{k, v})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].v != l[j].v {
			return l[i].v > l[j].v
		}
		return l[i].k < l[j].k
	})
	s := ""
	for i := 0; i < n && i < len(l); i++ {
		s += fmt.Sprintf("+%d(x%d) ", l[i].k, l[i].v)
	}
	return s
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
