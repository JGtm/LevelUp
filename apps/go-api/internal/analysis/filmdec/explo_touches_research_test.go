package filmdec

// explo_touches_research_test.go — LOT 1 : les TOUCHES EXPLOSIVES (roquette, empaleur, ravageur,
// fusil a choc, mangler, stalker, bulldog, tige...) sont-elles dans le film, et attribuables a
// leur TIREUR + ARME ?
//
// RAISONNEMENT (utilisateur) : l'API du match donne une accuracy par joueur qui compte les tirs
// AU BUT sur TOUTES les armes, explosifs compris. Donc le moteur COMPTE les touches explosives,
// donc le film les CONTIENT. Le verdict precedent (« armes lourdes = 0 % de touches en type 0 »)
// mesurait un APPARIEMENT qui exige que le RESPONSABLE du degat (damage_aftermath ref1, dom1)
// SOIT le joueur, MEME attaquant que le tir. Pour un explosif, le degat est inflige par le
// PROJECTILE, pas le joueur : notre appariement le RATE alors que le degat EXISTE.
//
// CE QUE MESURE CET INSTRUMENT, sur les damage_aftermath (0xC0 type 0) :
//   M1 — CENSUS de la responsabilite ref1 : absente / resolue en bipede (tir direct, appariable)
//        / presente-mais-NON-bipede (candidate « projectile »). Magnitude, victime resolue,
//        diversite de tags source par classe.
//   M2 — CLUSTER de tags source : y a-t-il des tags source qui n'apparaissent QUE dans la classe
//        non-resolue (population de degats distincte = signature explosive) ?
//   M3 — COINCIDENCE avec les tirs LOURDS : une touche non-resolue tombe-t-elle dans la fenetre
//        de vol [ts-Wide, ts] APRES un tir lourd, au-dessus d'un temoin decale ? (Confondu par la
//        densite des tirs lourds sur les modes Fiesta : lire avec M5, densite-independant.)
//   M4 — ATTRIBUTION chiffree : pour une touche non-resolue coincidant, combien de TIREURS lourds
//        distincts dans la fenetre de vol ? Un seul = attribution non ambigue (tireur+arme du tir).
//   M5 — MAGNITUDE (discriminant densite-independant) : la classe non-bipede frappe-t-elle ~2x
//        plus fort que le tir direct (signature d'un explosif) ?
//   M6 — CHRONO : une ref1 non-bipede en fin de chunk est-elle un TIREUR mort depuis (confond de
//        lot1_monde_chrono) ou une entite qui n'a JAMAIS ete un joueur (projectile) ?
//
// MECANISME DEJA CONNU POUR LES KILLS (rappel, non re-mesure ici) : killsource attribue deja les
// morts explosives au joueur+arme via le dead-state i11 de la VICTIME (tag source +0x00, TUEUR
// +0x08) — 97,6 %, cf. project_killweapon_deadstate_solved. Le present instrument cherche le
// pendant NON FATAL.
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule, borne a
// deltaWitnessChunks (12). Lancer une fois par film (000d5950, 01e1f945, 00502e52).
// Collecteurs, types et utilitaires : explo_touches_helpers_test.go.

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

const (
	exploFlightW = uint64(2_000_000) // 2 s : fenetre de vol tir lourd -> impact
	exploOffWit  = uint64(3_000_000) // 3 s : decalage du temoin de densite
	exploBase    = lot1chReferenceBase
)

