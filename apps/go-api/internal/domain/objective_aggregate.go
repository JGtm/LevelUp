package domain

// ObjectiveAggregate — cumul des stats objectifs (CTF/Zones/Oddball) d'un joueur
// sur un ENSEMBLE de matchs (agrégat à la lecture, SUM sur match_objective_stats_latest).
// Sert les KPI « Objectifs » sobres des pages Synthèse et Escouade. Un scope peut
// mêler plusieurs modes → plusieurs blocs non nuls (le front n'affiche que les KPI
// pertinents, valeur > 0). Durées en secondes (cumul). Tous les champs omitempty :
// une valeur zéro disparaît du JSON (le front la traite comme absente / non pertinente).
type ObjectiveAggregate struct {
	// CTF (cumul)
	FlagCaptures       int     `json:"flag_captures,omitempty"`
	FlagReturns        int     `json:"flag_returns,omitempty"`
	FlagSteals         int     `json:"flag_steals,omitempty"`
	FlagCarrierSeconds float64 `json:"flag_carrier_seconds,omitempty"`
	// Zones (cumul)
	ZoneCaptures int     `json:"zone_captures,omitempty"`
	ZoneSecures  int     `json:"zone_secures,omitempty"`
	ZoneSeconds  float64 `json:"zone_seconds,omitempty"`
	// Oddball (cumul)
	SkullGrabs          int     `json:"skull_grabs,omitempty"`
	SkullCarrierSeconds float64 `json:"skull_carrier_seconds,omitempty"`
}

// IsEmpty indique qu'aucune stat objectif n'est présente (tous les cumuls à zéro)
// → le caller sert nil (bloc omis, section masquée côté front).
func (o ObjectiveAggregate) IsEmpty() bool {
	return o.FlagCaptures == 0 && o.FlagReturns == 0 && o.FlagSteals == 0 &&
		o.FlagCarrierSeconds == 0 && o.ZoneCaptures == 0 && o.ZoneSecures == 0 &&
		o.ZoneSeconds == 0 && o.SkullGrabs == 0 && o.SkullCarrierSeconds == 0
}
