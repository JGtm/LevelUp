// Package relations — rivalry.go : algorithmes purs des cartes « Revanche »
// (Phase 3a Moments & Rivalités). 0 DB, 0 HTTP. Entrée : séquence de duels
// ordonnée ancien→récent (un par match commun en ennemi). Sortie : frise
// d'outcomes, taux de victoire glissant, KPIs (récent vs global, série en
// cours, écart de frags cumulé).
package relations

// RollingWinRateWindow : fenêtre du taux de victoire glissant contre un rival
// (constante nommée, pas de magic number). 5 derniers duels.
const RollingWinRateWindow = 5

// DuelOutcome : résultat d'un duel (match commun en ennemi), du point de vue du
// joueur courant. Aligné sur la sémantique canonical (win / loss / autre).
type DuelOutcome int

const (
	// DuelOther : ni victoire ni défaite (égalité, DNF, ou inconnu).
	DuelOther DuelOutcome = iota
	// DuelWin : le joueur a gagné ce match.
	DuelWin
	// DuelLoss : le joueur a perdu ce match.
	DuelLoss
)

// Duel : un match commun joué EN ENNEMI contre un rival, ordonné ancien→récent.
// Outcome est dérivé en amont (service) via canonical (pas de 2/3 en dur ici).
// KillsOnRival / DeathsByRival : frags directionnels moi↔lui sur ce match.
type Duel struct {
	Outcome       DuelOutcome
	KillsOnRival  int
	DeathsByRival int
}

// RivalryMetrics : résultat agrégé d'une carte revanche.
type RivalryMetrics struct {
	// RollingWinRate : taux de victoire glissant (fenêtre RollingWinRateWindow),
	// aligné sur la séquence des duels (un point par duel, ancien→récent). Chaque
	// point = wins / decisifs sur la fenêtre se terminant à ce duel. nil ⇒ aucun
	// duel décisif dans la fenêtre (égalités/DNF seulement).
	RollingWinRate []*float64
	// RecentWinRate : taux de victoire sur les RollingWinRateWindow derniers
	// duels décisifs. nil si aucun.
	RecentWinRate *float64
	// GlobalWinRate : taux de victoire sur tous les duels décisifs. nil si aucun.
	GlobalWinRate *float64
	// CurrentStreak : série en cours (signe = sens : >0 victoires, <0 défaites,
	// 0 si dernier duel non décisif ou aucun duel). |valeur| = longueur.
	CurrentStreak int
	// FragGap : écart de frags cumulé (kills cumulés sur lui − morts cumulées
	// par lui), sur tous les duels. >0 = domination, <0 = dominé.
	FragGap int
	// DecisiveCount : nombre de duels décisifs (win+loss), pour exposer la base.
	DecisiveCount int
}

// ComputeRivalryMetrics calcule les indicateurs d'une carte revanche à partir
// de la séquence des duels (ordonnée ancien→récent). Pur, déterministe.
func ComputeRivalryMetrics(duels []Duel) RivalryMetrics {
	m := RivalryMetrics{
		RollingWinRate: make([]*float64, len(duels)),
	}
	wins, decisive := 0, 0
	for i := range duels {
		m.FragGap += duels[i].KillsOnRival - duels[i].DeathsByRival
		if duels[i].Outcome == DuelWin {
			wins++
		}
		if duels[i].Outcome == DuelWin || duels[i].Outcome == DuelLoss {
			decisive++
		}
		m.RollingWinRate[i] = windowWinRate(duels, i, RollingWinRateWindow)
	}
	m.DecisiveCount = decisive
	m.GlobalWinRate = ratioOrNil(wins, decisive)
	m.RecentWinRate = windowWinRate(duels, len(duels)-1, RollingWinRateWindow)
	m.CurrentStreak = currentStreak(duels)
	return m
}

// windowWinRate : taux de victoire sur la fenêtre de `window` duels se terminant
// à l'index `end` (inclus). nil si aucun duel décisif dans la fenêtre. end<0 ⇒ nil.
func windowWinRate(duels []Duel, end, window int) *float64 {
	if end < 0 || len(duels) == 0 {
		return nil
	}
	start := end - window + 1
	if start < 0 {
		start = 0
	}
	wins, decisive := 0, 0
	for i := start; i <= end; i++ {
		switch duels[i].Outcome {
		case DuelWin:
			wins++
			decisive++
		case DuelLoss:
			decisive++
		}
	}
	return ratioOrNil(wins, decisive)
}

// currentStreak : longueur signée de la série en cours, en partant du duel le
// plus récent. >0 victoires consécutives, <0 défaites consécutives, 0 si le
// dernier duel est non décisif ou s'il n'y a aucun duel.
func currentStreak(duels []Duel) int {
	if len(duels) == 0 {
		return 0
	}
	last := duels[len(duels)-1].Outcome
	if last != DuelWin && last != DuelLoss {
		return 0
	}
	streak := 0
	for i := len(duels) - 1; i >= 0; i-- {
		if duels[i].Outcome != last {
			break
		}
		streak++
	}
	if last == DuelLoss {
		return -streak
	}
	return streak
}

// ratioOrNil : wins/total en 0..1, nil si total==0.
func ratioOrNil(wins, total int) *float64 {
	if total == 0 {
		return nil
	}
	r := float64(wins) / float64(total)
	return &r
}

// Codes du résultat décidé en amont (SQL title-aware via outcomeSQLEq, pas de
// 2/3 en dur). 1 = victoire, 2 = défaite, tout le reste = non décisif.
const (
	ResultWin  = 1
	ResultLoss = 2
)

// ResultToDuel mappe un code de résultat décidé (ResultWin/ResultLoss, résolu
// title-aware en amont) vers DuelOutcome. Aucun code Halo brut ici.
func ResultToDuel(result int) DuelOutcome {
	switch result {
	case ResultWin:
		return DuelWin
	case ResultLoss:
		return DuelLoss
	default:
		return DuelOther
	}
}
