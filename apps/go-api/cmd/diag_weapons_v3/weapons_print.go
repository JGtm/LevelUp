package main

// weapons_print.go — impression stdout du rapport §4 (par match) + agrégat panel.

import "fmt"

// printWeaponReport imprime le rapport §4 d'un match.
func printWeaponReport(m matchRef, reg *registryRow, r weaponReport) {
	fmt.Printf("[%s] %s — variant=%q — %d kills\n", m.short, m.full, reg.variant, r.total)
	if r.total == 0 {
		fmt.Println("  Aucun kill highlight_events (rien à attribuer).")
		return
	}
	printConfBlock("1. Confidence v3 ", r.confV3, r.total)
	fmt.Printf("  2. No-weapon v3   : %d (%s)\n", r.nullV3, pct(r.nullV3, r.total))
	fmt.Printf("  3. Armes distinctes v3 (high-32) : %d\n", len(r.distinctV3))
	printSignalBlock(r)
	printVsV2Block(r)
	printRecoveryBlock(r)
	printAggBlock(r)
}

// printConfBlock imprime une distribution de confiance (ordre stable + %).
func printConfBlock(label string, dist map[string]int, total int) {
	fmt.Printf("  %s: ", label)
	for i, lvl := range confLevels {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Printf("%s=%d (%s)", lvl, dist[lvl], pct(dist[lvl], total))
	}
	fmt.Println()
}

// printSignalBlock imprime le breakdown SourceSignal v3 (§4.4).
func printSignalBlock(r weaponReport) {
	fmt.Print("  4. SourceSignal v3: ")
	for i, sig := range signalLevels {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Printf("%s=%d", sig, r.signalV3[sig])
	}
	fmt.Println()
}

// printVsV2Block imprime la distribution v2 + l'agreement v2=high (§4.5).
func printVsV2Block(r weaponReport) {
	if r.pairedN == 0 {
		fmt.Println("  5. vs v2          : aucune baseline weapon_kills v2 pour ce match (skip agreement)")
		return
	}
	printConfBlock("5. Confidence v2 ", r.confV2, r.pairedN)
	if r.v2HighN == 0 {
		fmt.Println("     Agreement v2=high : n/a (0 kill v2 high apparié)")
		return
	}
	a := pct(r.agreeN, r.v2HighN)
	flag := ""
	if ratio(r.agreeN, r.v2HighN) < agreementThreshold {
		flag = "  [REGRESSION <98%]"
	}
	fmt.Printf("     Agreement v2=high : %d/%d même high-32 (%s)%s\n", r.agreeN, r.v2HighN, a, flag)
}

// printRecoveryBlock imprime la récupération v2 none/null → v3 résolu (§4.6).
func printRecoveryBlock(r weaponReport) {
	tot := r.recoveredMelee + r.recoveredGrenade + r.recoveredFire
	fmt.Printf("  6. Récupération v2 none/null → v3 : %d (melee=%d grenade=%d fire=%d)\n",
		tot, r.recoveredMelee, r.recoveredGrenade, r.recoveredFire)
}

