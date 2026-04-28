package domain

import "time"

// EngagementScoreResult est le resultat canonique d'un calcul d'engagement
// pour un joueur sur un match.
//
// Reference conceptuelle : .ai/REFLEXION_ENGAGEMENT_SCORE_INTRA_MATCH.md
//
// Concept central : la forme/engagement du joueur = ecart entre son engagement
// observe et son engagement attendu (vu son style historique et le contexte
// equipe), normalise en percentile vs sa propre distribution historique.
//
// 50 = ecart moyen comme d'habitude. > 50 = plus engage que d'habitude. < 50 =
// moins engage. La metrique est strictement intra-personnelle (pas comparable
// inter-joueurs ; pour cela, voir EngagementCoefficient).
type EngagementScoreResult struct {
	// EngagementScore est le percentile 0-100 du residu sur l'historique du
	// joueur (200 derniers matchs sur la meme categorie de mode). Nil si
	// insufficient_history.
	EngagementScore *float64

	// ResidualBrut est la valeur brute du residu (mean joueur - attendu) sur
	// le match, en events/min. Conserve pour debug et pour alimenter
	// l'historique futur (HistoricalEngagementBrut).
	ResidualBrut float64

	// EngagementCurve est la trace temporelle des paces sur le match, pour
	// affichage Match View (Mock 10) ou Squad Page (Mock 15 v2). Nil si
	// match trop court.
	EngagementCurve []EngagementPoint

	// MatchIntensity est la caracteristique objective du match
	// (events/min/joueur du lobby). Independante du joueur cible. Stockee
	// dans match_registry.match_intensity. 0 si non calculable.
	MatchIntensity float64

	// Confidence reflete la qualite du score :
	//   - "full"      : >= 30 matchs d'historique, signal complet
	//   - "partial"   : 10..29 matchs (cold start partiel)
	//   - "insufficient_history" : < 10 matchs (score = nil)
	//
	// Les seuils sont calibres sur H5 (cf doc reflexion §10).
	Confidence string

	// NHistoryMatches est le nombre de matchs utilises pour calculer le
	// percentile. Permet a l'UI de communiquer le contexte (ex. "calcule sur
	// 47 matchs").
	NHistoryMatches int
}

// EngagementPoint est un echantillon temporel de la courbe d'engagement.
//
// Echantillonne tous les SamplingSeconds (default 10s) sur la duree du match,
// fenetre glissante de WindowSeconds (default 90s).
type EngagementPoint struct {
	TimeMS int64

	// PaceJoueur est le pace observe du joueur cible dans la fenetre.
	PaceJoueur float64

	// PaceTeam est le pace de l'equipe alliee per_player dans la fenetre.
	// = events_team_dans_window / N_team / (W/60s)
	PaceTeam float64

	// PaceAttendu est l'engagement que le joueur aurait du produire vu son
	// style historique : coef_team_share * pace_team_per_player.
	PaceAttendu float64

	// PaceLobby est le pace lobby per_player dans la fenetre. Utilise par la
	// vue Squad Page (Mock 15 v2) comme reference externe. = events_lobby /
	// N_humans_lobby / (W/60s).
	PaceLobby float64

	// PostDeathFlag est true si le point est dans une periode post-mort
	// active (entre une mort et le prochain event d'engagement du joueur).
	PostDeathFlag bool

	// IsPassiveDeath n'est positionne que sur les points qui correspondent
	// exactement a un event "death" du joueur, ET dont la mort etait
	// precedee par un creux > PassiveDeathThresholdMS (default 30s) sans
	// event d'engagement. Indique un joueur "caught off-guard".
	IsPassiveDeath bool
}

// HistoricalEngagementBrut est un point d'historique utilise pour la
// normalisation percentile. La distribution complete (200 derniers matchs sur
// la meme categorie de mode) sert de baseline pour calculer
// EngagementScoreResult.EngagementScore.
type HistoricalEngagementBrut struct {
	MatchID string
	// Brut est le ResidualBrut historique de ce match (mean joueur - attendu).
	Brut float64
}

// EngagementCoefficient est une statistique personnelle exposee independamment
// du EngagementScore. Caracterise le style intra-equipe et le style absolu du
// joueur sur une categorie de mode donnee.
//
// Mediane glissante sur les 200 derniers matchs. Recalcule periodiquement
// (idealement incremental, cf. plan d'implementation §4.4).
type EngagementCoefficient struct {
	XUID         string
	ModeCategory string

	// CoefTeamShare = mediane historique de (pace_joueur / pace_team_per_player).
	// > 1 = leader intra-equipe. < 1 = support / passif. ~1 = fait sa part.
	CoefTeamShare float64

	// CoefLobbyShare = mediane historique de (pace_joueur / pace_lobby_per_player).
	// Caracterise le style absolu (mixe style + skill + qualite des equipes
	// habituelles). Comparable inter-joueurs (contrairement a CoefTeamShare).
	CoefLobbyShare float64

	// NMatches est le nombre de matchs utilises pour calculer ces medianes.
	NMatches int

	// LastUpdated est l'heure du dernier recalcul. Permet le check de
	// freshness (recalcul si > N jours d'inactivite).
	LastUpdated time.Time
}
