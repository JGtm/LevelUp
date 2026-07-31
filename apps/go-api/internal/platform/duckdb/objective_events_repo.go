// Package duckdb — objective_events_repo.go : write+read repo des events objectif
// v3 (shared.match_objective_events + shared.match_objective_event_players).
//
// Source : pipeline diagnostic v3 (décodage film) qui produit des
// []domain.ObjectiveEvent par match. Ce repo persiste / relit ces events.
//
// Connexion : comme tous les writers/readers shared (cf. weapon_kills_repo,
// highlight_events_repo + InsertWeaponKills), on passe par
// SharedReadDB().Get(ctx) qui retourne (ADR 0016) une connexion DIRECTE à
// shared_matches_v2.duckdb (tables à la racine du catalogue, PAS de préfixe
// `shared.`). En mode legacy / CLI backfill (SharedReader == LegacySharedReader),
// ce handle est RW : WriteMatch peut donc écrire dessus. Le backfill v3 tourne
// HORS chemin live (MaxOpenConns(1), sérialisé) — pas de pression concurrente ART
// (cf. .ai/PLAN_WEAPON_ATTRIBUTION_V3.md §10).
//
// Écriture par match : DELETE FROM les deux tables WHERE match_id = ?, puis INSERT
// de tous les events, dans une seule transaction (idempotent + atomique) — même
// pattern que InsertWeaponKills (DELETE-then-INSERT-par-match).
//
// Capability gating : si les tables n'existent pas (titre sans capability film /
// migration non appliquée), DuckDB remonte "Table with name ... does not exist",
// intercepté via isTableNotFoundErr -> games.ErrCapabilityNotSupported (comme
// WeaponKillsRepo).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// ObjectiveEventsRepo persiste et relit les events objectif v3 dans la DB shared.
type ObjectiveEventsRepo struct {
	pdb *PlayerDB
}

// NewObjectiveEventsRepo crée un ObjectiveEventsRepo lié à un PlayerDB.
func NewObjectiveEventsRepo(pdb *PlayerDB) *ObjectiveEventsRepo {
	return &ObjectiveEventsRepo{pdb: pdb}
}

// WriteMatch remplace atomiquement TOUS les events objectif d'un match.
//
// Pattern DELETE-then-INSERT-par-match (cf. InsertWeaponKills) : DELETE FROM les
// deux tables WHERE match_id = ?, puis INSERT de tous les events + leurs joueurs,
// dans une seule transaction. Ré-exécuter pour le même matchID écrase l'existant
// (idempotent) — la décision "remplacer" est explicite côté backfill.
//
// events == nil/[] est un no-op total (ni DELETE ni INSERT) : un décodage qui
// n'a rien produit ne doit pas effacer les events déjà persistés, garde-fou
// anti-perte de données aligné sur InsertWeaponKills.
//
// Retourne games.ErrCapabilityNotSupported si les tables n'existent pas.
func (r *ObjectiveEventsRepo) WriteMatch(ctx context.Context, matchID string, events []domain.ObjectiveEvent) error {
	if len(events) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return fmt.Errorf("ObjectiveEventsRepo.WriteMatch: shared reader: %w", err)
	}
	defer release()

	if err := writeObjectiveEventsTx(ctx, db, matchID, events); err != nil {
		if isTableNotFoundErr(err) {
			return games.ErrCapabilityNotSupported
		}
		return fmt.Errorf("ObjectiveEventsRepo.WriteMatch(%s): %w", matchID, err)
	}
	return nil
}

