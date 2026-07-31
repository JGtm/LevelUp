// tmp_defstate — THROWAWAY : mesure empirique de la default-state (Skip width) entre
// le typeIndex R(6) et le gate/mask d'un record biped #35 dans le keyframe type-2.
//
// QUESTION (mission) : la région "default-state" (380 bits pour le record Hydra calibré
// start=194126/defaultBits=380/rsp=2) est-elle CONSTANTE par archétype biped #35 ?
//
// MÉTHODE : pour CHAQUE littéral d'arme complet du payload (high32<<32|low32 catalogué),
// on calcule gateBit = litBit-1 (= StartBit du WST porteur, car gate@litBit-1, handle
// =high32 @litBit, variant=low32 @litBit+32). On brute-force (start, defaultBits, rsp)
// pour trouver le record biped #35 dont un weapon-state-type-info tombe PILE sur ce
// gateBit avec l'id64 reconstruit == l'arme cataloguée. On retient le hit au DesyncAt
// le plus profond (traversée la plus saine). On reporte (start, defaultBits, rsp) +
// l'arme + DesyncAt, et on conclut si defaultBits/rsp sont constants.
//
// NE COMMIT JAMAIS. Lecture seule. N'altère aucune logique de decode.
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
	"levelup/go-api/internal/analysis/filmdec"
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

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v = v << 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

func knownHigh32(v uint32) bool {
	for id := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return true
		}
	}
	return false
}

// lit = un littéral d'arme complet trouvé dans le payload.
type lit struct {
	bit  int
	id64 uint64
	name string
}

// carrier = un record biped #35 candidat porteur d'un littéral.
type carrier struct {
	start, defaultBits int
	rsp                uint32
	ti                 uint32
	compIdx            int
	desync             int
	ncomp              int
	endBit             int
}

