package replaydoc

// spawn_forecast.go — LE CORPS DE `GET /matches/{match_id}/replay/reapparitions` : la prevision de
// reapparition de chaque mort d'un match, seconde par seconde pendant le compte a rebours.
//
// UNE PREVISION APPARTIENT AU JOUEUR MORT : chaque mort porte le xuid de ce joueur, et le calque du
// rejeu la montre ou la masque avec lui (Seul / Masquer). Les zones sont celles de
// `GET /matches/{match_id}/replay/callouts` (meme `volume_index`) : la prevision ne repete pas leur
// forme, elle les designe. Une probabilite de zone est la SOMME des probabilites de ses points
// candidats ; la part des candidats qu'aucune zone ne porte est `outside_zone`.
//
// UN REFUS EST UNE REPONSE : une prevision qui n'a pas lieu d'etre (carte trop pauvre, modele
// absent) se sert en `status: unavailable` avec sa raison, jamais en erreur.

// Statuts et raisons d'une prevision de reapparition.
const (
	SpawnForecastStatusReady       = "ready"
	SpawnForecastStatusUnavailable = "unavailable"

	// Raisons d'un statut `unavailable`.
	SpawnForecastReasonNoModel    = "no_model"
	SpawnForecastReasonMapUnknown = "map_unknown"
	SpawnForecastReasonNotEnough  = "not_enough_matches"
	SpawnForecastReasonNoZone     = "no_zone"
	SpawnForecastReasonNoRespawn  = "no_respawn"
)

// SpawnForecast est la prevision de reapparition d'un match.
type SpawnForecast struct {
	Status string `json:"status" enum:"ready,unavailable"`
	// Reason dit pourquoi aucune prevision n'est servie (statut `unavailable`) ; Message est son
	// libelle, dans la langue de la requete.
	Reason  string `json:"reason,omitempty" enum:"no_model,map_unknown,not_enough_matches,no_zone,no_respawn"`
	Message string `json:"message,omitempty"`
	// Family est la famille de modes du match (cle de donnee du titre, pas un libelle) ; Model le
	// jeu de poids employe (la famille, ou `tous` quand elle n'a pas ses propres poids) et ModelDate
	// le jour de son ajustement.
	Family    string `json:"family,omitempty"`
	Model     string `json:"model,omitempty"`
	ModelDate string `json:"model_date,omitempty"`
	// MatchesOnMap : matchs de la famille sur la carte, ce match compris ; Candidates : points
	// candidats de la carte, hors ce match (une prevision ne voit pas la reponse).
	MatchesOnMap int `json:"matches_on_map"`
	Candidates   int `json:"candidates"`
	// DiscardedDeaths : reapparitions que la prevision ne peut pas servir (equipe inconnue, aucun
	// adversaire vivant pendant le compte a rebours).
	DiscardedDeaths int `json:"discarded_deaths"`
	// SampleStepMs est l'ecart entre deux echantillons d'une mort.
	SampleStepMs int `json:"sample_step_ms"`
	// Zones : les zones nommees que les previsions et les points reels designent, libelle dans la
	// langue de la requete, triees par `volume_index`.
	Zones []SpawnForecastZone `json:"zones"`
	// Points sont les points candidats de la carte, servis UNE fois : chaque echantillon de chaque mort
	// en porte la probabilite, alignee sur cette liste (`points_p`).
	Points []SpawnForecastCandidate `json:"points"`
	Deaths []SpawnForecastDeath     `json:"deaths"`
}

// SpawnForecastZone designe une zone nommee de la carte (`volume_index` de `/replay/callouts`).
type SpawnForecastZone struct {
	VolumeIndex int    `json:"volume_index"`
	Name        string `json:"name"`
}

// SpawnForecastCandidate est un point candidat de reapparition (metres monde, au centimetre) et la zone qui
// le nomme : `zone_index` est son indice dans `zones[]`, -1 quand aucune zone servie ne le nomme.
type SpawnForecastCandidate struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	ZoneIndex int     `json:"zone_index"`
}

// SpawnForecastPoint est une position en metres monde, au centimetre.
type SpawnForecastPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// SpawnForecastDeath est la prevision d'une mort suivie d'une reapparition : de l'instant de la
// mort (`death_frame`, le premier echantillon) a celui de la reapparition (`respawn_frame`).
type SpawnForecastDeath struct {
	XUID         string             `json:"xuid"`
	DeathFrame   int                `json:"death_frame"`
	RespawnFrame int                `json:"respawn_frame"`
	Death        SpawnForecastPoint `json:"death"`
	// Actual est le point reel de la reapparition ; ActualZone la zone qui le nomme (`volume_index`),
	// absente quand aucune zone ne le nomme.
	Actual     SpawnForecastPoint    `json:"actual"`
	ActualZone *int                  `json:"actual_zone,omitempty"`
	Samples    []SpawnForecastSample `json:"samples"`
}

// SpawnForecastSample est la prevision a un instant du compte a rebours : les zones de la plus a la
// moins probable (une probabilite arrondie a zero n'est pas servie).
type SpawnForecastSample struct {
	Frame       int                       `json:"frame"`
	Zones       []SpawnForecastZoneChance `json:"zones"`
	OutsideZone float64                   `json:"outside_zone"`
	// PointsP est la probabilite de chaque point de `points`, dans le meme ordre, arrondie au millieme
	// (somme egale a 1 a l'arrondi pres). La probabilite d'une zone est la somme de ses points.
	PointsP []float64 `json:"points_p"`
}

// SpawnForecastZoneChance est la probabilite d'une zone, entre 0 et 1.
type SpawnForecastZoneChance struct {
	Zone int     `json:"zone"`
	P    float64 `json:"p"`
}
