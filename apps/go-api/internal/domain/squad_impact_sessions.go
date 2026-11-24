package domain

// squad_impact_sessions.go — L'IMPACT DES COÉQUIPIERS PAR SESSION, publié sur /pages/teammates
// (onglet Contributions de l'Escouade) : un tableau d'une ligne par session, chaque ligne
// dépliable sur la matrice de ses matchs, plus une ligne « Toutes les sessions ».
//
// Une session = un libellé de session de la composition, tel que l'app découpe ses sessions
// (deux sessions du même jour restent deux sessions). Les rôles sont ceux de
// analysis/squadimpact.RolesOfMatch et le barème celui de squadimpact.Weight : le calcul vit
// côté Go, l'écran ne recalcule aucun point. Un tableau ou une liste vide se sert `[]`, jamais
// `null`.

// SquadImpactSessionsMax — le nombre de sessions publiées : la session affichée et les dix
// sessions précédentes de la composition.
const SquadImpactSessionsMax = 11

// SquadImpactSessions — le bloc publié.
type SquadImpactSessions struct {
	// Players : les joueurs de l'escouade dans l'ordre de la page (joueur principal, puis les
	// coéquipiers dans l'ordre de la sélection). Chaque session porte une entrée par joueur, dans
	// cet ordre, et Totals aussi.
	Players []string `json:"players"`
	// Scale : le barème, dans l'ordre d'empilement (les rôles qui ajoutent des points, du plus
	// fort au plus faible, puis ceux qui en retirent, du plus fort au plus faible). Le front en
	// tire le sens d'un rôle (Points < 0 : un compte plus élevé est pire) pour le surlignage du
	// meilleur et du pire ; il ne le recalcule pas. Les rôles hors barème n'y figurent pas.
	Scale []SquadImpactSessionsRoleWeight `json:"scale"`
	// RoleOrder : l'ordre canonique des colonnes de rôles (analysis/squadimpact.RoleOrder). Les
	// listes Counts de Sessions et de Totals suivent cet ordre et portent une entrée pour chaque
	// rôle, comptes nuls compris.
	RoleOrder []string `json:"role_order"`
	// Sessions : les sessions, de la plus récente à la plus ancienne (au plus
	// SquadImpactSessionsMax : la population affichée, puis les sessions antérieures de la
	// composition).
	Sessions []SquadImpactSessionsSession `json:"sessions"`
	// Totals : une entrée par joueur de Players, dans le même ordre, sommée sur les seules
	// Sessions publiées.
	Totals []SquadImpactSessionsTotal `json:"totals"`
}

// SquadImpactSessionsRoleWeight — un rôle et ses points par occurrence.
type SquadImpactSessionsRoleWeight struct {
	Role   string  `json:"role"`
	Points float64 `json:"points"`
}

// SquadImpactSessionsSession — une session de la composition.
type SquadImpactSessionsSession struct {
	// SessionLabel : le libellé de session (suffixe de compte « (N) » compris, tel que servi par
	// l'historique des matchs).
	SessionLabel string `json:"session_label"`
	// StartTime : l'heure (UTC, RFC 3339) du premier match de la session.
	StartTime string `json:"start_time"`
	// Matches : TOUS les matchs de la session, avec ou sans rôle attribué, du plus ancien au
	// plus récent.
	Matches []SquadImpactSessionsMatch `json:"matches"`
	// Players : une entrée par joueur de Players, dans le même ordre.
	Players []SquadImpactSessionsPlayer `json:"players"`
}

// SquadImpactSessionsMatch — un match d'une session.
type SquadImpactSessionsMatch struct {
	MatchID string `json:"match_id"`
	// Outcome : l'issue du match pour le joueur principal (codes d'issue de la page : 2 victoire,
	// 3 défaite, autre = ni l'un ni l'autre).
	Outcome int `json:"outcome"`
}

// SquadImpactSessionsPlayer — un joueur sur une session.
type SquadImpactSessionsPlayer struct {
	Player string `json:"player"`
	// MatchesPlayed : les matchs de la session où le joueur est dans l'équipe alliée du joueur
	// principal, ou où un rôle lui est attribué (un rôle ne tombe que sur un joueur du match). Le
	// bloc n'est publié que si l'équipe alliée a été lue : sans elle seuls les rôles feraient
	// présence. 0 : le joueur n'a pas joué la session (Points et tous les comptes de Counts valent alors 0).
	MatchesPlayed int `json:"matches_played"`
	// Points : le net de la session, somme du barème sur tous ses rôles. Le net par match que le
	// tableau affiche est Points / MatchesPlayed (calculé par l'écran, MatchesPlayed > 0).
	Points float64 `json:"points"`
	// Counts : le nombre d'occurrences de chaque rôle sur la session, dans l'ordre de RoleOrder,
	// une entrée par rôle de RoleOrder (0 compris).
	Counts []SquadImpactSessionsRoleCount `json:"counts"`
	// PerMatch : les rôles du joueur match par match, une entrée par match où il en a au moins un,
	// dans l'ordre de Matches. Un match de Matches absent de PerMatch : aucun rôle pour ce joueur.
	PerMatch []SquadImpactSessionsPerMatch `json:"per_match"`
}

// SquadImpactSessionsRoleCount — le nombre d'occurrences d'un rôle.
type SquadImpactSessionsRoleCount struct {
	Role  string `json:"role"`
	Count int    `json:"count"`
}

// SquadImpactSessionsPerMatch — les rôles d'un joueur sur un match.
type SquadImpactSessionsPerMatch struct {
	MatchID string `json:"match_id"`
	// Roles : les rôles obtenus sur ce match, dans l'ordre de RoleOrder ; un rôle compté deux
	// fois sur le même match y figure deux fois.
	Roles []string `json:"roles"`
}

// SquadImpactSessionsTotal — un joueur sur l'ensemble des sessions publiées.
type SquadImpactSessionsTotal struct {
	Player string `json:"player"`
	// MatchesPlayed : somme des MatchesPlayed du joueur sur les sessions publiées.
	MatchesPlayed int `json:"matches_played"`
	// Points : somme des Points du joueur sur les sessions publiées.
	Points float64 `json:"points"`
	// Counts : somme des Counts du joueur sur les sessions publiées, même ordre et mêmes
	// entrées que ceux d'une session.
	Counts []SquadImpactSessionsRoleCount `json:"counts"`
}
