// Package duckdb — weapon_kills_v3_repo.go : write+read repo de la table shadow
// shared.weapon_kills_v3 (pipeline d'attribution v3 film).
//
// Source : pipeline diagnostic v3 (décodage film) qui produit des
// []domain.WeaponKillV3Row par (match, joueur). Ce repo les persiste / relit, EN
// PARALLÈLE de la v2 weapon_kills — jamais d'écriture croisée (table additive).
//
// Connexion : comme tous les writers/readers shared (cf. ObjectiveEventsRepo,
// WeaponKillsRepo + InsertWeaponKills), on passe par SharedReadDB().Get(ctx) qui
// retourne (ADR 0016) une connexion DIRECTE à shared_matches_v2.duckdb (tables à la
// racine du catalogue, PAS de préfixe `shared.`). En mode legacy / CLI backfill
// (SharedReader == LegacySharedReader), ce handle est RW : WriteMatch peut écrire
// dessus. Le backfill v3 tourne HORS chemin live (MaxOpenConns(1), sérialisé) — pas
// de pression concurrente ART (cf. .ai/PLAN_WEAPON_ATTRIBUTION_V3.md §10).
//
// Écriture par (match, joueur) : DELETE FROM weapon_kills_v3 WHERE match_id=? AND
// xuid=?, puis INSERT batch, dans une seule transaction — même pattern que
// InsertWeaponKills (DELETE-then-INSERT-par-match-joueur). UBIGINT (weapon_id,
// reconciled_as) injectés via string décimale + CAST(? AS UBIGINT) : le driver
// duckdb-go rejette les uint64 bit63=1 en arg direct (cf. InsertWeaponKills).
//
// rows == nil/[] est un no-op total (ni DELETE ni INSERT) : un décodage qui n'a rien
// produit ne doit pas effacer les rows déjà persistées (garde-fou anti-perte aligné
// sur InsertWeaponKills).
//
// Capability gating : si la table n'existe pas (titre sans capability film /
// migration non appliquée), DuckDB remonte "Table with name ... does not exist",
// intercepté via isTableNotFoundErr -> games.ErrCapabilityNotSupported.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// WeaponKillsV3Repo persiste et relit les attributions v3 dans la DB shared.
type WeaponKillsV3Repo struct {
	pdb *PlayerDB
}

// NewWeaponKillsV3Repo crée un WeaponKillsV3Repo lié à un PlayerDB.
func NewWeaponKillsV3Repo(pdb *PlayerDB) *WeaponKillsV3Repo {
	return &WeaponKillsV3Repo{pdb: pdb}
}

// WriteMatch remplace atomiquement TOUTES les attributions v3 d'un (match, joueur).
//
// Pattern DELETE-then-INSERT (cf. InsertWeaponKills) : DELETE WHERE match_id=? AND
// xuid=?, puis INSERT batch, dans une seule transaction. Ré-exécuter pour le même
// (matchID, xuid) écrase l'existant (idempotent).
//
// rows == nil/[] est un no-op total (garde-fou anti-perte). Retourne
// games.ErrCapabilityNotSupported si la table n'existe pas.
func (r *WeaponKillsV3Repo) WriteMatch(ctx context.Context, matchID, xuid string, rows []domain.WeaponKillV3Row) error {
	if len(rows) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return fmt.Errorf("WeaponKillsV3Repo.WriteMatch: shared reader: %w", err)
	}
	defer release()

	if err := writeWeaponKillsV3Tx(ctx, db, matchID, xuid, rows); err != nil {
		if isTableNotFoundErr(err) {
			return games.ErrCapabilityNotSupported
		}
		return fmt.Errorf("WeaponKillsV3Repo.WriteMatch(%s/%s): %w", matchID, xuid, err)
	}
	return nil
}

