package duckdb

// replay_mode_gate.go — LE REGISTRE DIT QUELS MATCHS LE REJEU NE GERE PAS (Firefight, decision
// du 2026-10-15). Implemente [port.ReplayModeGate] sur le meme repo que la resolution de carte :
// le service de rejeu n'a qu'une dependance base, et elle lit le meme registre partage.
//
// Le predicat est `analysis.SQLModeNonGere`, jamais un nom de mode ni un slug.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"levelup/go-api/internal/analysis"
	titlePkg "levelup/go-api/internal/domain/title"
)

// ModeNonGere dit si le registre classe ce match (identifiant complet) dans un mode non gere.
// Un match absent du registre n'est PAS non gere : on ne refuse pas ce qu'on ne sait pas classer.
func (r *ReplayMapRepo) ModeNonGere(ctx context.Context, matchID string) (bool, error) {
	if r == nil || r.shared == nil {
		return false, nil
	}
	db, release, err := r.shared.Get(ctx)
	if err != nil {
		return false, fmt.Errorf("replay mode: lecteur shared indisponible: %w", err)
	}
	defer release()
	var nonGere bool
	err = db.QueryRowContext(ctx,
		`SELECT `+analysis.SQLModeNonGere("")+` FROM match_registry WHERE match_id = ?`, matchID,
	).Scan(&nonGere)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("replay mode: lecture match_registry: %w", err)
	}
	return nonGere, nil
}

// ModesNonGeresShortIDs rend, en forme COURTE (cle des artefacts), tous les matchs du registre
// dont le mode n'est pas gere. UNE requete : c'est la forme qu'emploie la liste des rejeux
// disponibles, jamais un appel par ligne.
func (r *ReplayMapRepo) ModesNonGeresShortIDs(ctx context.Context) (map[string]struct{}, error) {
	if r == nil || r.shared == nil {
		return nil, nil
	}
	db, release, err := r.shared.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("replay mode: lecteur shared indisponible: %w", err)
	}
	defer release()
	rows, err := db.QueryContext(ctx,
		`SELECT match_id FROM match_registry WHERE `+analysis.SQLModeNonGere(""))
	if err != nil {
		return nil, fmt.Errorf("replay mode: lecture match_registry: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("replay mode: lecture match_registry (scan): %w", err)
		}
		out[titlePkg.FilmShortMatchID(id)] = struct{}{}
	}
	return out, rows.Err()
}
