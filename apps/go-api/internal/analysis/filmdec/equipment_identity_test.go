package filmdec

// equipment_identity_test.go — INSTRUMENT DE MESURE de l'identité des entités `ti=37`
// (lot R3, cf. .ai/V7.5/replay2d/PLAN_R3_IDENTITE_TI37.md).
//
// CE QU'IL MESURE, dans l'ordre du plan :
//
//	phase 1.2/1.3 — les dénominateurs du balayage, et L'ORACLE BIT-EXACT : la marche complète
//	  d'un record `ti=37` d'image-clé doit atterrir EXACTEMENT sur le premier bit du record
//	  suivant. La MATRICE des huit combinaisons de grammaire est probée sur le même
//	  dénominateur — c'est elle qui tranche, pas une supposition.
//	phase 2.1/2.2 — par champ, et UNIQUEMENT sur les records bit-exacts : cardinalité des
//	  valeurs, couverture des vies d'objet, et STABILITÉ par vie (une identité qui change au
//	  cours d'une vie n'est pas une identité).
//
// Il ne NOMME rien : il rend des entiers et leurs dénominateurs.
//
// LECTURE SEULE, gardé par EQUIP_ID_FILM, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process (globaux de paquet) : le verrou est pris pour tout le test.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 EQUIP_ID_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestEquipmentIdentity$' -timeout 30m -v

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

const equipIDFilmEnv = "EQUIP_ID_FILM"

