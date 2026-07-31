// tmp_catalystprobe — sonde « la plage de déquantification est-elle per-map ? ».
//
// Décode les positions bipeds d'un film AVEC la plage actuelle (QuantRangeCEBiped, relevée
// sur la map de 000d5950 / Cliffhanger) et sort les métriques qui trahissent une plage
// fausse : enveloppe, pas moyen entre échantillons consécutifs d'un slot, distribution des
// quanta bruts. Exporte le CSV avec les INDICES DE QUANTUM BRUTS pour permettre une
// re-déquantification ultérieure sans re-décoder.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_catalystprobe <filmDir> <out.csv>
package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: tmp_catalystprobe <filmDir> <out.csv>")
		os.Exit(2)
	}
	dir, outPath := os.Args[1], os.Args[2]

	// Jeu 1 : pipeline COMPLET (filtres par défaut) — ce que produit le décodeur livré.
	full, err := filmdec.ScanFilmBipedPositions(dir, filmdec.DefaultScanFilmOptions())
	if err != nil {
		fmt.Println("scan(default):", err)
		os.Exit(1)
	}
	// Jeu 2 : SANS les filtres dépendants de la plage (vitesse / isolement). Seul
	// DropSaturated est conservé : il opère sur les quanta, pas sur les mètres.
	rawOpt := filmdec.DefaultScanFilmOptions()
	rawOpt.MaxSpeedMPS, rawOpt.IsolationGapMS = 0, 0
	raw, err := filmdec.ScanFilmBipedPositions(dir, rawOpt)
	if err != nil {
		fmt.Println("scan(raw):", err)
		os.Exit(1)
	}

	fmt.Printf("=== FILM %s ===\n", dir)
	report("PIPELINE COMPLET (filtres vitesse+isolement, plage Cliffhanger)", full)
	report("BRUT (sans filtre dépendant de la plage)", raw)

	if err := writeCSV(outPath, raw); err != nil {
		fmt.Println("csv:", err)
		os.Exit(1)
	}
	fmt.Printf("\nCSV (jeu BRUT, quanta inclus) -> %s (%d lignes)\n", outPath, len(raw))
}

func writeCSV(path string, recs []filmdec.BipedPosition) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	w := bufio.NewWriterSize(f, 1<<20)
	fmt.Fprintln(w, "slot,chunk,packet,ts,qx,qy,qz,x,y,z")
	for _, r := range recs {
		fmt.Fprintf(w, "%d,%d,%d,%d,%d,%d,%d,%.5f,%.5f,%.5f\n",
			r.Slot, r.Chunk, r.PacketIndex, r.TimestampUS, r.Q[0], r.Q[1], r.Q[2], r.X, r.Y, r.Z)
	}
	return w.Flush()
}

func report(title string, recs []filmdec.BipedPosition) {
	fmt.Printf("\n--- %s ---\n", title)
	if len(recs) == 0 {
		fmt.Println("aucune position")
		return
	}
	slots := map[uint32]int{}
	var lo, hi [3]float64
	var qlo, qhi [3]uint32
	for a := 0; a < 3; a++ {
		lo[a], hi[a] = math.Inf(1), math.Inf(-1)
		qlo[a], qhi[a] = math.MaxUint32, 0
	}
	for _, r := range recs {
		slots[r.Slot]++
		v := [3]float64{float64(r.X), float64(r.Y), float64(r.Z)}
		for a := 0; a < 3; a++ {
			lo[a] = math.Min(lo[a], v[a])
			hi[a] = math.Max(hi[a], v[a])
			if r.Q[a] < qlo[a] {
				qlo[a] = r.Q[a]
			}
			if r.Q[a] > qhi[a] {
				qhi[a] = r.Q[a]
			}
		}
	}
	fmt.Printf("positions=%d slots=%d\n", len(recs), len(slots))
	fmt.Printf("enveloppe monde : X[%.3f,%.3f] span %.3f | Y[%.3f,%.3f] span %.3f | Z[%.3f,%.3f] span %.3f\n",
		lo[0], hi[0], hi[0]-lo[0], lo[1], hi[1], hi[1]-lo[1], lo[2], hi[2], hi[2]-lo[2])
	fmt.Printf("quanta bruts    : qx[%d,%d]/8192 (%.1f%%) qy[%d,%d]/8192 (%.1f%%) qz[%d,%d]/16384 (%.1f%%)\n",
		qlo[0], qhi[0], 100*float64(qhi[0]-qlo[0])/8192,
		qlo[1], qhi[1], 100*float64(qhi[1]-qlo[1])/8192,
		qlo[2], qhi[2], 100*float64(qhi[2]-qlo[2])/16384)
	stepStats(recs)
	quantumStepStats(recs)
}

