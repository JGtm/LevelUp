package replay

// zone_gauge_temoin_test.go — LOT C-ter VOLET 3 (CT.3.3) : LE TEMOIN DE LA JAUGE EN DIRECT, relu
// SUR L'ARTEFACT cuit par le CLI (`cmd/replay-build --facts`), jamais sur les compteurs internes.
//
// CE QU'IL MESURE, sur un artefact de schema 17 :
//
//	POIDS         les octets de `zoneStates[].gauge` (serialisation JSON des seules series) contre
//	              la taille du fichier — c'est la part de la jauge en direct, independamment des
//	              autres calques qui ont pu bouger entre deux cuissons.
//	POINTS        par zone et au total ; egalite avec `coverage.zones.gaugePoints`.
//	CADENCE       les ecarts entre points consecutifs d'une meme rampe (v qui ne baisse pas) : la
//	              tenue cote client est d'UNE seconde (ZONE_GAUGE_HOLD_MS) — un ecart intra-rampe
//	              plus long ferait clignoter l'arc, et il faut le voir.
//	AVANT/APRES   « la jauge MONTE avant la bascule du proprietaire » : pour chaque intervalle
//	              dont le proprietaire est un camp et differe du precedent (une capture), existe-t-il
//	              un pas MONTANT de la serie dont l'instant tombe dans [t0 - 5 s ; t0 + 2 s] ?
//	              Seuil ecrit AVANT la mesure : >= 90 % des bascules (Bastion). Publie avec son
//	              temoin (bascules decalees de +20 s) et le NIVEAU DU HASARD (part de l axe de la
//	              zone couverte par les fenetres de ses pas montants), regle du depot pour toute
//	              clause temporelle.
//
// SOUS GARDE D'ENVIRONNEMENT (`ZONE_ARTEFACT` = chemin absolu de l'artefact JSON), un artefact
// par processus :
//
//	$env:CGO_ENABLED=0
//	$env:ZONE_ARTEFACT="C:/Users/Guillaume/Projects/LevelUp-wt-jauge-live/data/cache/replays/halo_infinite/7344d24f.json"
//	go test -count=1 -run TestZoneGaugeTemoin -v ./internal/analysis/replay/

import (
	"encoding/json"
	"os"
	"testing"
)

const (
	gaugeTemoinEnv = "ZONE_ARTEFACT"
	// gaugeTemoinBeforeMS / gaugeTemoinAfterMS : la fenetre autour d'une bascule dans laquelle un
	// pas montant de la jauge est cherche. Le canal de propriete replique JUSTE APRES la capture
	// (fenetre d'appariement zoneWindowMS = 2 s), et une capture de Bastion se joue en quelques
	// secondes : 5 s avant, 2 s apres.
	gaugeTemoinBeforeMS = 5000
	gaugeTemoinAfterMS  = 2000
	// gaugeTemoinShiftMS : le decalage du temoin (les bascules poussees de +20 s).
	gaugeTemoinShiftMS = 20000
	// gaugeTemoinSeuil : la clause, ecrite avant la mesure.
	gaugeTemoinSeuil = 0.90
)

// TestZoneGaugeTemoin relit UN artefact et publie les chiffres de CT.3.3.
func TestZoneGaugeTemoin(t *testing.T) {
	path := os.Getenv(gaugeTemoinEnv)
	if path == "" {
		t.Skipf("temoin de la jauge en direct : poser %s=<artefact.json>", gaugeTemoinEnv)
	}
	blob, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("artefact illisible : %v", err)
	}
	var doc ReplayDocument
	if err := json.Unmarshal(blob, &doc); err != nil {
		t.Fatalf("artefact non decodable : %v", err)
	}
	t.Logf("ARTEFACT %s — schema %d, %d frames de %d ms, %d octets, %d zone(s) publiee(s)",
		doc.MatchID, doc.SchemaVersion, doc.FrameCount, doc.FrameIntervalMS, len(blob), len(doc.ZoneStates))
	if doc.Coverage == nil || doc.Coverage.Zones == nil {
		t.Fatalf("aucune couverture de zone : rien a mesurer")
	}
	cov := doc.Coverage.Zones
	t.Logf("  COUVERTURE : methode %s · catalogue %d · apparies %d · intervalles %d · periodes colline %d"+
		" · gaugePoints %d", cov.Method, cov.Catalog, cov.Paired, cov.Spans, cov.HillPeriods, cov.GaugePoints)
	hold := zoneGaugeGapFrames(doc.FrameIntervalMS)
	gaugeTemoinPoids(t, doc, len(blob), hold)
	gaugeTemoinBascules(t, doc)
}

