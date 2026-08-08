package main

// measures.go — LES PASSES DE MESURE. main.go porte la lecture du film et la reference ;
// fit.go porte la deconvolution. Ce fichier porte les quatre passes descriptives.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

// ─────────────────────────────────────────────────────────────────────────────
//  Passe principale
// ─────────────────────────────────────────────────────────────────────────────

func runMain(root string, pfxs []string, refs map[string]*matchRef, outCSV, outPlr string) {
	matchW, closeMatch := openCSV(outCSV, []string{
		"pfx", "famille", "pair_name", "roster_api", "indices_film",
		"records_longs", "records_courts", "pas_total", "pas_total_avec_courts",
		"porteurs_tir", "porteurs_tous", "api_shots_fired", "api_shots_hit",
	})
	defer closeMatch()
	plrW, closePlr := openCSV(outPlr, []string{
		"pfx", "famille", "indice", "records_longs", "records_courts",
		"pas_total", "pas_1", "pas_2", "pas_3plus", "pas_gros", "pas_ambigus",
	})
	defer closePlr()

	for _, pfx := range pfxs {
		m := refs[pfx]
		recs := scanRecords(filepath.Join(root, pfx), bitCounter)
		agg := aggregate(recs)

		var recLong, recShort, gapSum, gapAll int
		for _, a := range agg {
			recLong += a.records
			recShort += a.shortRecs
			gapSum += a.gapSum
			gapAll += a.gapSumAll
		}
		// OU SONT LES TOUCHES DE PROJECTILE ? Un degat applique produit un record type 105
		// a table non vide (un PORTEUR). Sur arme a trace instantanee c est le record de tir
		// lui-meme ; sur projectile le degat arrive plus tard, donc le porteur d impact — s il
		// existe — est un AUTRE record, et rien ne dit qu il porte l identifiant d arme du
		// suffixe commun. On compte donc les porteurs DEUX FOIS : sur la population de tir
		// (`isFire`) et sur TOUS les records longs, filtre d arme compris. L ecart entre les
		// deux est exactement la population que `isFire` jette.
		var portFire, portAll int
		for _, r := range recs {
			if !r.long || !r.porteur {
				continue
			}
			portAll++
			if isFire(r) {
				portFire++
			}
		}
		var apiFired, apiHit int
		for _, p := range m.players {
			apiFired += p.shotsFired
			apiHit += p.shotsHit
		}
		fam := familyOf(m.pairName)
		writeRow(matchW, pfx, fam, m.pairName, len(m.players), len(agg),
			recLong, recShort, gapSum, gapAll, portFire, portAll, apiFired, apiHit)

		if plrW != nil {
			for _, pi := range sortedKeys(agg) {
				a := agg[pi]
				var s1, s2, s3, big int
				for step, n := range a.steps {
					switch {
					case step == 1:
						s1 += n
					case step == 2:
						s2 += n
					case step >= 3 && step < 32:
						s3 += n
					default:
						big += n
					}
				}
				writeRow(plrW, pfx, fam, pi, a.records, a.shortRecs, a.gapSum, s1, s2, s3, big, a.gapAmbig)
			}
		}
		fmt.Fprintf(os.Stderr, "%s %-10s longs=%6d courts=%6d pas=%7d api_tirs=%6d api_touches=%6d\n",
			pfx, fam, recLong, recShort, gapSum, apiFired, apiHit)
	}
}

// runAlign est le GATE DE REPRODUCTION : P(pas = +1) doit valoir ~0.9738 a la position
// retenue, ~0.50 a -1 bit et ~0.002 a +1 bit (7ter.80 (1)). Un instrument qui ne
// reproduit pas ce profil n est pas aligne sur le champ, et rien de ce qu il mesure
// ensuite ne vaut.
func runAlign(root string, pfxs []string) {
	for _, off := range []int{bitCounter - 1, bitCounter, bitCounter + 1} {
		var pairs, plus1, zero int
		hist := map[int]int{}
		for _, pfx := range pfxs {
			recs := scanRecords(filepath.Join(root, pfx), off)
			byPlayer := map[int][]rec{}
			for _, r := range recs {
				if isFire(r) {
					byPlayer[r.shooter] = append(byPlayer[r.shooter], r)
				}
			}
			for _, rs := range byPlayer {
				for i := 1; i < len(rs); i++ {
					step := (rs[i].counter - rs[i-1].counter + counterMod) % counterMod
					pairs++
					hist[step]++
					if step == 1 {
						plus1++
					}
					if step == 0 {
						zero++
					}
				}
			}
		}
		fmt.Printf("bit %2d : paires=%8d  P(pas=+1)=%.4f  P(pas=0)=%.4f\n",
			off, pairs, ratio(plus1, pairs), ratio(zero, pairs))
		if off == bitCounter {
			fmt.Print("  histogramme des pas : ")
			for _, s := range []int{0, 1, 2, 3, 4, 5} {
				fmt.Printf("%d:%.4f ", s, ratio(hist[s], pairs))
			}
			var ge6, ge32 int
			for s, n := range hist {
				if s >= 6 {
					ge6 += n
				}
				if s >= 32 {
					ge32 += n
				}
			}
			fmt.Printf(">=6:%.4f >=32:%.4f\n", ratio(ge6, pairs), ratio(ge32, pairs))
		}
	}
}

