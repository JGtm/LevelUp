package domain

// match_view_kill_distance.go — LOT G.3-POC (2026-08-30, plan
// .ai/PLAN_RETOURS_UTILISATEUR_2026-08-29.md §3bis, DEC-8) : kills mesurés et
// distance tueur-victime moyenne PAR ARME, PAR JOUEUR, pour UN match. Périmètre
// fermé par décision utilisateur (cadrage mot pour mot) : « mettre le nombre de
// kills par armes sur la distance et indiquer la distance moyenne pour chaque
// arme... pour chaque joueur. Pour le moment c'est tout ce qu'on va faire au
// niveau de la lecture de la distance. » — pas d'agrégat multi-matchs, pas de
// portée par arme (RRR), pas d'arme ni de distance côté ASSISTANT.
//
// Source : shared.kill_positions_latest × shared.match_kill_events_latest,
// jointes par (match_id, killer_xuid = feed_killer_xuid, time_ms) — voir
// platform/duckdb.KillDistanceRepo. La distance (hypot 3D tueur-victime) n'est
// JAMAIS stockée : calculée à la lecture, comme le veut G.0 (doctrine « on
// stocke une mesure, pas une résolution améliorable »).

// MatchKillDistanceWeapon — une arme, ses kills mesurés et sa distance
// tueur-victime pour UN joueur sur CE match.
type MatchKillDistanceWeapon struct {
	// WeaponKey : clé canonique du registre (résolue depuis source_tag via
	// port.KillSourceClassifier, même chaîne que le breakdown kills-hors-arme).
	WeaponKey string `json:"weapon_key"`
	// Label / LabelEN : FR-first / EN-first, même chaîne de résolution que
	// weapon_kills (weapon_resolver.go, table weapon_name_labels, lot V2.1).
	// Vide si la metadata n'est pas seedée — le front replie sur WeaponKey.
	Label   string `json:"label,omitempty"`
	LabelEN string `json:"label_en,omitempty"`
	// MeasuredKills : kills DE CETTE ARME dont la position tueur ET victime sont
	// TOUTES LES DEUX mesurées pour ce joueur sur ce match. Jamais le total de
	// kills à l'arme (weapon_kills) : la couverture positions est mesurée à
	// 75,8 % plancher (G.0) — ce compte est TOUJOURS ≤ au total réel.
	MeasuredKills int `json:"measured_kills"`
	// AvgDistanceM / MinDistanceM / MaxDistanceM : distance tueur-victime en
	// mètres monde (hypot 3D), calculée à la lecture — jamais stockée.
	AvgDistanceM float64 `json:"avg_distance_m"`
	MinDistanceM float64 `json:"min_distance_m"`
	MaxDistanceM float64 `json:"max_distance_m"`
}

// MatchKillDistancePlayer — le regroupement par joueur (xuid) des armes
// mesurées, pour un match. Gamertag délibérément ABSENT : le front le résout
// depuis le scoreboard déjà chargé (team_tab.scoreboard, même pattern que
// MatchObjectivesSection) plutôt que de le dupliquer ici.
type MatchKillDistancePlayer struct {
	XUID    string                    `json:"xuid"`
	Weapons []MatchKillDistanceWeapon `json:"weapons"`
}
