package domain

// squad_impact_history.go — LES POINTS D'IMPACT PAR SOIRÉE ET PAR RÔLE, publiés sur
// /pages/teammates (onglet Contributions de l'Escouade, sous la matrice d'impact).
//
// Une soirée = un libellé de session de la composition, tel que l'app découpe ses sessions
// (deux sessions du même jour restent deux soirées). Les rôles sont ceux de la matrice d'impact
// (analysis/squadimpact.RolesOfMatch), le barème est celui du score de la matrice : le calcul vit
// côté Go, l'écran ne recalcule aucun point.

// SquadImpactHistoryMaxEvenings — le nombre de soirées publiées : la soirée affichée et les dix
// soirées précédentes de la composition.
const SquadImpactHistoryMaxEvenings = 11

// SquadImpactHistory — le bloc publié.
type SquadImpactHistory struct {
	// Players : les joueurs de l'escouade dans l'ordre de la page (joueur principal, puis les
	// coéquipiers dans l'ordre de la sélection) — l'ordre des barres d'une soirée.
	Players []string `json:"players"`
	// Scale : le barème, dans l'ordre d'empilement : les rôles qui ajoutent des points, du plus
	// fort au plus faible, puis ceux qui en retirent, du plus fort au plus faible.
	Scale []SquadImpactRoleWeight `json:"scale"`
	// Evenings : les soirées, de la plus ancienne à la plus récente (au plus
	// SquadImpactHistoryMaxEvenings).
	Evenings []SquadImpactEvening `json:"evenings"`
}

// SquadImpactRoleWeight — un rôle et ses points par occurrence.
type SquadImpactRoleWeight struct {
	Role   string  `json:"role"`
	Points float64 `json:"points"`
}

// SquadImpactEvening — une soirée de la composition.
type SquadImpactEvening struct {
	// SessionLabel : le libellé de session de la soirée.
	SessionLabel string `json:"session_label"`
	// StartTime : l'heure (UTC, RFC 3339) du premier match de la soirée.
	StartTime string `json:"start_time"`
	// Matches : les matchs de la soirée, avec ou sans rôle attribué.
	Matches int `json:"matches"`
	// Wins : parmi eux, ceux que le joueur principal a gagnés.
	Wins int `json:"wins"`
	// Players : une entrée par joueur de Players, dans le même ordre.
	Players []SquadImpactEveningPlayer `json:"players"`
}

// SquadImpactEveningPlayer — les rôles d'un joueur sur une soirée.
type SquadImpactEveningPlayer struct {
	Player string `json:"player"`
	// Roles : les rôles obtenus au moins une fois, dans l'ordre du barème (Scale).
	Roles []SquadImpactRoleCount `json:"roles"`
	// Points : le net de la soirée, somme des Points de Roles.
	Points float64 `json:"points"`
}

// SquadImpactRoleCount — un rôle obtenu sur une soirée : son nombre d'occurrences et les points
// qu'elles valent (nombre × barème).
type SquadImpactRoleCount struct {
	Role   string  `json:"role"`
	Count  int     `json:"count"`
	Points float64 `json:"points"`
}
