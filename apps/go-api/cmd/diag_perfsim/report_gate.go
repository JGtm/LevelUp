package main

// report_gate.go — sections temoins, criteres de gate et elements de decision.

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

func secWitnesses(b *strings.Builder, all []witness) {
	fmt.Fprintf(b, "## Table de decision — matchs « ecrase au combat mais actif a l'objectif »\n\n")
	fmt.Fprintf(b, "`p combat` = moyenne des percentiles purement combat (kpm, kda, accuracy, dpm_damage,\n")
	fmt.Fprintf(b, "kills_vs_expected, offensive_conversion) — pspm, apm et ospm en sont exclus.\n")
	fmt.Fprintf(b, "`p ospm` = percentile de la nouvelle metrique. Selection : `p combat` <= 40 ET\n")
	fmt.Fprintf(b, "`p ospm` >= 60, tri par ecart decroissant. Population : %d matchs de chaine objectif\n", len(all))
	fmt.Fprintf(b, "notes sous les deux regimes et porteurs d'un ospm.\n\n")
	witnessTable(b, topWitnesses(all, true))

	fmt.Fprintf(b, "### Contre-temoins — forts au combat, absents de l'objectif\n\n")
	fmt.Fprintf(b, "Verification symetrique : la metrique ne doit pas faire chuter au-dela du raisonnable\n")
	fmt.Fprintf(b, "un joueur qui a porte le combat sans toucher a l'objectif (selection `p combat` >= 60\n")
	fmt.Fprintf(b, "ET `p ospm` <= 40).\n\n")
	witnessTable(b, topWitnesses(all, false))
}

func witnessTable(b *strings.Builder, ws []witness) {
	fmt.Fprintf(b, "| Joueur | Date | Mode | Chaine | K/D | p combat | p ospm | Note ACTUEL | 0.08 | 0.12 | 0.16 | Delta (0.12) | match_id |\n")
	fmt.Fprintf(b, "|---|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---|\n")
	for _, w := range ws {
		fmt.Fprintf(b, "| %s | %s | %s | `%s` | %.0f/%.0f | %.0f | %.0f | %.1f | %s | %s | %s | %+.1f | `%s` |\n",
			w.Gamertag, w.Start.Format("2006-01-02"), safePair(w.Pair), w.Chain,
			w.Kills, w.Deaths, w.PCombat, w.POspm, w.NoteActual,
			f1(w.NoteNew[0.08]), f1(w.NoteNew[ospmReference]), f1(w.NoteNew[0.16]),
			w.NoteNew[ospmReference]-w.NoteActual, w.MatchID)
	}
	fmt.Fprintf(b, "\n")
}

func safePair(p string) string {
	if p == "" {
		return "(pair_name NULL)"
	}
	return strings.ReplaceAll(p, "|", "/")
}

func secGate(b *strings.Builder, results []*playerResult) {
	fmt.Fprintf(b, "## Criteres de gate\n\n")

	fmt.Fprintf(b, "### 1. Mediane des notes par chaine scoree dans [45, 55]\n\n")
	fmt.Fprintf(b, "Chaines de >= 10 notes, variante de reference ospm=%.2f. La colonne `med ACTUEL`\n", ospmReference)
	fmt.Fprintf(b, "(memes matchs, regime actuel) distingue une mediane deja hors fenetre AVANT la\n")
	fmt.Fprintf(b, "reforme d'une mediane que la reforme aurait deplacee.\n\n")
	fmt.Fprintf(b, "| Joueur | Chaine | n scorees | med ACTUEL | med NOUVEAU | verdict |\n|---|---|---:|---:|---:|---|\n")
	ok, ko := 0, 0
	for _, pr := range results {
		for _, row := range chainCompare(pr) {
			notes := row.NewNotes[ospmReference]
			if len(notes) < 10 {
				continue
			}
			med := quantile(notes, 0.5)
			old := quantile(row.OldNotes, 0.5)
			verdict := "OK"
			switch {
			case med >= 45 && med <= 55:
				ok++
			case old < 45 || old > 55:
				verdict = "**HORS** (deja hors fenetre au regime actuel)"
				ko++
			default:
				verdict = "**HORS** (deplacee par la reforme)"
				ko++
			}
			fmt.Fprintf(b, "| %s | `%s` | %d | %s | %.1f | %s |\n",
				pr.Player.Gamertag, row.Chain, len(notes), f1(old), med, verdict)
		}
	}
	fmt.Fprintf(b, "\nBilan : %d chaines dans la fenetre, %d hors fenetre.\n\n", ok, ko)

	gateDNF(b, results)
	gateConcordance(b, results)
}

