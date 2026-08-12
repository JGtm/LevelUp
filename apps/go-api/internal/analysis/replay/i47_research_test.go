package replay

// i47_research_test.go — INSTRUMENT DE MESURE de la GRENADE SÉLECTIONNÉE (i47) dans les
// records de biped des images-clés (lot portage POC, phase A.1 —
// .ai/V7.5/replay2d/PLAN_PORTAGE_POC_FICHES_KILLFEED.md).
//
// CE QU'IL CHERCHE. La grammaire d'i47 est ÉTABLIE par la capture du dispatch
// (.ai/V7.5/replay2d/RECETTE_LOADOUT_2026-07-27.md §1) : 9 bits, [6 bits masque][3 bits
// sélection], le masque étant EXACTEMENT le bitmap des compteurs i22 non nuls et la sélection
// l'index en base 1 (accord i22↔i47 : 194/194 au dispatch). Ce qui MANQUE est sa POSITION dans
// le record d'image-clé — le POC ne l'a jamais localisé (ses 98 sélections affichées viennent
// toutes de la déduction « unique compteur non nul »).
//
// MÉTHODE. Pour chaque record biped dont i22 est lu (mêmes ancres que le décodeur de
// production), on balaye le record à la recherche du motif attendu et on publie la DISTRIBUTION
// des positions par rapport à quatre repères : début du record, fin du bloc de munitions (i42),
// première famille d'arme, fin du record. Les records à UN SEUL type porté déterminent le motif
// en entier (sélection unique possible) : ce sont eux qui établissent la position. Les records
// à DEUX types portés la confrontent — là, la sélection est une INFORMATION, pas une déduction.
//
// IL NE MODIFIE RIEN : lecture seule, sous garde d'environnement (I47_FILM), sauté partout
// ailleurs, CI comprise.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I47_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/replay/ -run I47Location -timeout 10m -v

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const i47FilmEnv = "I47_FILM"

// i47Hit est une occurrence du motif attendu dans un record.
type i47Hit struct {
	bit      int  // position absolue du motif dans le payload
	fromRec  int  // offset depuis le début du record
	fromAmmo int  // offset depuis le bit d'arrivée du bloc i30..i42 (-1 si bloc non résolu)
	fromFam  int  // offset depuis la première famille d'arme (-1 si aucune)
	fromLast int  // offset depuis la FIN (+32) de la dernière famille d'arme (-1 si aucune)
	toEnd    int  // distance à la fin du record
	sel      int  // sélection lue (base 1)
	alt      bool // motif trouvé sous l'orientation ALTERNATIVE du masque (bit 3-r)
}