func main() {
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	payload := extractType2(inflate(cache + "/chunk_02.bin"))
	fmt.Printf("registre : %d archétypes ; keyframe type-2 : %d octets (%d bits)\n",
		len(reg.Archetypes), len(payload), len(payload)*8)

	// 1) Collecte de TOUS les littéraux d'arme complets (high32<<32|low32 catalogué).
	var lits []lit
	total := len(payload) * 8
	for bp := 0; bp+64 <= total; bp++ {
		v := uint32(bitsAt(payload, bp, 32))
		if !knownHigh32(v) {
			continue
		}
		low := uint32(bitsAt(payload, bp+32, 32))
		id64 := (uint64(v) << 32) | uint64(low)
		if nm, ok := analysis.WeaponIDToName[id64]; ok {
			lits = append(lits, lit{bp, id64, nm})
		}
	}
	sort.Slice(lits, func(i, j int) bool { return lits[i].bit < lits[j].bit })
	fmt.Printf("\n%d littéraux d'arme complets dans le payload :\n", len(lits))
	for i, l := range lits {
		fmt.Printf("  [%2d] bit=%-7d gateBit=%-7d id64=0x%016x  %s\n", i, l.bit, l.bit-1, l.id64, l.name)
	}

	// 2) Cibles demandées par la mission : Hydra (contrôle calibré), Shock Rifle @198140,
	//    M41 SPNKr @200933, Mangler @203736. On résout AU MOINS ces 4 par leur bit littéral.
	//    On les retrouve dans la liste des littéraux par proximité de bit.
	type target struct {
		label  string
		litBit int
	}
	targets := []target{
		{"Hydra (contrôle)", 195323},
		{"Shock Rifle", 198140},
		{"M41 SPNKr", 200933},
		{"Mangler", 203736},
	}

	fmt.Printf("\n================ BRUTE-FORCE (start, defaultBits, rsp) par cible ================\n")
	type winner struct {
		label string
		c     carrier
		name  string
		litB  int
	}
	var winners []winner

	for _, tg := range targets {
		// Trouve le littéral le plus proche du bit annoncé (tolérance ±4 bits).
		var L *lit
		for i := range lits {
			if lits[i].bit >= tg.litBit-4 && lits[i].bit <= tg.litBit+4 {
				L = &lits[i]
				break
			}
		}
		if L == nil {
			fmt.Printf("\n  %-18s @~%d : AUCUN littéral d'arme catalogué à proximité\n", tg.label, tg.litBit)
			continue
		}
		best, allHits := bestCarrier(reg, payload, L.bit, L.id64)
		fmt.Printf("\n  %-18s  litBit=%d gateBit=%d  arme=%s  (%d hits #35)\n",
			tg.label, L.bit, L.bit-1, L.name, len(allHits))
		// Dump des hits triés par desync décroissant (-1 d'abord), pour voir si un hit
		// à defaultBits~380/rsp=2 existe et traverse proprement.
		for i, h := range allHits {
			if i >= 12 {
				fmt.Printf("        ... (%d hits supplémentaires)\n", len(allHits)-12)
				break
			}
			mark := ""
			if h.desync == -1 {
				mark = "  <<< TRAVERSÉE COMPLÈTE"
			}
			fmt.Printf("        start=%-7d d=%-3d rsp=%d compIdx=i%-2d desyncAt=i%-3d ncomp=%-2d endBit=%d%s\n",
				h.start, h.defaultBits, h.rsp, h.compIdx, h.desync, h.ncomp, h.endBit, mark)
		}
		if best == nil {
			fmt.Printf("      AUCUN record biped #35 ne place un WST exact sur gateBit=%d\n", L.bit-1)
			continue
		}
		fmt.Printf("      GAGNANT (desync le plus profond) : start=%-7d defaultBits=%-3d rsp=%d  ti=%d compIdx=i%d desyncAt=i%d ncomp=%d endBit=%d\n",
			best.start, best.defaultBits, best.rsp, best.ti, best.compIdx, best.desync, best.ncomp, best.endBit)
		winners = append(winners, winner{tg.label, *best, L.name, L.bit})
	}

	// 2bis) TEST DÉCISIF : fixer (defaultBits=380, rsp=2) = la calibration Hydra, et
	// chercher pour CHAQUE littéral un record biped #35 qui traverse PROPREMENT
	// (DesyncAt tardif/-1) en plaçant son WST sur gateBit. Si ça marche pour les 8
	// bipeds => 380/2 est CONSTANT par archétype. Sinon => non.
	fmt.Printf("\n================ TEST DÉCISIF : defaultBits=380, rsp=2 FIXES ================\n")
	fmt.Printf("(on cherche un start tel que la traversée biped #35 atteigne un WST exact sur gateBit\n")
	fmt.Printf(" AVEC une traversée saine — DesyncAt élevé/-1, pas un alignement fortuit start+d)\n")
	filmdec.SetRecordStateParam(2)
	for _, l := range lits {
		gateBit := l.bit - 1
		// Le record commence ~1766 bits avant son endBit (~litBit+~570). On balaie start.
		type fixHit struct {
			start, desync, ncomp, endBit, compIdx int
		}
		var fhits []fixHit
		for start := gateBit - 3200; start <= gateBit-20; start++ {
			if uint32(bitsAt(payload, start, 6)) != 35 {
				continue
			}
			br := filmdec.NewBitReader(payload)
			br.Skip(start)
			t := filmdec.TraverseEntity(br, reg, 380)
			if t.TypeIndex != 35 {
				continue
			}
			for _, c := range t.Comps {
				if c.Name != "weapon-state-type-info" || c.StartBit != gateBit {
					continue
				}
				h := uint32(bitsAt(payload, gateBit+1, 32))
				v := uint32(bitsAt(payload, gateBit+33, 32))
				if (uint64(h)<<32)|uint64(v) != l.id64 {
					continue
				}
				fhits = append(fhits, fixHit{start, t.DesyncAt, len(t.Comps), t.EndBit, c.Index})
			}
		}
		// Tri : desync -1 d'abord, puis le plus profond.
		sort.Slice(fhits, func(i, j int) bool {
			di, dj := fhits[i].desync, fhits[j].desync
			if di == -1 {
				di = 1 << 30
			}
			if dj == -1 {
				dj = 1 << 30
			}
			return di > dj
		})
		if len(fhits) == 0 {
			fmt.Printf("  %-16s gateBit=%-7d : AUCUN record #35 sain @380/2 ne place le WST ici\n", l.name, gateBit)
			continue
		}
		b := fhits[0]
		mark := ""
		if b.desync == -1 {
			mark = "  <<< TRAVERSÉE COMPLÈTE"
		} else if b.desync >= 44 {
			mark = "  (desync tardif = plausible)"
		}
		fmt.Printf("  %-16s gateBit=%-7d : start=%-7d desyncAt=i%-3d ncomp=%-2d compIdx=i%-2d endBit=%d (%d hits)%s\n",
			l.name, gateBit, b.start, b.desync, b.ncomp, b.compIdx, b.endBit, len(fhits), mark)
	}

	// 2quater) Pour les 4 cibles : trouver le START du biped #35 (R6==35) le plus
	// PROCHE en amont du WST principal, et mesurer à 380/2 jusqu'où la traversée va
	// (compIdx max atteint) et où tomberait le WST. Si à 380/2 chaque biped place un
	// WST sur SON gateBit avec une traversée profonde => 380 constant. On scanne aussi
	// d in {bestPerWeapon} pour voir quelle largeur fait atterrir le WST proprement.
	fmt.Printf("\n========== PROFIL biped #35 par cible : quel defaultBits aligne le WST sainement ==========\n")
	for _, tg := range targets {
		var L *lit
		for i := range lits {
			if lits[i].bit >= tg.litBit-4 && lits[i].bit <= tg.litBit+4 {
				L = &lits[i]
				break
			}
		}
		if L == nil {
			continue
		}
		gateBit := L.bit - 1
		// Pour chaque start #35 plausible, balaye d et retiens le hit qui (a) place le WST
		// sur gateBit avec id64 correct ET (b) a le desync le plus profond. On reporte le d.
		type prof struct {
			start, d, desync, ncomp int
		}
		var best *prof
		for start := gateBit - 2200; start <= gateBit-400; start++ {
			if uint32(bitsAt(payload, start, 6)) != 35 {
				continue
			}
			for d := 1; d <= 420; d++ {
				filmdec.SetRecordStateParam(2)
				br := filmdec.NewBitReader(payload)
				br.Skip(start)
				t := filmdec.TraverseEntity(br, reg, d)
				if t.TypeIndex != 35 {
					continue
				}
				for _, c := range t.Comps {
					if c.Name != "weapon-state-type-info" || c.StartBit != gateBit {
						continue
					}
					h := uint32(bitsAt(payload, gateBit+1, 32))
					v := uint32(bitsAt(payload, gateBit+33, 32))
					if (uint64(h)<<32)|uint64(v) != L.id64 {
						continue
					}
					dsc := t.DesyncAt
					eff := dsc
					if eff == -1 {
						eff = 1 << 30
					}
					if best == nil || eff > func() int {
						b := best.desync
						if b == -1 {
							return 1 << 30
						}
						return b
					}() {
						best = &prof{start, d, dsc, len(t.Comps)}
					}
				}
			}
		}
		if best == nil {
			fmt.Printf("  %-16s : aucun start #35 ne place le WST sur gateBit=%d (d in 1..420, rsp=2)\n", L.name, gateBit)
			continue
		}
		fmt.Printf("  %-16s : MEILLEUR start=%-7d d=%-3d (rsp=2) desyncAt=i%-3d ncomp=%-2d  [vrai biped si d=380 & desync profond]\n",
			L.name, best.start, best.d, best.desync, best.ncomp)
	}

	// 2-trace) Trace COMPLÈTE du record Hydra (référence) + meilleure tentative Shock Rifle
	// à 380/2 depuis le start #35 le plus proche, pour voir OÙ ça desync.
	fmt.Printf("\n========== TRACE référence biped Hydra (start=194126 d=380 rsp=2) ==========\n")
	traceRecord(reg, payload, 194126, 380, 2)

	// Liste des starts R(6)==35 dans la fenêtre amont de chaque WST principal, et trace
	// à 380/2 du plus proche pour Shock Rifle / M41.
	for _, tg := range []target{{"Shock Rifle", 198140}, {"M41 SPNKr", 200933}} {
		gateBit := tg.litBit - 1
		fmt.Printf("\n========== starts R6==35 en amont de %s (gateBit=%d) ==========\n", tg.label, gateBit)
		var nearest = -1
		for start := gateBit - 2200; start <= gateBit-400; start++ {
			if uint32(bitsAt(payload, start, 6)) == 35 {
				if nearest < 0 {
					nearest = start
				}
			}
		}
		// Compte total + trace du plus proche à 380/2.
		cnt := 0
		for start := gateBit - 2200; start <= gateBit-400; start++ {
			if uint32(bitsAt(payload, start, 6)) == 35 {
				cnt++
			}
		}
		fmt.Printf("  %d positions R6==35 dans [%d..%d] ; trace du plus proche=%d à 380/2 :\n",
			cnt, gateBit-2200, gateBit-400, nearest)
		if nearest >= 0 {
			traceRecord(reg, payload, nearest, 380, 2)
		}
	}

	// 2-offset) HYPOTHÈSE STRUCTURE RÉGULIÈRE : delta(start->WST) du biped Hydra = 1196.
	// Si les bipeds ont la MÊME structure, chaque biped-i démarre à WST_i - 1196 et
	// traverse à 380/2 jusqu'à -1. On teste start = gateBit+1-1196 (±60) à 380/2 pour
	// chaque cible, en EXIGEANT le WST exact sur gateBit ET desync profond.
	const deltaStartToWST = 195322 - 194126 // 1196 (gateBit Hydra - start Hydra)
	fmt.Printf("\n========== HYPOTHÈSE structure régulière : start = gateBit - %d (±80), 380/2 ==========\n", deltaStartToWST)
	for _, l := range lits {
		gateBit := l.bit - 1
		expStart := gateBit - deltaStartToWST
		type h2 struct{ start, desync, ncomp int }
		var bestH *h2
		for start := expStart - 80; start <= expStart+80; start++ {
			if start < 0 || uint32(bitsAt(payload, start, 6)) != 35 {
				continue
			}
			filmdec.SetRecordStateParam(2)
			br := filmdec.NewBitReader(payload)
			br.Skip(start)
			t := filmdec.TraverseEntity(br, reg, 380)
			if t.TypeIndex != 35 {
				continue
			}
			for _, c := range t.Comps {
				if c.Name != "weapon-state-type-info" || c.StartBit != gateBit {
					continue
				}
				hh := uint32(bitsAt(payload, gateBit+1, 32))
				vv := uint32(bitsAt(payload, gateBit+33, 32))
				if (uint64(hh)<<32)|uint64(vv) != l.id64 {
					continue
				}
				eff := t.DesyncAt
				if eff == -1 {
					eff = 1 << 30
				}
				cur := -2
				if bestH != nil {
					cur = bestH.desync
					if cur == -1 {
						cur = 1 << 30
					}
				}
				if eff > cur {
					bestH = &h2{start, t.DesyncAt, len(t.Comps)}
				}
			}
		}
		if bestH == nil {
			fmt.Printf("  %-16s expStart≈%-7d : pas de biped #35 sain @380/2 plaçant son WST ici\n", l.name, expStart)
		} else {
			tag := ""
			if bestH.desync == -1 {
				tag = "  <<< COMPLÈTE => 380/2 VALIDE pour ce biped"
			}
			fmt.Printf("  %-16s expStart≈%-7d : start=%-7d desyncAt=i%-3d ncomp=%-2d%s\n",
				l.name, expStart, bestH.start, bestH.desync, bestH.ncomp, tag)
		}
	}

	// 2ter) WALK CONSÉCUTIF depuis le endBit du record Hydra (195892). Le keyframe est
	// une cascade de records ; on décode le SUIVANT. Question : est-ce un autre biped #35
	// (typeIndex R6==35) ou un autre archétype ? Et à 380/2 traverse-t-il ? Cela tranche
	// entre "defaultBits variable" et "walk inter-archétype où 380 ne vaut que biped".
	fmt.Printf("\n================ WALK CONSÉCUTIF depuis endBit Hydra=195892 ================\n")
	walkRecords(reg, payload, 195892, 8)

	// 3) Synthèse : defaultBits et rsp sont-ils constants ?
	fmt.Printf("\n================ SYNTHÈSE ================\n")
	if len(winners) == 0 {
		fmt.Printf("Aucun gagnant — impossible de conclure.\n")
		return
	}
	dbSet := map[int]int{}
	rspSet := map[uint32]int{}
	fmt.Printf("%-18s %-9s %-12s %-5s %-8s %s\n", "cible", "start", "defaultBits", "rsp", "desyncAt", "arme")
	for _, w := range winners {
		dbSet[w.c.defaultBits]++
		rspSet[w.c.rsp]++
		fmt.Printf("%-18s %-9d %-12d %-5d i%-7d %s\n",
			w.label, w.c.start, w.c.defaultBits, w.c.rsp, w.c.desync, w.name)
	}
	fmt.Printf("\ndefaultBits distincts : ")
	for v, n := range dbSet {
		fmt.Printf("%d (×%d) ", v, n)
	}
	fmt.Printf("\nrsp distincts        : ")
	for v, n := range rspSet {
		fmt.Printf("%d (×%d) ", v, n)
	}
	fmt.Printf("\n")
	if len(dbSet) == 1 {
		fmt.Printf("=> defaultBits CONSTANT sur les %d records biped #35.\n", len(winners))
	} else {
		fmt.Printf("=> defaultBits VARIABLE => default-state NON fixe (content-dependante ou structure parseable).\n")
	}
	if len(rspSet) == 1 {
		fmt.Printf("=> recordStateParam CONSTANT.\n")
	} else {
		fmt.Printf("=> recordStateParam VARIABLE.\n")
	}

	// 4) Corrélation distance inter-records (le start gagnant - start gagnant précédent)
	//    vs defaultBits, pour détecter une dépendance au contenu.
	fmt.Printf("\n--- distances inter-records (start) ---\n")
	sort.Slice(winners, func(i, j int) bool { return winners[i].c.start < winners[j].c.start })
	for i, w := range winners {
		d := 0
		if i > 0 {
			d = w.c.start - winners[i-1].c.start
		}
		fmt.Printf("  start=%-7d (Δprev=%-6d) defaultBits=%-3d rsp=%d  %s\n",
			w.c.start, d, w.c.defaultBits, w.c.rsp, w.label)
	}
}