// exploM1Census : census de la responsabilite ref1 + profils de classe.
func exploM1Census(t *testing.T, dmg []exploDmg) {
	t.Helper()
	var nAbs, nBip, nNon int
	var magAbs, magBip, magNon float64
	vicAbs, vicBip, vicNon := 0, 0, 0
	srcAbs, srcBip, srcNon := map[uint64]int{}, map[uint64]int{}, map[uint64]int{}
	tiHist := map[int]int{} // archetype de (base+ref1) pour la classe non-bipede (ti>=0)
	nonBound := 0           // non-bipede qui ne lie a AUCUN archetype
	for _, e := range dmg {
		switch exploClassify(e) {
		case exploRespAbsent:
			nAbs++
			magAbs += e.mag
			if e.vicBiped {
				vicAbs++
			}
			if e.hasSrc {
				srcAbs[e.src]++
			}
		case exploRespBiped:
			nBip++
			magBip += e.mag
			if e.vicBiped {
				vicBip++
			}
			if e.hasSrc {
				srcBip[e.src]++
			}
		case exploRespNonBiped:
			nNon++
			magNon += e.mag
			if e.vicBiped {
				vicNon++
			}
			if e.hasSrc {
				srcNon[e.src]++
			}
			if e.respTI >= 0 {
				tiHist[e.respTI]++
			} else {
				nonBound++
			}
		}
	}
	tot := len(dmg)
	t.Logf("M1 CENSUS responsabilite (ref1) sur %d damage_aftermath :", tot)
	t.Logf("   ref1 ABSENTE          : %d (%.1f %%) · mag moy %.2f · victime bipede %d/%d (%.1f %%) · %d tags source",
		nAbs, lot1Pct(nAbs, tot), exploAvg(magAbs, nAbs), vicAbs, nAbs, lot1Pct(vicAbs, nAbs), len(srcAbs))
	t.Logf("   ref1 -> BIPEDE (direct): %d (%.1f %%) · mag moy %.2f · victime bipede %d/%d (%.1f %%) · %d tags source",
		nBip, lot1Pct(nBip, tot), exploAvg(magBip, nBip), vicBip, nBip, lot1Pct(vicBip, nBip), len(srcBip))
	t.Logf("   ref1 NON-BIPEDE       : %d (%.1f %%) · mag moy %.2f · victime bipede %d/%d (%.1f %%) · %d tags source",
		nNon, lot1Pct(nNon, tot), exploAvg(magNon, nNon), vicNon, nNon, lot1Pct(vicNon, nNon), len(srcNon))
	t.Logf("   NON-BIPEDE : lie a un archetype %d (top ti %s) · lie a RIEN %d",
		nNon-nonBound, exploTopTI(tiHist, 6), nonBound)
	t.Logf("   LECTURE : une classe non-bipede a victime RESOLUE et magnitude elevee est le reservoir")
	t.Logf("   candidat des touches dont le responsable est un projectile (confirme M5/M6).")
}

// exploM2Cluster : tags source exclusifs a la classe non-resolue (signature de population).
func exploM2Cluster(t *testing.T, dmg []exploDmg) {
	t.Helper()
	inBip, inNon := map[uint64]int{}, map[uint64]int{}
	for _, e := range dmg {
		if !e.hasSrc {
			continue
		}
		switch exploClassify(e) {
		case exploRespBiped:
			inBip[e.src]++
		case exploRespNonBiped:
			inNon[e.src]++
		}
	}
	exclusiveNon, sharedTags, exclusiveNonEvts := 0, 0, 0
	for s, c := range inNon {
		if inBip[s] == 0 {
			exclusiveNon++
			exclusiveNonEvts += c
		} else {
			sharedTags++
		}
	}
	t.Logf("M2 CLUSTER tags source : classe bipede %d tags · classe non-bipede %d tags",
		len(inBip), len(inNon))
	t.Logf("   tags EXCLUSIFS a la classe non-bipede : %d tags / %d evenements · tags PARTAGES : %d",
		exclusiveNon, exclusiveNonEvts, sharedTags)
	t.Logf("   top tags non-bipede : %s", lot1TopU64(inNon, 8))
	t.Logf("   LECTURE : peu de tags exclusifs => le tag SOURCE (effet jpt!) ne suffit pas seul a")
	t.Logf("   isoler l'explosif ; le discriminant fiable est la magnitude + la non-resolution (M5/M6).")
}

// exploM3Coincidence : les touches non-resolues coincident-elles avec les tirs LOURDS ?
func exploM3Coincidence(t *testing.T, shots []exploShot, dmg []exploDmg) {
	t.Helper()
	var heavyTs, allTs []uint64
	for _, s := range shots {
		if !s.has {
			continue
		}
		allTs = append(allTs, s.ts)
		if s.heavy {
			heavyTs = append(heavyTs, s.ts)
		}
	}
	sort.Slice(heavyTs, func(a, b int) bool { return heavyTs[a] < heavyTs[b] })
	sort.Slice(allTs, func(a, b int) bool { return allTs[a] < allTs[b] })
	t.Logf("M3 COINCIDENCE tir lourd -> touche (fenetre vol %ds · temoin +%ds) : %d tirs lourds, %d tirs total",
		exploFlightW/1_000_000, exploOffWit/1_000_000, len(heavyTs), len(allTs))
	if len(heavyTs) == 0 {
		t.Logf("   AUCUN tir lourd sur ce film : coincidence non mesurable (attendu sans explosif)")
		return
	}
	type cnt struct{ n, near, wit, nearAll int }
	var non, bip cnt
	tally := func(e exploDmg, c *cnt) {
		c.n++
		if exploPrecede(heavyTs, e.ts, exploFlightW) {
			c.near++
		}
		if exploPrecede(heavyTs, e.ts+exploOffWit, exploFlightW) {
			c.wit++
		}
		if exploPrecede(allTs, e.ts, exploFlightW) {
			c.nearAll++
		}
	}
	for _, e := range dmg {
		switch exploClassify(e) {
		case exploRespNonBiped:
			tally(e, &non)
		case exploRespBiped:
			tally(e, &bip)
		}
	}
	t.Logf("   NON-BIPEDE : %d/%d (%.1f %%) precedes d'un tir lourd · temoin %.1f %% · (tout tir %.1f %%)",
		non.near, non.n, lot1Pct(non.near, non.n), lot1Pct(non.wit, non.n), lot1Pct(non.nearAll, non.n))
	t.Logf("   BIPEDE (contr): %d/%d (%.1f %%) precedes d'un tir lourd · temoin %.1f %% · (tout tir %.1f %%)",
		bip.near, bip.n, lot1Pct(bip.near, bip.n), lot1Pct(bip.wit, bip.n), lot1Pct(bip.nearAll, bip.n))
	real := non.n >= 10 && lot1Pct(non.near, non.n) >= 1.5*maxF(1, lot1Pct(non.wit, non.n)) &&
		lot1Pct(non.near, non.n) > lot1Pct(bip.near, bip.n)
	t.Logf("   VERDICT (non-bipede enrichi en tirs lourds, >=1.5x temoin ET > classe bipede) : %s",
		lot1Verdict(real))
}