func TestI47LocationInKeyframeRecords(t *testing.T) {
	filmDir := os.Getenv(i47FilmEnv)
	if filmDir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i47FilmEnv)
	}
	known := loadoutFamilies()
	n := filmdec.CountFilmChunks(filmDir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", filmDir)
	}

	// Par nombre de types portés (1 ou 2+), par repère : distribution des offsets observés.
	offRec := map[int]map[int]int{1: {}, 2: {}}
	offAmmo := map[int]map[int]int{1: {}, 2: {}}
	offFam := map[int]map[int]int{1: {}, 2: {}}
	offLast := map[int]map[int]int{1: {}, 2: {}}
	offEnd := map[int]map[int]int{1: {}, 2: {}}
	records, withI22, hitsTotal, noHit, multiHit, altHits := 0, 0, 0, 0, 0, 0

	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(filmDir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			pay := p.Payload(chunk)
			for _, sp := range invRecordSpans(pay) {
				if sp.ti != invBipedTI {
					continue
				}
				records++
				hits, kinds, ok := i47ScanRecord(pay, sp, known)
				if !ok {
					continue
				}
				withI22++
				switch len(hits) {
				case 0:
					noHit++
					continue
				case 1:
				default:
					multiHit++
				}
				hitsTotal += len(hits)
				for _, h := range hits {
					if h.alt {
						altHits++
						continue // l'orientation alternative est comptée, pas mélangée aux offsets
					}
					offRec[kinds][h.fromRec]++
					if h.fromAmmo >= 0 {
						offAmmo[kinds][h.fromAmmo]++
					}
					if h.fromFam >= 0 {
						offFam[kinds][h.fromFam]++
					}
					if h.fromLast >= 0 {
						offLast[kinds][h.fromLast]++
					}
					offEnd[kinds][h.toEnd]++
				}
			}
		}
	}

	t.Logf("RECORDS biped %d · i22 lu %d · motifs trouvés %d (dont %d sous orientation ALT du masque)"+
		" · records sans motif %d · records à motifs multiples %d",
		records, withI22, hitsTotal, altHits, noHit, multiHit)
	for _, k := range []int{1, 2} {
		t.Logf("---- %d type(s) porté(s) ----", k)
		i47LogTop(t, "depuis le début du record", offRec[k])
		i47LogTop(t, "depuis la fin du bloc i30..i42", offAmmo[k])
		i47LogTop(t, "depuis la 1re famille d'arme", offFam[k])
		i47LogTop(t, "depuis la fin de la DERNIÈRE famille", offLast[k])
		i47LogTop(t, "jusqu'à la fin du record", offEnd[k])
	}

	// ANCRE CANDIDATE : pour les hits de la bande dominante (+190..+220 après la dernière
	// famille), les bits qui PRÉCÈDENT le motif portent-ils une constante ? Si oui, elle vaut
	// ancre à valeur imposée — le seul ancrage qui échappe au jitter.
	for _, span := range []int{16, 24, 32} {
		pre := map[uint32]int{}
		i47ForEachBandHit(t, filmDir, known, func(pay []byte, h i47Hit) {
			pre[invBits(pay, h.bit-span, span)]++
		})
		i47LogTopU32(t, fmt.Sprintf("préfixe %d bits avant le motif", span), pre)
	}

	// VALIDATION de la lecture « fenêtre unanime » : couverture, stabilité intra-vie, et
	// l'oracle des DÉCRÉMENTS — quand un compteur i22 décroît entre deux keyframes consécutifs
	// du même slot, un lancer de ce type a eu lieu entre les deux ; la sélection lue doit le
	// refléter. C'est la même méthode interne qui a établi la bijection type↔compteur (35/35).
	type i47State struct {
		ts     uint64
		gren   [invGrenadeSlots]uint32
		types  int
		selRk  int // rang sélectionné lu (fenêtre unanime), -1 sinon
		hadHit bool
	}
	bySlot := map[uint32][]i47State{}
	ambiguous := 0
	i47ForEachI22Record(t, filmDir, known, func(pay []byte, sp invRecordSpan, ts uint64,
		gren [invGrenadeSlots]uint32, types int, hits []i47Hit) {
		st := i47State{ts: ts, gren: gren, types: types, selRk: -1}
		sel := -2
		for _, h := range hits {
			if h.alt || h.fromLast < 200 || h.fromLast > 210 {
				continue
			}
			st.hadHit = true
			if sel == -2 {
				sel = h.sel
			} else if sel != h.sel {
				sel = -1
			}
		}
		if sel > 0 {
			st.selRk = sel - 1
		} else if sel == -1 {
			ambiguous++
		}
		bySlot[uint32(sp.slot)] = append(bySlot[uint32(sp.slot)], st)
	})
	read1, read2, total1, total2 := 0, 0, 0, 0
	for _, sts := range bySlot {
		for _, st := range sts {
			if st.types > 1 {
				total2++
				if st.selRk >= 0 {
					read2++
				}
			} else {
				total1++
				if st.selRk >= 0 {
					read1++
				}
			}
		}
	}
	t.Logf("LECTURE fenêtre [+200..+210] unanime — 1 type : %d/%d · 2+ types : %d/%d · ambiguës %d",
		read1, total1, read2, total2, ambiguous)

	// Stabilité : entre keyframes consécutifs d'un même slot (une vie), la sélection lue ne
	// doit changer que rarement — et l'oracle des décréments doit la voir juste. Le SEUL cas
	// probant est celui où le porteur a PLUSIEURS types (sinon sélection == seul type == type
	// lancé, tautologie) : il est compté à part.
	stable, changed := 0, 0
	decrTotal, before, beforeOK, after, afterOK, before2, before2OK := 0, 0, 0, 0, 0, 0, 0
	for _, sts := range bySlot {
		sort.Slice(sts, func(i, j int) bool { return sts[i].ts < sts[j].ts })
		for i := 1; i < len(sts); i++ {
			a, b := sts[i-1], sts[i]
			if a.selRk >= 0 && b.selRk >= 0 {
				if a.selRk == b.selRk {
					stable++
				} else {
					changed++
				}
			}
			// Décrément d'un SEUL rang entre a et b = un lancer de ce type entre les deux.
			down, downs := -1, 0
			for r := 0; r < invGrenadeSlots; r++ {
				if b.gren[r] < a.gren[r] {
					down = r
					downs++
				}
			}
			if downs != 1 {
				continue
			}
			decrTotal++
			if a.selRk >= 0 {
				before++
				if a.selRk == down {
					beforeOK++
				}
				if a.types > 1 {
					before2++
					if a.selRk == down {
						before2OK++
					}
				}
			}
			if b.selRk >= 0 {
				after++
				if b.selRk == down {
					afterOK++
				}
			}
		}
	}
	t.Logf("STABILITÉ intra-vie — paires lisibles stables %d · changements %d", stable, changed)
	t.Logf("ORACLE décréments — %d décréments unitaires · AVANT lisible %d dont accord %d"+
		" · dont NON TAUTOLOGIQUES (2+ types) %d dont accord %d · APRÈS lisible %d dont accord %d",
		decrTotal, before, beforeOK, before2, before2OK, after, afterOK)
}

