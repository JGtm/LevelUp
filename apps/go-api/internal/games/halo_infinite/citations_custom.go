// Package halo_infinite â€” citations_custom.go : 12 fonctions custom du moteur citations.
//
// Port Go de src/analysis/citations/custom_rules.py.
// 0 accÃ¨s DB â€” toutes les donnÃ©es arrivent via domain.CitationContext.
package halo_infinite

import (
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// init enregistre DispatchCustom auprès du moteur citations canonical
// (P5.4) — évite l'import cycle analysis → halo_infinite.
func init() {
	analysis.RegisterCustomDispatcher(DispatchCustom)
}

// DispatchCustom route vers la bonne fonction par nom de custom_function.
// P5.4 (ADR 0012) : public car appelé depuis `analysis/citations_engine.go`
// (pont vers les règles citations Halo-only).
func DispatchCustom(fnName string, ctx domain.CitationContext) int {
	switch fnName {
	case "compute_bulldozer":
		return computeBulldozer(ctx)
	case "compute_wins_ctf":
		return computeWinsCTF(ctx)
	case "compute_wins_firefight":
		return computeWinsFirefight(ctx)
	case "compute_wins_slayer":
		return computeWinsSlayer(ctx)
	case "compute_wins_strongholds":
		return computeWinsStrongholds(ctx)
	case "compute_annexion_forcee":
		return computeAnnexionForcee(ctx)
	case "compute_flag_em_down":
		return computeFlagEmDown(ctx)
	case "compute_hijack":
		return computeHijack(ctx)
	case "compute_vandalism":
		return computeVandalism(ctx)
	case "compute_wraith_destroyer":
		return computeWraithDestroyer(ctx)
	case "compute_mongoose_destroyer":
		return computeMongooseDestroyer(ctx)
	case "compute_warthog_destroyer":
		return computeWarthogDestroyer(ctx)
	}
	return 0
}

// computeBulldozer : KDA > 8 en Slayer/Assassin hors firefight/BTB.
func computeBulldozer(ctx domain.CitationContext) int {
	if ctx.IsFirefight {
		return 0
	}
	kda := ctx.Stats["kda"]
	if kda <= 8 {
		return 0
	}
	pl := ctx.Playlist
	gv := ctx.GameVariant
	if strings.Contains(pl, "btb") || strings.Contains(pl, "big team") {
		return 0
	}
	if strings.Contains(pl, "slayer") || strings.Contains(pl, "assassin") ||
		strings.Contains(gv, "slayer") || strings.Contains(gv, "assassin") {
		return 1
	}
	return 0
}

// computeWinsCTF : victoire en CTF.
func computeWinsCTF(ctx domain.CitationContext) int {
	if ctx.Outcome != domain.OutcomeWin {
		return 0
	}
	pl := ctx.Playlist
	gv := ctx.GameVariant
	if strings.Contains(pl, "ctf") || strings.Contains(pl, "capture") ||
		strings.Contains(pl, "drapeau") || strings.Contains(pl, "neutral flag") ||
		strings.Contains(gv, "ctf") || strings.Contains(gv, "capture") ||
		strings.Contains(gv, "flag") {
		return 1
	}
	return 0
}

// computeWinsFirefight : victoire en Firefight.
func computeWinsFirefight(ctx domain.CitationContext) int {
	if ctx.Outcome != domain.OutcomeWin {
		return 0
	}
	pl := ctx.Playlist
	gv := ctx.GameVariant
	if ctx.IsFirefight ||
		strings.Contains(pl, "firefight") || strings.Contains(pl, "baptême") ||
		strings.Contains(pl, "bapteme") || strings.Contains(gv, "firefight") {
		return 1
	}
	return 0
}

// computeWinsSlayer : victoire en Slayer/Assassin.
func computeWinsSlayer(ctx domain.CitationContext) int {
	if ctx.Outcome != domain.OutcomeWin {
		return 0
	}
	pl := ctx.Playlist
	gv := ctx.GameVariant
	if strings.Contains(pl, "slayer") || strings.Contains(pl, "assassin") ||
		strings.Contains(gv, "slayer") || strings.Contains(gv, "assassin") {
		return 1
	}
	return 0
}

// computeWinsStrongholds : victoire en Strongholds/Bases.
func computeWinsStrongholds(ctx domain.CitationContext) int {
	if ctx.Outcome != domain.OutcomeWin {
		return 0
	}
	pl := ctx.Playlist
	gv := ctx.GameVariant
	if strings.Contains(pl, "stronghold") || strings.Contains(pl, "bases") ||
		strings.Contains(gv, "stronghold") || strings.Contains(gv, "bases") {
		return 1
	}
	return 0
}

// computeAnnexionForcee : sÃ©ries de 3 captures de zone sans mort entre elles.
// Utilise highlight_events si disponibles, sinon fallback sur awards["zone_captured"] / 3.
func computeAnnexionForcee(ctx domain.CitationContext) int {
	if len(ctx.Events) > 0 {
		streak := 0
		total := 0
		for _, e := range ctx.Events {
			switch e.EventType {
			case analysis.EventTypeMode:
				streak++
				if streak%3 == 0 {
					total++
				}
			case analysis.EventTypeDeath:
				if e.XUID == ctx.PlayerXUID {
					streak = 0
				}
			}
		}
		return total
	}
	return ctx.Awards["zone_captured"] / 3
}

// computeFlagEmDown : arrÃªts de porteur de drapeau.
func computeFlagEmDown(ctx domain.CitationContext) int {
	return sumAwardsExact(ctx.Awards,
		"runner_stopped", "Porteur arrÃªtÃ©", "Flag Carrier Kill", "Flag Carrier Killed",
	)
}

// computeHijack : piratage de vÃ©hicule/tourelle.
func computeHijack(ctx domain.CitationContext) int {
	total := sumAwardsWithPrefix(ctx.Awards, "hijacked_")
	total += sumAwardsContaining(ctx.Awards, "hijack")
	total += sumAwardsContaining(ctx.Awards, "skyjack")
	return total
}

// computeVandalism : destruction d'objectifs ennemis.
func computeVandalism(ctx domain.CitationContext) int {
	total := sumAwardsWithPrefix(ctx.Awards, "destroyed_")
	total += sumAwardsContaining(ctx.Awards, "destroyed")
	total += sumAwardsContaining(ctx.Awards, "destruction")
	return total
}

// computeWraithDestroyer : destruction de Wraith.
func computeWraithDestroyer(ctx domain.CitationContext) int {
	return sumAwardsExact(ctx.Awards,
		"destroyed_wraith", "Wraith Destroyed", "Wraith destroyed",
	)
}

// computeMongooseDestroyer : destruction de Mongoose.
func computeMongooseDestroyer(ctx domain.CitationContext) int {
	return sumAwardsExact(ctx.Awards,
		"destroyed_mongoose", "Mongoose Destroyed", "Mongoose destroyed",
	)
}

// computeWarthogDestroyer : destruction de Warthog (standard + roquettes).
func computeWarthogDestroyer(ctx domain.CitationContext) int {
	return sumAwardsExact(ctx.Awards,
		"destroyed_warthog", "destroyed_rocket_warthog",
		"Warthog Destroyed", "Warthog destroyed",
		"Rocket Warthog Destroyed",
	)
}

// --- helpers awards ---

func sumAwardsExact(awards map[string]int, names ...string) int {
	total := 0
	for _, n := range names {
		total += awards[n]
	}
	return total
}

func sumAwardsWithPrefix(awards map[string]int, prefix string) int {
	total := 0
	for name, count := range awards {
		if strings.HasPrefix(name, prefix) {
			total += count
		}
	}
	return total
}

func sumAwardsContaining(awards map[string]int, substr string) int {
	total := 0
	for name, count := range awards {
		if strings.Contains(name, substr) {
			total += count
		}
	}
	return total
}