// printAggBlock imprime la comparaison melee/grenade v3 vs API (§4.7).
func printAggBlock(r weaponReport) {
	if !r.aggHasCols {
		fmt.Println("  7. vs agrégats    : colonnes melee_kills/grenade_kills absentes — SKIP")
		return
	}
	if len(r.aggLines) == 0 {
		fmt.Println("  7. vs agrégats    : aucun melee/grenade (v3 ou API) sur ce match")
		return
	}
	fmt.Println("  7. vs agrégats (melee/grenade v3 vs API, ±1) :")
	for _, l := range r.aggLines {
		fmt.Printf("       %-22s melee v3=%d api=%d %s · grenade v3=%d api=%d %s\n",
			l.xuid, l.v3Melee, l.apiMelee, okStr(l.meleeWithinTol),
			l.v3Grenade, l.apiGrenade, okStr(l.grenWithin))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Agrégat panel
// ─────────────────────────────────────────────────────────────────────────────

// agreementThreshold — plancher d'agreement v2=high (§0 : ≥ 98%).
const agreementThreshold = 0.98

// panelAgg cumule les métriques §4 sur l'ensemble du panel pour le verdict §0.
type panelAgg struct {
	matches    int
	total      int
	confV3     map[string]int
	nullV3     int
	signalV3   map[string]int
	distinctV3 map[uint32]bool

	pairedN int
	confV2  map[string]int
	v2HighN int
	agreeN  int

	recMelee, recGrenade, recFire int

	aggMatches  int
	aggMeleeOK  int
	aggMeleeTot int
	aggGrenOK   int
	aggGrenTot  int
}

// newPanelAgg initialise les maps de l'agrégat panel.
func newPanelAgg() *panelAgg {
	return &panelAgg{
		confV3:     map[string]int{},
		signalV3:   map[string]int{},
		distinctV3: map[uint32]bool{},
		confV2:     map[string]int{},
	}
}

// add cumule un rapport de match dans l'agrégat panel.
func (p *panelAgg) add(r weaponReport) {
	if r.total == 0 {
		return
	}
	p.matches++
	p.total += r.total
	p.nullV3 += r.nullV3
	addInto(p.confV3, r.confV3)
	addInto(p.signalV3, r.signalV3)
	for h := range r.distinctV3 {
		p.distinctV3[h] = true
	}
	p.pairedN += r.pairedN
	addInto(p.confV2, r.confV2)
	p.v2HighN += r.v2HighN
	p.agreeN += r.agreeN
	p.recMelee += r.recoveredMelee
	p.recGrenade += r.recoveredGrenade
	p.recFire += r.recoveredFire
	p.cumulAgg(r)
}

// cumulAgg cumule la cohérence agrégats §4.7 (taux de joueurs dans la tolérance).
func (p *panelAgg) cumulAgg(r weaponReport) {
	if !r.aggHasCols || len(r.aggLines) == 0 {
		return
	}
	p.aggMatches++
	for _, l := range r.aggLines {
		p.aggMeleeTot++
		p.aggGrenTot++
		if l.meleeWithinTol {
			p.aggMeleeOK++
		}
		if l.grenWithin {
			p.aggGrenOK++
		}
	}
}

// addInto additionne src dans dst (compteurs par clé).
func addInto(dst, src map[string]int) {
	for k, v := range src {
		dst[k] += v
	}
}

// print imprime la synthèse panel + le verdict §0 (seuils HIGH/NULL/agreement).
func (p *panelAgg) print() {
	fmt.Printf("=== SYNTHÈSE PANEL (%d match[s], %d kills) ===\n", p.matches, p.total)
	if p.total == 0 {
		fmt.Println("  Aucun kill agrégé.")
		return
	}
	printConfBlock("Confidence v3", p.confV3, p.total)
	fmt.Printf("  No-weapon v3      : %d (%s)\n", p.nullV3, pct(p.nullV3, p.total))
	fmt.Printf("  Armes distinctes  : %d\n", len(p.distinctV3))
	printSignalPanel(p)
	if p.pairedN > 0 {
		printConfBlock("Confidence v2", p.confV2, p.pairedN)
	}
	printAgreementPanel(p)
	fmt.Printf("  Récupération v2→v3: %d (melee=%d grenade=%d fire=%d)\n",
		p.recMelee+p.recGrenade+p.recFire, p.recMelee, p.recGrenade, p.recFire)
	printAggPanel(p)
	printVerdict(p)
}

// printSignalPanel imprime le breakdown SourceSignal agrégé.
func printSignalPanel(p *panelAgg) {
	fmt.Print("  SourceSignal v3   : ")
	for i, sig := range signalLevels {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Printf("%s=%d", sig, p.signalV3[sig])
	}
	fmt.Println()
}

// printAgreementPanel imprime l'agreement v2=high agrégé.
func printAgreementPanel(p *panelAgg) {
	if p.v2HighN == 0 {
		fmt.Println("  Agreement v2=high : n/a (0 kill v2 high)")
		return
	}
	flag := "OK (>=98%)"
	if ratio(p.agreeN, p.v2HighN) < agreementThreshold {
		flag = "ÉCHEC (<98%)"
	}
	fmt.Printf("  Agreement v2=high : %d/%d (%s) -> %s\n",
		p.agreeN, p.v2HighN, pct(p.agreeN, p.v2HighN), flag)
}

// printAggPanel imprime la cohérence agrégats §4.7 agrégée.
func printAggPanel(p *panelAgg) {
	if p.aggMatches == 0 {
		fmt.Println("  Cohérence agrégats: n/a (colonnes absentes / 0 melee-grenade)")
		return
	}
	fmt.Printf("  Cohérence agrégats: melee %d/%d (%s) dans ±1 · grenade %d/%d (%s) dans ±1\n",
		p.aggMeleeOK, p.aggMeleeTot, pct(p.aggMeleeOK, p.aggMeleeTot),
		p.aggGrenOK, p.aggGrenTot, pct(p.aggGrenOK, p.aggGrenTot))
}

// printVerdict imprime le verdict §0 : HIGH≥88%, NULL≤5%, agreement≥98%.
func printVerdict(p *panelAgg) {
	highPct := ratio(p.confV3["high"], p.total)
	nullPct := ratio(p.nullV3, p.total)
	agreePct := 1.0
	if p.v2HighN > 0 {
		agreePct = ratio(p.agreeN, p.v2HighN)
	}
	fmt.Println("  --- VERDICT §0 ---")
	fmt.Printf("    HIGH      : %s (cible >=88%%) -> %s\n", pctF(highPct), passStr(highPct >= 0.88))
	fmt.Printf("    NULL      : %s (cible <=5%%)  -> %s\n", pctF(nullPct), passStr(nullPct <= 0.05))
	fmt.Printf("    Agreement : %s (cible >=98%%) -> %s\n", pctF(agreePct), passStr(agreePct >= agreementThreshold))
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers de formatage
// ─────────────────────────────────────────────────────────────────────────────

// pct rend "x/total" en pourcentage formaté (ou "0.0%" si total nul).
func pct(n, total int) string { return pctF(ratio(n, total)) }

// pctF formate un ratio 0..1 en pourcentage.
func pctF(r float64) string { return fmt.Sprintf("%.1f%%", r*100) }

// ratio rend n/total (0 si total nul).
func ratio(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total)
}

// passStr rend PASS/FAIL pour un seuil §0.
func passStr(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}
