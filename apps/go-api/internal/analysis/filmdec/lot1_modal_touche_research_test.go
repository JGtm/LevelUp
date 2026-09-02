package filmdec

// lot1_modal_touche_research_test.go — LOT 1 : « MODAL = RATÉ ? ». Le record
// action_weapon_fire (0xD2, type 36) est MODAL quand il ne porte NI cible NI composante de
// dégât en ligne. Question utilisateur : « modal » = un tir qui ne touche pas (raté), ou le
// coup au but est-il simplement rangé dans un damage_aftermath (0xC0, type 0) séparé ?
//
// PRINCIPE DE L'APPARIEMENT — sans résoudre aucun handle en slot. L'attaquant d'un tir (ref0,
// domaine 1, lu par lot1RefDom1 AVANT tout champ contesté de la grammaire) et le responsable
// d'un damage_aftermath (ref1, domaine 1, MÊME encodage) vivent dans le MÊME espace d'index
// (lot1_degats_blesse : les deux se résolvent avec la base 512 vers les slots bipèdes). On
// peut donc APPARIER un tir à un dégât par (même index de tireur, fenêtre temporelle), index
// brut contre index brut, sans base. Le drapeau MODAL vient du décodeur de PRODUCTION
// (modalAimBit, fire_aim_modal.go, grammaire Ghidra la plus récemment revérifiée) : zéro copie
// de grammaire ajoutée. L'attaquant se lit avant le champ « d » dont la polarité diffère entre
// lot1_tirs (recherche) et fire_aim_modal (production) — la lecture d'attaquant est donc, elle,
// non ambiguë.
//
// SEUILS / TÉMOINS ÉCRITS AVANT LA MESURE :
//
//	W    = 250 ms : fenêtre d'appariement (|ts_dégât - ts_tir| <= W). Sweep 60/130/250/500 ms
//	       publié pour le contexte ; le verdict porte sur W = 250 ms.
//	OFF  = 3 s : décalage du TÉMOIN. Un lien causal tir->dégât (quasi même tick) doit
//	       s'effondrer quand on cherche le dégât du même tireur autour de T+OFF au lieu de T.
//	heals: les damage_aftermath à magnitude négative (Kscale=-1, soin) sont EXCLUS du jeu de
//	       « coups au but » (un soin n'est pas un tir qui touche).
//
// VERDICT — DISCRIMINANT = LE RATIO AU TÉMOIN DÉCALÉ (pas un taux absolu).
//	Un seuil ABSOLU de coïncidence a été écarté APRÈS coup, et le dire est le point : le taux
//	absolu tir-modal -> dégât est plafonné par la DENSITÉ des damage_aftermath dans le flux
//	(01e1f945 : 30 dégâts pour 491 tirs modaux), donc un seuil absolu mesure l'échantillonnage,
//	pas l'hypothèse. Le test sound compare la coïncidence même-tireur à la coïncidence FORTUITE
//	(même tireur, fenêtre décalée de OFF).
//	   « modal ≠ raté ; le coup au but vit dans un damage_aftermath séparé » est SOUTENU quand la
//	   coïncidence même-tireur dépasse le témoin décalé dans les DEUX sens — AVANT (le tir modal
//	   a un dégât) >= 1,5x, ARRIÈRE (le dégât vient d'un tir modal) >= 2x — l'arrière primant
//	   quand les dégâts sont rares.
//	   « modal ≈ raté » se lirait à l'inverse : coïncidence même-tireur AU NIVEAU du témoin
//	   (un tir raté ne produit aucun dégât).
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule.

import (
	"os"
	"sort"
	"testing"
)

// lot1mtFire : un action_weapon_fire horodaté. att = index d'attaquant (ref0 dom1) brut.
type lot1mtFire struct {
	ts    uint64
	att   uint64
	has   bool
	modal bool
}

// lot1mtHit : un damage_aftermath (non-soin) horodaté. resp = responsable (ref1 dom1) brut.
type lot1mtHit struct {
	ts   uint64
	resp uint64
	has  bool
}

// lot1mtDecodeFire lit l'attaquant (ref0) du record 0xD2 type 36 et son drapeau MODAL (via le
// décodeur de production). Rend ok=false si le paquet n'est pas un type 36.
func lot1mtDecodeFire(pay []byte, ts uint64) (lot1mtFire, bool) {
	br := NewBitReader(pay)
	br.Skip(2) // config + continuation
	if br.ReadBits(7) != 36 {
		return lot1mtFire{}, false
	}
	f := lot1mtFire{ts: ts}
	if att, ok := lot1RefDom1(br); ok { // ref0 = l'attaquant (domaine 1, sonde)
		f.att, f.has = att, true
	}
	_, f.modal = modalAimBit(pay) // production : ok == record MODAL (0 cible, 0 composante)
	return f, true
}

