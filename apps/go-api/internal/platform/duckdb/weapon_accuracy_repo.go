// Package duckdb — weapon_accuracy_repo.go : implémentation DuckDB du loader
// agrégat weapon_accuracy (port.WeaponAccuracyRepository).
//
// Source : table weapon_accuracy (1 row par (match, xuid, weapon) effectivement
// tirée — alimentée par halo_5/ingest/weapon_accuracy.go depuis les events
// weapon_drop natifs). Agrégation côté DB via GROUP BY (xuid, weapon_id) +
// SUM(shots_fired)/SUM(shots_landed).
//
// ADR 0016 : queries exécutées via SharedReadDB().Get(ctx) — connexion directe au
// fichier shared_matches_v2.duckdb du titre, pas de préfixe `shared.` (l'ATTACH a
// été retiré ; tables/vues dans le schéma `main`).
//
// Capability gating : si la table weapon_accuracy est absente (titre qui ne
// peuple pas cette donnée, ex. Halo Infinite), DuckDB remonte "Table with name
// ... does not exist" → interceptée via isTableNotFoundErr et convertie en
// games.ErrCapabilityNotSupported (dégradation gracieuse côté service).
//
// Labels EN/FR : résolus en post-traitement Go via resolveWeaponMeta (registre +
// weapon_labels de la metadata du titre) — la metadata DB est séparée, pas de
// jointure SQL pure sans ATTACH. Calqué sur WeaponKillsRepo.attachWeaponMeta.
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// WeaponAccuracyRepo implémente port.WeaponAccuracyRepository.
type WeaponAccuracyRepo struct {
	pdb *PlayerDB
}

// NewWeaponAccuracyRepo crée un WeaponAccuracyRepo lié à un PlayerDB.
func NewWeaponAccuracyRepo(pdb *PlayerDB) *WeaponAccuracyRepo {
	return &WeaponAccuracyRepo{pdb: pdb}
}

// LoadWeaponAccuracyAggregated charge la précision agrégée par (xuid, weapon_id).
//
// L'appelant DOIT avoir validé les filtres ; le repo re-valide en défense.
// Retourne games.ErrCapabilityNotSupported si weapon_accuracy n'existe pas dans
// la DB cible.
func (r *WeaponAccuracyRepo) LoadWeaponAccuracyAggregated(
	ctx context.Context,
	slug string,
	filters port.WeaponAccuracyFilters,
) ([]port.WeaponAccuracyRow, error) {
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("WeaponAccuracyRepo.LoadWeaponAccuracyAggregated: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := r.queryWeaponAccuracy(ctx, filters)
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "WeaponAccuracyRepo: weapon_accuracy missing",
				"slug", slug, "match_count", len(filters.MatchIDs))
			return nil, games.ErrCapabilityNotSupported
		}
		slog.ErrorContext(ctx, "WeaponAccuracyRepo: query failed",
			"slug", slug, "match_count", len(filters.MatchIDs), "err", err)
		return nil, fmt.Errorf("WeaponAccuracyRepo.LoadWeaponAccuracyAggregated: %w", err)
	}

	r.attachWeaponLabels(ctx, slug, rows)
	return rows, nil
}

// queryWeaponAccuracy exécute le SELECT agrégé (xuid, weapon_id, Σ shots_fired,
// Σ shots_landed) sur weapon_accuracy, filtré par MatchIDs + xuid.
func (r *WeaponAccuracyRepo) queryWeaponAccuracy(
	ctx context.Context,
	filters port.WeaponAccuracyFilters,
) ([]port.WeaponAccuracyRow, error) {
	q, args := buildWeaponAccuracyQuery(filters)
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	dbRows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer dbRows.Close()

	var out []port.WeaponAccuracyRow
	for dbRows.Next() {
		var (
			xuid     string
			weaponID UBigint // UBIGINT côté DuckDB (cf. ubigint_scanner.go)
			fired    int
			landed   int
		)
		if err := dbRows.Scan(&xuid, &weaponID, &fired, &landed); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, port.WeaponAccuracyRow{
			XUID:        xuid,
			WeaponID:    weaponID.Int64(),
			ShotsFired:  fired,
			ShotsLanded: landed,
		})
	}
	if err := dbRows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}

// buildWeaponAccuracyQuery compose le SELECT agrégé sur weapon_accuracy.
//
// Filtres :
//   - MatchIDs (requis) → wa.match_id IN (?,...)
//   - Gamertag XOR XUIDs → filtre sur wa.xuid (résolution gamertag→xuid via
//     xuid_aliases si Gamertag fourni), réutilise appendXUIDFilter.
//
// weapon_id reste en UBIGINT côté SQL (pas de cast ::BIGINT) car certaines armes
// (filmshell hashes bit63=1) ont des IDs hors INT64. Le scan Go capture en uint64
// puis réinterprète bit-à-bit en int64 (cf. UBigint.Int64).
func buildWeaponAccuracyQuery(f port.WeaponAccuracyFilters) (string, []any) {
	var sb strings.Builder
	args := make([]any, 0, len(f.MatchIDs)+len(f.XUIDs)+1)

	matchPlaceholders := Placeholders(len(f.MatchIDs))
	for _, id := range f.MatchIDs {
		args = append(args, id)
	}

	sb.WriteString(`
SELECT
    wa.xuid,
    wa.weapon_id,
    SUM(wa.shots_fired)::INTEGER  AS shots_fired,
    SUM(wa.shots_landed)::INTEGER AS shots_landed
FROM weapon_accuracy wa
WHERE wa.match_id IN (`)
	sb.WriteString(matchPlaceholders)
	sb.WriteString(`)`)

	// appendXUIDFilter attend un port.WeaponKillFilters (Gamertag/XUIDs). On
	// projette WeaponAccuracyFilters dessus pour réutiliser le helper unique
	// (résolution gamertag→xuid via xuid_aliases identique).
	appendXUIDFilter(&sb, &args, "wa", port.WeaponKillFilters{
		Gamertag: f.Gamertag,
		XUIDs:    f.XUIDs,
	})

	sb.WriteString(`
GROUP BY wa.xuid, wa.weapon_id`)
	return sb.String(), args
}

// attachWeaponLabels renseigne le Label EN/FR, la Class ET le Role (registre) par weapon_id
// via resolveWeaponMeta (registre + weapon_labels). Best-effort : meta absent → no-op (Label/
// Class/Role vides, le service filtre). La Class sert à écarter du graphe précision les classes
// sans précision pertinente (grenade/mêlée/capacités) ; le Role sert à AGRÉGER PAR RÔLE la
// précision de l'Escouade (regroupement des ~30 armes par rôle pour la lisibilité).
func (r *WeaponAccuracyRepo) attachWeaponLabels(ctx context.Context, slug string, rows []port.WeaponAccuracyRow) {
	if r.pdb == nil || r.pdb.Metadata == nil || len(rows) == 0 {
		return
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.WeaponID)
	}
	meta := resolveWeaponMeta(ctx, r.pdb.Metadata, slug, ids)
	for i := range rows {
		m, ok := meta[rows[i].WeaponID]
		if !ok {
			continue
		}
		if m.label != "" {
			rows[i].Label = m.label
		}
		rows[i].Class = m.class
		rows[i].Role = m.role
	}
}