// exploM4Attribution : une touche non-resolue coincidante a-t-elle UN seul tireur lourd candidat ?
func exploM4Attribution(t *testing.T, shots []exploShot, dmg []exploDmg) {
	t.Helper()
	type hs struct {
		ts       uint64
		att, wid uint64
	}
	var heavy []hs
	for _, s := range shots {
		if s.has && s.heavy {
			heavy = append(heavy, hs{s.ts, s.att, s.wid})
		}
	}
	sort.Slice(heavy, func(a, b int) bool { return heavy[a].ts < heavy[b].ts })
	if len(heavy) == 0 {
		t.Logf("M4 ATTRIBUTION : aucun tir lourd — attribution non evaluable sur ce film")
		return
	}
	tsList := make([]uint64, len(heavy))
	for i, h := range heavy {
		tsList[i] = h.ts
	}
	coinc, uniqueAtt, uniqueWid := 0, 0, 0
	for _, e := range dmg {
		if exploClassify(e) != exploRespNonBiped {
			continue
		}
		lo := uint64(0)
		if e.ts > exploFlightW {
			lo = e.ts - exploFlightW
		}
		i := sort.Search(len(tsList), func(i int) bool { return tsList[i] >= lo })
		atts, wids := map[uint64]bool{}, map[uint64]bool{}
		for ; i < len(heavy) && heavy[i].ts <= e.ts; i++ {
			atts[heavy[i].att] = true
			wids[heavy[i].wid] = true
		}
		if len(atts) == 0 {
			continue
		}
		coinc++
		if len(atts) == 1 {
			uniqueAtt++
		}
		if len(wids) == 1 {
			uniqueWid++
		}
	}
	t.Logf("M4 ATTRIBUTION (touche non-bipede coincidant un tir lourd dans la fenetre de vol) :")
	t.Logf("   %d touches candidates · tireur UNIQUE %d (%.1f %%) · arme UNIQUE %d (%.1f %%)",
		coinc, uniqueAtt, lot1Pct(uniqueAtt, coinc), uniqueWid, lot1Pct(uniqueWid, coinc))
	t.Logf("   LECTURE : tireur unique = attribution non ambigue (tireur+arme du seul tir lourd dans")
	t.Logf("   la fenetre) ; ambigu = plusieurs tireurs lourds simultanes (gros teamfight).")
}

// exploM5Magnitude : la magnitude est le discriminant DENSITE-INDEPENDANT. Un explosif frappe
// fort ; un tir direct chippe. On mesure la part de HAUTE magnitude (>= seuil) par classe.
func exploM5Magnitude(t *testing.T, dmg []exploDmg) {
	t.Helper()
	const hi = 2.5 // au-dela : coup lourd (une barre de bouclier ~ plusieurs balles)
	type mstat struct {
		n, high int
		sum     float64
	}
	var bip, non, abs mstat
	add := func(m *mstat, mag float64) {
		m.n++
		m.sum += mag
		if mag >= hi {
			m.high++
		}
	}
	for _, e := range dmg {
		switch exploClassify(e) {
		case exploRespBiped:
			add(&bip, e.mag)
		case exploRespNonBiped:
			add(&non, e.mag)
		case exploRespAbsent:
			add(&abs, e.mag)
		}
	}
	t.Logf("M5 MAGNITUDE (discriminant densite-independant · haute = mag >= %.1f) :", hi)
	t.Logf("   ref1 -> BIPEDE (direct): moy %.2f · haute %d/%d (%.1f %%)",
		exploAvg(bip.sum, bip.n), bip.high, bip.n, lot1Pct(bip.high, bip.n))
	t.Logf("   ref1 NON-BIPEDE       : moy %.2f · haute %d/%d (%.1f %%)",
		exploAvg(non.sum, non.n), non.high, non.n, lot1Pct(non.high, non.n))
	t.Logf("   ref1 ABSENTE          : moy %.2f · haute %d/%d (%.1f %%)",
		exploAvg(abs.sum, abs.n), abs.high, abs.n, lot1Pct(abs.high, abs.n))
	enriched := non.n >= 8 && bip.n >= 8 && lot1Pct(non.high, non.n) >= 2*maxF(1, lot1Pct(bip.high, bip.n))
	t.Logf("   VERDICT (non-bipede enrichi >= 2x en coups lourds vs direct) : %s", lot1Verdict(enriched))
}

