package domain

// timeseries_lives.go — « MES VIES : PRÈS D'UN COÉQUIPIER OU SEUL » (Séries temporelles › Usages,
// plan `.ai/V7.5/PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05.md`, décision D5 ; calcul :
// analysis/coordination/vies_pres_ou_seul.go).
//
// Une vie du joueur terminée par une mort est rangée selon la distance au coéquipier VISIBLE le
// plus proche À L'INSTANT DE LA MORT (`match_death_context`) : à portée du radar du match (borne
// incluse) ou au-delà. Les frags d'une vie sont ceux tombés entre son début et le début de sa vie
// suivante. Les vies qui ne se rangent pas sont écartées et COMPTÉES, jamais rangées d'office.

// TimeseriesLivesNearTeammate — le bloc publié sous `lives_near_teammate`. Nil sans
// `film.kill_positions`, sur lecture en échec ou sans aucune vie du joueur sur la fenêtre.
type TimeseriesLivesNearTeammate struct {
	// Near / Alone : les vies terminées par une mort à portée du radar d'un coéquipier, ou au-delà,
	// et les frags tombés pendant ces vies.
	Near  LivesSideCount `json:"near"`
	Alone LivesSideCount `json:"alone"`
	// ExcludedUnlocated : vies terminées par une mort sans coéquipier situé à cet instant (aucun
	// visible, équipe à terre, ou mort sans contexte). ExcludedNoRadar : vies d'un match dont la
	// variante n'a pas de portée de radar connue. ExcludedUnpublishable : vies d'un match dont la
	// dernière passe du journal des morts n'est pas publiable (juste en agrégat, fausse frag par
	// frag) — ses frags ne sont pas lus, ses vies ne se rangent donc pas. Comptées pour le journal
	// de la lecture (service/solo_lives_block.go), JAMAIS publiées : aucun inconnu à l'écran.
	ExcludedUnlocated     int `json:"-"`
	ExcludedNoRadar       int `json:"-"`
	ExcludedUnpublishable int `json:"-"`
	// MatchesRead : matchs où le joueur a au moins une vie lue ; MatchesWithoutRadar : parmi eux,
	// ceux sans portée de radar.
	MatchesRead         int `json:"matches_read"`
	MatchesWithoutRadar int `json:"matches_without_radar"`
}

// LivesSideCount — des vies et les frags tombés pendant ces vies.
type LivesSideCount struct {
	Lives int `json:"lives"`
	Kills int `json:"kills"`
}

// VieLue — une vie du joueur lue de `match_lives_latest` (entrée du calcul).
type VieLue struct {
	MatchID  string
	StartMS  int64
	EndCause string
}

// MortSituee — une mort du joueur lue de `match_death_context_latest` et la distance au coéquipier
// visible le plus proche (nil : aucun coéquipier visible).
type MortSituee struct {
	MatchID     string
	TimeMS      int64
	PlusProcheM *float64
}

// FragLu — un frag publiable du joueur (`match_kill_events_latest`) et les camps du tueur et de la
// victime (`match_participants` ; nil : inconnu).
type FragLu struct {
	MatchID                string
	TimeMS                 int64
	CampTueur, CampVictime *int
}

// ViesLues — la lecture bornée d'un joueur sur une fenêtre de matchs (port.SoloLivesRepository).
type ViesLues struct {
	Vies  []VieLue
	Morts []MortSituee
	Frags []FragLu
	// Variantes : match -> game_variant_name, la clé de la portée du radar.
	Variantes map[string]string
	// JournalNonPubliable : les matchs dont la dernière passe de `match_kill_events_latest` n'est
	// pas publiable (le drapeau vaut pour la passe entière).
	JournalNonPubliable map[string]bool
}
