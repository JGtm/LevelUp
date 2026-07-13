// Package analysis — indicators.go : helpers canoniques pour les indicateurs
// produit (CombatEfficiency, KDR, WinRate, Accuracy, KillsPerGame,
// DeathsPerGame, PerfTier).
//
// ADR 0006 (canonical-indicators-and-units) acte les formules et l'unité
// canonique côté API : toutes les fonctions retournent des ratios 0..1
// (sauf CombatEfficiency/KDR qui sont des ratios non bornés). Le formatage
// `*100` + arrondi décimal se fait UNIQUEMENT à l'affichage côté front via
// `formatPercent(ratio, decimals)`.
//
// Source de vérité côté front : apps/web/src/lib/accessibility/scales/instances.ts
// (tests Vitest table-driven sur perfScale).
package analysis

// CombatEfficiency retourne (kills + assists) / max(1, deaths) — une efficacité
// de combat (division par les morts, toujours >= 0).
//
// ATTENTION : ce N'EST PAS le KDA affiché. Le KDA per-match est un NET
// ((Kills + Assists/3) − Deaths, possiblement négatif) lu tel quel depuis l'API
// (Infinite) ou le FDA natif (Halo 5) — jamais un quotient par les morts. Cette
// fonction est une métrique INTERNE au perf score uniquement ; ne pas l'utiliser
// pour produire ou afficher un KDA. Renvoie 0 si k+a == 0.
func CombatEfficiency(kills, assists, deaths int) float64 {
	d := deaths
	if d < 1 {
		d = 1
	}
	return float64(kills+assists) / float64(d)
}

// AggregateKDA retourne le KDA AGRÉGAT canonique (ADR 0006) sur un ensemble de
// matchs : ((frags + assists/3) − morts) / nb_matchs. Ce N'EST JAMAIS un
// quotient par les morts (cf. CombatEfficiency) : c'est un NET moyen par match,
// possiblement négatif. Renvoie 0 si matches == 0.
//
// À utiliser partout où un KDA agrégé est nécessaire plutôt que de réinliner la
// formule (elle existe déjà inlinée dans explorer_target_stats.go — dette notée).
func AggregateKDA(kills, assists, deaths, matches int) float64 {
	if matches <= 0 {
		return 0
	}
	return (float64(kills) + float64(assists)/3.0 - float64(deaths)) / float64(matches)
}

// KDR retourne le K/D ratio canonique kills / max(1, deaths). Distinct du
// KDA — exclut les assists. Halo expose les deux indicateurs en parallèle.
func KDR(kills, deaths int) float64 {
	d := deaths
	if d < 1 {
		d = 1
	}
	return float64(kills) / float64(d)
}

// WinRate retourne wins / total en ratio 0..1. Renvoie 0 si total == 0.
// Convention API canonique (jamais de pourcent côté API).
func WinRate(wins, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(wins) / float64(total)
}

// Accuracy retourne hits / fired en ratio 0..1. Renvoie 0 si fired == 0.
// Convention API canonique (jamais de pourcent côté API).
func Accuracy(hits, fired int) float64 {
	if fired <= 0 {
		return 0
	}
	return float64(hits) / float64(fired)
}

// KillsPerGame retourne kills / matches. Renvoie 0 si matches == 0.
func KillsPerGame(kills, matches int) float64 {
	if matches <= 0 {
		return 0
	}
	return float64(kills) / float64(matches)
}

// DeathsPerGame retourne deaths / matches. Renvoie 0 si matches == 0.
func DeathsPerGame(deaths, matches int) float64 {
	if matches <= 0 {
		return 0
	}
	return float64(deaths) / float64(matches)
}

// Tier représente le palier de performance (1 = meilleur, 5 = plus faible).
// 5 paliers canoniques alignés sur perfScale côté front.
type Tier int

// Constantes des 5 paliers de performance (ADR 0006).
const (
	TierExcellent Tier = 1 // score >= 80
	TierBon       Tier = 2 // score >= 65
	TierCorrect   Tier = 3 // score >= 50
	TierFaible    Tier = 4 // score >= 35
	TierMauvais   Tier = 5 // score <  35
)

// Token retourne le token sémantique correspondant au palier ("perf-tier-1"
// .. "perf-tier-5"). Aligné sur les tokens CSS définis côté front
// (apps/web/src/lib/accessibility/semantic-tokens.ts).
func (t Tier) Token() string {
	switch t {
	case TierExcellent:
		return "perf-tier-1"
	case TierBon:
		return "perf-tier-2"
	case TierCorrect:
		return "perf-tier-3"
	case TierFaible:
		return "perf-tier-4"
	default:
		return "perf-tier-5"
	}
}

// PerfTier retourne le palier de performance pour un score 0..100 selon les
// seuils canoniques [80, 65, 50, 35]. Source de vérité testée côté front
// (apps/web/src/lib/accessibility/scales/instances.ts:20 + instances.test.ts).
//
// Ne pas utiliser de seuils alternatifs (80/60/40 fait passer un score 62
// d'un palier à l'autre selon la surface — bug visuel cf. revue 2026-04-29
// axe 6 BLOQUANT).
func PerfTier(score float64) Tier {
	switch {
	case score >= 80:
		return TierExcellent
	case score >= 65:
		return TierBon
	case score >= 50:
		return TierCorrect
	case score >= 35:
		return TierFaible
	default:
		return TierMauvais
	}
}