// writeWeaponKillsV3Tx exécute le DELETE+INSERT batch dans une transaction.
func writeWeaponKillsV3Tx(ctx context.Context, db *sql.DB, matchID, xuid string, rows []domain.WeaponKillV3Row) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM weapon_kills_v3 WHERE match_id = ? AND xuid = ?`, matchID, xuid); err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	for _, row := range rows {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO weapon_kills_v3 (
				match_id, xuid, time_ms, weapon_id, reconciled_as,
				delta_ms, confidence, attribution_path, swap_detected, delayed_damage,
				player_index, source_signal, high_weapon_id, killing_shot_hit,
				burst_final, shot_counter
			) VALUES (?, ?, ?, CAST(? AS UBIGINT), CAST(? AS UBIGINT), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			matchID, xuid, row.TimeMS, ubigintV3Arg(row.WeaponID), ubigintV3Arg(row.ReconciledAs),
			nullableInt(row.DeltaMS), row.Confidence, row.AttributionPath, row.SwapDetected, row.DelayedDamage,
			nullableInt(row.PlayerIndex), nullableString(row.SourceSignal), uint32V3Arg(row.HighWeaponID), nullableBool(row.KillingShotHit),
			nullableBool(row.BurstFinal), nullableInt(row.ShotCounter),
		); err != nil {
			return fmt.Errorf("insert time_ms=%d: %w", row.TimeMS, err)
		}
	}
	return tx.Commit()
}

// LoadMatch relit toutes les attributions v3 d'un match, ordonnées par (xuid, time_ms).
// Retourne games.ErrCapabilityNotSupported si la table n'existe pas.
func (r *WeaponKillsV3Repo) LoadMatch(ctx context.Context, matchID string) ([]domain.WeaponKillV3Row, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("WeaponKillsV3Repo.LoadMatch: shared reader: %w", err)
	}
	defer release()

	rows, err := loadWeaponKillsV3Rows(ctx, db, matchID)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, games.ErrCapabilityNotSupported
		}
		return nil, fmt.Errorf("WeaponKillsV3Repo.LoadMatch(%s): %w", matchID, err)
	}
	return rows, nil
}

// loadWeaponKillsV3Rows lit la table et scanne chaque colonne (UBIGINT via
// NullableUBigint, NULL-able via sql.Null*).
func loadWeaponKillsV3Rows(ctx context.Context, db *sql.DB, matchID string) ([]domain.WeaponKillV3Row, error) {
	dbRows, err := db.QueryContext(ctx, `
		SELECT time_ms, weapon_id, reconciled_as, delta_ms, confidence,
		       attribution_path, swap_detected, delayed_damage, player_index,
		       source_signal, high_weapon_id, killing_shot_hit, burst_final, shot_counter
		FROM weapon_kills_v3
		WHERE match_id = ?
		ORDER BY xuid ASC, time_ms ASC`, matchID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer dbRows.Close()

	var out []domain.WeaponKillV3Row
	for dbRows.Next() {
		row, err := scanWeaponKillV3Row(dbRows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := dbRows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}

// scanWeaponKillV3Row scanne une ligne en domain.WeaponKillV3Row.
func scanWeaponKillV3Row(dbRows *sql.Rows) (domain.WeaponKillV3Row, error) {
	var (
		row                               domain.WeaponKillV3Row
		weaponID, reconciledAs            NullableUBigint
		deltaMS, playerIndex, shotCounter sql.NullInt64
		highWeaponID                      sql.NullInt64
		conf, attrPath, sourceSignal      sql.NullString
		killingShotHit, burstFinal        sql.NullBool
	)
	if err := dbRows.Scan(&row.TimeMS, &weaponID, &reconciledAs, &deltaMS, &conf,
		&attrPath, &row.SwapDetected, &row.DelayedDamage, &playerIndex,
		&sourceSignal, &highWeaponID, &killingShotHit, &burstFinal, &shotCounter); err != nil {
		return row, fmt.Errorf("scan: %w", err)
	}
	row.WeaponID = uint64PtrFromNullableUBigint(weaponID)
	row.ReconciledAs = uint64PtrFromNullableUBigint(reconciledAs)
	row.DeltaMS = intPtrFromNull(deltaMS)
	row.Confidence = conf.String
	row.AttributionPath = attrPath.String
	row.PlayerIndex = intPtrFromNull(playerIndex)
	row.SourceSignal = sourceSignal.String
	row.HighWeaponID = uint32PtrFromNull(highWeaponID)
	row.KillingShotHit = boolPtrFromNull(killingShotHit)
	row.BurstFinal = boolPtrFromNull(burstFinal)
	row.ShotCounter = intPtrFromNull(shotCounter)
	return row, nil
}

// ubigintV3Arg sérialise un *uint64 en string décimale pour CAST(? AS UBIGINT), ou
// nil pour un NULL. Workaround driver duckdb-go (cf. InsertWeaponKills.ubigintArg).
func ubigintV3Arg(p *uint64) any {
	if p == nil {
		return nil
	}
	return strconv.FormatUint(*p, 10)
}

// uint32V3Arg convertit un *uint32 (colonne UINTEGER high_weapon_id) en arg SQL.
func uint32V3Arg(p *uint32) any {
	if p == nil {
		return nil
	}
	return int64(*p)
}

// nullableBool convertit un *bool en arg SQL (NULL si nil).
func nullableBool(p *bool) any {
	if p == nil {
		return nil
	}
	return *p
}

// uint64PtrFromNullableUBigint convertit un NullableUBigint en *uint64 (nil si NULL).
func uint64PtrFromNullableUBigint(n NullableUBigint) *uint64 {
	if !n.Valid {
		return nil
	}
	v := uint64(n.Value.Int64()) //nolint:gosec // bit-preserving reinterpret
	return &v
}

// uint32PtrFromNull convertit un sql.NullInt64 (UINTEGER) en *uint32 (nil si NULL).
func uint32PtrFromNull(n sql.NullInt64) *uint32 {
	if !n.Valid {
		return nil
	}
	v := uint32(n.Int64) //nolint:gosec // UINTEGER tient dans uint32
	return &v
}

// boolPtrFromNull convertit un sql.NullBool en *bool (nil si NULL).
func boolPtrFromNull(n sql.NullBool) *bool {
	if !n.Valid {
		return nil
	}
	v := n.Bool
	return &v
}
