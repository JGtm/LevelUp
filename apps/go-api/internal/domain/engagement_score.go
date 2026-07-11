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
	EngagementScore *float64 `json:"engagement_score"`

	// ResidualBrut est la valeur brute du residu (mean joueur - attendu) sur
	// le match, en events/min. Conserve pour debug et pour alimenter
	// l'historique futur (HistoricalEngagementBrut).
	ResidualBrut float64 `json:"residual_brut"`

	// EngagementCurve est la trace temporelle des paces sur le match, pour
	// affichage Match View (Mock 10) ou Squad Page (Mock 15 v2). Nil si
	// match trop court.
	EngagementCurve []EngagementPoint `json:"engagement_curve"`

	// MatchIntensity est la caracteristique objective du match
	// (events/min/joueur du lobby). Independante du joueur cible. Stockee
	// dans match_registry.match_intensity. 0 si non calculable.
	MatchIntensity float64 `json:"match_intensity"`

	// Confidence reflete la qualite du score :
	//   - "full"      : >= 30 matchs d'historique, signal complet
	//   - "partial"   : 10..29 matchs (cold start partiel)
	//   - "insufficient_history" : < 10 matchs (score = nil)
	//
	// Les seuils sont calibres sur H5 (cf doc reflexion §10).
	Confidence string `json:"confidence"`

	// NHistoryMatches est le nombre de matchs utilises pour calculer le
	// percentile. Permet a l'UI de communiquer le contexte (ex. "calcule sur
	// 47 matchs").
	NHistoryMatches int `json:"n_history_matches"`

	// MeanPaceJoueur / MeanPaceTeam / MeanPaceLobby sont les means agreges
	// de la courbe (events/min/joueur). Persistes dans player_match_enrichment
	// pour alimenter le recompute des coefficients sans avoir a rejouer toute
	// la courbe sur l'historique. Identiques aux means utilises par
	// EngagementMatchSummary (Mock 11).
	MeanPaceJoueur float64 `json:"mean_pace_joueur"`
	MeanPaceTeam   float64 `json:"mean_pace_team"`
	MeanPaceLobby  float64 `json:"mean_pace_lobby"`

	// PlayerActivity = kills + assists + deaths du joueur sur le match.
	// Sert a detecter les quitters/AFK lors du calcul du coefficient
	// (cf. temporal.PlayerActivityMin).
	PlayerActivity int `json:"player_activity"`

	// ExpectedBasis qualifie la base de calcul de l'attendu (PaceAttendu),
	// modele lobby-anchored v2 (cf. .ai/V7/PLAN_ENGAGEMENT_REFONTE_LOBBY_2026-07.md) :
	//   - "bin"        : coef du bin d'intensite du match (>= MinMatchesForBin echantillons)
	//   - "global"     : coef lobby global (fallback, >= MinMatchesForCoef echantillons)
	//   - "cold_start" : aucun historique exploitable → coef 1.0 ; la serie
	//     « Joueur attendu » est MASQUEE cote front (masquage indexe sur ce champ,
	//     plus sur Confidence).
	//
	// Distinct de Confidence, qui qualifie l'historique du PERCENTILE (le score),
	// pas l'attendu. Deux signaux independants.
	ExpectedBasis string `json:"expected_basis"`

	// IntensityBin est le libelle du bin d'intensite retenu quand ExpectedBasis
	// vaut "bin" (calme / standard / chaotique). Vide sinon.
	IntensityBin string `json:"intensity_bin"`

	// SignalBasis qualifie la SUFFISANCE du vecteur de signaux du match (1re porte
	// de degradation, chantier F7 engagement title-agnostic gradue) :
	//   - "full"         : ensemble minimal + au moins un signal riche (objectif...)
	//   - "partial"      : ensemble minimal seul (pas de signal riche)
	//   - "insufficient" : ensemble minimal absent (un resultat servi n'atteint pas
	//     ce niveau — les gardes ErrMatchTooShort/ErrInsufficientData ont deja filtre)
	//
	// Distinct de Confidence (historique du percentile) et de ExpectedBasis (base de
	// l'attendu). Combine avec le statut de calibration du titre pour la 2e porte.
	SignalBasis string `json:"signal_basis,omitempty"`

	// Calibration est la 2e porte de degradation (chantier F7) : statut de calibration
	// des coefficients d'engagement DU TITRE pour ce score :
	//   - "validated"   : coefficients valides (gate humain passe) — score de confiance pleine
	//   - "provisional" : coefficients provisoires (titre en calibration, ex. H5 degraded) —
	//     le front affiche une mention « calibration provisoire » (DE-8)
	//
	// Vide = titre historique valide (Infinite) traite comme validated. Injecte au
	// niveau service (title-aware) ; le moteur temporal, pur, ne le connait pas.
	Calibration string `json:"calibration,omitempty"`
}

