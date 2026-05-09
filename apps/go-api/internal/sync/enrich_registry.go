// Package sync — enrich_registry.go : enrichissement post-Extract des
// MatchRegistryRow depuis metadata.asset_translations.
//
// Objectif : éviter d'écrire un UUID brut comme valeur de
// match_registry.playlist_name (ou map_name / pair_name / game_variant_name)
// quand l'API Halo Waypoint n'a pas retourné de nom localisé. ExtractRegistry
// fait un fallback `coalesceStrPtr(name, id)` historique qui écrit l'UUID en
// dernier ressort — ce helper court-circuite ce fallback en résolvant le nom
// canonique EN depuis asset_translations, populée en parallèle par
// cmd/populate-assets.
//
// Best-effort : si metadata est indisponible ou si l'asset n'a pas de
// traduction en-US, on retombe sur le comportement historique (UUID brut comme
// nom) sans casser la sync. Les UI consumers ont leur propre cascade de
// résolution (cf. analysis.ResolvePairNameFR + applyXxxFRTranslations).
//
// Cf. thought_log 2026-05-09 — root cause des UUIDs bruts visibles dans
// available_playlists.
package sync

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
)

// EnrichRegistryFromMetadata résout les noms d'assets (playlist, map, pair,
// game_variant) depuis metadata.asset_translations[lang='en-US'] quand le
// nom retourné par l'API Halo est vide ou égal à l'asset_id (cas du fallback
// UUID dans ExtractRegistry).
//
// Mute le row in-place. Best-effort : retourne nil sur erreur DB.
func EnrichRegistryFromMetadata(ctx context.Context, metadataDB *sql.DB, row *MatchRegistryRow) error {
	if metadataDB == nil || row == nil {
		return nil
	}

	// Liste des couples (asset_type, asset_id, name_field) à enrichir.
	type field struct {
		assetType string
		idPtr     *string
		namePtr   **string
	}
	fields := []field{
		{"playlist", row.PlaylistID, &row.PlaylistName},
		{"map", row.MapID, &row.MapName},
		{"pair", row.PairID, &row.PairName},
		{"game_variant", row.GameVariantID, &row.GameVariantName},
	}

	for _, f := range fields {
		if f.idPtr == nil {
			continue
		}
		assetID := strings.TrimSpace(*f.idPtr)
		if assetID == "" {
			continue
		}
		if !needsRegistryNameOverride(*f.namePtr, assetID) {
			continue
		}
		canonical, err := lookupAssetCanonicalEN(ctx, metadataDB, f.assetType, assetID)
		if err != nil {
			slog.WarnContext(ctx, "EnrichRegistryFromMetadata: lookup failed",
				"asset_type", f.assetType, "asset_id", assetID, "err", err)
			continue
		}
		if canonical == "" {
			continue
		}
		// Mute via le pointeur de pointeur — le row est modifié in-place.
		copied := canonical
		*f.namePtr = &copied
	}
	return nil
}

// needsRegistryNameOverride indique qu'un nom d'asset doit être ré-résolu
// depuis metadata. Cas couverts :
//   - Pointeur nil (champ absent du JSON brut)
//   - String vide (`coalesceStrPtr` n'a pas eu d'asset_id à substituer)
//   - String == asset_id (l'API n'a pas retourné de PublicName et le fallback
//     UUID a kické dans ExtractRegistry)
func needsRegistryNameOverride(name *string, assetID string) bool {
	if name == nil {
		return true
	}
	trimmed := strings.TrimSpace(*name)
	if trimmed == "" {
		return true
	}
	return strings.EqualFold(trimmed, assetID)
}

// lookupAssetCanonicalEN charge le nom canonique en-US depuis
// metadata.asset_translations pour un (asset_type, asset_id) donné.
// Retourne "" si absent — utile pour distinguer "pas de traduction" de
// "erreur DB".
func lookupAssetCanonicalEN(ctx context.Context, db *sql.DB, assetType, assetID string) (string, error) {
	var name sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT name FROM asset_translations
		WHERE asset_type = ? AND asset_id = ? AND lang = 'en-US'
		LIMIT 1`,
		assetType, assetID,
	).Scan(&name)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		// Table peut être absente sur fixtures de test → traiter comme "rien à faire".
		msg := err.Error()
		if strings.Contains(msg, "does not exist") || strings.Contains(msg, "Table with name") {
			return "", nil
		}
		return "", err
	}
	if !name.Valid {
		return "", nil
	}
	return strings.TrimSpace(name.String), nil
}
