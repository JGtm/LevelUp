// Package persist — kickoff_persister.go : LE persisteur du coup d'envoi d'un match et du temps
// de jeu de ses joueurs (ADR 0050).
//
// # CE QU'IL ÉCRIT
//
// Pour UN match et dans UNE transaction : `match_registry.real_start_time` et
// `match_registry.real_start_source`, puis `match_participants.time_played_seconds` des joueurs
// dont la valeur change. L'appelant (paquet `sync/kickoff`) a lu l'état rangé, résolu le coup
// d'envoi par `timeline.ResolveKickoff` et calculé le temps de jeu sur ce coup d'envoi ; il ne
// passe que ce qui diffère de la base. Ce fichier ÉCRIT, il ne décide rien. Hors de lui, seules
// les insertions de la synchronisation posent ces colonnes (garde-rail
// `archlint/kickoff_write_guard_test.go`).
//
// # ART-SAFETY
//
// UN `UPDATE ... WHERE match_id = ?` (registre) et UN `UPDATE ... WHERE match_id = ? AND xuid = ?`
// par joueur, séquentiels, sous transaction et sous le writer exclusif de l'appelant : la forme
// row-by-row autorisée par `internal/sync/no_art_patterns_test.go`. Aucune des trois colonnes
// n'est indexée.
package persist

import (
	"context"
	"errors"
	"fmt"
	"time"

	"levelup/go-api/pkg/duckdbkeys"
)

// KickoffValue est le coup d'envoi à ranger au registre d'un match.
type KickoffValue struct {
	// RealStart : l'instant du coup d'envoi en UTC ; nil = inconnu (colonne remise à NULL).
	RealStart *time.Time
	// Source : `film`, `api`, ou vide (inconnu, rangé NULL).
	Source string
}

// PlayerTimePlayed est le temps de jeu à ranger pour un joueur du match.
type PlayerTimePlayed struct {
	XUID    string
	Seconds int
}

// MatchKickoffWrite est l'écriture d'UN match.
type MatchKickoffWrite struct {
	MatchID string
	// Kickoff : le coup d'envoi à ranger ; nil = registre déjà à jour, rien à y écrire.
	Kickoff *KickoffValue
	// TimePlayed : les joueurs dont le temps de jeu change.
	TimePlayed []PlayerTimePlayed
}

// KickoffPersister écrit le coup d'envoi et le temps de jeu d'un match. `db` doit porter un write
// lease actif sur la base partagée du titre.
type KickoffPersister struct {
	db txBeginner
}

// NewKickoffPersister construit le persister.
func NewKickoffPersister(db txBeginner) *KickoffPersister {
	return &KickoffPersister{db: db}
}

// WriteMatch écrit, dans une transaction, le coup d'envoi et les temps de jeu d'un match. Une
// écriture vide ne fait rien ; une erreur annule tout le match.
func (p *KickoffPersister) WriteMatch(ctx context.Context, w MatchKickoffWrite) error {
	if w.MatchID == "" {
		return errors.New("persist: KickoffPersister.WriteMatch: matchID vide")
	}
	if w.Kickoff == nil && len(w.TimePlayed) == 0 {
		return nil
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: WriteMatch BeginTx %s: %w", w.MatchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op après Commit réussi

	if w.Kickoff != nil {
		var realStart any
		if w.Kickoff.RealStart != nil {
			realStart = w.Kickoff.RealStart.UTC()
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE match_registry
			SET real_start_time = ?, real_start_source = ?
			WHERE match_id = ?`, realStart, nullableStr(w.Kickoff.Source), w.MatchID); err != nil {
			return fmt.Errorf("persist: WriteMatch registre %s: %w", w.MatchID, err)
		}
	}
	for _, tp := range w.TimePlayed {
		if _, err := tx.ExecContext(ctx, `
			UPDATE match_participants
			SET time_played_seconds = ?
			WHERE match_id = ? AND xuid = ?`, tp.Seconds, w.MatchID, tp.XUID); err != nil {
			return fmt.Errorf("persist: WriteMatch temps de jeu %s/%s: %w", w.MatchID, tp.XUID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: WriteMatch Commit %s: %w", w.MatchID, err)
	}
	return nil
}

// StoreRevision range dans `sync_meta` de la base partagée la révision du rattrapage qui vient de
// réussir (SELECT puis UPDATE ou INSERT : jamais d'ON CONFLICT sur la clé, cf. duckdbkeys.Upsert).
func (p *KickoffPersister) StoreRevision(ctx context.Context, key, value string) error {
	if key == "" || value == "" {
		return errors.New("persist: StoreRevision: clé ou valeur vide")
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: StoreRevision BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op après Commit réussi
	now := time.Now().UTC()
	if err := (duckdbkeys.Upsert{
		ExistsSQL: `SELECT 1 FROM sync_meta WHERE key = ?`, ExistsArgs: []any{key},
		UpdateSQL:  `UPDATE sync_meta SET value = ?, updated_at = ? WHERE key = ?`,
		UpdateArgs: []any{value, now, key},
		InsertSQL:  `INSERT INTO sync_meta (key, value, updated_at) VALUES (?, ?, ?)`,
		InsertArgs: []any{key, value, now},
	}).Exec(ctx, tx); err != nil {
		return fmt.Errorf("persist: StoreRevision %s: %w", key, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: StoreRevision Commit: %w", err)
	}
	return nil
}