// writeObjectiveEventsTx exécute le DELETE+INSERT dans une transaction.
func writeObjectiveEventsTx(ctx context.Context, db *sql.DB, matchID string, events []domain.ObjectiveEvent) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM match_objective_event_players WHERE match_id = ?`, matchID); err != nil {
		return fmt.Errorf("delete players: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM match_objective_events WHERE match_id = ?`, matchID); err != nil {
		return fmt.Errorf("delete events: %w", err)
	}

	for _, ev := range events {
		if err := insertObjectiveEvent(ctx, tx, matchID, ev); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// insertObjectiveEvent insère un event + ses joueurs associés.
func insertObjectiveEvent(ctx context.Context, tx *sql.Tx, matchID string, ev domain.ObjectiveEvent) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO match_objective_events (
			match_id, seq, time_ms, objective_type, event_type,
			team_id, objective_id, value, source, confidence, details
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		matchID, ev.Seq, nullableInt(ev.TimeMS), ev.ObjectiveType, ev.EventType,
		nullableInt(ev.TeamID), nullableInt(ev.ObjectiveID), nullableInt(ev.Value),
		ev.Source, ev.Confidence, ev.Details,
	); err != nil {
		return fmt.Errorf("insert event seq=%d: %w", ev.Seq, err)
	}
	for _, p := range ev.Players {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO match_objective_event_players (match_id, seq, xuid, role)
			VALUES (?, ?, ?, ?)`,
			matchID, ev.Seq, p.XUID, p.Role,
		); err != nil {
			return fmt.Errorf("insert player seq=%d xuid=%s: %w", ev.Seq, p.XUID, err)
		}
	}
	return nil
}

// LoadMatch relit tous les events objectif d'un match, ordonnés par seq, avec
// leurs joueurs associés. Retourne games.ErrCapabilityNotSupported si les tables
// n'existent pas.
func (r *ObjectiveEventsRepo) LoadMatch(ctx context.Context, matchID string) ([]domain.ObjectiveEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ObjectiveEventsRepo.LoadMatch: shared reader: %w", err)
	}
	defer release()

	events, bySeq, err := loadObjectiveEventRows(ctx, db, matchID)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, games.ErrCapabilityNotSupported
		}
		return nil, fmt.Errorf("ObjectiveEventsRepo.LoadMatch(%s): %w", matchID, err)
	}
	if err := attachObjectiveEventPlayers(ctx, db, matchID, bySeq); err != nil {
		if isTableNotFoundErr(err) {
			return nil, games.ErrCapabilityNotSupported
		}
		return nil, fmt.Errorf("ObjectiveEventsRepo.LoadMatch(%s): players: %w", matchID, err)
	}
	return events, nil
}

// loadObjectiveEventRows lit la table parente. Retourne le slice ordonné par seq
// + un index seq -> *ObjectiveEvent (les pointeurs ciblent les éléments du slice).
func loadObjectiveEventRows(
	ctx context.Context, db *sql.DB, matchID string,
) ([]domain.ObjectiveEvent, map[int]*domain.ObjectiveEvent, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT seq, time_ms, objective_type, event_type,
		       team_id, objective_id, value, source, confidence, details
		FROM match_objective_events
		WHERE match_id = ?
		ORDER BY seq ASC`, matchID)
	if err != nil {
		return nil, nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var out []domain.ObjectiveEvent
	for rows.Next() {
		var (
			ev                                     domain.ObjectiveEvent
			timeMS, teamID, objectiveID, value     sql.NullInt64
			objType, evType, source, conf, details sql.NullString
		)
		if err := rows.Scan(&ev.Seq, &timeMS, &objType, &evType,
			&teamID, &objectiveID, &value, &source, &conf, &details); err != nil {
			return nil, nil, fmt.Errorf("scan event: %w", err)
		}
		ev.MatchID = matchID
		ev.TimeMS = intPtrFromNull(timeMS)
		ev.TeamID = intPtrFromNull(teamID)
		ev.ObjectiveID = intPtrFromNull(objectiveID)
		ev.Value = intPtrFromNull(value)
		ev.ObjectiveType = objType.String
		ev.EventType = evType.String
		ev.Source = source.String
		ev.Confidence = conf.String
		ev.Details = details.String
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("rows: %w", err)
	}

	bySeq := make(map[int]*domain.ObjectiveEvent, len(out))
	for i := range out {
		bySeq[out[i].Seq] = &out[i]
	}
	return out, bySeq, nil
}

// attachObjectiveEventPlayers lit la table joueurs et raccroche chaque joueur à
// son event parent via le seq.
func attachObjectiveEventPlayers(
	ctx context.Context, db *sql.DB, matchID string, bySeq map[int]*domain.ObjectiveEvent,
) error {
	if len(bySeq) == 0 {
		return nil
	}
	rows, err := db.QueryContext(ctx, `
		SELECT seq, xuid, role
		FROM match_objective_event_players
		WHERE match_id = ?
		ORDER BY seq ASC, xuid ASC`, matchID)
	if err != nil {
		return fmt.Errorf("query players: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			seq  int
			xuid string
			role sql.NullString
		)
		if err := rows.Scan(&seq, &xuid, &role); err != nil {
			return fmt.Errorf("scan player: %w", err)
		}
		if ev, ok := bySeq[seq]; ok {
			ev.Players = append(ev.Players, domain.ObjectiveEventPlayer{XUID: xuid, Role: role.String})
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}
	return nil
}

// nullableInt convertit un *int en argument SQL (NULL si nil).
func nullableInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

// intPtrFromNull convertit un sql.NullInt64 en *int (nil si NULL).
func intPtrFromNull(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}
