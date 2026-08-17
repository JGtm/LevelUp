package filmdec

// equipment_life_end_test.go — LA FIN DE VIE d'un objet d'équipement (ti=37) : ce que le film
// porte vraiment, mesuré au lieu d'être hérité.
//
// LA QUESTION, et elle est falsifiable. `splitLives` clôt une vie au premier record portant le
// composant i18 — la règle du PROJECTILE (ti=41), dont i18 s'appelle `projectile-at-rest-state`
// dans le registre. `ScanFilmWorldObjects` sert les DEUX archétypes avec le même découpage, si
// bien que l'équipement hérite d'une règle qui ne parle pas de lui. Cet instrument mesure :
//
//  1. le NOM du composant i18 dans le registre de ti=37 (et de ti=41, témoin) ;
//  2. la durée des vies confirmées, par identifiant `eqip`, sous DEUX règles — (a) coupure au
//     premier i18, celle de production ; (b) coupure au seul trou de 250 ms ;
//  3. le témoin OFFICIEL : le capteur de menaces dure 15 s (« Sensor Duration: 6.5 -> 15
//     secondes », Halo Waypoint, Sandbox Overview Season 4), le mur une dizaine de secondes ;
//  4. la SÉLECTIVITÉ d'un balayage du record DEL (`recDel`) : une fin explicite vaudrait mieux
//     qu'une dernière observation, encore faut-il qu'elle soit isolable.
//
// LECTURE SEULE, gardé par EQUIP_CREATION_FILM. UN SEUL décodage filmdec par process.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 EQUIP_CREATION_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestEquipmentLifeEnd$' -timeout 60m -v

import (
	"os"
	"sort"
	"testing"
)

func TestEquipmentLifeEnd(t *testing.T) {
	dir := os.Getenv(equipCreationFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", equipCreationFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("découpage i0 illisible dans %s : %v", dir, err)
	}
	prev := WorldObjectPrecision
	t.Cleanup(func() { WorldObjectPrecision = prev })
	SetWorldObjectPrecisionFromLayout(lay)
	prevW := CurrentMPPWidths()
	t.Cleanup(func() { SetMPPWidths(prevW) })

	t.Logf("FILM %s", dir)
	lifeEndRegistry(t, dir)

	n := CountFilmChunks(dir)
	band := worldObjectSlotBand(dir, n, EquipmentTypeIndex)
	if len(band) == 0 {
		t.Fatalf("aucun slot ti=%d", EquipmentTypeIndex)
	}
	raw := lifeEndRawSamples(dir, n, band)
	tracks, err := ScanFilmWorldObjects(dir, &equipCreationUnitRange, EquipmentTypeIndex)
	if err != nil {
		t.Fatalf("trajectoires ti=%d illisibles : %v", EquipmentTypeIndex, err)
	}
	spans := EquipmentLifeSpans(tracks)
	cal, ok := CalibrateMPPWidths(dir, &equipCreationUnitRange, band, spans)
	t.Logf("   LARGEUR : %s", cal)
	if !ok {
		t.Logf("   VERDICT : calibration non tranchée — aucune pose, rien à mesurer")
		return
	}
	SetMPPWidths(cal.Widths)
	confirmed := lifeEndDurations(t, dir, band, spans, raw)
	lifeEndKeyframeCensus(t, dir, n, confirmed)
	lifeEndTailProbe(t, dir, n, band, raw, confirmed)
	lifeEndDelSelectivity(t, dir, n, band, len(tracks))
}

// lifeEndRegistry publie le nom du composant i18 dans les deux archétypes. C'est LA PIÈCE : si
// ti=37 i18 ne s'appelle pas `projectile-at-rest-state`, la coupure de production repose sur un
// index emprunté à un autre archétype.
func lifeEndRegistry(t *testing.T, dir string) {
	t.Helper()
	rawReg, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(rawReg)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	for _, ti := range []int{EquipmentTypeIndex, ProjectileTypeIndex} {
		arch, okA := reg.Archetype(ti)
		if !okA {
			t.Logf("   REGISTRE ti=%d : archétype absent", ti)
			continue
		}
		t.Logf("   REGISTRE ti=%d : i%d = %q (sur %d composants)",
			ti, projectileRestComponent, arch.component(projectileRestComponent),
			len(arch.Components))
	}
}