// exploM6Chrono SEPARE le confond de lot1_monde_chrono : une ref1 non-bipede EN FIN DE CHUNK
// peut etre soit un TIREUR mort en cours de chunk (delie du monde -> mauvaise resolution d'un
// tir DIRECT), soit une entite qui n'a JAMAIS ete un joueur (projectile). On resout ref1 aux
// DEUX instants (chronologique = a l'instant de l'evenement ; fin de chunk) et on ventile les
// non-bipedes-fin selon leur resolution CHRONO. Un projectile reste non-bipede aux deux instants.
func exploM6Chrono(t *testing.T, dir string, reg *Registry, n int) {
	t.Helper()
	cfg := DefaultFrameConfig()
	var confN, projN int
	var confMag, projMag float64
	type ev struct {
		idx1        int
		mag         float64
		chronoBiped bool
	}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		w := NewWorld(reg)
		var events []ev
		for _, pk := range WalkPackets(data) {
			pay := pk.Payload(data)
			switch {
			case pk.Type == PacketTypeKeyframe:
				for _, r := range WalkKeyframeWorld(pay) {
					w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
				}
			case pk.Type == PacketTypeDelta && pk.Size >= 1 && pay[0]&0x40 == 0:
				br := NewBitReader(pay)
				_, _ = DecodeFrameRecords(br, w, cfg)
			case pk.Type == PacketTypeDelta && pk.Size >= 2 && pay[0] == 0xC0:
				br := NewBitReader(pay)
				br.Skip(2)
				if br.ReadBits(7) != 0 {
					continue
				}
				lot1RefDom1(br) // ref0
				i1, ok1 := lot1RefDom1(br)
				lot1RefDom(br, 7)
				r := lot1DecodeDamageAftermath(br)
				if !ok1 {
					continue
				}
				cb, _ := exploResolve(w, exploBase, int(i1))
				events = append(events, ev{idx1: int(i1), mag: r.dmgClear, chronoBiped: cb})
			}
		}
		for _, e := range events {
			if endB, _ := exploResolve(w, exploBase, e.idx1); endB {
				continue // resolu en fin de chunk : deja compte comme direct
			}
			if e.chronoBiped {
				confN++
				confMag += e.mag
			} else {
				projN++
				projMag += e.mag
			}
		}
	}
	tot := confN + projN
	t.Logf("M6 CHRONO (ref1 non-bipede en fin de chunk, ventile par resolution a l'instant) :")
	t.Logf("   CONFOND tir direct (chrono BIPEDE, tireur mort depuis) : %d/%d (%.1f %%) · mag moy %.2f",
		confN, tot, lot1Pct(confN, tot), exploAvg(confMag, confN))
	t.Logf("   PROJECTILE authentique (chrono NON-bipede aux deux instants) : %d/%d (%.1f %%) · mag moy %.2f",
		projN, tot, lot1Pct(projN, tot), exploAvg(projMag, projN))
	t.Logf("   LECTURE : le sous-groupe PROJECTILE porte la magnitude explosive ; le CONFOND, une fois")
	t.Logf("   resolu chrono, redevient un degat direct attribuable par l'appariement classique.")
}

// TestExploTouches produit les 6 mesures sur LOT1_TRAME_FILM.
func TestExploTouches(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	t.Logf("== film %s · %d chunks · base bipede %d ==", filepath.Base(dir), n, exploBase)

	shots := exploCollectShots(t, dir, n)
	dmg := exploCollectDamage(t, dir, reg, n)
	var nHeavy int
	for _, s := range shots {
		if s.has && s.heavy {
			nHeavy++
		}
	}
	t.Logf("collecte : %d tirs longs (%d lourds) · %d damage_aftermath", len(shots), nHeavy, len(dmg))

	exploM1Census(t, dmg)
	exploM2Cluster(t, dmg)
	exploM3Coincidence(t, shots, dmg)
	exploM4Attribution(t, shots, dmg)
	exploM5Magnitude(t, dmg)
	exploM6Chrono(t, dir, reg, n)
}
