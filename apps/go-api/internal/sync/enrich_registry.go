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

	"levelup/go-api/internal/games"
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
		{games.AssetKindPlaylist, row.PlaylistID, &row.PlaylistName},
		{games.AssetKindMap, row.MapID, &row.MapName},
		{games.AssetKindPair, row.PairID, &row.PairName},
		{games.AssetKindGameVariant, row.GameVariantID, &row.GameVariantName},
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

	// Fallback ultime pour la PAIRE : si après lookup asset_translations le
	// pair_name reste un GUID (== pair_id), on le CONSTRUIT depuis le mode
	// (game_variant) + la map, déjà résolus dans la boucle ci-dessus. Le nom
	// public Halo d'une paire EST littéralement "{Mode} on {Map}" (ex "Slayer on
	// Chasm"), et NormalizeModeLabel sait stripper le " on {map}". Sans ça, le
	// nouveau contenu (maps inédites → paires absentes du catalogue) exposait un
	// GUID brut comme libellé de mode tant que le drain catalogue n'avait pas
	// tourné. Pur, zéro HTTP — robuste même si le catalogue est en retard.
	if row.PairID != nil && needsRegistryNameOverride(row.PairName, *row.PairID) {
		if constructed, ok := constructPairName(
			row.GameVariantName, row.GameVariantID, row.MapName, row.MapID,
		); ok {
			row.PairName = &constructed
			row.ModeCategory = determineModeCategory(constructed)
		}
	}
	return nil
}

// constructPairName fabrique un nom de paire synthétique "{mode} on {map}" à
// partir du game_variant et de la map, quand l'API/le catalogue n'a pas fourni
// le nom public de la paire. Retourne ("", false) si l'un des deux est absent
// ou encore non résolu (== son asset_id brut) — on ne fabrique jamais un nom à
// partir d'un GUID.
func constructPairName(gameVariantName, gameVariantID, mapName, mapID *string) (string, bool) {
	gv := resolvedRegistryName(gameVariantName, gameVariantID)
	mp := resolvedRegistryName(mapName, mapID)
	if gv == "" || mp == "" {
		return "", false
	}
	return gv + " on " + mp, true
}

// resolvedRegistryName retourne le nom trimé s'il s'agit d'un vrai nom (non vide
// et différent de l'asset_id), sinon "". Sert à ne construire un libellé qu'à
// partir de valeurs réellement résolues.
func resolvedRegistryName(name, id *string) string {
	if name == nil {
		return ""
	}
	n := strings.TrimSpace(*name)
	if n == "" {
		return ""
	}
	if id != nil && strings.EqualFold(n, strings.TrimSpace(*id)) {
		return ""
	}
	return n
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