// traceRecord trace un record à (start, d, rsp) : tous les composants + WST décodés.
func traceRecord(reg *filmdec.Registry, payload []byte, start, d int, rsp uint32) {
	filmdec.SetRecordStateParam(rsp)
	br := filmdec.NewBitReader(payload)
	br.Skip(start)
	t := filmdec.TraverseEntity(br, reg, d)
	fmt.Printf("  typeIndex=%d desyncAt=i%d ncomp=%d endBit=%d\n", t.TypeIndex, t.DesyncAt, len(t.Comps), t.EndBit)
	for _, c := range t.Comps {
		extra := ""
		if c.Name == "weapon-state-type-info" {
			g := bitsAt(payload, c.StartBit, 1)
			h := uint32(bitsAt(payload, c.StartBit+1, 32))
			v := uint32(bitsAt(payload, c.StartBit+33, 32))
			id := (uint64(h) << 32) | uint64(v)
			nm := analysis.WeaponIDToName[id]
			extra = fmt.Sprintf("  gate=%d id64=0x%016x arme=%q", g, id, nm)
		}
		mark := ""
		if !c.Ported {
			mark = "  <<< DESYNC (non-porté)"
		}
		fmt.Printf("    i%-2d %-44s @bit%d%s%s\n", c.Index, c.Name, c.StartBit, extra, mark)
	}
}