// lifeEndRawSamples rejoue le balayage de ScanFilmWorldObjects SANS découper en vies : c'est la
// matière commune aux deux règles comparées.
func lifeEndRawSamples(
	dir string, n int, band map[uint32]bool,
) map[EquipmentLifeKey][]ProjectileSample {
	out := map[EquipmentLifeKey][]ProjectileSample{}
	for c := 1; c <= n; c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(chunk) {
			if p.Type != PacketTypeDelta {
				continue
			}
			pay := p.Payload(chunk)
			for _, s := range scanProjectileRecords(pay, band, &equipCreationUnitRange) {
				s.TimestampUS, s.Chunk = p.TimestampUS, c
				k := EquipmentLifeKey{s.slot, s.gen}
				out[k] = append(out[k], s.ProjectileSample)
			}
		}
	}
	for k := range out {
		pts := out[k]
		sort.Slice(pts, func(i, j int) bool { return lessSample(pts[i], pts[j]) })
		out[k] = pts
	}
	return out
}

// lifeEndGapOnly découpe au SEUL trou de 250 ms — la règle (b), sans la coupure i18.
func lifeEndGapOnly(pts []ProjectileSample) [][]ProjectileSample {
	var out [][]ProjectileSample
	start := 0
	flush := func(end int) {
		if end-start >= 3 {
			out = append(out, pts[start:end])
		}
	}
	for i := 1; i < len(pts); i++ {
		if pts[i].TimestampUS-pts[i-1].TimestampUS > projectileGapUS {
			flush(i)
			start = i
		}
	}
	flush(len(pts))
	return out
}

// lifeEndConfirmed est une pose confirmée par l'oracle, réduite à ce que la sonde de queue
// doit connaître : sa clé de vie, son instant de pose, la fin de son flux de POSITION.
type lifeEndConfirmed struct {
	key      EquipmentLifeKey
	gid      uint32
	t0, t1US uint64
}

// lifeEndDurations compare les deux règles sur la cohorte CONFIRMÉE (les poses que la
// production publie), identifiant par identifiant.
func lifeEndDurations(
	t *testing.T, dir string, band map[uint32]bool,
	spans map[EquipmentLifeKey][]EquipmentLifeSpan, raw map[EquipmentLifeKey][]ProjectileSample,
) []lifeEndConfirmed {
	t.Helper()
	var out []lifeEndConfirmed
	cre, _, err := ScanFilmEquipmentCreationsForBand(dir, &equipCreationUnitRange, band)
	if err != nil {
		t.Fatalf("balayage des créations impossible : %v", err)
	}
	eps := EquipmentPosEps(&equipCreationUnitRange)
	type pair struct{ a, b float64 } // durées en secondes, règle (a) puis (b)
	byID := map[uint32][]pair{}
	seen := map[[3]uint64]bool{}
	for _, c := range cre {
		k := EquipmentLifeKey{c.Slot, c.Gen}
		life, hit := MatchEquipmentLife(spans[k], [3]float32{c.X, c.Y, c.Z}, eps, c.TimestampUS)
		if !hit {
			continue
		}
		id := [3]uint64{uint64(c.Slot), uint64(c.Gen), life.T0US}
		if seen[id] {
			continue
		}
		seen[id] = true
		gid := uint32(c.MPPVal[MPPWord32])
		durA := float64(life.T1US-life.T0US) / 1e6
		durB := durA
		for _, seg := range lifeEndGapOnly(raw[k]) {
			if seg[0].TimestampUS <= life.T0US && seg[len(seg)-1].TimestampUS >= life.T0US {
				durB = float64(seg[len(seg)-1].TimestampUS-life.T0US) / 1e6
				break
			}
		}
		byID[gid] = append(byID[gid], pair{durA, durB})
		out = append(out, lifeEndConfirmed{key: k, gid: gid, t0: life.T0US, t1US: life.T1US})
	}
	ids := make([]uint32, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return len(byID[ids[i]]) > len(byID[ids[j]]) })
	t.Logf("   DURÉES DE VIE par identifiant `eqip` — (a) règle de production (coupure i18)" +
		" contre (b) coupure au seul trou de 250 ms. Secondes.")
	for _, id := range ids {
		ps := byID[id]
		a := make([]float64, len(ps))
		b := make([]float64, len(ps))
		for i, p := range ps {
			a[i], b[i] = p.a, p.b
		}
		t.Logf("      0x%08x  n=%3d   (a) med %5.2f  p90 %6.2f  max %6.2f   |"+
			"   (b) med %5.2f  p90 %6.2f  max %6.2f",
			id, len(ps), lifeEndQuantile(a, 0.5), lifeEndQuantile(a, 0.9), lifeEndQuantile(a, 1),
			lifeEndQuantile(b, 0.5), lifeEndQuantile(b, 0.9), lifeEndQuantile(b, 1))
	}
	return out
}

