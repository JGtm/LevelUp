package replaydoc

// objectives.go — LES OBJECTIFS VIVANTS DU MATCH : score dans le temps, drapeaux, zones,
// couronne, crane, bombe — tout ce dont l'etat evolue pendant la partie.

// ObjectiveAction est une action d'objectif posee sur l'axe de temps du rejeu.
type ObjectiveAction struct {
	T      int    `json:"t"`
	XUID   string `json:"xuid"`
	Stat   string `json:"stat"`
	TimeMS int    `json:"timeMs"`
}

// ScoreTimeline est le calque du score dans le temps.
type ScoreTimeline struct {
	Teams             []TeamScore   `json:"teams,omitempty"`
	Players           []PlayerScore `json:"players,omitempty"`
	TargetScore       *int          `json:"targetScore,omitempty"`
	HoldTicks         []TeamHold    `json:"holdTicks,omitempty"`
	HoldTicksPerPoint *int          `json:"holdTicksPerPoint,omitempty"`
}

// TeamScore est la courbe de score d'un camp.
type TeamScore struct {
	TeamID *int         `json:"teamId,omitempty"`
	Rounds []ScoreRound `json:"rounds,omitempty"`
	Total  []ScoreTick  `json:"total,omitempty"`
}

// ScoreRound est la courbe d'UNE manche : les valeurs propres a la manche, non cumulees.
type ScoreRound struct {
	Round  int         `json:"round"`
	Points []ScoreTick `json:"points"`
}

// ScoreTick est un point de courbe : un instant sur l'axe du rejeu et une valeur.
type ScoreTick struct {
	T int `json:"t"`
	V int `json:"v"`
}

// PlayerScore porte les compteurs vivants d'un joueur.
type PlayerScore struct {
	XUID    string      `json:"xuid"`
	Score   ScoreSeries `json:"score"`
	Kills   ScoreSeries `json:"kills"`
	Deaths  ScoreSeries `json:"deaths"`
	Assists ScoreSeries `json:"assists"`
}

// ScoreSeries est un compteur suivi dans le temps, sous ses deux formes.
type ScoreSeries struct {
	Rounds []ScoreRound `json:"rounds,omitempty"`
	Total  []ScoreTick  `json:"total,omitempty"`
}

// TeamHold est la barre de garde d'un camp : ses tics cumules, en escalier.
type TeamHold struct {
	TeamID *int        `json:"teamId,omitempty"`
	Ticks  []ScoreTick `json:"ticks,omitempty"`
}

// FlagCarry est LA VIE D'UN DRAPEAU sur toute la partie : une suite d'intervalles d'etat.
type FlagCarry struct {
	Team  int        `json:"team"`
	Spans []FlagSpan `json:"spans"`
}

// FlagSpan est UN intervalle d'etat du drapeau.
type FlagSpan struct {
	State string  `json:"state"`
	T0    int     `json:"t0"`
	T1    int     `json:"t1"`
	XUID  *string `json:"xuid"`
	X     float32 `json:"x"`
	Y     float32 `json:"y"`
}

// FlagReturnZone est LA REGLE DE RETOUR du drapeau, telle que le manifeste du titre la donne
// (schema 35). Elle ne decrit PAS ce match-ci : elle decrit le MODE, et c'est pour cela qu'elle
// est publiee une fois et non par lacher.
type FlagReturnZone struct {
	RadiusM      float32 `json:"radiusM"`
	ResetSeconds float32 `json:"resetSeconds"`
	SoloSeconds  float32 `json:"soloSeconds"`
}

// ObjectiveObjectLife est UNE VIE LIBRE d'un objet d'objectif : l'objet apparaît, réplique sa
// position, puis se tait — parce qu'on l'a ramassé, ou parce qu'il s'est immobilisé.
type ObjectiveObjectLife struct {
	Family string                 `json:"family"`
	En     string                 `json:"en"`
	Fr     string                 `json:"fr"`
	T0     int                    `json:"t0"`
	T1     int                    `json:"t1"`
	Pts    []ObjectiveObjectPoint `json:"pts"`
}

// ObjectiveObjectPoint est une position datée d'un objet d'objectif libre. Même axe et mêmes
// unités que `Point` — le client les dessine avec la même transformation, sans conversion.
type ObjectiveObjectPoint struct {
	T int     `json:"t"`
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

// ZoneState est L'ETAT D'UNE ZONE sur toute la partie : une suite d'intervalles.
type ZoneState struct {
	ZoneRef    int          `json:"zoneRef"`
	LetterRank *int         `json:"letterRank,omitempty"`
	Key        uint32       `json:"key,omitempty"`
	Spans      []ZoneSpan   `json:"spans"`
	Gauge      []GaugePoint `json:"gauge,omitempty"`
}

// ZoneSpan est UN intervalle d'etat d'une zone.
type ZoneSpan struct {
	T0       int      `json:"t0"`
	T1       int      `json:"t1"`
	Owner    *int     `json:"owner"`
	Progress *float32 `json:"progress,omitempty"`
	Active   bool     `json:"active"`
}

// GaugePoint est UN point de la jauge en direct : la frame et la valeur lue a cet instant.
type GaugePoint struct {
	T int     `json:"t"`
	V float32 `json:"v"`
}

// VipPeriod est UNE periode de port de la couronne : un joueur, un intervalle de frames.
type VipPeriod struct {
	XUID   string `json:"xuid"`
	T0     int    `json:"t0"`
	T1     int    `json:"t1"`
	Closed bool   `json:"closed"`
}

// SkullCarry est UNE periode de portage du crane : un joueur, un intervalle de frames.
type SkullCarry struct {
	XUID   string `json:"xuid"`
	T0     int    `json:"t0"`
	T1     int    `json:"t1"`
	Closed bool   `json:"closed"`
}

// BombArming est UN armement de la bombe d'Assaut : l'instant où le compte à rebours part,
// sur l'axe de frames du rejeu comme sur l'horloge du film, et la durée de la mèche.
type BombArming struct {
	T       int `json:"t"`
	TimeMS  int `json:"timeMs"`
	StartT  int `json:"startT"`
	StartMS int `json:"startMs"`
	FuseMS  int `json:"fuseMs"`
}

// BombCarry est UNE periode de portage de la bombe : un joueur, un intervalle de frames.
type BombCarry struct {
	XUID   string `json:"xuid"`
	T0     int    `json:"t0"`
	T1     int    `json:"t1"`
	Closed bool   `json:"closed"`
}

// BombMatchStats est le résultat par match : les joueurs et ce qui a été vu.
type BombMatchStats struct {
	Players  []BombPlayerStats `json:"players,omitempty"`
	Coverage BombStatsCoverage `json:"coverage"`
}

// BombPlayerStats porte les statistiques d'UN joueur. Un champ à `nil` n'a pas été mesuré.
type BombPlayerStats struct {
	XUID                 string   `json:"xuid"`
	Detonations          *int     `json:"detonations,omitempty"`
	Arms                 *int     `json:"arms,omitempty"`
	Grabs                *int     `json:"grabs,omitempty"`
	TimeAsCarrierSeconds *float64 `json:"timeAsCarrierSeconds,omitempty"`
	CarriersKilled       *int     `json:"carriersKilled,omitempty"`
}

// BombEvent est un fait daté de la bombe, sur l'horloge du FILM — la même que celle des
// `ObjectiveAction`, superposable sans recalage.
type BombEvent struct {
	Type        string `json:"type"`
	TimeMS      int    `json:"timeMs"`
	XUID        string `json:"xuid,omitempty"`
	ActorSource string `json:"actorSource,omitempty"`
}
