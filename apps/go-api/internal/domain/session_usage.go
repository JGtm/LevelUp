// Package domain — session_usage.go : le bloc « usages d'équipement, socles et
// objectifs » de la page détail de session (chantier session-usage, S2 —
// .ai/HANDOFF_SESSION_USAGE_BDD_2026-09-04.md §5/S2).
//
// DOCTRINE DU CONTRAT (§1 du handoff) : TOUT axe de comparaison est NORMALISÉ —
// parts en pourcentage, parités (100/effectif), cadences par dix minutes. Les
// comptes bruts (player_total, team_total, lobby_total, pad_unnamed_total,
// occupations de bonus) ne survivent que comme DÉNOMINATEURS D'HONNÊTETÉ, à
// afficher en texte à côté d'un taux — jamais comme axe. La référence (l'équipe
// d'en face) n'apparaît nulle part : elle est le complément du dénominateur.
//
// DEUX DÉNOMINATEURS PARTOUT : la part de MON ÉQUIPE (parité = 100/effectif
// d'équipe) et la part du LOBBY (parité = 100/effectif du lobby). Les grenades
// sont produites en S1 mais NE FONT PAS partie du contrat (décision utilisateur
// 2026-09-04 : « ce ne sont pas des équipements »).
//
// CONTEXTE ESCOUADE : le scope des matchs est résolu EN AMONT par
// Filters.MatchContext (le bloc agrège les matchs de la session affichée, quel
// que soit le contexte) ; en contexte escouade, les coéquipiers suivis reçoivent
// EN PLUS une ligne par grandeur (squad) — la « piste du lobby découpée par
// joueur » de la maquette en a besoin.
package domain

// Raisons machine du bloc indisponible (jamais un libellé : l'i18n vit au front).
const (
	// SessionUsageUnsupported : le titre ne déclare pas la capability
	// film.usage_summary — réponse partielle propre, jamais un 500 (ADR 0011).
	SessionUsageUnsupported = "unsupported"
	// SessionUsageLoadFailed : la lecture des vues a échoué (best-effort dégradé,
	// l'erreur est loggée côté service).
	SessionUsageLoadFailed = "load_failed"
)

// SessionUsageShares — le triplet joueur / son camp / lobby d'UNE grandeur, avec
// ses parts croisées. Les totaux sont les dénominateurs d'honnêteté (texte) ; les
// parts sont les axes. Une part est nil quand son dénominateur est nul (0/0 n'est
// pas 0 %) ou quand le camp du joueur est inconnu sur tout le scope.
//
// RÈGLE DE SCOPE (ronde de correction S2) : les grandeurs relatives à l'ÉQUIPE
// (team_total, player_share_of_team_pct, team_share_of_lobby_pct) se calculent
// sur le SOUS-ENSEMBLE des matchs du scope à camp CONNU — numérateurs ET
// dénominateurs. Sans quoi une session mêlant matchs en équipe et FFA croiserait
// deux scopes et player_share_of_team_pct dépasserait 100 %. Sous-ensemble
// vide : team_total est nil — jamais un 0 inventé pour dire « inconnu ».
// player_total, lobby_total et les parts joueur/lobby restent sur TOUT le scope.
type SessionUsageShares struct {
	PlayerTotal float64 `json:"player_total"`
	// TeamTotal : somme du camp du joueur sur les matchs à camp connu du scope.
	// nil quand aucun match du scope n'a de camp connu (session entièrement FFA).
	TeamTotal  *float64 `json:"team_total,omitempty"`
	LobbyTotal float64  `json:"lobby_total"`
	// TeamShareOfLobbyPct : « mon camp / lobby » (§7 du handoff, 1re colonne).
	TeamShareOfLobbyPct *float64 `json:"team_share_of_lobby_pct,omitempty"`
	// PlayerShareOfTeamPct : « joueur / son équipe » (2e colonne).
	PlayerShareOfTeamPct *float64 `json:"player_share_of_team_pct,omitempty"`
	// PlayerShareOfLobbyPct : « joueur / lobby » (3e colonne).
	PlayerShareOfLobbyPct *float64 `json:"player_share_of_lobby_pct,omitempty"`
}

// SessionUsageSquadPlayer — l'identité d'UN coéquipier suivi du contexte
// escouade (l'ordre de la liste est l'ordre d'affichage — jetons
// squad-player-1..3 côté front). Le joueur de la route n'y figure pas : ses
// grandeurs sont les champs Player* des métriques.
type SessionUsageSquadPlayer struct {
	XUID     string `json:"xuid"`
	Gamertag string `json:"gamertag"`
}

