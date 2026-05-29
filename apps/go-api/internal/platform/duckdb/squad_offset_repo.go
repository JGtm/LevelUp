// Package duckdb — squad_offset_repo.go : accès player_squad_offset (LUSR v2
// Sprint 1.C). Table append-only ; lecture via la vue player_squad_offset_latest,
// écriture par INSERT pur (jamais d'UPDATE/ON CONFLICT — anti-ART, cf. ADR 0019).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/domain"
)

// SquadOffsetRepo encapsule l'accès à player_squad_offset dans
// shared_matches_v2.duckdb. Reçoit *sql.DB directement (même convention que
// SkillV2Repo).
//
// Un cache run-scoped (map[xuid|group] → map[partner]offset) évite de re-requêter
// les mêmes offsets pour chaque match d'un même joueur pendant un run shadow.
// Les offsets ne changent pas pendant un run, donc le cache est toujours valide.
type SquadOffsetRepo struct {
	shared *sql.DB
	cache  map[string]map[string]float64
}

// NewSquadOffsetRepo construit un SquadOffsetRepo. Le cache est initialisé vide.
func NewSquadOffsetRepo(shared *sql.DB) *SquadOffsetRepo {
	return &SquadOffsetRepo{shared: shared, cache: make(map[string]map[string]float64)}
}

// LoadSquadOffsets retourne les offsets de synergie de `xuid` sur `group`,
// indexés par partner_xuid. Map vide si aucun offset enregistré. Mémoïsé.
func (r *SquadOffsetRepo) LoadSquadOffsets(ctx context.Context, xuid, playlistGroup string) (map[string]float64, error) {
	key := xuid + "|" + playlistGroup
	if cached, ok := r.cache[key]; ok {
		return cached, nil
	}
	rows, err := r.shared.QueryContext(ctx, `
		SELECT partner_xuid, offset_value
		FROM player_squad_offset_latest
		WHERE xuid = ? AND playlist_group = ?`,
		xuid, playlistGroup)
	if err != nil {
		return nil, fmt.Errorf("SquadOffsetRepo.LoadSquadOffsets(%s, %s): %w", xuid, playlistGroup, err)
	}
	defer rows.Close() //nolint:errcheck

	out := make(map[string]float64)
	for rows.Next() {
		var partner string
		var offset float64
		if err := rows.Scan(&partner, &offset); err != nil {
			return nil, fmt.Errorf("SquadOffsetRepo.LoadSquadOffsets scan: %w", err)
		}
		out[partner] = offset
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	r.cache[key] = out
	return out, nil
}

// UpsertSquadOffset écrit un nouveau snapshot d'offset (INSERT pur append-only).
// La vue _latest dédoublonne par (xuid, partner_xuid, playlist_group).
func (r *SquadOffsetRepo) UpsertSquadOffset(ctx context.Context, o domain.SquadOffset) error {
	_, err := r.shared.ExecContext(ctx, `
		INSERT INTO player_squad_offset
			(xuid, partner_xuid, playlist_group, offset_value, match_count, source)
		VALUES (?, ?, ?, ?, ?, ?)`,
		o.XUID, o.PartnerXUID, o.PlaylistGroup, o.OffsetValue, o.MatchCount, o.Source,
	)
	if err != nil {
		return fmt.Errorf("SquadOffsetRepo.UpsertSquadOffset(%s/%s/%s): %w",
			o.XUID, o.PartnerXUID, o.PlaylistGroup, err)
	}
	return nil
}