// gaugeTemoinPoids publie, par zone, les points, les rampes, les octets et les ecarts intra-rampe.
func gaugeTemoinPoids(t *testing.T, doc ReplayDocument, fileBytes, hold int) {
	t.Helper()
	total, totalBytes, trous, visibles, rampes := 0, 0, 0, 0, 0
	for _, st := range doc.ZoneStates {
		blob, err := json.Marshal(st.Gauge)
		if err != nil {
			t.Fatalf("serialisation de la jauge de la zone %d : %v", st.ZoneRef, err)
		}
		r, tr, vis, maxGap := gaugeTemoinRampes(st.Gauge, hold)
		t.Logf("  zone %d : %d intervalle(s), %d point(s) de jauge en %d rampe(s), %d octets, "+
			"ecart intra-rampe max %d frame(s), trous intra-rampe > %d frames : %d (dont %d a arc"+
			" VISIBLE, v >= 0,05 avant le trou)",
			st.ZoneRef, len(st.Spans), len(st.Gauge), r, len(blob), maxGap, hold, tr, vis)
		if len(st.Gauge) > 0 {
			extrait := st.Gauge
			if len(extrait) > 8 {
				extrait = extrait[:8]
			}
			b, _ := json.Marshal(extrait)
			t.Logf("    extrait : %s", b)
		}
		total += len(st.Gauge)
		totalBytes += len(blob)
		trous += tr
		visibles += vis
		rampes += r
	}
	if total != doc.Coverage.Zones.GaugePoints {
		t.Errorf("gaugePoints publie %d, points relus %d", doc.Coverage.Zones.GaugePoints, total)
	}
	t.Logf("  POINTS : %d en %d rampes · OCTETS des series : %d sur %d = %.3f %% de l'artefact"+
		" · trous intra-rampe > 1 s : %d, dont %d a arc VISIBLE (v >= 0,05 avant le trou)",
		total, rampes, totalBytes, fileBytes, 100*float64(totalBytes)/float64(fileBytes), trous,
		visibles)
}

// gaugeTemoinRampes decoupe une serie publiee en rampes (un point qui BAISSE ou qui suit un trou
// de plus de `hold` frames en ouvre une nouvelle) et compte les ecarts intra-rampe superieurs
// a `hold` — ceux pendant lesquels l'arc s'eteint cote client — en distinguant ceux ou l'arc
// etait VISIBLE (v >= 0,05 avant le trou : un clignotement) de ceux ou la jauge etait au repos
// (v < 0,05 : l'arc de zero ne trace rien, l'extinction ne se voit pas).
func gaugeTemoinRampes(g []GaugePoint, hold int) (rampes, trous, visibles, maxGap int) {
	for i, p := range g {
		if i == 0 {
			rampes = 1
			continue
		}
		gap := p.T - g[i-1].T
		if p.V < g[i-1].V {
			rampes++
			continue
		}
		if gap > hold {
			// Une montee qui reprend apres plus d'une seconde : cote client, l'arc s'est
			// eteint entre-temps. C'est soit une nouvelle rampe (jauge revenue au meme
			// niveau), soit un trou de cadence — les deux se comptent ici.
			trous++
			rampes++
			if g[i-1].V >= 0.05 {
				visibles++
			}
		}
		if gap > maxGap {
			maxGap = gap
		}
	}
	return rampes, trous, visibles, maxGap
}

// gaugeTemoinBascules publie la clause « la jauge monte avant la bascule du proprietaire ».
func gaugeTemoinBascules(t *testing.T, doc ReplayDocument) {
	t.Helper()
	if doc.FrameIntervalMS <= 0 {
		t.Logf("  BASCULES : sans objet (axe sans echelle)")
		return
	}
	before := gaugeTemoinBeforeMS / doc.FrameIntervalMS
	after := gaugeTemoinAfterMS / doc.FrameIntervalMS
	shift := gaugeTemoinShiftMS / doc.FrameIntervalMS
	ok, okShift, total, montees := 0, 0, 0, 0
	hasard := 0.0
	for _, st := range doc.ZoneStates {
		ups := gaugeTemoinMontees(st.Gauge)
		montees += len(ups)
		couverture := gaugeTemoinCouverture(ups, doc.FrameCount, before, after)
		for i := 1; i < len(st.Spans); i++ {
			sp := st.Spans[i]
			if sp.Owner == nil || gaugeTemoinSameOwner(st.Spans[i-1].Owner, sp.Owner) {
				continue
			}
			total++
			hasard += couverture
			if gaugeTemoinHasUp(ups, sp.T0-before, sp.T0+after) {
				ok++
			}
			if gaugeTemoinHasUp(ups, sp.T0+shift-before, sp.T0+shift+after) {
				okShift++
			}
		}
	}
	if total == 0 {
		t.Logf("  BASCULES : sans objet (aucun changement de proprietaire publie — mode a colline)")
		return
	}
	hasard /= float64(total)
	verdict := "NON TENU"
	if float64(ok)/float64(total) >= gaugeTemoinSeuil {
		verdict = "TENU"
	}
	t.Logf("  BASCULES du proprietaire : %d · precedees d'un pas MONTANT de la jauge dans"+
		" [t0 - %d ms ; t0 + %d ms] : %d/%d = %.1f %% (seuil %.0f %%) — %s",
		total, gaugeTemoinBeforeMS, gaugeTemoinAfterMS, ok, total, 100*float64(ok)/float64(total),
		100*gaugeTemoinSeuil, verdict)
	t.Logf("  TEMOIN (bascules decalees de +%d ms) : %d/%d = %.1f %% · NIVEAU DU HASARD (part de"+
		" l'axe de la zone ou une bascule tiree au hasard trouverait un de ses pas montants dans sa"+
		" fenetre, moyenne sur les %d bascules ; %d pas montants en tout) : %.1f %%",
		gaugeTemoinShiftMS, okShift, total, 100*float64(okShift)/float64(total), total, montees,
		100*hasard)
}

