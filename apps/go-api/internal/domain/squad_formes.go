package domain

// squad_formes.go — LE BLOC « FORMES RETENUES », réduit à ce que lisent les cartes d'objectif
// (Escouade › Contributions, Séries temporelles › Usages ; plan
// PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05, décision D7).
//
// # CE QUE CE BLOC PUBLIE, ET POURQUOI IL EST « PLAT »
//
// Les cartes d'objectif emploient plusieurs dénominateurs (mon camp, le lobby, l'escouade) sur la
// MÊME matière : ce que chaque joueur des DEUX camps a fait à l'objectif, match par match. Le bloc
// publie donc LA MATIÈRE — une feuille d'objectif par match — et les parts se calculent là où
// elles s'affichent, dans des modèles purs testés côté web.
//
// # CE QU'IL NE PORTE PAS
//
//   - Aucun libellé FR/EN : les clés de colonne d'objectif et de famille se traduisent côté web.
//   - Aucun match sans objectif : un mode sans objectif n'a rien à y dire, et les comptes
//     (MatchesTotal, MatchesMeasured) disent la portée entière.

// SquadFormesBlock — la matière des cartes d'objectif, sur le scope filtré de la page qui le
// publie.
type SquadFormesBlock struct {
	// Available : le titre porte la capability de mesure et les lectures ont
	// abouti. Faux ⇒ UnavailableReason dit laquelle des deux a manqué
	// (constantes SessionUsage* — le même vocabulaire que le bloc d'usage).
	Available         bool   `json:"available"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
	// MatchesTotal : les matchs du scope. MatchesMeasured : ceux qui portent un
	// film décodé. Le couple s'affiche tel quel (« 8 sur 9 »).
	MatchesTotal    int `json:"matches_total"`
	MatchesMeasured int `json:"matches_measured"`
	// MainXUID : le joueur de la page. Squad : lui EN TÊTE, puis les
	// coéquipiers sélectionnés, dans l'ordre d'affichage.
	MainXUID string                    `json:"main_xuid,omitempty"`
	Squad    []SessionUsageSquadPlayer `json:"squad,omitempty"`
	// Matches : les matchs À OBJECTIF du scope, dans l'ordre chronologique de la page.
	Matches []SquadFormesMatch `json:"matches,omitempty"`
}

// SquadFormesMatch — un match à objectif du scope : son identité d'affichage, son camp, sa
// feuille d'objectif.
type SquadFormesMatch struct {
	MatchID string `json:"match_id"`
	// StartTime : ISO 8601 UTC (l'heure affichée est locale, côté web).
	StartTime string `json:"start_time,omitempty"`
	// ModeLabel / MapLabel : libellés déjà résolus par les adapters du titre
	// (jamais une chaîne écrite en Go). Vides quand l'historique de la page ne
	// porte pas ce match.
	ModeLabel string `json:"mode_label,omitempty"`
	MapLabel  string `json:"map_label,omitempty"`
	// PlayerTeam : camp du joueur de la page (nil = inconnu, FFA).
	PlayerTeam *int `json:"player_team,omitempty"`
	// Objective : la feuille d'objectif du match.
	Objective *SquadFormesObjective `json:"objective,omitempty"`
}

// SquadFormesObjective — l'objectif d'un match : sa famille, les colonnes que ce
// mode publie VRAIMENT sur ce scope, et les valeurs de chaque joueur des deux
// camps.
type SquadFormesObjective struct {
	// Family : clé stable de famille de mode (`ctf`, `zones_koth`, ...).
	Family  string                       `json:"family"`
	Columns []SquadFormesObjectiveColumn `json:"columns,omitempty"`
	Players []SquadFormesObjectivePlayer `json:"players,omitempty"`
	// FlagJuggleWindowSeconds : la FENÊTRE DE JONGLAGE sous laquelle les prises
	// nettes de ce match ont été calculées. Absente = le match n'en porte pas
	// (pas de film lu, ou mode sans drapeau).
	//
	// ELLE VOYAGE AVEC LA MESURE parce que la mesure ne se lit pas sans elle :
	// « 4 prises nettes » n'a de sens qu'assorti de « les reprises de moins de
	// N secondes comptent pour une ». Le web en fait son infobulle ; le libellé,
	// lui, reste côté web (aucune chaîne de langue ne descend d'ici).
	FlagJuggleWindowSeconds float64 `json:"flag_juggle_window_seconds,omitempty"`
	// ExcludedFromBalance : le mode de ce match est écarté des parts de rôle de l'escouade
	// (« Rapport de force au fil de la session » et « soirée après soirée ») — le drapeau
	// neutre, où personne ne peut renvoyer le drapeau et où la part d'un camp est mécanique
	// (D6 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26). Même prédicat du titre que
	// l'historique d'objectif : la fin du fil égale le point « ce soir ».
	ExcludedFromBalance bool `json:"excluded_from_balance,omitempty"`
}

// SquadFormesObjectiveColumn — une grandeur du mode et son rôle.
type SquadFormesObjectiveColumn struct {
	// Key : nom de colonne de match_objective_stats (`flag_returns`). Le web le
	// traduit ; aucun libellé ne descend d'ici.
	Key string `json:"key"`
	// Role : `take`, `defend` ou `hold` (narrative.ObjectiveRole).
	Role string `json:"role"`
	// Duration : la colonne porte des SECONDES (elle s'affiche en mm:ss et ne
	// se compare jamais à un compte d'actions).
	Duration bool `json:"duration,omitempty"`
	// Optional : la grandeur PEUT MANQUER sur un match donné, et son absence
	// n'est pas un zéro.
	//
	// POURQUOI CE DRAPEAU EXISTE. Les colonnes de `match_objective_stats`
	// viennent du sync API : dès qu'un match a une ligne, toutes ses colonnes
	// ont une valeur, et un 0 y est une mesure. Les grandeurs lues du FILM (les
	// prises nettes de drapeau) n'existent que pour les matchs dont l'artefact
	// a été lu — un match sans film décodé n'a AUCUNE valeur, et l'afficher à
	// zéro dirait « il n'a rien pris » là où la vérité est « on n'a pas
	// regardé ». Le web rend alors « non mesuré ».
	//
	// Une grandeur optionnelle ABSENTE ne figure pas dans `values` du joueur :
	// c'est l'absence de clé qui porte l'information, jamais une valeur
	// sentinelle.
	Optional bool `json:"optional,omitempty"`
}

// SquadFormesObjectivePlayer — les valeurs d'un joueur sur les colonnes du match.
type SquadFormesObjectivePlayer struct {
	XUID   string             `json:"xuid"`
	TeamID *int               `json:"team_id,omitempty"`
	Values map[string]float64 `json:"values,omitempty"`
}
