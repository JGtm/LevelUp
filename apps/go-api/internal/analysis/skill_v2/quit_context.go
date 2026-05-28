package skill_v2

// quit_context.go — Sprint 2.A : situation de l'équipe d'un quitter AU MOMENT
// où il part, reconstruite depuis la timeline des frags (même logique que le
// graphe "tug of war" de la page match-view).
//
// Aujourd'hui le quit penalty se base sur l'outcome FINAL du match (cf.
// quitDeltaForTeam). Ici on raffine : si l'équipe du quitter MENAIT (ou était à
// égalité) au moment du quit, c'est un abandon d'une situation non-perdante →
// pénalité forte ("unrelated"). Si elle PERDAIT déjà, l'abandon est moins
// déterminant → pénalité modérée ("related").
//
// Fiabilité (validée produit) : 100% sur les modes non-objectifs (frags = score),
// signal suffisamment fort pour être accepté sur les modes à objectifs.
//
// Pur (0 accès DB). Le caller fournit les frags déjà attribués par équipe et le
// moment du quit dans le MÊME repère temporel que les frags (cf. note T0 / hook
// adapter côté sync, internal/sync/skill_v2_quit_penalty.go).

// QuitContext décrit la situation de l'équipe du quitter à l'instant du quit.
type QuitContext int

const (
	// QuitWhileTrailing : l'équipe perdait au moment du quit → "related".
	QuitWhileTrailing QuitContext = iota
	// QuitWhileTied : égalité au moment du quit → "unrelated" (situation non perdante).
	QuitWhileTied
	// QuitWhileLeading : l'équipe menait au moment du quit → "unrelated" (abandon net).
	QuitWhileLeading
)

// TeamFrag : un frag horodaté attribué à une équipe. TimeMs est exprimé dans le
// même repère que le quitMs passé à InferQuitContext (typiquement ms depuis le
// début du film).
type TeamFrag struct {
	TimeMs int64
	TeamID int
}

// InferQuitContext retourne la situation de l'équipe `quitterTeamID` à l'instant
// `quitMs`, en comptant les frags cumulés de chaque équipe AVANT (≤) ce moment.
//
//	frags de l'équipe du quitter > adverse → QuitWhileLeading
//	<                                        → QuitWhileTrailing
//	=                                        → QuitWhileTied
//
// Aucun frag avant le quit (0-0) → QuitWhileTied (début de match : situation non
// perdante). Le caller décide d'utiliser ce contexte ou de retomber sur
// l'outcome final quand la timeline est indisponible.
func InferQuitContext(frags []TeamFrag, quitMs int64, quitterTeamID int) QuitContext {
	var mine, theirs int
	for _, f := range frags {
		if f.TimeMs > quitMs {
			continue
		}
		if f.TeamID == quitterTeamID {
			mine++
		} else {
			theirs++
		}
	}
	switch {
	case mine > theirs:
		return QuitWhileLeading
	case mine < theirs:
		return QuitWhileTrailing
	default:
		return QuitWhileTied
	}
}
