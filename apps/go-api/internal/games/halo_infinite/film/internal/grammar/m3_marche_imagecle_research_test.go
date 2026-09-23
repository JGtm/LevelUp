//go:build research

package grammar

// m3_marche_imagecle_research_test.go — LOT M3.1, INSTRUMENT DE MESURE : la regle d election du
// balayeur d image-cle (production contre V2 « generation 1 la plus proche »), AVEC et SANS la
// fenetre de 120 000 bits, sur tous les paquets d image-cle d un film.
//
// Le « deraillement » que le lot 5.20.1 avait mesure sans fenetre (dad793c7, chunks 2 a 5 :
// 187 -> 127 records) se lit ici de trois facons : le nombre d ancres, les bipedes atteints, et
// la COHERENCE des ancres — une ancre dont l archetype contredit celui que le MEME slot porte
// dans la majorite des images-cles du film est une ancre prise a une position fausse.
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestM3MarcheV2$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

// m3ScanV2 est `kfScanNext` sous la regle V2 de la sonde P2 (`p2ElectionV2`) : un voisin
// immediat de generation 1 trouve N IMPORTE OU dans la fenetre gagne (chemin rapide inchange) ;
// sinon le candidat de generation 1 le PLUS PROCHE ; a defaut, la regle de production.
func m3ScanV2(buf []byte, from, prevSlot, total, maxWin int) (at int) {
	return m3ScanRegle(buf, from, prevSlot, total, maxWin, false)
}

// m3ScanV2c est V2, mais un voisin immediat de generation >= 2 garde la priorite sur un
// candidat non consecutif de generation 1 (la priorite « consecutif d abord » de la production).
func m3ScanV2c(buf []byte, from, prevSlot, total, maxWin int) (at int) {
	return m3ScanRegle(buf, from, prevSlot, total, maxWin, true)
}

func m3ScanRegle(buf []byte, from, prevSlot, total, maxWin int, consecutifDAbord bool) (at int) {
	at = -1
	best := kfCand{consecutive: -1, gen: 1 << 30, slot: 1 << 30, bit: 1 << 30}
	proche := -1
	end := from + maxWin
	if end > total {
		end = total
	}
	sentStreak := 0
	for q := from; q+64 <= end; q++ {
		id := kfReadBits(buf, q, 32)
		if id == kfSent {
			if sentStreak++; sentStreak >= 2048 {
				break
			}
			continue
		}
		sentStreak = 0
		s, _, g, ok := kfAnchorFromID(buf, q, id, prevSlot, total)
		if !ok {
			continue
		}
		if s == prevSlot+1 && g == 1 {
			return q
		}
		if g == 1 && proche < 0 {
			proche = q
		}
		cand := kfCand{gen: g, slot: s, bit: q}
		if s == prevSlot+1 {
			cand.consecutive = 1
		}
		if at < 0 || cand.betterThan(best) {
			at, best = q, cand
		}
	}
	if consecutifDAbord && best.consecutive == 1 {
		return at
	}
	if proche >= 0 {
		return proche
	}
	return at
}

// m3ScanRecal est la regle de PRODUCTION, plus un RECALAGE : si un en-tete EXACT de bipede
// (identifiant valide, mot d archetype de 32 bits egal a 35) precede l ancre que l election
// retient, c est lui qui est pris. Un tel en-tete a un faux positif de l ordre de 2^-32 par
// position ; une ancre elue PLUS LOIN que lui et de slot plus bas est necessairement fausse
// (la table est a slots croissants).
func m3ScanRecal(buf []byte, from, prevSlot, total, maxWin int) (at int) {
	at = -1
	best := kfCand{consecutive: -1, gen: 1 << 30, slot: 1 << 30, bit: 1 << 30}
	exact := -1
	end := from + maxWin
	if end > total {
		end = total
	}
	sentStreak := 0
	for q := from; q+64 <= end; q++ {
		id := kfReadBits(buf, q, 32)
		if id == kfSent {
			if sentStreak++; sentStreak >= 2048 {
				break
			}
			continue
		}
		sentStreak = 0
		s, _, g, ok := kfAnchorFromID(buf, q, id, prevSlot, total)
		if !ok {
			continue
		}
		if s == prevSlot+1 && g == 1 {
			return q
		}
		if exact < 0 && g == 1 && kfReadBits(buf, q+32, 32) == BipedTypeIndex {
			exact = q
		}
		cand := kfCand{gen: g, slot: s, bit: q}
		if s == prevSlot+1 {
			cand.consecutive = 1
		}
		if at < 0 || cand.betterThan(best) {
			at, best = q, cand
		}
	}
	if exact >= 0 && (at < 0 || exact < at) {
		return exact
	}
	return at
}

