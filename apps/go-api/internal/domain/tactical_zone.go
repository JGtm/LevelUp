package domain

// tactical_zone.go — CE QUE LE DÉTAIL D'UNE ZONE DU PLAN TACTIQUE PUBLIE EN PLUS DE SES
// CONTRIBUTIONS : le nom en jeu de la zone et le placement d'une mort (seul ou près d'un
// coéquipier) ; et le contexte de mort que la lecture lit pour ce placement.

// TacticalZoneNom est le nom en JEU d'une zone (catalogue des callouts), dans les deux langues :
// un nom de lieu du jeu, que l'i18n du produit ne connaît pas et ne peut pas traduire. Absent de
// la réponse (nil) quand aucune zone ne nomme la cellule — le web écrit alors « Zone sans nom ».
type TacticalZoneNom struct {
	NomFR string `json:"nom_fr"`
	NomEN string `json:"nom_en"`
}

// TacticalPlacement dit, pour une MORT, si la victime était près d'un coéquipier ou seule, à
// l'instant de la mort (contexte de `match_death_context`, portée du radar du match).
type TacticalPlacement struct {
	// Seul : aucun coéquipier visible à portée (borne de portée INCLUSIVE : à la portée, c'est
	// « près »).
	Seul bool `json:"seul"`
	// DistanceM : la distance au coéquipier VISIBLE le plus proche, en mètres. Absente (nil)
	// quand aucun coéquipier n'était visible — une absence de mesure, jamais une distance
	// infinie : la mort est alors « seule », sans distance.
	DistanceM *float64 `json:"distance_m,omitempty"`
}

// ContexteDeMort est une ligne de `match_death_context_latest` : le voisinage d'UNE mort, mesuré
// au sync par le collecteur de kills, sur l'horloge du MATCH.
type ContexteDeMort struct {
	MatchID    string
	VictimXUID string
	TimeMs     int64
	// PlusProcheM : distance au coéquipier VISIBLE le plus proche ; nil = aucun visible.
	PlusProcheM *float64
	// Visibles, HorsDeVue : les coéquipiers vivants, vus et non vus, à cet instant.
	Visibles, HorsDeVue int
}