// lot1mtDecodeHit lit le responsable (ref1) d'un damage_aftermath (0xC0 type 0) et si c'est un
// soin. Rend (hit, estSoin, ok). ok=false si ce n'est pas un type 0.
func lot1mtDecodeHit(pay []byte, ts uint64) (lot1mtHit, bool, bool) {
	br := NewBitReader(pay)
	br.Skip(2)
	if br.ReadBits(7) != 0 { // type 1 (damage_section_response), pas damage_aftermath
		return lot1mtHit{}, false, false
	}
	h := lot1mtHit{ts: ts}
	lot1RefDom1(br) // ref0 = blessé (non utilisé ici)
	if resp, ok := lot1RefDom1(br); ok {
		h.resp, h.has = resp, true // ref1 = responsable (l'attaquant du dégât)
	}
	lot1RefDom(br, 7) // ref2 dom7
	r := lot1DecodeDamageAftermath(br)
	return h, r.negatif, true
}

// lot1mtCollect balaie n chunks et rend les tirs, les coups au but (soins exclus) et le
// nombre de soins écartés.
func lot1mtCollect(t *testing.T, dir string, n int) (fires []lot1mtFire, hits []lot1mtHit, soins int) {
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 4 {
				continue
			}
			pay := pk.Payload(data)
			switch pay[0] {
			case 0xD2:
				if f, ok := lot1mtDecodeFire(pay, pk.TimestampUS); ok {
					fires = append(fires, f)
				}
			case 0xC0:
				if h, soin, ok := lot1mtDecodeHit(pay, pk.TimestampUS); ok && h.has {
					if soin {
						soins++
						continue
					}
					hits = append(hits, h)
				}
			}
		}
	}
	return fires, hits, soins
}

// lot1mtNear rend vrai si sorted (croissant) contient une valeur dans [T-W, T+W].
func lot1mtNear(sorted []uint64, T, W uint64) bool {
	lo := uint64(0)
	if T > W {
		lo = T - W
	}
	i := sort.Search(len(sorted), func(i int) bool { return sorted[i] >= lo })
	return i < len(sorted) && sorted[i] <= T+W
}

// lot1mtIndexByKey range les horodatages par clé (tireur ou responsable), triés croissants.
func lot1mtIndexByKey(ts []uint64, keys []uint64) map[uint64][]uint64 {
	m := map[uint64][]uint64{}
	for i, k := range keys {
		m[k] = append(m[k], ts[i])
	}
	for k := range m {
		sort.Slice(m[k], func(a, b int) bool { return m[k][a] < m[k][b] })
	}
	return m
}

// lot1mtTally : compteurs d'un groupe de tirs (modal ou non-modal).
type lot1mtTally struct {
	n, anyNear, sameNear, shiftNear int
}

func (ta *lot1mtTally) add(f lot1mtFire, allHit []uint64, byResp map[uint64][]uint64, W, off uint64) {
	ta.n++
	if lot1mtNear(allHit, f.ts, W) {
		ta.anyNear++
	}
	if lot1mtNear(byResp[f.att], f.ts, W) {
		ta.sameNear++
	}
	if lot1mtNear(byResp[f.att], f.ts+off, W) {
		ta.shiftNear++
	}
}