// i47ForEachBandHit rejoue le balayage et rappelle fn pour chaque hit DIRECT de la bande
// dominante (+190..+220 depuis la fin de la dernière famille).
func i47ForEachBandHit(t *testing.T, filmDir string, known map[uint32]bool, fn func([]byte, i47Hit)) {
	t.Helper()
	i47ForEachI22Record(t, filmDir, known, func(pay []byte, _ invRecordSpan, _ uint64,
		_ [invGrenadeSlots]uint32, _ int, hits []i47Hit) {
		for _, h := range hits {
			if !h.alt && h.fromLast >= 190 && h.fromLast <= 220 {
				fn(pay, h)
			}
		}
	})
}

// i47ForEachI22Record rejoue le balayage et rappelle fn pour chaque record biped d'image-clé
// dont i22 est lu, avec ses compteurs, son nombre de types portés et ses hits de motif.
func i47ForEachI22Record(t *testing.T, filmDir string, known map[uint32]bool,
	fn func(pay []byte, sp invRecordSpan, ts uint64, gren [invGrenadeSlots]uint32, types int, hits []i47Hit)) {
	t.Helper()
	n := filmdec.CountFilmChunks(filmDir)
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(filmDir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			pay := p.Payload(chunk)
			for _, sp := range invRecordSpans(pay) {
				if sp.ti != invBipedTI {
					continue
				}
				hits, _, ok := i47ScanRecord(pay, sp, known)
				if !ok {
					continue
				}
				gren, types, ok := i47GrenOf(pay, sp)
				if !ok {
					continue
				}
				fn(pay, sp, p.TimestampUS, gren, types, hits)
			}
		}
	}
}

// i47GrenOf relit les compteurs i22 du record (mêmes ancres que le décodeur de production).
func i47GrenOf(pay []byte, sp invRecordSpan) ([invGrenadeSlots]uint32, int, bool) {
	var zero [invGrenadeSlots]uint32
	hits := invAbilityIn(pay, sp.from, sp.to)
	if len(hits) != 1 {
		return zero, 0, false
	}
	gren, ok := invGrenadesAfter(pay, hits[0].anchorBit, sp.to, DefaultGrenadeMax)
	if !ok {
		return zero, 0, false
	}
	types := 0
	for _, v := range gren {
		if v > 0 {
			types++
		}
	}
	return gren, types, true
}

// i47LogTopU32 publie les valeurs les plus fréquentes d'une distribution de mots.
func i47LogTopU32(t *testing.T, label string, dist map[uint32]int) {
	type kv struct {
		v uint32
		n int
	}
	all := make([]kv, 0, len(dist))
	total := 0
	for v, c := range dist {
		all = append(all, kv{v, c})
		total += c
	}
	sort.Slice(all, func(i, j int) bool { return all[i].n > all[j].n })
	s := ""
	for i, e := range all {
		if i == 6 {
			s += " …"
			break
		}
		s += fmt.Sprintf(" 0x%X×%d", e.v, e.n)
	}
	t.Logf("  %-32s %d valeurs distinctes / %d hits :%s", label, len(dist), total, s)
}