// Valeurs de EngagementScoreResult.ExpectedBasis (chaine de fallback de l'attendu).
const (
	ExpectedBasisBin       = "bin"
	ExpectedBasisGlobal    = "global"
	ExpectedBasisColdStart = "cold_start"
)

// Valeurs de EngagementScoreResult.Calibration (2e porte de degradation F7).
const (
	CalibrationValidated   = "validated"
	CalibrationProvisional = "provisional"
)

// EngagementResponseBins porte les coefficients de reponse du joueur par bin
// d'intensite (tercile de pace_lobby), pour une categorie de mode. Modele
// lobby-anchored v2 (2026-07-07) : l'attendu du joueur est « sa reponse
// habituelle a un match d'intensite similaire », pas une part relative a son
// equipe. cf .ai/V7/PLAN_ENGAGEMENT_REFONTE_LOBBY_2026-07.md.
type EngagementResponseBins struct {
	XUID         string                   `json:"xuid,omitempty"`
	ModeCategory string                   `json:"mode_category,omitempty"`
	Bins         []EngagementIntensityBin `json:"bins"`
}

// EngagementIntensityBin est un bin d'intensite (tercile de pace_lobby) avec son
// coefficient de reponse lobby-anchored.
type EngagementIntensityBin struct {
	// Bin est le libelle du tercile : calme / standard / chaotique.
	Bin string `json:"bin"`
	// LowerBound / UpperBound bornent l'intensite (pace_lobby moyen du match)
	// couverte par ce bin. Bornes contiguës en terciles ; extremes ouverts a la
	// resolution (cf. ResolveBin).
	LowerBound float64 `json:"lower_bound"`
	UpperBound float64 `json:"upper_bound"`
	// CoefLobby = mediane(pace_joueur / pace_lobby) des matchs du bin, clampee.
	CoefLobby float64 `json:"coef_lobby"`
	// NMatches = echantillons valides dans le bin. Le serving n'exploite le coef
	// du bin (ExpectedBasis "bin") que si NMatches >= temporal.MinMatchesForBin.
	NMatches int `json:"n_matches"`
}

// ResolveBin retourne le bin dont la borne inferieure est la plus grande sans
// depasser l'intensite fournie (terciles contigus ; extremes ouverts : une
// intensite au-dela du dernier tercile reste « chaotique », en-deca du premier
// reste « calme »). Retourne (zero, false) si aucun bin n'est defini.
func (b *EngagementResponseBins) ResolveBin(intensity float64) (EngagementIntensityBin, bool) {
	if b == nil || len(b.Bins) == 0 {
		return EngagementIntensityBin{}, false
	}
	best := -1
	for i := range b.Bins {
		if b.Bins[i].LowerBound <= intensity {
			if best < 0 || b.Bins[i].LowerBound > b.Bins[best].LowerBound {
				best = i
			}
		}
	}
	if best < 0 {
		// Intensite sous toutes les bornes inf (calme.lower devrait valoir 0, donc
		// rare) : rabattre sur le bin de plus basse borne.
		best = 0
		for i := range b.Bins {
			if b.Bins[i].LowerBound < b.Bins[best].LowerBound {
				best = i
			}
		}
	}
	return b.Bins[best], true
}

