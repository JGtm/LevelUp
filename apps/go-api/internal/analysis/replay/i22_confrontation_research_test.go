package replay

// i22_confrontation_research_test.go — SONDE JETABLE (etude de faisabilite du 2026-08-24).
//
// CE QU'ELLE CONFRONTE : les comptes de grenades lus dans les PAQUETS DELTA (sonde
// filmdec/i22_delta_research_test.go, dont la sortie JSON est passee par I22_DELTA_JSON) aux
// comptes lus aux IMAGES-CLES (ScanFilmKeyframeInventory, le canal de production).
//
// LE TEST REFUTABLE : pour chaque lecture d'image-cle, la DERNIERE lecture delta anterieure du
// meme slot doit donner le meme quadruplet. Si les deux canaux divergent, l'un des deux lit a
// cote et le suivi delta est mort ; s'ils concordent, les deltas interpolent legitimement entre
// deux images-cles.
//
// LECTURE SEULE, gate par I22_FILM. Un seul decodage filmdec par process.
//
// USAGE (depuis apps/go-api, apres la sonde filmdec) :
//
//	CGO_ENABLED=0 I22_FILM=<film> I22_DELTA_JSON=<sortie de la sonde filmdec> \
//	  go test ./internal/analysis/replay/ -run '^TestI22Confrontation$' -timeout 30m -v

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"
)

type i22DeltaRead struct {
	Slot        uint32   `json:"slot"`
	TimestampUS uint64   `json:"ts_us"`
	Values      []uint64 `json:"values"`
}

func TestI22Confrontation(t *testing.T) {
	dir := os.Getenv("I22_FILM")
	js := os.Getenv("I22_DELTA_JSON")
	if dir == "" || js == "" {
		t.Skip("I22_FILM / I22_DELTA_JSON non definis : sonde de recherche sautee")
	}
	raw, err := os.ReadFile(js)
	if err != nil {
		t.Fatalf("lecture %s : %v", js, err)
	}
	var deltas []i22DeltaRead
	if err := json.Unmarshal(raw, &deltas); err != nil {
		t.Fatalf("json : %v", err)
	}
	bySlot := map[uint32][]i22DeltaRead{}
	for _, d := range deltas {
		bySlot[d.Slot] = append(bySlot[d.Slot], d)
	}
	for s := range bySlot {
		sort.Slice(bySlot[s], func(i, j int) bool {
			return bySlot[s][i].TimestampUS < bySlot[s][j].TimestampUS
		})
	}

	// La telemetrie de couverture (KeyframeInventoryStats, lot 2 du 2026-08-25) ne concerne
	// pas cette sonde : elle confronte les lectures, pas la sante du scan.
	kf, _, err := ScanFilmKeyframeInventory(dir, loadoutFamilies(), 0)
	if err != nil {
		t.Fatalf("keyframes : %v", err)
	}
	var total, withPrior, agree int
	var mismatches []string
	for _, k := range kf {
		if !k.GrenadesRead {
			continue
		}
		total++
		seq := bySlot[k.Slot]
		var prior *i22DeltaRead
		for i := range seq {
			if seq[i].TimestampUS <= k.TimestampUS {
				prior = &seq[i]
			}
		}
		if prior == nil {
			continue
		}
		withPrior++
		same := len(prior.Values) == 4
		if same {
			for i := 0; i < 4; i++ {
				if uint32(prior.Values[i]) != k.Grenades[i] {
					same = false
				}
			}
		}
		if same {
			agree++
		} else if len(mismatches) < 20 {
			mismatches = append(mismatches, fmt.Sprintf(
				"slot %d t=%.1fs kf=%v delta(t=%.1fs)=%v",
				k.Slot, float64(k.TimestampUS)/1e6, k.Grenades,
				float64(prior.TimestampUS)/1e6, prior.Values))
		}
	}
	t.Logf("images-cles avec grenades lues = %d | dont un delta anterieur du meme slot = %d",
		total, withPrior)
	if withPrior > 0 {
		t.Logf("CONCORDANCE = %d / %d = %.1f %%", agree, withPrior,
			100*float64(agree)/float64(withPrior))
	}
	for _, m := range mismatches {
		t.Logf("DIVERGENCE  %s", m)
	}

	// GAIN DE FRAICHEUR : age median de la derniere lecture connue, a 1 Hz sur la duree de vie
	// de chaque slot, avec les images-cles seules puis avec les deux canaux fusionnes.
	kfBySlot := map[uint32][]uint64{}
	for _, k := range kf {
		if k.GrenadesRead {
			kfBySlot[k.Slot] = append(kfBySlot[k.Slot], k.TimestampUS)
		}
	}
	var ageKF, ageMerged []float64
	var newInfo int
	for s, kts := range kfBySlot {
		sort.Slice(kts, func(i, j int) bool { return kts[i] < kts[j] })
		merged := append([]uint64(nil), kts...)
		for _, d := range bySlot[s] {
			merged = append(merged, d.TimestampUS)
			if d.TimestampUS > kts[0] && d.TimestampUS < kts[len(kts)-1] {
				newInfo++
			}
		}
		sort.Slice(merged, func(i, j int) bool { return merged[i] < merged[j] })
		for t0 := kts[0]; t0 <= kts[len(kts)-1]; t0 += 1_000_000 {
			ageKF = append(ageKF, lastAgeS(kts, t0))
			ageMerged = append(ageMerged, lastAgeS(merged, t0))
		}
	}
	t.Logf("lectures delta STRICTEMENT entre deux images-cles du meme slot = %d / %d", newInfo, len(deltas))
	t.Logf("age median de la derniere lecture : images-cles seules = %.2f s | fusionne = %.2f s",
		medianOf(ageKF), medianOf(ageMerged))
}

func lastAgeS(ts []uint64, at uint64) float64 {
	var last uint64
	var got bool
	for _, t := range ts {
		if t <= at {
			last, got = t, true
		}
	}
	if !got {
		return 0
	}
	return float64(at-last) / 1e6
}

func medianOf(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	return c[len(c)/2]
}