// walkRecords décode n records consécutifs à partir de `at`. Pour chaque record on lit
// le typeIndex R(6), on affiche l'archétype, puis on tente la traversée à 380/2 et on
// reporte desync/endBit. Cela montre si les records consécutifs sont tous biped #35
// (=> 380 testable comme constante) ou des archétypes mélangés.
func walkRecords(reg *filmdec.Registry, payload []byte, at, n int) {
	filmdec.SetRecordStateParam(2)
	for k := 0; k < n; k++ {
		if at+6 > len(payload)*8 {
			fmt.Printf("  (fin du payload)\n")
			return
		}
		ti := uint32(bitsAt(payload, at, 6))
		arch, ok := reg.Archetype(int(ti))
		head := "?"
		if ok && len(arch.Components) > 0 {
			head = arch.Components[0]
		}
		br := filmdec.NewBitReader(payload)
		br.Skip(at)
		t := filmdec.TraverseEntity(br, reg, 380)
		// On affiche aussi le 1er WST atteint éventuel.
		wst := "—"
		for _, c := range t.Comps {
			if c.Name == "weapon-state-type-info" {
				h := uint32(bitsAt(payload, c.StartBit+1, 32))
				v := uint32(bitsAt(payload, c.StartBit+33, 32))
				id := (uint64(h) << 32) | uint64(v)
				nm := analysis.WeaponIDToName[id]
				if nm == "" {
					nm = fmt.Sprintf("0x%016x?", id)
				}
				wst = fmt.Sprintf("WST@%d=%s", c.StartBit, nm)
				break
			}
		}
		fmt.Printf("  rec[%d] @bit%-7d ti=%-3d [%-42s] @380/2: desyncAt=i%-3d ncomp=%-2d endBit=%d  %s\n",
			k, at, ti, head, t.DesyncAt, len(t.Comps), t.EndBit, wst)
		if t.DesyncAt != -1 {
			// Traversée non complète : le endBit n'est pas fiable pour continuer le walk.
			fmt.Printf("       (desync => endBit non fiable, walk interrompu)\n")
			return
		}
		at = t.EndBit
	}
}