// EngagementPoint est un echantillon temporel de la courbe d'engagement.
//
// Echantillonne tous les SamplingSeconds (default 10s) sur la duree du match,
// fenetre glissante de WindowSeconds (default 90s).
type EngagementPoint struct {
	TimeMS int64 `json:"time_ms"`

	// PaceJoueur est le pace observe du joueur cible dans la fenetre.
	PaceJoueur float64 `json:"pace_joueur"`

	// PaceTeam est le pace de l'equipe alliee per_player dans la fenetre.
	// = events_team_dans_window / N_team / (W/60s)
	PaceTeam float64 `json:"pace_team"`

	// PaceAttendu est l'engagement que le joueur produit D'HABITUDE dans un match
	// d'intensite similaire (modele lobby-anchored) : coef[bin] * pace_lobby.
	PaceAttendu float64 `json:"pace_attendu"`

	// PaceLobby est le pace lobby per_player dans la fenetre. Utilise par la
	// vue Squad Page (Mock 15 v2) comme reference externe. = events_lobby /
	// N_humans_lobby / (W/60s).
	PaceLobby float64 `json:"pace_lobby"`

	// PostDeathFlag est true si le point est dans une periode post-mort
	// active (entre une mort et le prochain event d'engagement du joueur).
	PostDeathFlag bool `json:"post_death_flag"`

	// IsPassiveDeath n'est positionne que sur les points qui correspondent
	// exactement a un event "death" du joueur, ET dont la mort etait
	// precedee par un creux > PassiveDeathThresholdMS (default 30s) sans
	// event d'engagement. Indique un joueur "caught off-guard".
	IsPassiveDeath bool `json:"is_passive_death"`
}

// HistoricalEngagementBrut est un point d'historique utilise pour la
// normalisation percentile. La distribution complete (200 derniers matchs sur
// la meme categorie de mode) sert de baseline pour calculer
// EngagementScoreResult.EngagementScore.
type HistoricalEngagementBrut struct {
	MatchID string  `json:"match_id"`
	Brut    float64 `json:"brut"`
}

// EngagementCoefficient est une statistique personnelle exposee independamment
// du EngagementScore. Caracterise le style intra-equipe et le style absolu du
// joueur sur une categorie de mode donnee.
//
// Mediane glissante sur les 200 derniers matchs. Recalcule periodiquement
// (idealement incremental, cf. plan d'implementation §4.4).
// EngagementTimeseriesRequest est le corps de POST /engagement/timeseries.
// Aligne sur TimeseriesQueryRequest (meme `filters`) pour que le scope (period,
// cascade, sessions, match_context) suive celui de la page Timeseries Mock 11.
// `limit` borne le nombre de matchs PvP retournes (defaut 50, max 500).
type EngagementTimeseriesRequest struct {
	Filters FilterContextInput `json:"filters"`
	Limit   int                `json:"limit,omitempty"`
}

// EngagementMatchSummary est un point Mock 11 — soit 1 match (granularite
// "match"), soit l'agregat de N matchs sur une session/semaine/mois.
//
// Pour les granularites agregees :
//   - MatchID est vide (les paces sont une moyenne ponderee de plusieurs matchs)
//   - Label sert d'identifiant lisible (ex session_label, "2026-S18", "2026-05")
//   - MatchCount > 1 ; les paces sont les moyennes du bucket
//   - EngagementScore est la moyenne des scores non-nuls du bucket (nil si aucun)
type EngagementMatchSummary struct {
	MatchID         string    `json:"match_id"`
	Label           string    `json:"label"`
	MapName         *string   `json:"map_name,omitempty"`
	StartedAt       time.Time `json:"started_at"`
	PaceJoueur      float64   `json:"pace_joueur"`
	PaceTeam        float64   `json:"pace_team"`
	PaceAttendu     float64   `json:"pace_attendu"`
	PaceLobby       float64   `json:"pace_lobby"`
	EngagementScore *float64  `json:"engagement_score"`
	// MatchCount : nombre de matchs representes par ce point (1 pour granularite
	// "match", >1 pour session/week/month). Permet au front d'afficher "n=N"
	// dans le tooltip et de moduler l'opacite des markers.
	MatchCount int `json:"match_count"`
}

// EngagementGranularity enumere les granularites d'un EngagementTimeseriesResponse.
const (
	EngagementGranularityMatch   = "match"
	EngagementGranularitySession = "session"
	EngagementGranularityWeek    = "week"
	EngagementGranularityMonth   = "month"
)

