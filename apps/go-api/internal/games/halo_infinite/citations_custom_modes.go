package halo_infinite

import (
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// citations_custom_modes.go — les PRÉDICATS DE MODE des citations custom, et les citations qui
// ne lisent que le mode et l'issue. Une seule définition par mode : le garde-rail
// `TestCitationsCustom_ModesParPredicat` interdit de retester la playlist ou la variante en
// dehors de ce fichier.
//
// Playlist et variante arrivent normalisées en minuscules (domain.CitationContext).

// containsAny dit si s contient l'un des motifs.
func containsAny(s string, motifs ...string) bool {
	for _, m := range motifs {
		if strings.Contains(s, m) {
			return true
		}
	}
	return false
}

func isSlayerMode(ctx domain.CitationContext) bool {
	return containsAny(ctx.Playlist, "slayer", "assassin") ||
		containsAny(ctx.GameVariant, "slayer", "assassin")
}

func isBigTeam(ctx domain.CitationContext) bool {
	return containsAny(ctx.Playlist, "btb", "big team")
}

func isCTFMode(ctx domain.CitationContext) bool {
	return containsAny(ctx.Playlist, "ctf", "capture", "drapeau", "neutral flag") ||
		containsAny(ctx.GameVariant, "ctf", "capture", "flag")
}

func isFirefightMode(ctx domain.CitationContext) bool {
	return ctx.IsFirefight ||
		containsAny(ctx.Playlist, "firefight", "baptême", "bapteme") ||
		containsAny(ctx.GameVariant, "firefight")
}

func isStrongholdsMode(ctx domain.CitationContext) bool {
	return containsAny(ctx.Playlist, "stronghold", "bases") ||
		containsAny(ctx.GameVariant, "stronghold", "bases")
}

// isAssaultMode lit la VARIANTE seule : « Assault » est aussi un conteneur de playlist qui
// regroupe d'autres modes (skill halo-modes). Variantes du registre : « Assault:One Bomb »,
// « Assault:Neutral Bomb », « Assault:Neutral Bomb Squad », « Husky Raid:Assault ».
func isAssaultMode(ctx domain.CitationContext) bool {
	return containsAny(ctx.GameVariant, "assault", "bomb")
}

// winIn vaut 1 pour une victoire dans un mode que `mode` reconnaît.
func winIn(ctx domain.CitationContext, mode func(domain.CitationContext) bool) int {
	if ctx.Outcome == domain.OutcomeWin && mode(ctx) {
		return 1
	}
	return 0
}

// computeWinsAssault : victoire en Assaut (« Tick Tick Boom »).
func computeWinsAssault(ctx domain.CitationContext) int {
	return winIn(ctx, isAssaultMode)
}

// computeFirstStrike : 1 si le joueur a fait le premier frag du match (« Premier coup »). La
// règle est celle du badge « Premier sang » de l'impact du match (analysis.FirstBlood), sur les
// mêmes événements highlight : un frag de bot ouvre le match sans créditer personne.
func computeFirstStrike(ctx domain.CitationContext) int {
	events := make([]analysis.ImpactEvent, 0, len(ctx.Events))
	for _, e := range ctx.Events {
		events = append(events, analysis.ImpactEvent{TimeMS: e.TimeMS, EventType: e.EventType, ActorXUID: e.XUID})
	}
	if fb := analysis.FirstBlood(events); fb != nil && fb.ActorXUID != "" && fb.ActorXUID == ctx.PlayerXUID {
		return 1
	}
	return 0
}
