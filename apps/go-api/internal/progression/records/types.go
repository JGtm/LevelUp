// Package records — types et constantes pour les Personal Records (PB).
//
// On track le PB de chaque métrique sur 3 fenêtres glissantes : 30 jours
// (forme actuelle), 90 jours (tendance moyen terme), et all-time (carrière).
//
// Le mot « fenêtre » est porté par la colonne `period` côté DB (pas `window`,
// qui est un mot réservé DuckDB). Domaine inchangé : période temporelle de
// référence pour le PB.
//
// Persistance :
//   - table `player_records` dans shared_social.duckdb (étendue avec period,
//     previous_value, previous_achieved_at — cf. migration
//     `extend_player_records_with_window`)
//   - table `record_history` dans stats.duckdb (timeline chronologique
//     pour l'affichage en page profil)
package records

import "time"

// RecordPeriod identifie la fenêtre temporelle d'un PB.
type RecordPeriod string

const (
	// RecordPeriod30d : forme actuelle, fenêtre glissante de 30 jours.
	RecordPeriod30d RecordPeriod = "30d"
	// RecordPeriod90d : tendance moyen terme, fenêtre glissante de 90 jours.
	RecordPeriod90d RecordPeriod = "90d"
	// RecordPeriodAllTime : record carrière, sans limite temporelle.
	// Valeur par défaut pour les lignes existantes avant la migration period.
	RecordPeriodAllTime RecordPeriod = "all_time"
)

// AllRecordPeriods liste les périodes supportées.
func AllRecordPeriods() []RecordPeriod {
	return []RecordPeriod{RecordPeriod30d, RecordPeriod90d, RecordPeriodAllTime}
}

// MinMatchesForRecord est le nombre minimum de matchs dans la fenêtre avant
// d'enregistrer un PB. Évite les faux positifs sur petit échantillon
// (un match exceptionnel ne fait pas un PB).
const MinMatchesForRecord = 10

// NearMissRatio est la distance relative au PB considérée comme "proche".
// Si valeur courante >= PB × (1 - NearMissRatio), on émet une notif near-miss.
const NearMissRatio = 0.05

// NearMissMinGapRatio (DP11) est l'écart relatif MINIMAL sous le PB pour qu'un
// near-miss soit significatif pour un joueur. Sous ce seuil (value trop proche
// du PB), la notif est du bruit : incident prod 2026-07-03 « 73.33 vs 73.333336 »
// rendu « 73.33 vs 73.33 ». La bande d'émission est donc
// PB×(1-NearMissRatio) <= value <= PB×(1-NearMissMinGapRatio). 2 % ≈ 0.1 de KDA
// sur un PB à 5.0, ~1.5 pt d'accuracy sur un PB à 73. Relatif (pas d'absolu :
// insignifiant sur toutes les échelles).
const NearMissMinGapRatio = 0.02

// PersonalRecord représente le meilleur score d'une métrique sur une période.
//
// Stocké dans `player_records` (shared_social.duckdb). La clé primaire est
// (XUID, Metric, Period) — plusieurs périodes possibles pour la même métrique.
type PersonalRecord struct {
	XUID               string       `json:"xuid"`
	Metric             string       `json:"metric"`
	Period             RecordPeriod `json:"period"`
	Value              float64      `json:"value"`
	AchievedAt         *time.Time   `json:"achieved_at,omitempty"`
	AchievedMatchID    string       `json:"achieved_match_id,omitempty"`
	PreviousValue      *float64     `json:"previous_value,omitempty"`
	PreviousAchievedAt *time.Time   `json:"previous_achieved_at,omitempty"`
	UpdatedAt          time.Time    `json:"updated_at"`
}

// RecordHistory est une entrée de timeline pour un PB battu.
//
// Stocké dans `record_history` (stats.duckdb par joueur). Une nouvelle ligne
// est créée à chaque fois qu'un PB est dépassé — historique append-only.
type RecordHistory struct {
	ID         string       `json:"id"`
	UserID     string       `json:"user_id"`
	TitleSlug  string       `json:"title_slug"`
	Metric     string       `json:"metric"`
	Period     RecordPeriod `json:"period"`
	Value      float64      `json:"value"`
	AchievedAt time.Time    `json:"achieved_at"`
}
