package domain

// match_emprise.go — LES BLOCS DE L'EMPRISE SUR LA VUE MATCH (plan
// `.ai/PLAN_MATCHVIEW_EMPRISE_2026-10-06.md`) : un match, en comptes exhaustifs, les joueurs de
// l'équipe du joueur de la page connus un par un.

// MatchViewEmpriseFields — embarqué dans MatchViewResponse : les champs s'aplatissent dans le JSON
// de la réponse.
type MatchViewEmpriseFields struct {
	// Emprise : « Contrôle des ressources, par match », « Prises par joueur », « Frags par
	// ressource », « Rendement par ressource ». Nil sans joueur de la page connu au tableau des scores.
	Emprise *MatchEmpriseBlock `json:"emprise,omitempty"`
	// LivesNearTeammate : « Isolement, par joueur ». Nil sans `film.kill_positions`, sur lecture en
	// échec ou sans aucune vie lue.
	LivesNearTeammate *MatchLivesNearTeammate `json:"lives_near_teammate,omitempty"`
}

// MatchEmpriseBlock — l'Emprise d'UN match (`Matches` a un seul élément), fiches = les joueurs de
// l'équipe présents à la fin (le joueur de la page d'abord, puis les profils suivis, puis le reste
// dans l'ordre du tableau des scores ; bots et partis comptés dans le reste de l'équipe).
type MatchEmpriseBlock struct {
	SquadEmpriseBlock
	// KillJournalPublishable : le journal des morts du match se lit ligne à ligne. Faux : les frags
	// pendant l'effet d'un bonus ne sont pas une mesure (le web le dit au lieu d'écrire zéro).
	KillJournalPublishable bool `json:"kill_journal_publishable"`
}

// MatchLivesNearTeammate — « Isolement, par joueur » : les vies de chaque joueur de l'équipe, dans
// l'ordre des fiches de l'Emprise. Les vies d'un match au journal non publiable sont écartées et
// comptées (ExcludedUnpublishable), jamais rangées avec zéro frag.
type MatchLivesNearTeammate struct {
	Players []MatchLivesPlayer `json:"players"`
}

// MatchLivesPlayer — le bilan des vies d'un joueur (à zéro s'il n'a aucune vie lue).
type MatchLivesPlayer struct {
	XUID string `json:"xuid"`
	TimeseriesLivesNearTeammate
}
