// report.go — le rapport : la COURBE taux(seuil), avec ses deux temoins.
//
// Pourquoi une courbe et pas un chiffre. Le test strict (« le joueur est DANS la forme »)
// rend un taux bas, et un seuil choisi pour le remonter serait ajuste sur le resultat
// qu'il doit prouver. Publier la courbe entiere, avec le meme calcul applique a deux
// temoins, laisse le lecteur voir ou le signal se separe du bruit — et si jamais il ne
// s'en separe pas.
package main

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis/replay"
)

// thresholdsM : les seuils de distance balayes, en metres. Zero = appartenance stricte ;
// 20 m est la distance au-dela de laquelle plus AUCUNE action n'est observee sur le
// corpus, donc la borne au-dela de laquelle le seuil ne discrimine plus rien.
var thresholdsM = []float64{0, 2, 4, 6, 8, 10, 15, 20}

// sweepRow est une ligne de la courbe.
type sweepRow struct {
	threshold float64
	real      replay.ZoneCoverage
	shifted   replay.ZoneCoverage // temoin TEMPOREL : memes zones, instants decales
	moved     replay.ZoneCoverage // temoin SPATIAL : memes instants, zones deplacees
}

func printResults(res []result, tune runTuning) {
	printPerMatch(res, tune)
	printOriginCorrection(res, tune)
	rows := sweep(res, tune)
	printSweep(rows)
	printClockSweep(res, tune)
	printAgreement(res, tune)
	printDiagnostics(res)
	if tune.dump {
		printActionDump(res, tune)
	}
}

// printOriginCorrection publie le AVANT/APRES de la correction d'horloge LUE.
//
// Ce n'est PAS un balayage : aucun decalage n'est cherche, celui du film est applique
// (cf. correctedActions). La colonne « temoin » reste la garde — une correction qui
// remonterait AUSSI le temoin temporel ne mesurerait qu'un relachement, pas un recalage.
func printOriginCorrection(res []result, tune runTuning) {
	fmt.Println("CORRECTION D'HORLOGE — origine LUE de l'artefact appliquee aux instants (aucun balayage)")
	fmt.Printf("%-9s %-14s %9s %6s %6s %6s %6s %7s %7s %7s %7s\n",
		"film", "carte", "originMs", "posAV", "dansAV", "posAP", "dansAP",
		"dehorsAP", "sansPosAP", "tauxAV", "tauxAP")
	var totAV, totAP, totNull replay.ZoneCoverage
	sansOrigine := 0
	for _, r := range res {
		if r.err != nil {
			continue
		}
		if !r.hasOrigin {
			sansOrigine++
			fmt.Printf("%-9s %-14s %9s (origine non publiee — non corrigeable)\n",
				r.m.short, r.m.mapName, "-")
			continue
		}
		_, av := r.attribute(r.actions, r.zones, tune.maxGap, 0)
		_, ap := r.attribute(r.corrected, r.zones, tune.maxGap, 0)
		_, nu := r.attribute(r.correctedNull, r.zones, tune.maxGap, 0)
		totAV, totAP, totNull = addCoverage(totAV, av), addCoverage(totAP, ap), addCoverage(totNull, nu)
		fmt.Printf("%-9s %-14s %9d %6d %6d %6d %6d %7d %7d %6.1f%% %6.1f%%\n",
			r.m.short, r.m.mapName, r.originMS,
			av.Actions, av.Attributed, ap.Actions, ap.Attributed, ap.Outside, ap.NoPosition,
			pct(av), pct(ap))
	}
	fmt.Printf("%-9s %-14s %9s %6d %6d %6d %6d %7d %7d %6.1f%% %6.1f%%\n",
		"TOTAL", "", "", totAV.Actions, totAV.Attributed, totAP.Actions, totAP.Attributed,
		totAP.Outside, totAP.NoPosition, pct(totAV), pct(totAP))
	fmt.Printf("  temoin temporel APRES correction : %.1f%%  (rapport reel/temoin %s)\n",
		pct(totNull), ratioOf(pct(totAP), pct(totNull)))
	if sansOrigine > 0 {
		fmt.Printf("  %d film(s) sans origine publiee : exclus du APRES, comptes nulle part ailleurs\n",
			sansOrigine)
	}
	fmt.Println()
}

// ratioOf rend le rapport signal/temoin, ou « inf » quand le temoin est nul.
func ratioOf(real, witness float64) string {
	if witness <= 0 {
		return "inf"
	}
	return fmt.Sprintf("%.1f", real/witness)
}

