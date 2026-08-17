package objectiveevents

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// score_measure_rounds_check_test.go — le TEST de la grammaire etendue (item A.0b.1) : il
// confronte le score reconstruit (manches figees + manche en cours) et la somme des frags a
// l'oracle, et publie la ventilation par forme d'en-tete.
//
// La grammaire elle-meme vit dans score_measure_rounds_test.go — separee pour tenir le seuil de
// 500 lignes par fichier.

// TestLotAPhase0bisManches mesure la grammaire etendue contre l'oracle : score de mode
// reconstruit (manches figees + manche en cours) et somme des frags des joueurs.
//
// Memes gardes que l'instrument de la phase 0 (`SCORE_FILM`, `SCORE_ORACLE`), un film par
// processus. Sortie : `<dossier de l'oracle>/lotA/<short8>_manches.tsv`.
func TestLotAPhase0bisManches(t *testing.T) {
	filmDir, oraclePath := os.Getenv(scoreFilmEnv), os.Getenv(scoreOracleEnv)
	if filmDir == "" || oraclePath == "" {
		t.Skipf("instrument non arme (%s=%q, %s=%q)", scoreFilmEnv, filmDir, scoreOracleEnv, oraclePath)
	}
	short := filepath.Base(filepath.Clean(filmDir))
	root := filepath.Dir(filepath.Dir(filepath.Clean(filmDir)))
	or := loadOracle(t, oraclePath, short)

	t.Setenv(filmCacheEnv, root)
	src, ok := newDiskFilmSource(t, short)
	if !ok {
		t.Fatalf("manifeste du film %s absent sous %s", short, root)
	}
	// SCORE_LOW_MAX eleve la borne des identifiants : c est la sonde des entites recreees.
	if v := os.Getenv("SCORE_LOW_MAX"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &extLowMax); err != nil {
			t.Fatalf("SCORE_LOW_MAX=%q illisible : %v", v, err)
		}
	}
	start := time.Now()
	recs := statRecordsExt(src)
	decodeMS := time.Since(start).Milliseconds()
	if len(recs) == 0 {
		t.Fatalf("aucun enregistrement lu par la grammaire etendue dans %s", filmDir)
	}

	w := &measureRows{}
	writeExtMeta(w, short, or, recs, decodeMS)
	writeExtScore(w, recs, or)
	writeExtKills(w, recs, or)
	writeExtSegments(w, recs, modeScoreComp)
	writeExtSegments(w, recs, coreKillsComp)
	writeExtSerie(w, recs)
	writeExtHeaders(w, recs, modeScoreComp)
	writeExtHeaders(w, recs, coreKillsComp)
	writeExtHeaderVerdict(w, recs, or)
	writeExtSerieH(w, recs)

	out := filepath.Join(filepath.Dir(oraclePath), "lotA", short+"_manches.tsv")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatalf("creation du dossier de sortie : %v", err)
	}
	if err := os.WriteFile(out, []byte(w.b.String()), 0o644); err != nil {
		t.Fatalf("ecriture de %s : %v", out, err)
	}
	t.Logf("mesure etendue ecrite : %s (%d enregistrements, %d ms)", out, len(recs), decodeMS)
}

// writeExtMeta ecrit la ventilation des enregistrements par forme de liste.
func writeExtMeta(m *measureRows, short string, or oracleMatch, recs []statRecordExt, decodeMS int64) {
	dense, baseline, finVals := 0, 0, 0
	tMin, tMax := recs[0].TimeMS, recs[0].TimeMS
	byGen := map[int]int{}
	for _, r := range recs {
		if r.Form.Dense {
			dense++
		}
		if r.Form.Baseline {
			baseline++
		}
		byGen[r.Form.Gen]++
		finVals += len(r.Fin)
		if r.TimeMS < tMin {
			tMin = r.TimeMS
		}
		if r.TimeMS > tMax {
			tMax = r.TimeMS
		}
	}
	m.row("metaext", short, or.Variant, or.Team0, or.Team1, or.DurationS,
		len(recs), dense, len(recs)-dense, baseline, finVals, tMin, tMax, decodeMS)
	for _, g := range sortedSlots(byGen) {
		m.row("forme_gen", g, byGen[g])
	}
}

// writeExtScore ecrit les manches figees du score de mode et le total reconstruit.
func writeExtScore(m *measureRows, recs []statRecordExt, or oracleMatch) {
	seen := roundsSeen(recs, modeScoreComp)
	for _, slot := range []int{6, 8} {
		m.row("manches", slot, len(seen[slot]), fmt.Sprint(seen[slot]))
	}
	for _, f := range dedupFinalized(recs, modeScoreComp) {
		m.row("figee", f.Slot, f.Round, f.Value, f.TimeMS)
	}
	tot := roundsTotal(recs, modeScoreComp)
	film := []int64{tot[6], tot[8]}
	oracle := []int64{int64(or.Team0), int64(or.Team1)}
	sort.Slice(film, func(i, j int) bool { return film[i] < film[j] })
	sort.Slice(oracle, func(i, j int) bool { return oracle[i] < oracle[j] })
	verdict := "ecart"
	if film[0] == oracle[0] && film[1] == oracle[1] {
		verdict = "exact"
	}
	m.row("total_score", verdict, fmt.Sprintf("%d/%d", tot[6], tot[8]),
		fmt.Sprintf("%d/%d", or.Team0, or.Team1))
}

