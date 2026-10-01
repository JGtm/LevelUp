package domain

// squad_emprise_placement.go — LE BLOC « GROUPÉS OU ISOLÉS » de l'onglet Emprise (plan
// `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`, lot V3, décisions V3, V4, V5, V8 ; rendu : §2 du plan).
//
// Une vie = une ligne de `match_life_placement_latest`, écrite AU SYNC par le collecteur de kills.
// Chaque vie MESURÉE de la composition (le joueur de la page et les coéquipiers sélectionnés,
// dans l'ordre des fiches de l'Emprise ; le reste du camp n'est pas tracé) porte deux
// coordonnées :
//
//   - X : la distance médiane au coéquipier vivant le plus proche pendant la vie, rapportée à la
//     portée du radar du match (RadarRatio ; 1,0 = « à la portée du radar ») ;
//   - Y : les frags de la vie (Kills).
//
// Et UN quart, tranché ICI et jamais par le client (décision V4) : isolé = RadarRatio ≥
// IsolatedFromRatio ; rentable = Kills ≥ ProductiveFromKills. Une vie non mesurée (moins de
// 2 000 ms mesurées) n'est ni tracée ni classée : elle est comptée (Coverage).

// Les quatre quarts du placement, dans l'ordre de publication (barre « Part des vies par
// placement » : à portée et rentable, isolé et rentable, à portée et coûteux, isolé et coûteux).
const (
	EmprisePlacementInRangeProductive  = "in_range_productive"
	EmprisePlacementIsolatedProductive = "isolated_productive"
	EmprisePlacementInRangeCostly      = "in_range_costly"
	EmprisePlacementIsolatedCostly     = "isolated_costly"
)

// EmprisePlacementIsolatedFromRatio : une vie est ISOLÉE quand sa distance médiane au coéquipier
// le plus proche atteint la portée du radar (décision V4, borne comprise).
const EmprisePlacementIsolatedFromRatio = 1.0

// EmprisePlacementProductiveFromKills : une vie est RENTABLE à partir d'un frag (décision V4).
const EmprisePlacementProductiveFromKills = 1

// SquadEmprisePlacement — le bloc publié (SquadEmpriseBlock.Placement). Absent quand le titre ne
// mesure pas les vies (capability `film.kill_positions`), quand la lecture a échoué, ou quand
// aucun match du périmètre n'a de vie écrite pour la composition : une OMISSION, jamais un
// nuage vide qui se lirait comme une mesure.
type SquadEmprisePlacement struct {
	// IsolatedFromRatio / ProductiveFromKills republient les deux seuils des quarts : le client
	// trace le repère du radar et la frontière des frags sans les coder en dur.
	IsolatedFromRatio   float64 `json:"isolated_from_ratio"`
	ProductiveFromKills int     `json:"productive_from_kills"`
	// Players : un par joueur de la composition, dans l'ordre des fiches de l'Emprise
	// (SquadEmpriseBlock.Players), même sans vie mesurée.
	Players  []SquadEmprisePlacementPlayer `json:"players"`
	Coverage SquadEmprisePlacementCoverage `json:"coverage"`
}

// SquadEmprisePlacementPlayer — un joueur : ses vies tracées, son gros point, ses quarts.
type SquadEmprisePlacementPlayer struct {
	XUID     string `json:"xuid"`
	Gamertag string `json:"gamertag"`
	// LivesTotal : ses vies retenues (matchs à portée courante connue, lignes non périmées),
	// mesurées ou non. LivesMeasured : celles qui sont tracées et classées — le « N vies » des
	// infobulles du gros point et de la barre, et la taille du gros point.
	LivesTotal    int `json:"lives_total"`
	LivesMeasured int `json:"lives_measured"`
	// MedianRadarRatio / MedianKills : le gros point (médianes sur ses vies mesurées). Nil sans
	// vie mesurée : pas de gros point.
	MedianRadarRatio *float64 `json:"median_radar_ratio,omitempty"`
	MedianKills      *float64 `json:"median_kills,omitempty"`
	// Quadrants : les quatre quarts, toujours présents et dans l'ordre de publication.
	Quadrants []SquadEmprisePlacementQuadrant `json:"quadrants"`
	// Lives : ses vies mesurées, dans l'ordre chronologique du périmètre (match, puis début).
	Lives []SquadEmprisePlacementLife `json:"lives"`
}

// SquadEmprisePlacementQuadrant — un quart d'un joueur.
type SquadEmprisePlacementQuadrant struct {
	Quadrant string `json:"quadrant" enum:"in_range_productive,isolated_productive,in_range_costly,isolated_costly"`
	Lives    int    `json:"lives"`
	// Share : Lives / LivesMeasured du joueur, unité 0..1 (ADR 0006). Nil sans vie mesurée.
	Share *float64 `json:"share,omitempty"`
}

// SquadEmprisePlacementLife — une vie tracée (un petit point du nuage).
type SquadEmprisePlacementLife struct {
	// MatchID / StartMS : la clé de la vie avec le joueur (même clé que `match_lives`) ; le
	// client en dérive un décalage vertical déterministe.
	MatchID    string `json:"match_id"`
	StartMS    int64  `json:"start_ms"`
	DurationMS int64  `json:"duration_ms"`
	// RadarRatio : distance médiane / portée du radar du match, valeur vraie (le client la pose
	// à 2 au-delà, l'infobulle garde celle-ci).
	RadarRatio float64 `json:"radar_ratio"`
	// OutOfRadarShare : part du temps mesuré hors de portée du radar, unité 0..1.
	OutOfRadarShare float64 `json:"out_of_radar_share"`
	Kills           int     `json:"kills"`
	Quadrant        string  `json:"quadrant" enum:"in_range_productive,isolated_productive,in_range_costly,isolated_costly"`
}

// SquadEmprisePlacementCoverage — ce que le bloc a mesuré, et ce qu'il a écarté, pour la
// composition sur le périmètre D2 (l'infobulle de la carte).
type SquadEmprisePlacementCoverage struct {
	// MatchesTotal : les matchs du périmètre. MatchesWithPlacement : ceux qui portent au moins une
	// vie écrite pour la composition.
	MatchesTotal         int `json:"matches_total"`
	MatchesWithPlacement int `json:"matches_with_placement"`
	// MatchesWithoutRange : parmi eux, ceux dont la variante n'a pas de portée de radar connue
	// AUJOURD'HUI : ils sortent de l'univers, leurs vies ne sont ni retenues ni comptées ailleurs.
	MatchesWithoutRange int `json:"matches_without_range"`
	// StaleLives : vies écartées parce que la portée écrite au sync diffère de la portée courante
	// de la variante (la table a changé depuis : un rattrapage est dû).
	StaleLives int `json:"stale_lives"`
	// LivesTotal = LivesMeasured + LivesUnmeasured : les vies retenues de la composition.
	LivesTotal      int `json:"lives_total"`
	LivesMeasured   int `json:"lives_measured"`
	LivesUnmeasured int `json:"lives_unmeasured"`
	// Cumuls en ms des vies retenues, par cause (grille de 100 ms bornes incluses, décision V3) :
	// le temps mesuré, puis chaque cause d'exclusion de la mesure.
	MeasuredMS         int64 `json:"measured_ms"`
	CarrierMS          int64 `json:"carrier_ms"`
	TeamDownMS         int64 `json:"team_down_ms"`
	UnplacedMS         int64 `json:"unplaced_ms"`
	TeammateUnplacedMS int64 `json:"teammate_unplaced_ms"`
}
