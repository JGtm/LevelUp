// Package domain — synthesis_weapon_records.go : contrat de la section « Records de distance
// par arme » de la Synthèse (plan .ai/V7.5/PLAN_RECORDS_DISTANCE_2026-09-20.md).
//
// CE QUE LA SECTION PUBLIE : pour chaque arme, LE frag mesuré le plus lointain — identifié
// (match, instant) pour que le joueur aille le revoir — avec la médiane de l'arme comme repère
// d'habitude et l'effectif pour peser le record. Aucun seuil d'effectif (décision utilisateur
// du 2026-09-20). Aucun libellé produit ne dit « portée de l'arme » : c'est un usage mesuré.
//
// CE QUI EST ÉCARTÉ EST NOMMÉ : les classes d'arme pour lesquelles une distance n'a pas de
// sens (corps à corps, chute et environnement, équipement) sortent de la règle mais restent
// dans `Excluded`, avec leur effectif, jamais tues en silence (D9).
package domain

import "time"

// SynthesisWeaponRecords — la section entière.
type SynthesisWeaponRecords struct {
	// Weapons : une ligne par arme retenue, triée par record CROISSANT (ordre imposé côté Go,
	// jamais rejoué côté web). Toujours sérialisée, jamais `null` : le front itère sans garde.
	Weapons []WeaponDistanceRecordRow `json:"weapons"`
	// MeasuredKills / TotalKills : la couverture. Mesurés = frags du joueur dont les deux
	// positions sont décodées (classes écartées COMPRISES : elles sont mesurées, seulement pas
	// tracées) ; total = frags du scope canonique. Même doctrine que la portée : le dénominateur
	// est ce que le joueur a fait, le numérateur ce que le décodeur a su placer.
	MeasuredKills int `json:"measured_kills"`
	TotalKills    int `json:"total_kills"`
	// Excluded nomme les armes écartées par classe, avec leur effectif, par effectif décroissant.
	Excluded []WeaponExcludedFromRecords `json:"excluded,omitempty"`
}

// WeaponDistanceRecordRow : une arme et son record.
type WeaponDistanceRecordRow struct {
	// WeaponKey est toujours renseignée : c'est le repli d'affichage quand la metadata n'a pas
	// de libellé pour la clé.
	WeaponKey string `json:"weapon_key"`
	Label     string `json:"label,omitempty"`
	LabelEN   string `json:"label_en,omitempty"`
	// Class est la classe de registre (épaule, poing, lourde, grenade...) : elle porte la
	// couleur côté web (famille `frag-*`, la même que le sunburst voisin). Vide si le registre
	// ne connaît pas la clé — le web retombe sur la couleur neutre, jamais une classe inventée.
	Class string `json:"class,omitempty"`
	// Measured est le nombre de frags mesurés de l'arme ; MedianM leur médiane en mètres.
	Measured int     `json:"measured"`
	MedianM  float64 `json:"median_m"`
	// RecordM est la distance du frag le plus lointain, en mètres ; Record l'identifie.
	RecordM float64          `json:"record_m"`
	Record  WeaponRecordFrag `json:"record"`
}

// WeaponRecordFrag identifie LE frag du record et le situe : c'est ce qui le rend vérifiable.
type WeaponRecordFrag struct {
	MatchID string `json:"match_id"`
	// TimeMS est l'instant du frag sur l'HORLOGE DU MATCH (`match_kill_events.time_ms`, cf.
	// TacticalClockMatch) : le web ouvre le rejeu avec `?t=<time_ms>&clock=match`, et c'est la
	// route du rejeu qui recale sur l'axe du film une fois le document chargé.
	TimeMS int64 `json:"time_ms"`
	// StartedAt, MapLabel, MapLabelEN viennent du scope canonique déjà chargé par la page
	// (aucune requête neuve). Omis si le match n'y figure pas.
	StartedAt  *time.Time `json:"started_at,omitempty"`
	MapLabel   string     `json:"map_label,omitempty"`
	MapLabelEN string     `json:"map_label_en,omitempty"`
}

// WeaponExcludedFromRecords : une arme écartée de la règle, nommée avec sa classe et son effectif.
type WeaponExcludedFromRecords struct {
	WeaponKey string `json:"weapon_key"`
	Label     string `json:"label,omitempty"`
	LabelEN   string `json:"label_en,omitempty"`
	Class     string `json:"class"`
	Measured  int    `json:"measured"`
}
