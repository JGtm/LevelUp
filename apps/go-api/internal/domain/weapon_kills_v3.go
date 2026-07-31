// Package domain — weapon_kills_v3.go : struct neutre miroir de la table shadow
// shared.weapon_kills_v3 (pipeline d'attribution v3 film).
//
// **Localisation domain** : comme WeaponKillRow (match_rows.go) et ObjectiveEvent
// (objective_events.go), cette struct est consommée par plusieurs packages — le
// backfill diagnostic v3 (producteur) et la couche persistence
// (internal/platform/duckdb, WeaponKillsV3Repo). Zone neutre = pas de cycle d'import.
//
// Schéma : voir internal/migration/steps_shared_weapon_kills_v3.go. Les pointeurs
// modélisent les colonnes NULL-able (weapon_id/reconciled_as = pas d'arme attribuée ;
// high_weapon_id/killing_shot_hit/burst_final/shot_counter = signal v3 non disponible
// selon la source). Confidence/AttributionPath/SourceSignal tracent la provenance.

package domain

// WeaponKillV3Row représente une ligne de shared.weapon_kills_v3.
//
// WeaponID / ReconciledAs sont des UBIGINT côté DuckDB (hash filmshell bit63=1
// possible) — persistés via CAST(? AS UBIGINT) (cf. InsertWeaponKills). HighWeaponID
// est le high-32 de l'ID arme (UINTEGER). Les champs v3 (SourceSignal, HighWeaponID,
// KillingShotHit, BurstFinal, ShotCounter) tracent le signal de tir décodé du film.
type WeaponKillV3Row struct {
	TimeMS          int
	WeaponID        *uint64
	ReconciledAs    *uint64
	DeltaMS         *int
	Confidence      string
	AttributionPath string
	SwapDetected    bool
	DelayedDamage   bool
	PlayerIndex     *int
	SourceSignal    string
	HighWeaponID    *uint32
	KillingShotHit  *bool
	BurstFinal      *bool
	ShotCounter     *int
}