// lifeEndKeyframeCensus interroge le RECENSEMENT des keyframes : la seule source du film qui
// dise « cette entité existe » sans rien demander à son mouvement. Le walker est durci et validé
// (249/250 entités, cf. keyframe_world.go), donc une présence y est une présence.
//
// SA RÉSOLUTION EST SA LIMITE, et c'est le point de la mesure : les keyframes sont espacés d'une
// vingtaine de secondes. Le recensement peut donc PROUVER qu'un objet survit à son flux de
// position ; il ne peut pas DATER sa disparition à la seconde.
func lifeEndKeyframeCensus(t *testing.T, dir string, n int, confirmed []lifeEndConfirmed) {
	t.Helper()
	type kf struct {
		ts   uint64
		live map[EquipmentLifeKey]bool
	}
	var kfs []kf
	for c := 1; c <= n; c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(chunk) {
			if p.Type != PacketTypeKeyframe {
				continue
			}
			live := map[EquipmentLifeKey]bool{}
			for _, r := range WalkKeyframeWorld(p.Payload(chunk)) {
				if r.TI == EquipmentTypeIndex {
					live[EquipmentLifeKey{uint32(r.Slot), uint32(r.Gen)}] = true
				}
			}
			kfs = append(kfs, kf{ts: p.TimestampUS, live: live})
		}
	}
	sort.Slice(kfs, func(i, j int) bool { return kfs[i].ts < kfs[j].ts })
	var gaps []float64
	for i := 1; i < len(kfs); i++ {
		gaps = append(gaps, float64(kfs[i].ts-kfs[i-1].ts)/1e6)
	}
	survivors, seen := 0, 0
	var lastSeen []float64
	for _, c := range confirmed {
		last := uint64(0)
		after := false
		for _, k := range kfs {
			if !k.live[c.key] {
				continue
			}
			if k.ts >= c.t0 {
				last = k.ts
			}
			if k.ts > c.t1US+1_000_000 {
				after = true
			}
		}
		if last > 0 {
			seen++
			lastSeen = append(lastSeen, float64(last-c.t0)/1e6)
		}
		if after {
			survivors++
		}
	}
	t.Logf("   RECENSEMENT keyframes : %d keyframes, espacement med %.1f s ·"+
		" %d poses sur %d recensées après leur pose, dont %d ENCORE recensées plus d'une"+
		" seconde après la fin de leur flux de position", len(kfs),
		lifeEndQuantile(gaps, 0.5), seen, len(confirmed), survivors)
	t.Logf("      dernier recensement - pose : med %.1f s · p90 %.1f s · max %.1f s"+
		" (borne INFÉRIEURE de la durée de vie, à la résolution des keyframes)",
		lifeEndQuantile(lastSeen, 0.5), lifeEndQuantile(lastSeen, 0.9),
		lifeEndQuantile(lastSeen, 1))
	byID := map[uint32][]float64{}
	for _, c := range confirmed {
		last := uint64(0)
		for _, k := range kfs {
			if k.live[c.key] && k.ts >= c.t0 {
				last = k.ts
			}
		}
		if last > 0 {
			byID[c.gid] = append(byID[c.gid], float64(last-c.t0)/1e6)
		}
	}
	ids := make([]uint32, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return len(byID[ids[i]]) > len(byID[ids[j]]) })
	for _, id := range ids {
		v := byID[id]
		t.Logf("      0x%08x  recensées %3d fois — dernier recensement - pose :"+
			" med %6.1f s · p90 %6.1f s · max %6.1f s", id, len(v),
			lifeEndQuantile(v, 0.5), lifeEndQuantile(v, 0.9), lifeEndQuantile(v, 1))
	}
}