// m3ScanRecalExt est RECAL dont une fenetre SANS AUCUN candidat n arrete plus la marche : la
// recherche glisse de fenetre en fenetre jusqu au premier candidat (ou la fin du payload).
func m3ScanRecalExt(buf []byte, from, prevSlot, total, maxWin int) int {
	for f := from; f+64 <= total; f += maxWin {
		if at := m3ScanRecal(buf, f, prevSlot, total, maxWin); at >= 0 {
			return at
		}
		if m3SentinelleAvant(buf, f, total, maxWin) {
			return -1
		}
	}
	return -1
}

// m3SentinelleAvant dit si la fenetre [f, f+maxWin) contient la fin de table (2048 sentinelles).
func m3SentinelleAvant(buf []byte, f, total, maxWin int) bool {
	end := f + maxWin
	if end > total {
		end = total
	}
	streak := 0
	for q := f; q+32 <= end; q++ {
		if kfReadBits(buf, q, 32) == kfSent {
			if streak++; streak >= 2048 {
				return true
			}
			continue
		}
		streak = 0
	}
	return false
}

// m3ScanProdAncien est `kfScanNext` TEL QU IL ETAIT sur la base fe7079f41 (avant le lot M3.1) :
// voisin immediat, sinon election `betterThan`, fenetre qui ARRETE la marche quand elle est vide.
func m3ScanProdAncien(buf []byte, from, prevSlot, total, maxWin int) (at int) {
	at = -1
	best := kfCand{consecutive: -1, gen: 1 << 30, slot: 1 << 30, bit: 1 << 30}
	end := from + maxWin
	if end > total {
		end = total
	}
	sentStreak := 0
	for q := from; q+64 <= end; q++ {
		id := kfReadBits(buf, q, 32)
		if id == kfSent {
			if sentStreak++; sentStreak >= 2048 {
				break
			}
			continue
		}
		sentStreak = 0
		s, _, g, ok := kfAnchorFromID(buf, q, id, prevSlot, total)
		if !ok {
			continue
		}
		if s == prevSlot+1 && g == 1 {
			return q
		}
		cand := kfCand{gen: g, slot: s, bit: q}
		if s == prevSlot+1 {
			cand.consecutive = 1
		}
		if at < 0 || cand.betterThan(best) {
			at, best = q, cand
		}
	}
	return
}

// m3ScanProdM31 est la regle de production du lot M3.1 (voisin, recalage, election, fenetre
// glissante), vue au travers de la signature des variantes.
func m3ScanProdM31(buf []byte, from, prevSlot, total, maxWin int) int {
	at, _, _ := kfScanGlissant(buf, from, prevSlot, total, maxWin)
	return at
}

// m3Walk est `walkKeyframeWorldFenetre` avec une fonction de balayage injectee.
func m3Walk(buf []byte, maxWin int, scan func(buf []byte, from, prevSlot, total, maxWin int) int) []KeyframeRec {
	total := len(buf) * 8
	if maxWin <= 0 {
		maxWin = total
	}
	width := map[int]int{}
	seen := map[int]int{}
	var out []KeyframeRec
	pos, prev := 1, -1
	if _, _, _, ok := kfValidAnchor(buf, pos, prev, total); !ok {
		pos = scan(buf, pos, prev, total, maxWin)
	}
	for pos >= 0 {
		slot, ti, gen, ok := kfValidAnchor(buf, pos, prev, total)
		if !ok {
			break
		}
		startState := pos + 64
		var nat int
		if w, has := width[ti]; has {
			if _, _, jg, vok := kfValidAnchor(buf, startState+w, slot, total); vok && jg == 1 {
				nat = startState + w
			} else {
				nat = scan(buf, startState, slot, total, maxWin)
			}
		} else {
			nat = scan(buf, startState, slot, total, maxWin)
		}
		out = append(out, KeyframeRec{Slot: slot, TI: ti, Gen: gen, Bit: pos})
		prev = slot
		if nat < 0 {
			break
		}
		w := nat - startState
		if prevW, s := seen[ti]; s {
			if prevW == w {
				width[ti] = w
			} else {
				delete(width, ti)
			}
		} else {
			seen[ti] = w
		}
		pos = nat
	}
	return out
}

