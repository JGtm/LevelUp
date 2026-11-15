package duckdb

// replay_registry_repo.go : le registre des matchs vu par la mise en file d'un rejeu, derrière
// `port.ReplayRegistryRepo`. Lecture seule, courte ; le repo n'ouvre ni ne ferme rien (les handles sont
// relâchés par l'appelant AVANT le décodage).

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/platform/duckdb/sqlfrag"
	"levelup/go-api/internal/port"
)

var _ port.ReplayRegistryRepo = (*ReplayRegistryRepo)(nil)

// ReplayRegistryRepo lit match_registry (base partagée) et asset_translations (metadata).
type ReplayRegistryRepo struct {
	shared *sql.DB
	meta   *sql.DB
}

// NewReplayRegistryRepo crée le repo sur des connexions DÉJÀ ouvertes. meta peut être nil (dégradation :
// aucun nom EN de carte).
func NewReplayRegistryRepo(shared, meta *sql.DB) *ReplayRegistryRepo {
	return &ReplayRegistryRepo{shared: shared, meta: meta}
}

// MatchesByIDOrPrefix rend au plus trois matchs dont l'identifiant est matchID ou commence par lui (forme
// courte acceptée : match_registry est indexé par le match_id COMPLET).
func (r *ReplayRegistryRepo) MatchesByIDOrPrefix(ctx context.Context, matchID string) ([]port.ReplayRegistryMatch, error) {
	rows, err := r.shared.QueryContext(ctx,
		`SELECT match_id, map_name, map_id, `+sqlfrag.SQLModeNonGere("")+` FROM match_registry
		 WHERE match_id = ? OR match_id LIKE ? LIMIT 3`,
		matchID, matchID+"%")
	if err != nil {
		return nil, fmt.Errorf("lecture match_registry: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var hits []port.ReplayRegistryMatch
	for rows.Next() {
		var id string
		var rawName, mapID sql.NullString
		var nonGere bool
		if err := rows.Scan(&id, &rawName, &mapID, &nonGere); err != nil {
			return nil, fmt.Errorf("lecture match_registry (scan): %w", err)
		}
		hits = append(hits, port.ReplayRegistryMatch{ID: id, RawMapName: strings.TrimSpace(rawName.String),
			MapID: strings.TrimSpace(mapID.String), UnmanagedMode: nonGere})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hits, nil
}

// EnglishMapName rend le nom EN technique d'une carte (traduction d'asset dans la langue de repli des noms de
// jeu, i18n.Fallback, dont l'étiquette Halo est `en-US`) ; vide sans metadata. C'est la clé du catalogue de
// modules du rejeu : si la langue de repli de config/locales.toml cessait d'être l'anglais, cette clé changerait.
func (r *ReplayRegistryRepo) EnglishMapName(ctx context.Context, mapID string) (string, error) {
	if r.meta == nil {
		return "", nil
	}
	var en string
	if err := r.meta.QueryRowContext(ctx,
		`SELECT name FROM asset_translations WHERE asset_type = 'map' AND asset_id = ? AND lang = ?`,
		mapID, i18n.HaloTags(i18n.Fallback)[0]).Scan(&en); err != nil {
		return "", err
	}
	return en, nil
}

// UnmanagedModeMatchIDs rend les identifiants complets des matchs refusés par règle : le MÊME prédicat
// (sqlfrag.SQLModeNonGere, lu au registre) que le refus de la cuisson.
func (r *ReplayRegistryRepo) UnmanagedModeMatchIDs(ctx context.Context) ([]string, error) {
	rows, err := r.shared.QueryContext(ctx, `SELECT match_id FROM match_registry WHERE `+sqlfrag.SQLModeNonGere(""))
	if err != nil {
		return nil, fmt.Errorf("registre des matchs (modes non gérés): %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("registre des matchs (modes non gérés, scan): %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