// lifeEndTailProbe cherche la QUEUE de la vie : des records de la même paire (slot, génération)
// APRÈS que le flux de position se soit tu. L'objet posé cesse de bouger, donc cesse d'émettre
// `object-position-component` — mais il reste répliqué, et ses autres composants (déployé,
// activé, énergie, `item-at-rest-component`) pourraient continuer de l'être.
//
// LE TÉMOIN EST INDISPENSABLE et il est intégré : le balayage sans contrainte de position est
// beaucoup moins sélectif (aucune position à déquantifier), donc une queue « trouvée » ne vaut
// que comparée au BRUIT. Le témoin est l'ensemble des clés dont le slot appartient à la bande
// mais qui ne portent AUCUNE vie décodée du film : ce qu'on y compte est du faux par
// construction, sur la même durée et le même balayage.
func lifeEndTailProbe(
	t *testing.T, dir string, n int, band map[uint32]bool,
	raw map[EquipmentLifeKey][]ProjectileSample, confirmed []lifeEndConfirmed,
) {
	t.Helper()
	if len(confirmed) == 0 {
		t.Logf("   QUEUE : aucune pose confirmée — rien à sonder")
		return
	}
	all := lifeEndAllRecords(dir, n, band)

	// Fenêtre de sondage : 30 s après la dernière position. Au-delà, un match ordinaire a
	// recyclé le slot.
	const windowUS = 30_000_000
	var tails []float64
	hits := 0
	for _, c := range confirmed {
		last := uint64(0)
		for _, ts := range all[c.key] {
			if ts > c.t1US && ts <= c.t1US+windowUS && ts > last {
				last = ts
			}
		}
		if last == 0 {
			continue
		}
		hits++
		tails = append(tails, float64(last-c.t0)/1e6)
	}
	// TÉMOIN : mêmes fenêtres, sur des clés que le film ne porte pas.
	ctrl, ctrlHits := 0, 0
	for slot := range band {
		for gen := uint32(0); gen < 4; gen++ {
			k := EquipmentLifeKey{slot, gen}
			if len(raw[k]) > 0 {
				continue
			}
			ctrl++
			if len(all[k]) > 0 {
				ctrlHits++
			}
		}
	}
	t.Logf("   QUEUE (records de la même clé sans contrainte de position, 30 s après la"+
		" dernière position) : %d poses sur %d en portent — durée totale med %.2f s,"+
		" p90 %.2f s, max %.2f s",
		hits, len(confirmed), lifeEndQuantile(tails, 0.5), lifeEndQuantile(tails, 0.9),
		lifeEndQuantile(tails, 1))
	t.Logf("      TÉMOIN : %d clés de la bande sans aucune vie décodée, dont %d (%.1f %%)"+
		" portent tout de même des records — c'est le plancher de bruit du balayage",
		ctrl, ctrlHits, 100*float64(ctrlHits)/float64(max(ctrl, 1)))
}

// lifeEndAllRecords collecte les instants de TOUS les records d'objet du monde de la bande,
// SANS exiger la position (i0) : c'est le balayage le plus large possible sur cet archétype.
func lifeEndAllRecords(dir string, n int, band map[uint32]bool) map[EquipmentLifeKey][]uint64 {
	out := map[EquipmentLifeKey][]uint64{}
	posBits := projPosBits()
	for c := 1; c <= n; c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(chunk) {
			if p.Type != PacketTypeDelta {
				continue
			}
			pay := p.Payload(chunk)
			limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits + posBits)
			for bit := 0; bit <= limit; bit++ {
				rec, okR := matchWorldObjectRecord(pay, bit, band)
				if !okR || len(rec.Idx) < 2 {
					continue // >= 2 composants : un masque à un seul index n'est pas sélectif
				}
				k := EquipmentLifeKey{rec.Slot, rec.Gen}
				out[k] = append(out[k], p.TimestampUS)
				bit = rec.After - 1
			}
		}
	}
	return out
}

func lifeEndQuantile(v []float64, q float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return s[int(q*float64(len(s)-1))]
}

// lifeEndDelSelectivity mesure ce qu'un balayage bit à bit du record DEL rendrait. Un record DEL
// est `0` + `R(2)=2` + slot(13) + génération(2) + R(32) : trois bits contraints et un slot dans
// la bande, aucune contrainte sur les 32 bits de corps. La question n'est pas « existe-t-il ? »
// mais « combien de faux ? » — si la densité de candidats dépasse de plusieurs ordres le nombre
// de vies, aucune fin explicite n'en sort.
func lifeEndDelSelectivity(t *testing.T, dir string, n int, band map[uint32]bool, lives int) {
	t.Helper()
	cands, payloads := 0, 0
	limit := n
	if limit > 4 {
		limit = 4 // quatre chunks suffisent à établir un ordre de grandeur
	}
	for c := 1; c <= limit; c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(chunk) {
			if p.Type != PacketTypeDelta {
				continue
			}
			payloads++
			cands += lifeEndCountDel(p.Payload(chunk), band)
		}
	}
	t.Logf("   DEL : %d candidats sur %d payloads (%d chunks lus) pour %d vies au film —"+
		" un candidat n'est une suppression que si quelque chose le confirme",
		cands, payloads, limit, lives)
}

func lifeEndCountDel(pay []byte, band map[uint32]bool) int {
	cnt := 0
	last := len(pay)*8 - (3 + 13 + 2 + 32)
	for p := 0; p <= last; p++ {
		if PeekBits(pay, p, 1) != 0 || PeekBits(pay, p+1, 2) != uint64(recDel) {
			continue
		}
		if band[uint32(PeekBits(pay, p+3, 13))] {
			cnt++
		}
	}
	return cnt
}