// SessionUsageSquadShare — la ligne d'UN coéquipier suivi sur UNE grandeur,
// mêmes conventions ET mêmes scopes que les champs Player* (total = dénominateur
// d'honnêteté sur tout le scope ; part d'équipe sur les matchs à camp connu ;
// cadence sur les matchs à durée connue ; nil quand le dénominateur est nul).
// Alignée sur SessionUsageBlock.SquadPlayers par XUID.
type SessionUsageSquadShare struct {
	XUID            string   `json:"xuid"`
	Total           float64  `json:"total"`
	ShareOfTeamPct  *float64 `json:"share_of_team_pct,omitempty"`
	ShareOfLobbyPct *float64 `json:"share_of_lobby_pct,omitempty"`
	Per10Min        *float64 `json:"per_10min,omitempty"`
}

// SessionUsageMatchPoint — les parts d'UN match mesuré (une case de la bande de
// régularité ; l'étendue de la jauge double s'en dérive). Une part nil = non
// calculable sur ce match (dénominateur nul, camp inconnu).
type SessionUsageMatchPoint struct {
	MatchID               string   `json:"match_id"`
	PlayerShareOfTeamPct  *float64 `json:"player_share_of_team_pct,omitempty"`
	PlayerShareOfLobbyPct *float64 `json:"player_share_of_lobby_pct,omitempty"`
	TeamShareOfLobbyPct   *float64 `json:"team_share_of_lobby_pct,omitempty"`
}

// SessionUsageMetric — UNE grandeur agrégée sur les matchs MESURÉS de la session.
// Clés servies : "grapple_pulls", "camo_episodes", "overshield_episodes",
// "dropped_objects", "pad_pickups", et "deployed_<famille>" par famille déployée
// observée (ensemble ouvert — manifeste du titre, ex. "deployed_wall").
type SessionUsageMetric struct {
	Key string `json:"key"`
	SessionUsageShares
	// Cadences par DIX MINUTES de jeu mesuré À DURÉE CONNUE — numérateur ET
	// dénominateur sur ces seuls matchs (un match sans échelle de temps n'entre
	// dans aucune cadence : compté au numérateur seul, il la gonflerait en
	// silence). TeamPer10Min : scope camp connu en plus (durée mesurée des matchs
	// à camp connu) — nil quand ce scope est vide, jamais une cadence inventée.
	PlayerPer10Min *float64 `json:"player_per_10min,omitempty"`
	TeamPer10Min   *float64 `json:"team_per_10min,omitempty"`
	LobbyPer10Min  *float64 `json:"lobby_per_10min,omitempty"`
	// Étendue match par match de la part joueur/équipe (bornes de la jauge).
	PlayerShareOfTeamMinPct *float64 `json:"player_share_of_team_min_pct,omitempty"`
	PlayerShareOfTeamMaxPct *float64 `json:"player_share_of_team_max_pct,omitempty"`
	// Matchs où la part du joueur dépasse LA PARITÉ DU MATCH (100/effectif de CE
	// match, pas la moyenne de session), contre chacun des deux dénominateurs.
	// MatchesAboveTeamParity : nil quand aucun match mesuré n'a de camp connu
	// (le compte serait invérifiable — même règle de scope que team_total).
	MatchesAboveTeamParity  *int `json:"matches_above_team_parity,omitempty"`
	MatchesAboveLobbyParity int  `json:"matches_above_lobby_parity"`
	// PerMatch : une entrée par match MESURÉ, dans l'ordre de la session.
	PerMatch []SessionUsageMatchPoint `json:"per_match,omitempty"`
	// Squad : une ligne par coéquipier suivi (contexte escouade uniquement),
	// alignée sur SessionUsageBlock.SquadPlayers.
	Squad []SessionUsageSquadShare `json:"squad,omitempty"`
}

// SessionUsagePadFamily — ventilation des prises de socle d'ARME nommées par clé
// de famille NORMALISÉE (replay.PadWeaponFamilyKey : huit hexa minuscules).
type SessionUsagePadFamily struct {
	FamilyKey string `json:"family_key"`
	SessionUsageShares
}

// SessionUsagePowerup — occupations de socle de BONUS, ANONYMES par construction
// (un bonus s'identifie par un nom, jamais rattachable à un joueur — §4 du
// handoff). Grandeur de MATCH : ni part d'équipe ni part de joueur, seulement le
// total (texte) et sa cadence.
type SessionUsagePowerup struct {
	FamilyKey   string   `json:"family_key"` // nom canonique ("powerup_camo", ...)
	Occupations int      `json:"occupations"`
	Per10Min    *float64 `json:"per_10min,omitempty"`
}

// SessionObjectiveRoleMetric — un rôle d'objectif agrégé. Role est une clé de
// narrative.ObjectiveRole ("take" | "defend" | "hold" — la classification des
// colonnes par rôle vit en SOURCE UNIQUE dans analysis/narrative,
// objective_roles.go). IsDuration : la grandeur est en secondes (rôle « tenir »)
// — les parts restent des pourcentages, seuls les totaux changent d'unité.
type SessionObjectiveRoleMetric struct {
	Role       string `json:"role"`
	IsDuration bool   `json:"is_duration,omitempty"`
	SessionUsageShares
	// Squad : une ligne par coéquipier suivi (contexte escouade uniquement),
	// alignée sur SessionUsageBlock.SquadPlayers. Per10Min reste nil : les rôles
	// d'objectif se publient en parts, jamais en cadence.
	Squad []SessionUsageSquadShare `json:"squad,omitempty"`
}