// quantumStepStats mesure le pas entre échantillons consécutifs EN INDICES DE QUANTUM.
// Cette métrique ne dépend d'AUCUNE plage de déquantification : elle sépare « la plage est
// fausse » (pas en quanta cohérent, seule l'échelle en mètres change) de « le décodage ne
// trouve pas de trajectoire » (pas en quanta ~ uniforme sur 2^w).
func quantumStepStats(recs []filmdec.BipedPosition) {
	bySlot := map[uint32][]filmdec.BipedPosition{}
	for _, r := range recs {
		bySlot[r.Slot] = append(bySlot[r.Slot], r)
	}
	var dq [3][]float64
	for _, s := range bySlot {
		for i := 1; i < len(s); i++ {
			dt := float64(s[i].TimestampUS-s[i-1].TimestampUS) / 1e6
			if dt <= 0 || dt > 0.5 {
				continue
			}
			for a := 0; a < 3; a++ {
				d := float64(s[i].Q[a]) - float64(s[i-1].Q[a])
				dq[a] = append(dq[a], math.Abs(d))
			}
		}
	}
	if len(dq[0]) == 0 {
		return
	}
	// Référence « bruit blanc » : si les quanta étaient uniformes sur [0,2^w[, l'écart
	// absolu moyen entre deux tirages vaut 2^w/3.
	widths := [3]float64{8192, 8192, 16384}
	names := [3]string{"qx", "qy", "qz"}
	for a := 0; a < 3; a++ {
		sort.Float64s(dq[a])
		fmt.Printf("|d%s| entre échantillons : median %.0f p90 %.0f | attendu si BRUIT UNIFORME %.0f -> %.0f%% du bruit\n",
			names[a], med(dq[a]), pct(dq[a], 90), widths[a]/3, 100*med(dq[a])/(widths[a]/3))
	}
}

// stepStats : distance et vitesse entre échantillons consécutifs du même slot. Une plage
// de déquantification fausse en ÉCHELLE gonfle ou écrase mécaniquement ces valeurs.
func stepStats(recs []filmdec.BipedPosition) {
	bySlot := map[uint32][]filmdec.BipedPosition{}
	for _, r := range recs {
		bySlot[r.Slot] = append(bySlot[r.Slot], r)
	}
	var dists, speeds, dts []float64
	for _, s := range bySlot {
		for i := 1; i < len(s); i++ {
			dt := float64(s[i].TimestampUS-s[i-1].TimestampUS) / 1e6
			if dt <= 0 || dt > 0.5 {
				continue
			}
			d := math.Sqrt(sq(float64(s[i].X-s[i-1].X)) + sq(float64(s[i].Y-s[i-1].Y)) + sq(float64(s[i].Z-s[i-1].Z)))
			dists = append(dists, d)
			speeds = append(speeds, d/dt)
			dts = append(dts, dt)
		}
	}
	if len(dists) == 0 {
		fmt.Println("pas de paires consécutives exploitables")
		return
	}
	sort.Float64s(dists)
	sort.Float64s(speeds)
	sort.Float64s(dts)
	fmt.Printf("pas consécutifs (n=%d) : dt median %.4f s | dist median %.4f wu p90 %.4f p99 %.4f max %.3f\n",
		len(dists), med(dts), med(dists), pct(dists, 90), pct(dists, 99), dists[len(dists)-1])
	fmt.Printf("vitesse : median %.3f wu/s p90 %.3f p99 %.3f max %.2f | >35 wu/s : %.2f%%\n",
		med(speeds), pct(speeds, 90), pct(speeds, 99), speeds[len(speeds)-1], 100*fracAbove(speeds, 35))
}

func sq(v float64) float64 { return v * v }
func med(v []float64) float64 {
	return v[len(v)/2]
}
func pct(v []float64, p int) float64 {
	i := len(v) * p / 100
	if i >= len(v) {
		i = len(v) - 1
	}
	return v[i]
}
func fracAbove(sorted []float64, t float64) float64 {
	i := sort.SearchFloat64s(sorted, t)
	return float64(len(sorted)-i) / float64(len(sorted))
}