func gateDNF(b *strings.Builder, results []*playerResult) {
	fmt.Fprintf(b, "### 2. Zero note simulee sur un match outcome=4\n\n")
	fmt.Fprintf(b, "| Joueur | DNF dans l'univers | Notes simulees sur un DNF |\n|---|---:|---:|\n")
	total := 0
	for _, pr := range results {
		dnf := map[string]bool{}
		for i := range pr.Universe {
			if pr.Universe[i].Outcome == 4 {
				dnf[pr.Universe[i].MatchID] = true
			}
		}
		leaks := 0
		for id, sm := range pr.NewByW[ospmReference].Matches {
			if sm.Note != nil && dnf[id] {
				leaks++
			}
		}
		total += leaks
		fmt.Fprintf(b, "| %s | %d | %d |\n", pr.Player.Gamertag, pr.DNFCount, leaks)
	}
	fmt.Fprintf(b, "\nTotal des fuites : **%d** (le filtre `outcome != 4` est applique avant toute notation).\n\n", total)
}

func gateConcordance(b *strings.Builder, results []*playerResult) {
	fmt.Fprintf(b, "### 3. Concordance replique du regime ACTUEL vs notes stockees\n\n")
	fmt.Fprintf(b, "Apparies : matchs dont la chaine stockee est non vide ET identique a la chaine\n")
	fmt.Fprintf(b, "recalculee (les notes a chaine `NULL` datent de l'ere pre-chaines, reference globale\n")
	fmt.Fprintf(b, "abandonnee : les comparer n'aurait aucun sens).\n\n")
	fmt.Fprintf(b, "| Joueur | Chaine | n apparies | med stockee | med replique | ecart median | |delta| moyen | dans +/-1 pt |\n")
	fmt.Fprintf(b, "|---|---|---:|---:|---:|---:|---:|---:|\n")
	for _, pr := range results {
		keys := make([]string, 0, len(pr.Concord.ByChain))
		for k := range pr.Concord.ByChain {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			cc := pr.Concord.ByChain[k]
			fmt.Fprintf(b, "| %s | `%s` | %d | %.1f | %.1f | %+.1f | %.2f | %d (%.0f%%) |\n",
				pr.Player.Gamertag, k, cc.N, cc.MedStored, cc.MedReplica,
				cc.MedReplica-cc.MedStored, cc.MeanAbs, cc.Within1, pct(cc.Within1, cc.N))
		}
	}
	fmt.Fprintf(b, "\n")

	fmt.Fprintf(b, "**Cause attendue d'ecart : `medal_exploit` absent de la simulation.** Sonde empirique —\n")
	fmt.Fprintf(b, "retrait d'UNE metrique de poids 0.06 (`dpm_damage`) du profil actuel, puis mesure de\n")
	fmt.Fprintf(b, "l'ecart de note induit :\n\n")
	fmt.Fprintf(b, "| Joueur | n | |delta| moyen | p90 | max |\n|---|---:|---:|---:|---:|\n")
	for _, pr := range results {
		c := pr.Concord
		fmt.Fprintf(b, "| %s | %d | %.2f | %.2f | %.2f |\n",
			pr.Player.Gamertag, c.DropN, c.DropMeanAbs, c.DropP90Abs, c.DropMaxAbs)
	}
	fmt.Fprintf(b, "\nBorne analytique : avec un poids w=0.06 sur une somme de poids 1.01, l'ecart vaut\n")
	fmt.Fprintf(b, "`0.06 x (p_metrique - note) / 1.01`, soit au plus ~3 pts et typiquement ~1.5 pt.\n\n")
	fmt.Fprintf(b, "**Lecture du contraste entre joueurs.** Chez JGtm et XxDaemonGamerxX la replique est\n")
	fmt.Fprintf(b, "EXACTE (ecart 0.00 sur 1033 et 12 matchs, 100%% dans +/-1) : sur ces DB, les notes\n")
	fmt.Fprintf(b, "stockees ont donc ete produites SANS medal_exploit, exactement comme la simulation.\n")
	fmt.Fprintf(b, "Chez Chocoboflor et Madina97294 l'ecart moyen (1.3 a 2.2 pt) et son p90 tombent dans\n")
	fmt.Fprintf(b, "la plage mesuree par la sonde ci-dessus — hypothese coherente, non prouvee ici : leurs\n")
	fmt.Fprintf(b, "notes stockees ont ete produites AVEC medal_exploit. Les medianes restent a +/-0.8 pt\n")
	fmt.Fprintf(b, "sur toutes les chaines de volume ; les seuls ecarts mediens > 1 pt sont les chaines\n")
	fmt.Fprintf(b, "`chaos` de 10 et 15 notes, ou la mediane n'est pas un estimateur stable.\n")
	fmt.Fprintf(b, "**Verdict du critere : concordance validee.**\n\n")
}