// isFire retient les records de TIR : variante longue et moitie basse d identifiant
// d arme egale au suffixe commun. Les autres records type 105 (12.8 % en mode standard)
// portent P(pas = 0) = 0.63 — leur champ 26..32 n est pas un compteur de tir.
// LIMITE ASSUMEE : quelques armes reelles ont une autre moitie basse (MA5K) et sortent
// donc de cette population. C est une restriction, pas une correction.
func isFire(r rec) bool {
	return r.long && r.hasWeap && uint32(r.weapon) == commonWeaponSuffix
}

// runHdr ventile les records type 105 par CLASSE D EN-TETE (bits 8..11) et mesure
// P(pas = +1) dans chaque classe. But : retrouver la population exacte sur laquelle
// 7ter.80 a mesure 0.9738 — l ancien scanneur exigeait `hdr == 6` (ses bits de marqueur)
// et un identifiant d arme du catalogue ; le balayage par paquets n exige ni l un ni l autre.
func runHdr(root string, pfxs []string) {
	byHdr := map[int]*cls{}
	bySuffix := map[string]*cls{}
	for _, pfx := range pfxs {
		recs := scanRecords(filepath.Join(root, pfx), bitCounter)
		// chaines par (indice, classe) : un pas ne se calcule qu entre records comparables
		chains := map[[2]int][]rec{}
		sufChains := map[[2]int][]rec{}
		for _, r := range recs {
			if !r.long {
				continue
			}
			c := byHdr[r.hdr]
			if c == nil {
				c = &cls{}
				byHdr[r.hdr] = c
			}
			c.n++
			chains[[2]int{r.shooter, r.hdr}] = append(chains[[2]int{r.shooter, r.hdr}], r)
			s := 0
			if r.hasWeap && uint32(r.weapon) == commonWeaponSuffix {
				s = 1
			}
			k := "arme_autre"
			if s == 1 {
				k = "arme_42c9679f"
			}
			sc := bySuffix[k]
			if sc == nil {
				sc = &cls{}
				bySuffix[k] = sc
			}
			sc.n++
			sufChains[[2]int{r.shooter, s}] = append(sufChains[[2]int{r.shooter, s}], r)
		}
		for k, rs := range chains {
			accumSteps(byHdr[k[1]], rs)
		}
		for k, rs := range sufChains {
			name := "arme_autre"
			if k[1] == 1 {
				name = "arme_42c9679f"
			}
			accumSteps(bySuffix[name], rs)
		}
	}
	fmt.Println("— par classe d en-tete (bits 8..11) —")
	keys := make([]int, 0, len(byHdr))
	for k := range byHdr {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		c := byHdr[k]
		fmt.Printf("hdr=%2d  records=%8d  paires=%8d  P(+1)=%.4f  P(0)=%.4f\n",
			k, c.n, c.pairs, ratio(c.plus1, c.pairs), ratio(c.zero, c.pairs))
	}
	fmt.Println("— par moitie basse d identifiant d arme —")
	for _, k := range []string{"arme_42c9679f", "arme_autre"} {
		c := bySuffix[k]
		if c == nil {
			continue
		}
		fmt.Printf("%-14s records=%8d  paires=%8d  P(+1)=%.4f  P(0)=%.4f\n",
			k, c.n, c.pairs, ratio(c.plus1, c.pairs), ratio(c.zero, c.pairs))
	}
}

// commonWeaponSuffix est la moitie basse partagee par 95 % des identifiants d arme.
const commonWeaponSuffix = 0x42c9679f

// cls compte une population de records et les pas de compteur mesures dedans.
type cls struct{ n, pairs, plus1, zero int }

func accumSteps(c *cls, rs []rec) {
	for i := 1; i < len(rs); i++ {
		step := (rs[i].counter - rs[i-1].counter + counterMod) % counterMod
		c.pairs++
		if step == 1 {
			c.plus1++
		}
		if step == 0 {
			c.zero++
		}
	}
}