func TestEquipmentIdentity(t *testing.T) {
	dir := os.Getenv(equipIDFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", equipIDFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	reads, st, err := ScanFilmEquipmentIdentity(dir, EquipmentIdentityLayouts[0])
	if err != nil {
		t.Fatalf("balayage d'identité impossible : %v", err)
	}

	t.Logf("== DÉNOMINATEURS — %s ==", dir)
	t.Logf("  chunks lus %d · images-clés %d · records keyframe TOUS archétypes %d",
		st.Chunks, st.Keyframes, st.AllRecords)
	t.Logf("  records ti=%d %d · bornés par un voisin %d (les seuls contrôlables)",
		EquipmentTypeIndex, st.Records, st.Bounded)

	eqidLogMatrix(t, st)
	eqidLogGaps(t, st)
	if len(reads) == 0 {
		t.Log("VERDICT phase 1 : aucune marche conclusive — la voie H1 (état par défaut au " +
			"keyframe) est CLOSE NÉGATIVE sur ce film")
		return
	}
	exact := eqidExactOnly(reads)
	t.Logf("== RECORDS RETENUS : %d bit-exacts sur %d marches conclusives ==",
		len(exact), len(reads))
	if len(exact) == 0 {
		t.Log("VERDICT phase 1 : AUCUN record bit-exact — les valeurs lues NE SONT PAS des " +
			"mesures. Elles ne sont pas publiées.")
		return
	}
	eqidLogFields(t, exact)
	eqidLogLives(t, exact)
}

// eqidExactOnly ne garde que les records dont la marche est bit-exacte. C'est la règle du
// lot : une valeur extraite d'un record qui ne retombe pas juste n'est pas une mesure.
func eqidExactOnly(reads []EquipmentIdentityRead) []EquipmentIdentityRead {
	out := make([]EquipmentIdentityRead, 0, len(reads))
	for _, r := range reads {
		if r.Exact {
			out = append(out, r)
		}
	}
	return out
}

// eqidLogMatrix publie la MATRICE des combinaisons de grammaire (phase 1.3). C'est la
// mesure qui décide de la grammaire du record keyframe — ou qui dit qu'aucune ne marche.
func eqidLogMatrix(t *testing.T, st EquipmentIdentityStats) {
	t.Helper()
	t.Logf("== MATRICE DES GRAMMAIRES (phase 1.3) — %d records bornés ==", st.Bounded)
	best, bestK := -1, -1
	for k, l := range EquipmentIdentityLayouts {
		pct := 0.0
		if st.Bounded > 0 {
			pct = 100 * float64(st.LayoutExact[k]) / float64(st.Bounded)
		}
		t.Logf("  [%d] %s · BIT-EXACT %5d (%5.1f %%) · désync %5d",
			k, l, st.LayoutExact[k], pct, st.LayoutDesync[k])
		if st.LayoutExact[k] > best {
			best, bestK = st.LayoutExact[k], k
		}
	}
	if best <= 0 {
		t.Log("  VERDICT MATRICE : AUCUNE combinaison ne rend une seule marche bit-exacte — " +
			"la grammaire du record keyframe `ti=37` n'est PAS celle du record NEW delta")
		return
	}
	t.Logf("  VERDICT MATRICE : la combinaison [%d] domine (%d marches bit-exactes)", bestK, best)
}

// eqidLogGaps publie la distribution des écarts finaux sous la combinaison retenue.
func eqidLogGaps(t *testing.T, st EquipmentIdentityStats) {
	t.Helper()
	t.Logf("== COMBINAISON RETENUE [0] — bit-exactes %d · ratées %d · désync %d ==",
		st.Exact, st.Inexact, st.Desync)
	gaps := make([]int, 0, len(st.GapHist))
	for g := range st.GapHist {
		gaps = append(gaps, g)
	}
	sort.Slice(gaps, func(i, j int) bool { return st.GapHist[gaps[i]] > st.GapHist[gaps[j]] })
	for i, g := range gaps {
		if i >= 6 {
			break
		}
		t.Logf("    écart final %6d bits · %5d fois", g, st.GapHist[g])
	}
}

// eqidLogFields publie, par champ, la cardinalité et les valeurs dominantes (phase 2.1).
func eqidLogFields(t *testing.T, reads []EquipmentIdentityRead) {
	t.Helper()
	t.Logf("== CHAMPS (phase 2.1) — %d lectures BIT-EXACTES ==", len(reads))
	for f := 0; f < EquipIDFieldCount; f++ {
		hist := map[uint64]int{}
		gated := 0
		for _, r := range reads {
			if r.Present[f] {
				hist[r.Val[f]]++
			} else {
				gated++
			}
		}
		t.Logf("  %-32s porte OUVERTE %5d · fermée %5d · valeurs DISTINCTES %d",
			EquipIDField(f), len(reads)-gated, gated, len(hist))
		for _, line := range eqidTopValues(hist, 6) {
			t.Logf("      %s", line)
		}
	}
}

// eqidTopValues rend les n valeurs les plus fréquentes d'un histogramme, formatées.
func eqidTopValues(hist map[uint64]int, n int) []string {
	vals := make([]uint64, 0, len(hist))
	for v := range hist {
		vals = append(vals, v)
	}
	sort.Slice(vals, func(i, j int) bool {
		if hist[vals[i]] != hist[vals[j]] {
			return hist[vals[i]] > hist[vals[j]]
		}
		return vals[i] < vals[j]
	})
	out := make([]string, 0, n)
	for i, v := range vals {
		if i >= n {
			break
		}
		// Le décalage d'un bit est affiché parce que c'est ainsi que les tags `proj` des
		// grenades se lisent dans le flux (cf. grenade_events.go) — l'afficher coûte zéro et
		// évite de re-mesurer plus tard.
		out = append(out, fmt.Sprintf("0x%08x (>>1 = 0x%08x) · %5d fois", v, v>>1, hist[v]))
	}
	return out
}

// eqidLifeKey identifie une VIE d'objet : la paire (slot, gen), jamais le slot seul.
type eqidLifeKey struct{ slot, gen uint32 }

// eqidLogLives publie la STABILITÉ par vie d'objet (phase 2.2) : une valeur qui change au
// cours d'une vie n'identifie pas cette vie.
func eqidLogLives(t *testing.T, reads []EquipmentIdentityRead) {
	t.Helper()
	lives := map[eqidLifeKey]bool{}
	for _, r := range reads {
		lives[eqidLifeKey{r.Slot, r.Gen}] = true
	}
	t.Logf("== STABILITÉ PAR VIE (phase 2.2) — %d vies d'objet bit-exactes ==", len(lives))
	for f := 0; f < EquipIDFieldCount; f++ {
		perLife := map[eqidLifeKey]map[uint64]bool{}
		for _, r := range reads {
			if !r.Present[f] {
				continue
			}
			k := eqidLifeKey{r.Slot, r.Gen}
			if perLife[k] == nil {
				perLife[k] = map[uint64]bool{}
			}
			perLife[k][r.Val[f]] = true
		}
		stable, total := 0, len(perLife)
		for _, vs := range perLife {
			if len(vs) == 1 {
				stable++
			}
		}
		pct := 0.0
		if total > 0 {
			pct = 100 * float64(stable) / float64(total)
		}
		t.Logf("  %-32s vies couvertes %4d / %4d · à valeur UNIQUE %4d (%.1f %%)",
			EquipIDField(f), total, len(lives), stable, pct)
	}
}