// i47ScanRecord balaye UN record : rend les occurrences du motif attendu, le nombre de types
// portés (1 ou 2 pour « 2 ou plus »), et ok=false si i22 n'est pas lu dans ce record.
func i47ScanRecord(pay []byte, sp invRecordSpan, known map[uint32]bool) ([]i47Hit, int, bool) {
	hits := invAbilityIn(pay, sp.from, sp.to)
	if len(hits) != 1 {
		return nil, 0, false
	}
	gren, ok := invGrenadesAfter(pay, hits[0].anchorBit, sp.to, DefaultGrenadeMax)
	if !ok {
		return nil, 0, false
	}
	// Deux orientations candidates du masque : bit r (LSB = rang 0) et bit 3-r. La recette dit
	// « bitmap des compteurs non nuls » sans fixer le sens ; l'instrument mesure les deux.
	var mask, maskAlt uint32
	types := 0
	for r, v := range gren {
		if v > 0 {
			mask |= 1 << uint(r)
			maskAlt |= 1 << uint(3-r)
			types++
		}
	}
	kinds := 1
	if types > 1 {
		kinds = 2
	}
	// Repères : familles d'arme (toutes les occurrences), bloc munitions (mêmes chemins que le
	// décodeur de production).
	fams := i47AllFamilies(pay, sp.from, sp.to, known)
	famBit, hasFam := 0, len(fams) > 0
	lastFamEnd := -1
	if hasFam {
		famBit = fams[0]
		lastFamEnd = fams[len(fams)-1] + 32
	}
	ammoEnd := -1
	if hasFam {
		end := famBit - 1
		lo := end - invAmmoSearchSpan
		if lo < sp.from {
			lo = sp.from
		}
		if sols := invSolveAmmoBlock(pay, end, lo); len(sols) > 0 {
			_, _, e, okp := invParseAmmoBlock(pay, sols[0], end+1)
			if okp {
				ammoEnd = e
			}
		}
	}
	var out []i47Hit
	for b := sp.from; b+9 <= sp.to; b++ {
		w := invBits(pay, b, 6)
		direct := w == mask
		alt := !direct && w == maskAlt && mask != maskAlt
		if !direct && !alt {
			continue
		}
		sel := int(invBits(pay, b+6, 3))
		if sel < 1 || sel > invGrenadeSlots {
			continue
		}
		if direct && mask&(1<<uint(sel-1)) == 0 {
			continue
		}
		if alt && maskAlt&(1<<uint(invGrenadeSlots-sel)) == 0 {
			continue
		}
		h := i47Hit{bit: b, fromRec: b - sp.from, fromAmmo: -1, fromFam: -1, fromLast: -1,
			toEnd: sp.to - b, sel: sel, alt: alt}
		if ammoEnd >= 0 {
			h.fromAmmo = b - ammoEnd
		}
		if hasFam {
			h.fromFam = b - famBit
			h.fromLast = b - lastFamEnd
		}
		out = append(out, h)
	}
	return out, kinds, true
}

// i47AllFamilies rend la position bit de TOUTES les occurrences de famille connue du record,
// dans l'ordre des bits (invFirstFamily n'en rend que la première).
func i47AllFamilies(pay []byte, from, to int, known map[uint32]bool) []int {
	var out []int
	var w uint32
	for b := from; b < to; b++ {
		w = w<<1 | invBitAt(pay, b)
		if b-from < 31 {
			continue
		}
		if known[w] {
			out = append(out, b-31)
		}
	}
	return out
}

// i47LogTop publie les offsets les plus fréquents d'une distribution (jusqu'à 8).
func i47LogTop(t *testing.T, label string, dist map[int]int) {
	if len(dist) == 0 {
		t.Logf("  %-32s (aucune donnée)", label)
		return
	}
	type kv struct{ off, n int }
	all := make([]kv, 0, len(dist))
	total := 0
	for o, c := range dist {
		all = append(all, kv{o, c})
		total += c
	}
	sort.Slice(all, func(i, j int) bool { return all[i].n > all[j].n })
	s := ""
	for i, e := range all {
		if i == 8 {
			s += " …"
			break
		}
		s += fmt.Sprintf(" %+d×%d", e.off, e.n)
	}
	t.Logf("  %-32s %d offsets distincts / %d hits :%s", label, len(dist), total, s)
}
