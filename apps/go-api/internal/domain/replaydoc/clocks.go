package replaydoc

// clocks.go — L'HORLOGE DE MANCHE, LES COMPTES A REBOURS DE REAPPARITION ET LES REGLES DE LA VARIANTE.

// ClockSample est une lecture de l'horloge de manche. `t` est une frame, negative avant la frame 0 ;
// `left` et `limit` sont en secondes (`left` : temps restant quand la minuterie descend, temps ecoule
// quand elle monte ; `limit` zero : manche sans limite) ; `state` vaut 1 (arretee), 6 (descend) ou 19 (monte).
type ClockSample struct {
	T     int     `json:"t"`
	Left  float64 `json:"left"`
	Limit float64 `json:"limit"`
	State int     `json:"state"`
}

// RespawnCountdown est une periode de compte a rebours de reapparition d'un joueur du film : de la
// frame `from` (compte egal a `seconds`) a la frame `to` (zero). `startRead` / `endRead` disent si le
// debut / la fin est lu, faux quand l'instant est deduit des lectures.
type RespawnCountdown struct {
	FilmIndex int  `json:"filmIndex"`
	From      int  `json:"from"`
	To        int  `json:"to"`
	Seconds   int  `json:"seconds"`
	StartRead bool `json:"startRead"`
	EndRead   bool `json:"endRead"`
}

// MatchRules est ce que la variante du match ecrit : delai de reapparition, penalites de suicide et de
// trahison (secondes) et tirs allies. Un champ que la variante n'ecrit pas est absent.
type MatchRules struct {
	MinRespawnS      *int64 `json:"minRespawnS,omitempty"`
	SuicidePenaltyS  *int64 `json:"suicidePenaltyS,omitempty"`
	BetrayalPenaltyS *int64 `json:"betrayalPenaltyS,omitempty"`
	FriendlyFire     *bool  `json:"friendlyFire,omitempty"`
}

// ClocksCoverage dit ce que la pose des horloges a lu et ce qu'elle a laisse de cote : trames lues,
// lectures de l'horloge publiees ou ecartees, comptes a rebours publies ou retenus (par cause).
type ClocksCoverage struct {
	Frames                  int `json:"frames"`
	FramesUnread            int `json:"framesUnread"`
	ClockReads              int `json:"clockReads"`
	ClockPublished          int `json:"clockPublished"`
	ClockImpossible         int `json:"clockImpossible"`
	ClockUnmeasuredState    int `json:"clockUnmeasuredState"`
	CountdownReads          int `json:"countdownReads"`
	CountdownImpossible     int `json:"countdownImpossible"`
	CountdownNoPlayer       int `json:"countdownNoPlayer"`
	CountdownIsolated       int `json:"countdownIsolated"`
	WithheldUnknownIndex    int `json:"withheldUnknownIndex"`
	WithheldReusedIndex     int `json:"withheldReusedIndex"`
	WithheldOutsidePresence int `json:"withheldOutsidePresence"`
	Countdowns              int `json:"countdowns"`
	StartDeduced            int `json:"startDeduced"`
	EndDeduced              int `json:"endDeduced"`
}