// clockOffsets : decalages appliques a l'INSTANT des actions, en frames de 100 ms.
//
// Ils testent le confondant le plus probable d'un taux d'appartenance faible : le
// statborg replique ses compteurs a SON rythme (intervalles mesures de 1 a 31 s), donc
// l'instant lu est POSTERIEUR a l'action. Si c'est la cause, reculer l'instant doit faire
// MONTER le taux reel en laissant le temoin temporel plat — un maximum net a un decalage
// donne serait la signature de la latence, et sa valeur.
//
// Si au contraire le taux reel reste colle au temoin quel que soit le decalage, ce n'est
// pas un probleme d'horloge.
//
// [2026-08-25] Borne elargie de -10 s a -60 s (registre REGISTRE_REPORTS.md L13, mesure
// executee depuis le worktree wt/clock-offsets). Au lot 4 (2026-08-08), 3 films sur 8
// piquaient PILE sur l'ancienne borne (-100 = -10 s) : mesure tronquee. Le lot du
// 2026-08-14 (`PLAN_CONTAINMENT_ZONES.md` etape 1, thought_log meme date) a depuis EXPLIQUE
// et CORRIGE ce retard : ce n'est pas une latence a chercher par balayage, c'est le
// decalage de referentiel LU sur l'artefact (`originMs`, mesure 3,6-50,8 s selon le film,
// cf. `correctedActions` dans measure.go) — une correction EXACTE par film qui fait mieux
// que n'importe quel decalage balaye ici, sur chacun des 8 films du corpus d'origine. Ce
// balayage reste donc un DIAGNOSTIC de confirmation (la borne elargie retrouve-t-elle bien
// un pic INTERIEUR pres de l'origine lue de chaque film ?), plus jamais le mecanisme de
// correction — voir printOriginCorrection pour celui-ci.
var clockOffsets = []int{-600, -500, -400, -300, -200, -150, -100, -50, -30, -20, -10, -5, 0, 10}

// clockThresholdsM : deux seuils suffisent — l'appartenance stricte et un seuil relache
// representatif. Balayer les deux axes en entier n'apprendrait rien de plus.
var clockThresholdsM = []float64{0, 5}

// printClockSweep imprime taux(decalage d'horloge) pour deux seuils, avec le temoin.
func printClockSweep(res []result, tune runTuning) {
	fmt.Println("BALAYAGE D'HORLOGE — l'instant du statborg est-il en retard sur l'action ?")
	fmt.Printf("%10s %12s %12s %12s %12s\n",
		"decalage", "reel 0 m", "temoin 0 m", "reel 5 m", "temoin 5 m")
	for _, off := range clockOffsets {
		cells := make([]string, 0, 4)
		for _, th := range clockThresholdsM {
			var real, null replay.ZoneCoverage
			for _, r := range res {
				if r.err != nil {
					continue
				}
				_, a := r.attribute(shiftBy(r.actions, r.frameCount, off), r.zones, tune.maxGap, th)
				_, b := r.attribute(shiftBy(r.nullAction, r.frameCount, off), r.zones, tune.maxGap, th)
				real, null = addCoverage(real, a), addCoverage(null, b)
			}
			cells = append(cells, fmt.Sprintf("%11.1f%%", pct(real)), fmt.Sprintf("%11.1f%%", pct(null)))
		}
		fmt.Printf("%9.1fs %s %s %s %s\n", float64(off)/10, cells[0], cells[1], cells[2], cells[3])
	}
	fmt.Println()
	printPerMatchPeak(res, tune)
}

// printPerMatchPeak donne, POUR CHAQUE MATCH, le decalage qui maximise le taux.
//
// C'est le controle qui separe une vraie latence d'un chiffre ajuste. Un maximum global
// obtenu sur le corpus entier peut n'etre qu'un reglage sur le resultat qu'il doit
// prouver. Si en revanche les huit matchs — cartes, joueurs et dates differents — piquent
// INDEPENDAMMENT dans la meme region, c'est une propriete du format, pas du reglage.
func printPerMatchPeak(res []result, tune runTuning) {
	fmt.Println("  pic par match (controle anti-ajustement : memes valeurs sur 8 films independants ?)")
	fmt.Printf("  %-9s %-14s %10s %10s %10s\n", "film", "carte", "pic 0 m", "taux au pic", "taux a 0 s")
	for _, r := range res {
		if r.err != nil {
			continue
		}
		bestOff, bestRate, atZero := 0, -1.0, 0.0
		for _, off := range clockOffsets {
			_, cov := r.attribute(shiftBy(r.actions, r.frameCount, off), r.zones, tune.maxGap, 0)
			rate := pct(cov)
			if off == 0 {
				atZero = rate
			}
			if rate > bestRate {
				bestOff, bestRate = off, rate
			}
		}
		fmt.Printf("  %-9s %-14s %9.1fs %9.1f%% %9.1f%%\n",
			r.m.short, r.m.mapName, float64(bestOff)/10, bestRate, atZero)
	}
	fmt.Println()
}