// gaugeTemoinCouverture rend la part des frames de l'axe qui ont un pas montant de CETTE zone dans
// leur fenetre [f - before ; f + after] : la probabilite qu'une bascule posee au hasard sur cette
// zone passe la clause. C'est l'UNION des fenetres, pas leur somme — les pas montants d'une rampe
// sont contigus et leurs fenetres se recouvrent presque entierement.
func gaugeTemoinCouverture(ups []int, frames, before, after int) float64 {
	if frames <= 0 {
		return 0
	}
	marked := make([]bool, frames)
	for _, u := range ups {
		for f := max(0, u-after); f <= u+before && f < frames; f++ {
			marked[f] = true
		}
	}
	n := 0
	for _, m := range marked {
		if m {
			n++
		}
	}
	return float64(n) / float64(frames)
}

// gaugeTemoinMontees rend les instants des pas MONTANTS d'une serie (v strictement plus haut que
// le point precedent).
func gaugeTemoinMontees(g []GaugePoint) []int {
	var out []int
	for i := 1; i < len(g); i++ {
		if g[i].V > g[i-1].V {
			out = append(out, g[i].T)
		}
	}
	return out
}

// gaugeTemoinHasUp dit si un pas montant tombe dans [t0, t1].
func gaugeTemoinHasUp(ups []int, t0, t1 int) bool {
	for _, u := range ups {
		if u >= t0 && u <= t1 {
			return true
		}
	}
	return false
}

// gaugeTemoinSameOwner compare deux proprietaires (nil = personne).
func gaugeTemoinSameOwner(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// TestZoneGaugeEchelleTemoin — L'ECHELLE DE LA JAUGE, lue dans le FILM (garde `ZONE_FILM`, un
// film par processus, avant-plan) : pour chaque slot de jauge (tag 3), l'excursion brute, les
// quantiles, la valeur la plus frequente et les premieres emissions. C'est ce qui dit si le
// PLANCHER de l'echelle « excursion mesuree » (la convention du schema 16, retiree depuis) est
// la jauge au repos ou une valeur aberrante — la question que le temoin de l'artefact a soulevee
// (zone 0 de `7344d24f` : rampes de 0,694 a 1,0).
func TestZoneGaugeEchelleTemoin(t *testing.T) {
	dir := os.Getenv("ZONE_FILM")
	if dir == "" {
		t.Skip("echelle de la jauge : poser ZONE_FILM=<repertoire des chunks d'un film>")
	}
	sc := p2bScan(t, dir)
	c := zoneCtx{origin: 0, step: 100_000, frames: 1 << 30, intervalMS: 100}
	ser := zoneSeriesOf(sc.Reads, c)
	for _, slot := range sortedZoneSlots(ser.gauge) {
		ss := ser.gauge[slot]
		if len(ss) < 3 {
			continue
		}
		vals := make([]uint64, len(ss))
		freq := map[uint64]int{}
		for i, s := range ss {
			vals[i] = s.v
			freq[s.v]++
		}
		sortU64(vals)
		mode, modeN := uint64(0), 0
		for v, n := range freq {
			if n > modeN {
				mode, modeN = v, n
			}
		}
		ramps := findZoneRamps(slot, ss)
		up, down, flat, resets := 0, 0, 0, 0
		for i := 1; i < len(ss); i++ {
			switch {
			case ss[i].v > ss[i-1].v:
				up++
			case ss[i].v < ss[i-1].v:
				down++
				if ss[i].v <= zoneGaugeQuantZero {
					resets++
				}
			default:
				flat++
			}
		}
		t.Logf("slot %d tag3 : %d emissions, %d rampes · min %d · p01 %d · p10 %d · mediane %d · p90 %d · max %d"+
			" · mode %d (x%d) · pas montants %d, descendants %d (dont %d retours a zero), nuls %d · premieres : %v",
			slot, len(ss), len(ramps), vals[0], vals[len(vals)/100], vals[len(vals)/10], vals[len(vals)/2],
			vals[len(vals)*9/10], vals[len(vals)-1], mode, modeN, up, down, resets, flat, gaugeTemoinHead(ss, 6))
	}
}

// sortU64 trie des quanta.
func sortU64(v []uint64) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j] < v[j-1]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}

// gaugeTemoinHead rend les n premieres emissions (frame moteur/100 ms : valeur).
func gaugeTemoinHead(ss []zoneSample, n int) []zoneSample {
	if len(ss) < n {
		return ss
	}
	return ss[:n]
}
