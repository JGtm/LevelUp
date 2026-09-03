package filmdec

// respawn_calibration_research_test.go — CALIBRER L'UNITÉ DU COMPTE À REBOURS (ti=5 i1).
//
// LE PROBLÈME (registre 2026-09-02) : le binaire écrit T0/T1 en ENTIERS BRUTS, sans
// déquantification — l'unité n'est pas dans l'exe. LES ANCRES DE VÉRITÉ TERRAIN viennent de
// l'utilisateur (2026-09-02) : le HUD affiche 8 s sur une mort ordinaire, 10 s sur un
// suicide, et le compteur apparaît un court instant APRÈS la mort (latence à mesurer, pas à
// supposer).
//
// LA MÉTHODE NE SUPPOSE RIEN DE L'UNITÉ : sur chaque épisode actif (un slot, des lectures
// consécutives du compteur), la PENTE de la valeur brute contre l'horloge du film (µs) donne
// directement « unités par seconde » — un compte à rebours en dixièmes descend à -10/s, en
// tics 30 Hz à -30/s, en tics 60 Hz à -60/s. La valeur de DÉPART d'un épisode, convertie par
// cette pente, doit tomber sur les ancres 8 s / 10 s (moins la latence d'affichage).
//
// LECTURE SEULE, UN SEUL FILM (D17), gardé par GAME_FILM — sauté partout ailleurs, CI
// comprise. La chaîne séquentielle est la source : records CERTAINS uniquement.
//
// USAGE (depuis apps/go-api) :
//
//	GAME_FILM=C:/.../data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestRespawnTimerCalibration$' -timeout 30m -v

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

// respawnReading : une lecture certaine du compteur, datée sur l'horloge du film.
type respawnReading struct {
	tUS uint64
	t0  uint16
	t1  uint16
}

// respawnEpisode : des lectures ACTIVES consécutives d'un même slot. Un trou de plus de
// respawnEpisodeGapUS, ou une valeur qui REMONTE, ouvre l'épisode suivant.
type respawnEpisode struct {
	slot     uint32
	readings []respawnReading
}

const respawnEpisodeGapUS = 2_000_000

