package domain

// MatchReplayContext est ce que la page de rejeu lit de la base pour un match.
// Route : GET /api/v1/players/{slug}/matches/{match_id}/replay/context
//
// Quatre morceaux, chacun du TYPE EXACT de son homologue dans MatchViewResponse (Header,
// TeamTab.Scoreboard, CombatTab.HighlightEvents, MediaTab) et construits par les MÊMES
// builders : la parité avec la vue complète est structurelle, et prouvée par
// TestGetReplayContext_PariteAvecLaVueComplete (service). Les lignes du tableau de score n'y
// portent pas les champs que seuls les chargements lourds de la vue complète alimentent
// (attendus, extras des amis suivis) : le rejeu ne les lit pas.
type MatchReplayContext struct {
	Header          MatchViewHeader       `json:"header"`
	Scoreboard      []MatchScoreboardRow  `json:"scoreboard"`
	HighlightEvents []MatchHighlightEvent `json:"highlight_events"`
	Media           MatchMediaTab         `json:"media"`
	// SourceUnavailable : au moins une lecture a échoué pendant la construction. Réponse
	// transitoire (D-9), comme une vue complète construite avec une source en panne : le
	// handler pose `Cache-Control: no-store`, le client réessaie toutes les 30 s.
	SourceUnavailable bool `json:"source_unavailable" doc:"Une lecture a échoué côté serveur : contexte transitoire, à redemander (jamais conservé en cache)."`
}