// runWeapon est LE TEST DE LA PISTE E, et il ne demande NI reference API NI appariement
// indice -> xuid : le PAS MOYEN DU COMPTEUR entre deux records CONSECUTIFS DE LA MEME ARME
// chez le MEME tireur.
//
// Ce que chaque hypothese predit, et c est ce qui rend la mesure decisive :
//
//	H1 « le compteur compte les tirs, records d arme a projectile = TOUCHES »
//	    -> pas moyen ~1 sur arme a trace instantanee (un record par tir)
//	    -> pas moyen ~1/precision (~2.5 a 4) sur arme a projectile
//	H0 « le compteur n avance que sur ce qui emet un record »
//	    -> pas moyen ~1 PARTOUT, sans contraste entre les deux familles d armes
//
// Un pas moyen plat serait le refus : la precision par arme a projectile ne serait pas
// derivable de ce champ.
func runWeapon(root string, pfxs []string, names map[uint64]string, outCSV string) {
	type wagg struct {
		records  int
		porteurs int
		pairs    int
		steps    int
		hist     map[int]int
		dtMS     []float64 // ecarts de temps des paires a pas = +1 (cadence)
	}
	byW := map[uint64]*wagg{}
	get := func(w uint64) *wagg {
		a := byW[w]
		if a == nil {
			a = &wagg{hist: map[int]int{}}
			byW[w] = a
		}
		return a
	}
	var ambigPairs, ambigSteps int
	for _, pfx := range pfxs {
		recs := scanRecords(filepath.Join(root, pfx), bitCounter)
		byPlayer := map[int][]rec{}
		for _, r := range recs {
			if isFire(r) {
				byPlayer[r.shooter] = append(byPlayer[r.shooter], r)
			}
		}
		for _, rs := range byPlayer {
			for i := range rs {
				wa := get(rs[i].weapon)
				wa.records++
				if rs[i].porteur {
					wa.porteurs++
				}
				if i == 0 {
					continue
				}
				step := (rs[i].counter - rs[i-1].counter + counterMod) % counterMod
				if step >= 32 { // rupture de chaine (chunk absent, longue absence) : hors mesure
					continue
				}
				if rs[i].weapon != rs[i-1].weapon {
					ambigPairs++
					ambigSteps += step
					continue
				}
				a := get(rs[i].weapon)
				a.pairs++
				a.steps += step
				a.hist[step]++
				if step == 1 && rs[i].tsUS >= rs[i-1].tsUS {
					a.dtMS = append(a.dtMS, float64(rs[i].tsUS-rs[i-1].tsUS)/1000)
				}
			}
		}
	}
	w, closeW := openCSV(outCSV, []string{"weapon_id", "arme", "records", "porteurs", "taux_porteur", "paires", "pas_total", "pas_moyen", "p_pas1", "p_pas2", "p_pas3plus"})
	defer closeW()
	ids := make([]uint64, 0, len(byW))
	for id := range byW {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return byW[ids[i]].records > byW[ids[j]].records })
	fmt.Printf("%-22s %8s %9s %8s %9s %7s %9s %9s\n",
		"arme", "records", "tx_porteur", "paires", "pas_moyen", "P(1)", "cad_med_ms", "cad_q90_ms")
	for _, id := range ids {
		a := byW[id]
		if a.pairs < 30 {
			continue
		}
		var p3 int
		for s, n := range a.hist {
			if s >= 3 {
				p3 += n
			}
		}
		name := names[id]
		if name == "" {
			name = fmt.Sprintf("0x%016x", id)
		}
		med, q90 := quantile(a.dtMS, 0.5), quantile(a.dtMS, 0.9)
		fmt.Printf("%-22s %8d %9.4f %8d %9.4f %7.4f %9.1f %9.1f\n", name, a.records,
			ratio(a.porteurs, a.records), a.pairs, ratio(a.steps, a.pairs), ratio(a.hist[1], a.pairs), med, q90)
		writeRow(w, strconv.FormatUint(id, 10), name, a.records, a.porteurs,
			fmt.Sprintf("%.4f", ratio(a.porteurs, a.records)), a.pairs, a.steps,
			fmt.Sprintf("%.4f", ratio(a.steps, a.pairs)), fmt.Sprintf("%.4f", ratio(a.hist[1], a.pairs)),
			fmt.Sprintf("%.4f", ratio(a.hist[2], a.pairs)), fmt.Sprintf("%.4f", ratio(p3, a.pairs)),
			fmt.Sprintf("%.1f", med), fmt.Sprintf("%.1f", q90))
	}
	fmt.Printf("paires a armes DIFFERENTES (hors mesure) : %d, pas moyen %.4f\n",
		ambigPairs, ratio(ambigSteps, ambigPairs))
}