// bestCarrier brute-force (start, defaultBits, rsp) pour trouver le record biped #35
// dont un weapon-state-type-info tombe PILE sur gateBit=litBit-1 et reconstruit id64.
// Retourne le hit au DesyncAt le plus profond (traversée la plus saine) + TOUS les hits triés.
func bestCarrier(reg *filmdec.Registry, payload []byte, litBit int, id64 uint64) (*carrier, []carrier) {
	gateBit := litBit - 1

	var hits []carrier
	seen := map[int]bool{} // dédup par start
	// Fenêtre start : un record biped fait ~1766 bits (195892-194126). Le WST i45 est
	// vers la fin du record, donc le start peut être jusqu'à ~1800 bits avant le gate.
	// On balaie large (3200) pour couvrir les variations. defaultBits 1..420 (380 attendu).
	lo := gateBit - 3200
	if lo < 0 {
		lo = 0
	}
	hi := gateBit - 20
	for start := lo; start <= hi; start++ {
		// Court-circuit : le typeIndex R(6) du record doit valoir 35.
		if uint32(bitsAt(payload, start, 6)) != 35 {
			continue
		}
		for d := 1; d <= 420; d++ {
			for r := uint32(0); r <= 3; r++ {
				filmdec.SetRecordStateParam(r)
				br := filmdec.NewBitReader(payload)
				br.Skip(start)
				t := filmdec.TraverseEntity(br, reg, d)
				if t.TypeIndex != 35 {
					continue
				}
				for _, c := range t.Comps {
					if c.Name != "weapon-state-type-info" || c.StartBit != gateBit {
						continue
					}
					h := uint32(bitsAt(payload, gateBit+1, 32))
					v := uint32(bitsAt(payload, gateBit+33, 32))
					if (uint64(h)<<32)|uint64(v) != id64 {
						continue
					}
					if seen[start] {
						continue
					}
					seen[start] = true
					hits = append(hits, carrier{start, d, r, t.TypeIndex, c.Index, t.DesyncAt, len(t.Comps), t.EndBit})
				}
			}
		}
	}
	if len(hits) == 0 {
		return nil, nil
	}
	// Meilleur = DesyncAt le plus profond (et -1 = traversée complète gagne). On traite
	// -1 comme +infini (record entièrement consommé).
	sort.Slice(hits, func(i, j int) bool {
		di, dj := hits[i].desync, hits[j].desync
		if di == -1 {
			di = 1 << 30
		}
		if dj == -1 {
			dj = 1 << 30
		}
		if di != dj {
			return di > dj
		}
		return hits[i].start < hits[j].start
	})
	best := hits[0]
	return &best, hits
}
