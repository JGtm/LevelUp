// Package duckdb — SkillV2Repo : accès player_skill_state_v2 + lusr_hyperparams_v2.
//
// Tables append-only ; toutes les lectures passent par les vues *_latest. Les
// écritures sont des INSERT purs (jamais d'UPDATE / ON CONFLICT — voir ADR 0019
// pour le rationnel anti-corruption ART DuckDB).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/domain"
)

// SkillV2Repo encapsule les accès aux tables LUSR v2 dans shared_matches_v2.duckdb.
// Reçoit *sql.DB directement (pas de wrapper PlayerDB / SharedReadDB) : ce repo
// est destiné à être appelé depuis la pipeline sync, qui ouvre déjà la shared
// elle-même. Pas de timeout interne — le caller passe son ctx.
type SkillV2Repo struct {
	shared *sql.DB
}

// NewSkillV2Repo construit un SkillV2Repo depuis une connexion sql ouverte sur
// shared_matches_v2.duckdb (RW pour les UpsertState/UpsertHyperparam ;
// les Load* fonctionnent en RO).
func NewSkillV2Repo(shared *sql.DB) *SkillV2Repo {
	return &SkillV2Repo{shared: shared}
}

// LoadState retourne le posterior courant d'un joueur sur un groupe de modes.
// Retourne (nil, nil) si aucune row — caller doit initialiser depuis Priors.
func (r *SkillV2Repo) LoadState(ctx context.Context, xuid, playlistGroup string) (*domain.SkillV2State, error) {
	row := r.shared.QueryRowContext(ctx, `
		SELECT xuid, playlist_group, mu, sigma, experience,
		       last_match_id, last_match_at, written_at
		FROM player_skill_state_v2_latest
		WHERE xuid = ? AND playlist_group = ?`,
		xuid, playlistGroup)

	var s domain.SkillV2State
	var lastMatchID sql.NullString
	var lastMatchAt sql.NullTime
	if err := row.Scan(
		&s.XUID, &s.PlaylistGroup, &s.Mu, &s.Sigma, &s.Experience,
		&lastMatchID, &lastMatchAt, &s.WrittenAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("SkillV2Repo.LoadState(%s, %s): %w", xuid, playlistGroup, err)
	}
	if lastMatchID.Valid {
		s.LastMatchID = &lastMatchID.String
	}
	if lastMatchAt.Valid {
		s.LastMatchAt = &lastMatchAt.Time
	}
	return &s, nil
}

// LoadAllStates retourne tous les posteriors courants d'un joueur, un par
// playlist_group dans lequel il a un état. Vide si jamais vu. Utilisé par la
// Phase 4 (mode correlation) pour propager une mise à jour cross-mode.
func (r *SkillV2Repo) LoadAllStates(ctx context.Context, xuid string) ([]domain.SkillV2State, error) {
	rows, err := r.shared.QueryContext(ctx, `
		SELECT xuid, playlist_group, mu, sigma, experience,
		       last_match_id, last_match_at, written_at
		FROM player_skill_state_v2_latest
		WHERE xuid = ?`, xuid)
	if err != nil {
		return nil, fmt.Errorf("SkillV2Repo.LoadAllStates(%s): %w", xuid, err)
	}
	defer rows.Close() //nolint:errcheck

	var out []domain.SkillV2State
	for rows.Next() {
		var s domain.SkillV2State
		var lastMatchID sql.NullString
		var lastMatchAt sql.NullTime
		if err := rows.Scan(
			&s.XUID, &s.PlaylistGroup, &s.Mu, &s.Sigma, &s.Experience,
			&lastMatchID, &lastMatchAt, &s.WrittenAt,
		); err != nil {
			return nil, fmt.Errorf("SkillV2Repo.LoadAllStates scan: %w", err)
		}
		if lastMatchID.Valid {
			s.LastMatchID = &lastMatchID.String
		}
		if lastMatchAt.Valid {
			s.LastMatchAt = &lastMatchAt.Time
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// UpsertState écrit un nouveau snapshot du posterior. Append-only : chaque appel
// produit une nouvelle row ; la vue _latest s'occupe de filtrer.
func (r *SkillV2Repo) UpsertState(ctx context.Context, s domain.SkillV2State) error {
	_, err := r.shared.ExecContext(ctx, `
		INSERT INTO player_skill_state_v2
			(xuid, playlist_group, mu, sigma, experience, last_match_id, last_match_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		s.XUID, s.PlaylistGroup, s.Mu, s.Sigma, s.Experience,
		nullableStringPtr(s.LastMatchID), nullableTime(s.LastMatchAt),
	)
	if err != nil {
		return fmt.Errorf("SkillV2Repo.UpsertState(%s, %s): %w", s.XUID, s.PlaylistGroup, err)
	}
	return nil
}

// LoadHyperparams retourne tous les hyperparamètres latest pour un groupe.
// Map name → value. Vide si aucun param enregistré (caller doit appliquer
// les defaults Priors).
func (r *SkillV2Repo) LoadHyperparams(ctx context.Context, playlistGroup string) (map[string]float64, error) {
	rows, err := r.shared.QueryContext(ctx, `
		SELECT name, value
		FROM lusr_hyperparams_v2_latest
		WHERE playlist_group = ?`, playlistGroup)
	if err != nil {
		return nil, fmt.Errorf("SkillV2Repo.LoadHyperparams(%s): %w", playlistGroup, err)
	}
	defer rows.Close() //nolint:errcheck

	out := make(map[string]float64)
	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("SkillV2Repo.LoadHyperparams scan: %w", err)
		}
		out[name] = value
	}
	return out, rows.Err()
}

// UpsertHyperparam écrit un nouveau snapshot d'un hyperparamètre nommé.
func (r *SkillV2Repo) UpsertHyperparam(ctx context.Context, h domain.SkillV2Hyperparam) error {
	_, err := r.shared.ExecContext(ctx, `
		INSERT INTO lusr_hyperparams_v2
			(playlist_group, name, value, source)
		VALUES (?, ?, ?, ?)`,
		h.PlaylistGroup, h.Name, h.Value, h.Source,
	)
	if err != nil {
		return fmt.Errorf("SkillV2Repo.UpsertHyperparam(%s/%s): %w", h.PlaylistGroup, h.Name, err)
	}
	return nil
}

// nullableStringPtr convertit un *string en any (nil → SQL NULL). Local au
// repo car le nullableString(string) du package prend une string non-ptr.
func nullableStringPtr(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}