func TestRespawnTimerCalibration(t *testing.T) {
	dir := os.Getenv(gameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de calibration saute", gameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	recs, _, _, err := ScanFilmGameEntitiesChain(dir)
	if err != nil {
		t.Fatalf("chaine sequentielle impossible : %v", err)
	}
	episodes := respawnEpisodes(recs)
	t.Logf("EPISODES ACTIFS : %d (lectures certaines du compteur, chaine sequentielle)", len(episodes))

	var slopes0, slopes1 []float64
	var starts0, starts1 []float64
	for _, ep := range episodes {
		if len(ep.readings) < 2 {
			// Une lecture isolée ne fait pas de pente, mais sa VALEUR reste une pièce :
			// elle doit tomber dans la plage des épisodes régressés.
			r := ep.readings[0]
			t.Logf("  slot %5d · lecture isolee t=%.2fs · T0 %4d · T1 %4d",
				ep.slot, float64(r.tUS)/1e6, r.t0, r.t1)
			continue
		}
		s0 := respawnSlope(ep.readings, func(r respawnReading) float64 { return float64(r.t0) })
		s1 := respawnSlope(ep.readings, func(r respawnReading) float64 { return float64(r.t1) })
		slopes0, slopes1 = append(slopes0, s0), append(slopes1, s1)
		starts0 = append(starts0, float64(ep.readings[0].t0))
		starts1 = append(starts1, float64(ep.readings[0].t1))
		dur := float64(ep.readings[len(ep.readings)-1].tUS-ep.readings[0].tUS) / 1e6
		t.Logf("  slot %5d · %2d lectures sur %5.2f s · T0 %4d->%4d (pente %+7.2f/s) · "+
			"T1 %4d->%4d (pente %+7.2f/s)",
			ep.slot, len(ep.readings), dur,
			ep.readings[0].t0, ep.readings[len(ep.readings)-1].t0, s0,
			ep.readings[0].t1, ep.readings[len(ep.readings)-1].t1, s1)
	}
	if len(slopes0) == 0 {
		t.Log("AUCUN episode a >= 3 lectures : le film ne permet pas la regression")
		return
	}
	m0, m1 := respawnMedian(slopes0), respawnMedian(slopes1)
	t.Logf("PENTES MEDIANES : T0 %+8.3f unites/s · T1 %+8.3f unites/s (episodes %d)",
		m0, m1, len(slopes0))
	// Le mot qui COMPTE À REBOURS est celui dont la pente est franchement négative ; l'autre
	// est probablement la CIBLE (constante de l'épisode). L'unité = -pente en unités/seconde.
	for _, c := range []struct {
		name   string
		slope  float64
		starts []float64
	}{{"T0", m0, starts0}, {"T1", m1, starts1}} {
		if c.slope >= -0.5 {
			t.Logf("%s : pente %+0.3f/s — ne decompte pas (cible ou etat)", c.name, c.slope)
			continue
		}
		unit := -c.slope
		var startsS []float64
		for _, s := range c.starts {
			startsS = append(startsS, s/unit)
		}
		t.Logf("%s DECOMPTE a %.2f unites/s -> depart des episodes en SECONDES : %s",
			c.name, unit, respawnHist(startsS))
		t.Logf("%s ANCRES USER : mort ordinaire 8 s, suicide 10 s, moins la latence "+
			"d'apparition du compteur — comparer aux departs ci-dessus", c.name)
	}
}

// respawnEpisodes regroupe les lectures actives par slot puis les coupe en épisodes.
func respawnEpisodes(recs []GameEntityRecord) []respawnEpisode {
	bySlot := map[uint32][]respawnReading{}
	for _, r := range recs {
		if r.TI != PlayerEngineTypeIndex || !r.HasRespawn || !r.Respawn.Active {
			continue
		}
		bySlot[r.Slot] = append(bySlot[r.Slot], respawnReading{r.TimestampUS, r.Respawn.T0, r.Respawn.T1})
	}
	slots := make([]uint32, 0, len(bySlot))
	for s := range bySlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []respawnEpisode
	for _, s := range slots {
		rs := bySlot[s]
		sort.Slice(rs, func(i, j int) bool { return rs[i].tUS < rs[j].tUS })
		cur := respawnEpisode{slot: s}
		for _, r := range rs {
			if n := len(cur.readings); n > 0 {
				prev := cur.readings[n-1]
				if r.tUS-prev.tUS > respawnEpisodeGapUS || r.t0 > prev.t0+5 || r.t1 > prev.t1+5 {
					out = append(out, cur)
					cur = respawnEpisode{slot: s}
				}
			}
			cur.readings = append(cur.readings, r)
		}
		if len(cur.readings) > 0 {
			out = append(out, cur)
		}
	}
	return out
}

// respawnSlope : régression linéaire simple valeur = f(t), en unités par SECONDE.
func respawnSlope(rs []respawnReading, val func(respawnReading) float64) float64 {
	n := float64(len(rs))
	t0 := rs[0].tUS
	var sx, sy, sxx, sxy float64
	for _, r := range rs {
		x := float64(r.tUS-t0) / 1e6
		y := val(r)
		sx, sy, sxx, sxy = sx+x, sy+y, sxx+x*x, sxy+x*y
	}
	den := n*sxx - sx*sx
	if den == 0 {
		return 0
	}
	return (n*sxy - sx*sy) / den
}

func respawnMedian(v []float64) float64 {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return s[len(s)/2]
}

// respawnHist : les départs en secondes, arrondis au dixième, comptés — deux ancres
// attendues (8 s, 10 s), tout le reste est une découverte.
func respawnHist(v []float64) string {
	hist := map[string]int{}
	for _, x := range v {
		hist[fmt.Sprintf("%.1fs", x)]++
	}
	keys := make([]string, 0, len(hist))
	for k := range hist {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return hist[keys[i]] > hist[keys[j]] })
	if len(keys) > 8 {
		keys = keys[:8]
	}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s x%d", k, hist[k]))
	}
	return fmt.Sprint(parts)
}