// m3Variante2 nomme une lecture.
type m3VarMarche struct {
	nom    string
	maxWin int
	scan   func(buf []byte, from, prevSlot, total, maxWin int) int
}

func TestM3MarcheV2(t *testing.T) {
	if os.Getenv("M3_MARCHE") == "" {
		t.Skip("M3_MARCHE absent")
	}
	tc := t516Cadre(t)
	vars := []m3VarMarche{
		{"PROD", kfScanFenetreBits, m3ScanProdAncien},
		{"PROD-sans", 0, m3ScanProdAncien},
		{"V2", kfScanFenetreBits, m3ScanV2},
		{"V2-sans", 0, m3ScanV2},
		{"RECAL", kfScanFenetreBits, m3ScanRecal},
		{"RECAL-sans", 0, m3ScanRecal},
	}
	type lecture struct {
		recs []KeyframeRec
	}
	type kf struct {
		ch  int
		pay int
		par map[string]lecture
	}
	var kfs []kf
	// archetype MAJORITAIRE de chaque slot, sur les images-cles de la lecture de production
	vote := map[int]map[int]int{}
	for _, ch := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(ch)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			k := kf{ch: ch, pay: len(pay) * 8, par: map[string]lecture{}}
			for _, v := range vars {
				recs := m3Walk(pay, v.maxWin, v.scan)
				k.par[v.nom] = lecture{recs}
				if v.nom == "PROD" {
					for _, r := range recs {
						if vote[r.Slot] == nil {
							vote[r.Slot] = map[int]int{}
						}
						vote[r.Slot][r.TI]++
					}
				}
			}
			kfs = append(kfs, k)
		}
	}
	major := map[int]int{}
	for s, m := range vote {
		best, n := -1, 0
		for ti, c := range m {
			if c > n || c == n && ti < best {
				best, n = ti, c
			}
		}
		major[s] = best
	}
	type tot struct{ recs, bip, incoherents, inconnus, fin int }
	sommes := map[string]*tot{}
	for _, v := range vars {
		sommes[v.nom] = &tot{}
	}
	for _, k := range kfs {
		ligne := fmt.Sprintf("ch%-3d %7d bits :", k.ch, k.pay)
		for _, v := range vars {
			l := k.par[v.nom]
			s := sommes[v.nom]
			var bip, inco, inc int
			fin := 0
			for _, r := range l.recs {
				if r.TI == BipedTypeIndex {
					bip++
				}
				m, ok := major[r.Slot]
				switch {
				case !ok:
					inc++
				case m != r.TI:
					inco++
				}
				if r.Bit > fin {
					fin = r.Bit
				}
			}
			s.recs += len(l.recs)
			s.bip += bip
			s.incoherents += inco
			s.inconnus += inc
			s.fin += fin
			ligne += fmt.Sprintf(" | %s %d rec %d bip %d inco %d inc %.0f%%", v.nom, len(l.recs), bip, inco,
				inc, 100*float64(fin)/float64(k.pay))
		}
		t.Logf("   %s", ligne)
	}
	noms := make([]string, 0, len(vars))
	for _, v := range vars {
		noms = append(noms, v.nom)
	}
	sort.Strings(noms)
	for _, n := range noms {
		s := sommes[n]
		t.Logf("== %-9s : %d images-cles, %d ancres, %d bipedes, %d ancres INCOHERENTES (archetype contredit), "+
			"%d ancres de slot inconnu de la production", n, len(kfs), s.recs, s.bip, s.incoherents, s.inconnus)
	}
}