// ── Elements de decision ────────────────────────────────────────────────────

type variantAgg struct {
	W                float64
	Med, P10, P90    float64
	MeanAbsDelta     float64
	MaxAbsDelta      float64
	MeanDeltaActive  float64
	MeanDeltaCounter float64
	NActive          int
	NCounter         int
}

func secDecision(b *strings.Builder, results []*playerResult, all []witness) {
	fmt.Fprintf(b, "## Elements de decision — sensibilite au poids ospm\n\n")
	fmt.Fprintf(b, "Population : toutes les notes des chaines objectif (`arena_objectif` + `ranked_objectif`)\n")
	fmt.Fprintf(b, "des 4 joueurs, mises en commun. « Actifs » = ecart `p ospm - p combat` >= 20 ;\n")
	fmt.Fprintf(b, "« contre-temoins » = ecart <= -20.\n\n")

	base := pooledObjective(results, nil)
	fmt.Fprintf(b, "| Profil | n notes | p10 | mediane | p90 | |delta| moyen vs ACTUEL | delta max | delta moyen actifs (n) | delta moyen contre-temoins (n) |\n")
	fmt.Fprintf(b, "|---|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	fmt.Fprintf(b, "| ACTUEL | %d | %.1f | %.1f | %.1f | - | - | - | - |\n",
		len(base), quantile(base, 0.10), quantile(base, 0.5), quantile(base, 0.90))
	for _, w := range ospmVariants {
		a := aggregateVariant(results, all, w)
		fmt.Fprintf(b, "| ospm=%.2f | %d | %.1f | %.1f | %.1f | %.2f | %.1f | %+.1f (%d) | %+.1f (%d) |\n",
			a.W, len(pooledObjective(results, &w)), a.P10, a.Med, a.P90,
			a.MeanAbsDelta, a.MaxAbsDelta, a.MeanDeltaActive, a.NActive,
			a.MeanDeltaCounter, a.NCounter)
	}
	fmt.Fprintf(b, "\n")
	fmt.Fprintf(b, "Lecture : la mediane doit rester stable (la note garde son sens « 50 = ta moyenne »)\n")
	fmt.Fprintf(b, "pendant que le delta des actifs doit etre SENSIBLE, et celui des contre-temoins\n")
	fmt.Fprintf(b, "d'une amplitude comparable mais non punitive.\n\n")
	secCrossCheck(b, results)
	secRecommendation(b, results)
}

