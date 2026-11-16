package duckdb

// replay_spawn_registry.go — LE REGISTRE DES MATCHS, LU POUR LA PREVISION DE REAPPARITION.
// Implemente [port.ReplaySpawnRegistry] sur le meme repo que la resolution de carte : le service de
// rejeu n'a qu'une dependance base, et elle lit le meme registre partage.
//
// Les matchs que le rejeu ne gere pas (PvE) n'y sont jamais : meme predicat que la porte des modes
// (`sqlfrag.SQLModeNonGere`).

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"levelup/go-api/internal/platform/duckdb/sqlfrag"
	"levelup/go-api/internal/port"
)

const sqlSpawnMatchColonnes = `match_id, COALESCE(map_id, ''), COALESCE(game_variant_name, ''), COALESCE(mode_category, '')`

func scanSpawnMatch(row interface{ Scan(...any) error }) (port.SpawnMatch, error) {
	var m port.SpawnMatch
	err := row.Scan(&m.MatchID, &m.MapID, &m.Variant, &m.Category)
	return m, err
}

// SpawnMatchOf rend le match du registre. ErrMatchMapUnknown quand il est absent ou sans carte :
// sans carte, aucun candidat n'est cherchable.
func (r *ReplayMapRepo) SpawnMatchOf(ctx context.Context, matchID string) (port.SpawnMatch, error) {
	if r == nil || r.shared == nil {
		return port.SpawnMatch{}, ErrMatchMapUnknown
	}
	db, release, err := r.shared.Get(ctx)
	if err != nil {
		return port.SpawnMatch{}, fmt.Errorf("replay spawn: lecteur shared indisponible: %w", err)
	}
	defer release()
	m, err := scanSpawnMatch(db.QueryRowContext(ctx,
		`SELECT `+sqlSpawnMatchColonnes+` FROM match_registry WHERE match_id = ?`, matchID))
	if errors.Is(err, sql.ErrNoRows) {
		return port.SpawnMatch{}, ErrMatchMapUnknown
	}
	if err != nil {
		return port.SpawnMatch{}, fmt.Errorf("replay spawn: lecture match_registry: %w", err)
	}
	if strings.TrimSpace(m.MapID) == "" {
		return port.SpawnMatch{}, ErrMatchMapUnknown
	}
	return m, nil
}

// SpawnMatchesOnMap rend les matchs JcJ d'une carte, tries par identifiant (resultat
// deterministe). Une carte vide rend aucun match.
func (r *ReplayMapRepo) SpawnMatchesOnMap(ctx context.Context, mapID string) ([]port.SpawnMatch, error) {
	mapID = strings.TrimSpace(mapID)
	if r == nil || r.shared == nil || mapID == "" {
		return nil, nil
	}
	db, release, err := r.shared.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("replay spawn: lecteur shared indisponible: %w", err)
	}
	defer release()
	rows, err := db.QueryContext(ctx,
		`SELECT `+sqlSpawnMatchColonnes+` FROM match_registry mr
		 WHERE map_id = ? AND NOT `+sqlfrag.SQLModeNonGere("mr")+` ORDER BY match_id`, mapID)
	if err != nil {
		return nil, fmt.Errorf("replay spawn: lecture match_registry par carte: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []port.SpawnMatch
	for rows.Next() {
		m, err := scanSpawnMatch(rows)
		if err != nil {
			return nil, fmt.Errorf("replay spawn: lecture match_registry par carte (scan): %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

var _ port.ReplaySpawnRegistry = (*ReplayMapRepo)(nil)