// SessionObjectiveFamilyBlock — le détail d'une famille de mode (clés
// narrative : "ctf", "zones_koth", "zones_strongholds", "oddball", "stockpile",
// "extraction", "vip"). Un rôle sans colonne pour la famille (extraction n'a pas
// de « tenir ») est absent de Roles.
type SessionObjectiveFamilyBlock struct {
	Family  string                       `json:"family"`
	Matches int                          `json:"matches"`
	Roles   []SessionObjectiveRoleMetric `json:"roles"`
}

// SessionObjectivesBlock — le bloc 3 : objectifs par rôle et par famille, lus de
// match_objective_stats_latest (les deux camps y sont déjà — rien n'est produit,
// tout est agrégé à la lecture). Son scope est l'ensemble des matchs de la
// session portant des stats objectifs — INDÉPENDANT de la couverture des films.
type SessionObjectivesBlock struct {
	MatchesWithObjectives int      `json:"matches_with_objectives"`
	TeamSizeAvg           float64  `json:"team_size_avg,omitempty"`
	LobbySizeAvg          float64  `json:"lobby_size_avg,omitempty"`
	TeamParityPct         *float64 `json:"team_parity_pct,omitempty"`
	LobbyParityPct        *float64 `json:"lobby_parity_pct,omitempty"`
	// Roles : les trois rôles agrégés TOUTES familles confondues (le tableau §7).
	Roles []SessionObjectiveRoleMetric `json:"roles"`
	// Families : le même découpage, par famille de mode.
	Families []SessionObjectiveFamilyBlock `json:"families,omitempty"`
}

// SessionUsageBlock — le bloc complet servi avec la page détail de session
// (contexte Solo/Escouade déjà résolu en amont par Filters.MatchContext : le bloc
// agrège LES MATCHS DE LA SESSION AFFICHÉE, quel que soit le contexte).
type SessionUsageBlock struct {
	// Available : false = bloc non calculable (voir UnavailableReason) ; les
	// autres champs valent alors zéro sauf MatchesTotal.
	Available         bool   `json:"available"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
	// MatchesMeasured / MatchesTotal : « matchs mesurés N/M » — la couverture des
	// films n'est jamais totale et l'écran DOIT le dire (§5/S2). Un match est
	// mesuré s'il a une ligne match_usage_films_latest. 0/M est un état légitime :
	// le bloc est vide mais présent.
	MatchesMeasured int `json:"matches_measured"`
	MatchesTotal    int `json:"matches_total"`
	// MeasuredDurationSeconds : durée jouée cumulée des matchs MESURÉS à durée
	// CONNUE — le dénominateur des cadences joueur/lobby par dix minutes. Un
	// match mesuré sans échelle de temps (artefact sans durée, aucun repli) n'y
	// entre pas : il est exclu des cadences (numérateur et dénominateur) mais
	// reste compté dans les totaux et les parts.
	MeasuredDurationSeconds float64 `json:"measured_duration_seconds,omitempty"`
	// Effectifs MOYENS sur les matchs mesurés (joueurs présents à la fin, bots
	// inclus) et les deux parités qui s'en déduisent (100/effectif).
	TeamSizeAvg    float64  `json:"team_size_avg,omitempty"`
	LobbySizeAvg   float64  `json:"lobby_size_avg,omitempty"`
	TeamParityPct  *float64 `json:"team_parity_pct,omitempty"`
	LobbyParityPct *float64 `json:"lobby_parity_pct,omitempty"`
	// PadUnnamedTotal : prises de socle d'ARME dont le ramasseur n'est pas nommé
	// par le film — dénominateur d'honnêteté de la note de pied (§7 : 82 sur le
	// témoin). Jamais réparties : elles n'appartiennent à personne.
	PadUnnamedTotal int `json:"pad_unnamed_total,omitempty"`
	// SquadPlayers : les coéquipiers suivis du contexte escouade (ordre
	// d'affichage ; vide en contexte solo ou sans coéquipier commun à toute la
	// session). Les lignes Squad des métriques s'y alignent par XUID.
	SquadPlayers   []SessionUsageSquadPlayer `json:"squad_players,omitempty"`
	Metrics        []SessionUsageMetric      `json:"metrics,omitempty"`
	PadFamilies    []SessionUsagePadFamily   `json:"pad_families,omitempty"`
	PowerupPickups []SessionUsagePowerup     `json:"powerup_pickups,omitempty"`
	Objectives     *SessionObjectivesBlock   `json:"objectives,omitempty"`
}