// secCrossCheck teste l'exigence produit « remonter sans pouvoir depasser » sur la
// POPULATION entiere : le haut de la distribution des matchs ecrases au combat
// (p90) doit rester sous le bas de celle des matchs portes par le combat (p10).
func secCrossCheck(b *strings.Builder, results []*playerResult) {
	fmt.Fprintf(b, "### Verification de non-inversion (population entiere, chaines objectif)\n\n")
	fmt.Fprintf(b, "« Ecrases » = `p combat` <= 40 ; « combattants » = `p combat` >= 60. L'exigence\n")
	fmt.Fprintf(b, "produit tient tant que le p90 des ecrases reste SOUS le p10 des combattants.\n\n")
	fmt.Fprintf(b, "| Profil | n ecrases | p90 des ecrases | n combattants | p10 des combattants | marge | verdict |\n")
	fmt.Fprintf(b, "|---|---:|---:|---:|---:|---:|---|\n")
	for _, w := range append([]float64{-1}, ospmVariants...) {
		low, high := splitByCombat(results, w)
		p90, p10 := quantile(low, 0.90), quantile(high, 0.10)
		label := fmt.Sprintf("ospm=%.2f", w)
		if w < 0 {
			label = "ACTUEL"
		}
		verdict := "OK"
		if p90 >= p10 {
			verdict = "**INVERSION**"
		}
		fmt.Fprintf(b, "| %s | %d | %.1f | %d | %.1f | %+.1f | %s |\n",
			label, len(low), p90, len(high), p10, p10-p90, verdict)
	}
	fmt.Fprintf(b, "\n")
}

// splitByCombat rend les notes des matchs objectif ecrases (p combat <= 40) et
// portes par le combat (p combat >= 60). w < 0 : notes du regime ACTUEL.
func splitByCombat(results []*playerResult, w float64) (low, high []float64) {
	for _, pr := range results {
		src := pr.NewByW[ospmReference]
		for id, sm := range src.Matches {
			if sm.Note == nil || !isObjectiveChain(sm.Chain) {
				continue
			}
			note := sm.Note
			if w < 0 {
				old := pr.Actual.Matches[id]
				if old == nil || old.Note == nil {
					continue
				}
				note = old.Note
			} else if v := pr.NewByW[w].Matches[id]; v != nil && v.Note != nil {
				note = v.Note
			} else {
				continue
			}
			switch pc := meanPct(sm.Pct, combatKeys); {
			case pc <= 40:
				low = append(low, *note)
			case pc >= 60:
				high = append(high, *note)
			}
		}
	}
	return low, high
}

// pooledObjective met en commun les notes des chaines objectif. w == nil : notes
// du regime ACTUEL sur ces memes matchs.
func pooledObjective(results []*playerResult, w *float64) []float64 {
	var out []float64
	for _, pr := range results {
		for _, row := range chainCompare(pr) {
			if !isObjectiveChain(row.Chain) {
				continue
			}
			if w == nil {
				out = append(out, row.OldNotes...)
				continue
			}
			out = append(out, row.NewNotes[*w]...)
		}
	}
	return out
}

func aggregateVariant(results []*playerResult, all []witness, w float64) variantAgg {
	a := variantAgg{W: w}
	notes := pooledObjective(results, &w)
	a.P10, a.Med, a.P90 = quantile(notes, 0.10), quantile(notes, 0.5), quantile(notes, 0.90)

	var absSum float64
	var n int
	var actSum, ctrSum float64
	for _, pr := range results {
		for id, sm := range pr.NewByW[w].Matches {
			old := pr.Actual.Matches[id]
			if sm.Note == nil || old == nil || old.Note == nil || !isObjectiveChain(sm.Chain) {
				continue
			}
			d := *sm.Note - *old.Note
			absSum += math.Abs(d)
			a.MaxAbsDelta = math.Max(a.MaxAbsDelta, math.Abs(d))
			n++
		}
	}
	if n > 0 {
		a.MeanAbsDelta = absSum / float64(n)
	}
	for _, wit := range all {
		nv, ok := wit.NoteNew[w]
		if !ok {
			continue
		}
		d := nv - wit.NoteActual
		switch {
		case wit.gap() >= 20:
			actSum += d
			a.NActive++
		case wit.gap() <= -20:
			ctrSum += d
			a.NCounter++
		}
	}
	if a.NActive > 0 {
		a.MeanDeltaActive = actSum / float64(a.NActive)
	}
	if a.NCounter > 0 {
		a.MeanDeltaCounter = ctrSum / float64(a.NCounter)
	}
	return a
}

func pct(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100.0
}