// printPerMatch imprime le detail au seuil STRICT : c'est la proposition forte, celle qui
// dit « il etait dedans ».
func printPerMatch(res []result, tune runTuning) {
	fmt.Printf("ATTRIBUTION STRICTE (tolerance %d frame(s), temoin spatial a %.0f m)\n",
		tune.maxGap, tune.witness)
	fmt.Printf("%-9s %-14s %7s %7s %7s %7s %7s %7s\n",
		"film", "carte", "ident", "posees", "dedans", "dehors", "sansPos", "ambig")
	for _, r := range res {
		if r.err != nil {
			fmt.Printf("%-9s %-14s ERREUR: %v\n", r.m.short, r.m.mapName, r.err)
			continue
		}
		_, cov := r.attribute(r.actions, r.zones, tune.maxGap, 0)
		fmt.Printf("%-9s %-14s %7d %7d %7d %7d %7d %7d\n",
			r.m.short, r.m.mapName, r.identified, cov.Actions,
			cov.Attributed, cov.Outside, cov.NoPosition, cov.Ambiguous)
	}
	fmt.Println()
}

// attribute est le croisement d'un match a un seuil donne.
func (r result) attribute(actions []replay.ObjectiveAction, zones []replay.Zone,
	maxGap int, maxDist float64) ([]replay.ZoneAttribution, replay.ZoneCoverage) {
	return replay.AttributeZones(actions, r.tracks, zones,
		replay.AttributeOptions{MaxGapFrames: maxGap, MaxDistanceM: maxDist})
}

func sweep(res []result, tune runTuning) []sweepRow {
	rows := make([]sweepRow, 0, len(thresholdsM))
	for _, th := range thresholdsM {
		row := sweepRow{threshold: th}
		for _, r := range res {
			if r.err != nil {
				continue
			}
			moved := replay.TranslateZones(r.zones, mapvarOffset(tune.witness))
			_, a := r.attribute(r.actions, r.zones, tune.maxGap, th)
			_, b := r.attribute(r.nullAction, r.zones, tune.maxGap, th)
			_, c := r.attribute(r.actions, moved, tune.maxGap, th)
			row.real = addCoverage(row.real, a)
			row.shifted = addCoverage(row.shifted, b)
			row.moved = addCoverage(row.moved, c)
		}
		rows = append(rows, row)
	}
	return rows
}

func printSweep(rows []sweepRow) {
	fmt.Println("COURBE taux(seuil) — part des actions posees rattachees a une zone")
	fmt.Printf("%8s %14s %14s %14s %10s\n",
		"seuil m", "reel", "temoin temps", "temoin espace", "reel/pire")
	for _, r := range rows {
		pr, ps, pm := pct(r.real), pct(r.shifted), pct(r.moved)
		worst := ps
		if pm > worst {
			worst = pm
		}
		ratio := "inf"
		if worst > 0 {
			ratio = fmt.Sprintf("%.1f", pr/worst)
		}
		fmt.Printf("%8.0f %13.1f%% %13.1f%% %13.1f%% %10s\n", r.threshold, pr, ps, pm, ratio)
	}
	fmt.Println()
}

func pct(c replay.ZoneCoverage) float64 {
	if c.Actions == 0 {
		return 0
	}
	return 100 * float64(c.Attributed) / float64(c.Actions)
}

// printDiagnostics agrege la distribution des distances et le sort par statistique.
func printDiagnostics(res []result) {
	diag := newDistanceStats()
	for _, r := range res {
		if r.err != nil {
			continue
		}
		att, _ := r.attribute(r.actions, r.zones, replay.DefaultMaxGapFrames, 0)
		for _, a := range att {
			diag.add(a, r.zones)
		}
	}
	diag.print()
}

func addCoverage(a, b replay.ZoneCoverage) replay.ZoneCoverage {
	return replay.ZoneCoverage{
		Actions:    a.Actions + b.Actions,
		Attributed: a.Attributed + b.Attributed,
		NoPosition: a.NoPosition + b.NoPosition,
		Outside:    a.Outside + b.Outside,
		Ambiguous:  a.Ambiguous + b.Ambiguous,
	}
}

// sortedCopy rend une copie triee — les quantiles exigent un ordre, et trier en place
// modifierait la donnee de l'appelant.
func sortedCopy(v []float64) []float64 {
	out := append([]float64(nil), v...)
	sort.Float64s(out)
	return out
}