// writeExtKills ecrit les frags reconstruits par slot de joueur et leur somme.
func writeExtKills(m *measureRows, recs []statRecordExt, or oracleMatch) {
	tot := roundsTotal(recs, coreKillsComp)
	var sum int64
	for _, slot := range sortedSlots(tot) {
		if IsTeamSlot(slot) {
			continue
		}
		sum += tot[slot]
		m.row("frags", slot, tot[slot])
	}
	var oracleSum int64
	for _, l := range or.Lines {
		oracleSum += int64(l.Kills)
	}
	verdict := "ecart"
	if sum == oracleSum {
		verdict = "exact"
	}
	m.row("total_frags", verdict, sum, oracleSum, len(or.Lines))
}

// dedupFinalized rend une valeur figee par (slot, manche) : celle qui l'emporte au vote, avec
// l'instant de sa premiere emission.
func dedupFinalized(recs []statRecordExt, comp int) []finalizedValue {
	votes := map[[2]int]map[int64]int{}
	first := map[[2]int]int{}
	for _, r := range recs {
		for _, f := range r.Fin {
			if f.Comp != comp {
				continue
			}
			k := [2]int{f.Slot, f.Round}
			if votes[k] == nil {
				votes[k] = map[int64]int{}
				first[k] = f.TimeMS
			}
			votes[k][f.Value]++
		}
	}
	var out []finalizedValue
	for k, byVal := range votes {
		out = append(out, finalizedValue{TimeMS: first[k], Slot: k[0], Comp: comp,
			Round: k[1], Value: majorityValue(byVal)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Slot != out[j].Slot {
			return out[i].Slot < out[j].Slot
		}
		return out[i].Round < out[j].Round
	})
	return out
}

// writeExtSegments ecrit les SEGMENTS de la serie de la manche en cours, par slot : un compteur
// de manche repart de zero, donc une chute de valeur borne une manche. C'est la mesure qui dit
// si le film porte une ou plusieurs manches, et laquelle la grammaire voit.
func writeExtSegments(m *measureRows, recs []statRecordExt, comp int) {
	series := map[int][]ScorePoint{}
	for _, r := range recs {
		v, ok := r.Cur[comp]
		if !ok || v.A < 0 {
			continue
		}
		series[r.Slot] = append(series[r.Slot], ScorePoint{TimeMS: r.TimeMS, Slot: r.Slot, Value: v.A})
	}
	for _, slot := range sortedSlots(series) {
		for i, seg := range segmentsOf(series[slot]) {
			m.row("segment", comp, slot, i, len(seg),
				seg[0].TimeMS, seg[len(seg)-1].TimeMS, seg[len(seg)-1].Value)
		}
	}
}

// segmentsOf decoupe une serie chronologique a chaque CHUTE de valeur, puis filtre chaque
// segment par la plus longue suite croissante (le meme critere que la production) et ne garde
// que ceux d'au moins [roundSegmentMin] emissions retenues.
func segmentsOf(pts []ScorePoint) [][]ScorePoint {
	sort.SliceStable(pts, func(i, j int) bool { return pts[i].TimeMS < pts[j].TimeMS })
	var out [][]ScorePoint
	var cur []ScorePoint
	flush := func() {
		if kept := longestRun(cur, true); len(kept) >= roundSegmentMin {
			out = append(out, kept)
		}
		cur = nil
	}
	for _, p := range pts {
		if len(cur) > 0 && p.Value < cur[len(cur)-1].Value {
			flush()
		}
		cur = append(cur, p)
	}
	flush()
	return out
}

// roundSegmentMin borne la longueur d'un segment pris pour une manche : un ancrage fortuit
// arrive isole, une manche reelle emet plusieurs fois.
const roundSegmentMin = 3

// writeExtSerie ecrit la serie complete du score de mode des deux slots d'equipe. Elle sert a
// lire la valeur du film a un instant DONNE — par exemple a `time_played` quand l'oracle semble
// fige avant la fin du film (item A.0b.3).
func writeExtSerie(m *measureRows, recs []statRecordExt) {
	series := map[int][]ScorePoint{}
	for _, r := range recs {
		if !IsTeamSlot(r.Slot) {
			continue
		}
		if v, ok := r.Cur[modeScoreComp]; ok && v.A >= 0 {
			series[r.Slot] = append(series[r.Slot], ScorePoint{TimeMS: r.TimeMS, Slot: r.Slot, Value: v.A})
		}
	}
	for _, slot := range sortedSlots(series) {
		for _, p := range longestRun(series[slot], true) {
			m.row("serie", slot, p.TimeMS, p.Value)
		}
	}
}
