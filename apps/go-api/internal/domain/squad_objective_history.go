package domain

// squad_objective_history.go — LE RAPPORT DE FORCE À L'OBJECTIF, SOIRÉE APRÈS SOIRÉE
// (carte « Rapport de force, soirée après soirée » de l'onglet Contributions de l'Escouade,
// lot L3 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, décisions D6 et D7).
//
// LA MESURE : pour chacun des trois rôles de l'objectif (prendre, défendre, tenir — la
// partition de narrative/objective_roles.go), la part de notre camp dans ce que le lobby a
// fait, match par match ; la part d'une soirée est la MOYENNE de ces parts, chaque match
// pesant pareil (D7). Une part n'a pas d'unité : une soirée Bases + Drapeau se compare à une
// soirée tout Drapeau.
//
// LES SOIRÉES RETENUES (D6) : celles de la composition qui comptent au moins
// SquadObjectiveMinMatches matchs à objectif, tous modes confondus, drapeau neutre exclu. Les
// dix plus récentes AVANT la soirée affichée, puis la soirée affichée elle-même.

// SquadObjectiveMinMatches — le minimum de matchs à objectif d'une soirée pour qu'elle ait un
// point (D6). Sous ce seuil, la part d'une soirée dépend d'un ou deux matchs.
const SquadObjectiveMinMatches = 3

// SquadObjectiveHistoryMaxPrevious — le nombre de soirées précédentes publiées (D6).
const SquadObjectiveHistoryMaxPrevious = 10

// SquadObjectiveHistory — le bloc publié sur /pages/teammates.
type SquadObjectiveHistory struct {
	// Current : la soirée affichée (le périmètre D2 de la page). Toujours présente ; son
	// ObjectiveMatches dit si elle a un point (≥ SquadObjectiveMinMatches).
	Current SquadObjectiveEvening `json:"current"`
	// Previous : les soirées précédentes retenues, de la plus ancienne à la plus récente
	// (au plus SquadObjectiveHistoryMaxPrevious). Vide = aucun historique comparable.
	Previous []SquadObjectiveEvening `json:"previous"`
	// EveningsWithObjective : le nombre de soirées de la composition qui ont au moins un
	// match à objectif (drapeau neutre exclu), toutes périodes confondues.
	EveningsWithObjective int `json:"evenings_with_objective"`
	// EveningsBelowMinimum : parmi elles, celles qui en ont moins de SquadObjectiveMinMatches.
	EveningsBelowMinimum int `json:"evenings_below_minimum"`
	// MinObjectiveMatches : le seuil appliqué (SquadObjectiveMinMatches), publié pour que
	// l'écran le dise sans le recopier.
	MinObjectiveMatches int `json:"min_objective_matches"`
}

// SquadObjectiveEvening — une soirée de la composition.
type SquadObjectiveEvening struct {
	// SessionLabel : le libellé de session de la soirée (vide pour la soirée affichée quand
	// le périmètre couvre plusieurs sessions).
	SessionLabel string `json:"session_label"`
	// StartTime : l'heure (UTC, RFC 3339) du premier match à objectif de la soirée — l'écran
	// en tire la date. Vide si la soirée n'a aucun match à objectif.
	StartTime string `json:"start_time"`
	// ObjectiveMatches : ses matchs à objectif (drapeau neutre exclu).
	ObjectiveMatches int `json:"objective_matches"`
	// Wins : parmi eux, ceux que notre camp a gagnés.
	Wins int `json:"wins"`
	// Families : le mélange de modes, une entrée par famille (clé stable de
	// narrative.ObjectiveFamily), dans l'ordre décroissant du nombre de matchs.
	Families []SquadObjectiveFamilyCount `json:"families"`
	// Take / Defend / Hold : la part moyenne (0..1, ADR 0006) de notre camp dans le lobby pour
	// chaque rôle. Nil = aucun match de la soirée ne mesure ce rôle (lobby à zéro).
	Take   *float64 `json:"take,omitempty"`
	Defend *float64 `json:"defend,omitempty"`
	Hold   *float64 `json:"hold,omitempty"`
}

// SquadObjectiveFamilyCount — une famille de mode et son nombre de matchs dans la soirée.
type SquadObjectiveFamilyCount struct {
	Family  string `json:"family"`
	Matches int    `json:"matches"`
}