// EngagementTimeseriesResponse est la reponse de POST /engagement/timeseries.
//
// Mock 11 — granularite adaptative selon la densite :
//   - <= limit matchs filtres → "match" (1 point = 1 match)
//   - sinon, agregation par session_label → "session" si <= limit
//   - sinon, agregation par semaine ISO → "week" si <= limit
//   - sinon, agregation par mois → "month"
//
// TruncatedToRecent (optionnel) signale qu'on a borne le compute aux N matchs
// les plus recents avant binning (perfs : ComputeEngagementScore est O(events)
// + 3 queries DB par match). TotalMatches reflete le total filtre AVANT cap.
type EngagementTimeseriesResponse struct {
	Granularity       string                   `json:"granularity"`
	Points            []EngagementMatchSummary `json:"points"`
	TotalMatches      int                      `json:"total_matches"`
	TruncatedToRecent *int                     `json:"truncated_to_recent,omitempty"`
}

// SquadEngagementSession est le payload pour la Squad Page Mock 15 v2.
// Pour chaque match commun a la squad : 3 traces team-level + per-player paces.
type SquadEngagementSession struct {
	Labels         []string                `json:"labels"`
	MapNames       []string                `json:"map_names"`
	LobbyPerPlayer []float64               `json:"lobby_per_player"`
	TeamExpected   []float64               `json:"team_expected"`
	TeamObserved   []float64               `json:"team_observed"`
	Players        []SquadPlayerEngagement `json:"players"`
}

// SquadPlayerEngagement = pace observe d'un membre du squad sur la session.
type SquadPlayerEngagement struct {
	XUID         string    `json:"xuid"`
	Gamertag     string    `json:"gamertag"`
	PaceObserved []float64 `json:"pace_observed"`
}

// EngagementProfile est le profil engagement long-terme d'un joueur pour une
// categorie de mode, expose par GET /engagement_profile (modele lobby-anchored
// v2). Type DEDIE (cf. plan Phase 3) : distinct de EngagementCoefficient, qui
// reste le porteur interne xuid/gamertag cote squad. Ne porte PAS coef_team_share
// (deprecated, D5) et ajoute les bins de reponse par intensite.
type EngagementProfile struct {
	XUID         string `json:"xuid"`
	Gamertag     string `json:"gamertag,omitempty"`
	ModeCategory string `json:"mode_category"`

	// CoefLobbyShare = mediane historique de (pace_joueur / pace_lobby_per_player).
	// Coef lobby global du joueur (toutes intensites), fallback de l'attendu.
	CoefLobbyShare float64 `json:"coef_lobby_share"`

	// NMatches = matchs utilises pour la mediane globale.
	NMatches int `json:"n_matches"`

	// LastUpdated = heure du dernier recompute des coefficients.
	LastUpdated time.Time `json:"last_updated"`

	// Bins = coefs de reponse par bin d'intensite (terciles de pace_lobby).
	// Vide si aucun bin persiste (historique insuffisant).
	Bins []EngagementIntensityBin `json:"bins"`
}

// EngagementCoefficient est le porteur INTERNE du coef lobby global d'un joueur
// (+ xuid/gamertag, réutilisé côté squad). coef_team_share a été abandonné (D5,
// modèle lobby-anchored) ; il n'est plus porté par ce type. Le payload public
// GET /engagement_profile utilise EngagementProfile (avec les bins).
type EngagementCoefficient struct {
	XUID         string `json:"xuid"`
	Gamertag     string `json:"gamertag,omitempty"`
	ModeCategory string `json:"mode_category"`

	// CoefLobbyShare = mediane historique de (pace_joueur / pace_lobby_per_player).
	// Caracterise la reponse globale du joueur a l'action totale du lobby
	// (fallback de l'attendu, ExpectedBasis "global").
	CoefLobbyShare float64 `json:"coef_lobby_share"`

	// NMatches est le nombre de matchs utilises pour calculer la mediane.
	NMatches int `json:"n_matches"`

	// LastUpdated est l'heure du dernier recalcul. Permet le check de
	// freshness (recalcul si > N jours d'inactivite).
	LastUpdated time.Time `json:"last_updated"`
}