func TestLot1ModalTouche(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}

	const (
		W   = uint64(250_000)   // 250 ms
		OFF = uint64(3_000_000) // 3 s (témoin)
	)

	fires, hits, soins := lot1mtCollect(t, dir, n)

	// Index des coups au but : tous, et par responsable.
	allHitTs := make([]uint64, len(hits))
	hitTs := make([]uint64, len(hits))
	hitResp := make([]uint64, len(hits))
	for i, h := range hits {
		allHitTs[i], hitTs[i], hitResp[i] = h.ts, h.ts, h.resp
	}
	sort.Slice(allHitTs, func(a, b int) bool { return allHitTs[a] < allHitTs[b] })
	hitByResp := lot1mtIndexByKey(hitTs, hitResp)

	// Index des tirs MODAUX par attaquant (pour la mesure inverse).
	var mfTs, mfAtt []uint64
	var nModal, nNonModal int
	for _, f := range fires {
		if f.modal {
			nModal++
		} else {
			nNonModal++
		}
		if f.modal && f.has {
			mfTs = append(mfTs, f.ts)
			mfAtt = append(mfAtt, f.att)
		}
	}
	modalFireByAtt := lot1mtIndexByKey(mfTs, mfAtt)

	// Tallies par groupe.
	var modalT, nonModalT lot1mtTally
	for _, f := range fires {
		if !f.has {
			continue
		}
		if f.modal {
			modalT.add(f, allHitTs, hitByResp, W, OFF)
		} else {
			nonModalT.add(f, allHitTs, hitByResp, W, OFF)
		}
	}

	nFires := nModal + nNonModal
	nHits := len(hits)
	t.Logf("== film %s (%d chunks) : %d tirs (0xD2 t36) · %d coups au but (0xC0 t0, hors %d soins) ==",
		dir, n, nFires, nHits, soins)
	t.Logf("TIRS : modaux %d (%.1f %%) · non-modaux %d (%.1f %%)",
		nModal, lot1Pct(nModal, nFires), nNonModal, lot1Pct(nNonModal, nFires))

	logGrp := func(nom string, ta lot1mtTally) {
		t.Logf("  %-10s (att connu %d) : dégât MÊME TIREUR ±%dms %d (%.1f %%) · témoin +%ds %d (%.1f %%) · un dégât quelconque ±%dms %d (%.1f %%)",
			nom, ta.n, W/1000, ta.sameNear, lot1Pct(ta.sameNear, ta.n),
			OFF/1_000_000, ta.shiftNear, lot1Pct(ta.shiftNear, ta.n),
			W/1000, ta.anyNear, lot1Pct(ta.anyNear, ta.n))
	}
	logGrp("MODAUX", modalT)
	logGrp("NON-MODAUX", nonModalT)

	// Sweep de fenêtre sur le lien même-tireur des tirs MODAUX (contexte).
	for _, w := range []uint64{60_000, 130_000, 250_000, 500_000} {
		hit := 0
		for _, f := range fires {
			if f.modal && f.has && lot1mtNear(hitByResp[f.att], f.ts, w) {
				hit++
			}
		}
		t.Logf("  sweep W=%3dms : tir modal -> dégât même tireur %d/%d (%.1f %%)",
			w/1000, hit, modalT.n, lot1Pct(hit, modalT.n))
	}

	// MESURE INVERSE : un coup au but est-il précédé/accompagné d'un tir MODAL du même tireur ?
	revN, revSame, revShift := 0, 0, 0
	for _, h := range hits {
		revN++
		if lot1mtNear(modalFireByAtt[h.resp], h.ts, W) {
			revSame++
		}
		if lot1mtNear(modalFireByAtt[h.resp], h.ts+OFF, W) {
			revShift++
		}
	}
	t.Logf("INVERSE : coup au but avec un tir MODAL du même tireur ±%dms : %d/%d (%.1f %%) · témoin +%ds %d (%.1f %%)",
		W/1000, revSame, revN, lot1Pct(revSame, revN), OFF/1_000_000, revShift, lot1Pct(revShift, revN))

	// DISCRIMINANT = LE RATIO AU TÉMOIN DÉCALÉ, pas le taux absolu.
	//
	// Le taux ABSOLU de coïncidence tir-modal -> dégât même-tireur est PLAFONNÉ par la densité
	// des damage_aftermath dans le flux delta (ex. 01e1f945 : 30 dégâts pour 491 tirs modaux —
	// au plus ~6 % des tirs POURRAIENT s'apparier). Un seuil absolu testerait donc la densité
	// d'échantillonnage, pas l'hypothèse. Le test SOUND est : la coïncidence même-tireur
	// dépasse-t-elle la coïncidence FORTUITE (même tireur, fenêtre décalée de OFF) ? Un vrai
	// lien causal la dépasse largement ; « modal = raté » (le tir ne touche pas -> aucun dégât)
	// la mettrait AU NIVEAU du témoin. On mesure les deux sens (avant : le tir modal a-t-il un
	// dégât ; arrière : le dégât vient-il d'un tir modal), l'arrière étant le plus fiable quand
	// les dégâts sont rares. Plancher de 1 % au dénominateur pour ne pas diviser par ~0.
	same := lot1Pct(modalT.sameNear, modalT.n)
	shift := lot1Pct(modalT.shiftNear, modalT.n)
	revSameR := lot1Pct(revSame, revN)
	revShiftR := lot1Pct(revShift, revN)
	floor := func(v float64) float64 {
		if v < 1 {
			return 1
		}
		return v
	}
	fwdRatio := same / floor(shift)
	invRatio := revSameR / floor(revShiftR)
	nonModalPct := lot1Pct(nNonModal, nFires)
	// « modal ≠ raté » est SOUTENU si la coïncidence même-tireur bat le témoin décalé dans les
	// DEUX sens (avant >= 1,5x, arrière >= 2x) avec des effectifs suffisants.
	refute := modalT.n >= 20 && revN >= 10 && fwdRatio >= 1.5 && invRatio >= 2
	t.Logf("DISCRIMINANT même-tireur vs témoin décalé : AVANT %.1f %% / %.1f %% = %.1fx · ARRIÈRE %.1f %% / %.1f %% = %.1fx",
		same, shift, fwdRatio, revSameR, revShiftR, invRatio)
	t.Logf("CONTEXTE (bornes, non discriminantes) : tirs non-modaux %.1f %% · coups au but %d vs non-modaux %d · dégât modal même-tireur ±500ms borne haute observée",
		nonModalPct, nHits, nNonModal)
	if refute {
		t.Logf("VERDICT Q2 : « MODAL ≠ RATÉ » — un tir modal PEUT toucher, et son coup au but est rangé dans un damage_aftermath SÉPARÉ (co-incidence même-tireur %.1fx / %.1fx le témoin) — %s",
			fwdRatio, invRatio, lot1Verdict(true))
	} else {
		t.Logf("VERDICT Q2 : discriminant insuffisant sur ce film (avant %.1fx, arrière %.1fx, n=%d/%d) — lire les taux ci-dessus",
			fwdRatio, invRatio, modalT.n, revN)
	}
}
