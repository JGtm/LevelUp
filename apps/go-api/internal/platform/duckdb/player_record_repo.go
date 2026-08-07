// Package duckdb — player_record_repo.go : persistance des records personnels
// (player_records). Extrait de internal/api/post_sync_deltas_records.go (K1a,
// 2026-07-05) : la racine api/ ne doit plus porter de SQL — les lectures/écritures
// player_records vivent ici, à côté des autres repos DuckDB.
//
// Écriture = append-only via SocialPersister (ADR 0022, CHECKPOINT garanti) ;
// lecture = vue player_records_latest (jamais la table legacy). Cf. loadPlayerRecord
// d'origine pour l'historique du split-brain corrigé.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// PlayerRecord est un record personnel (meilleure valeur atteinte sur une métrique).
type PlayerRecord struct {
	Loaded        bool
	Value         float64
	AchievedMatch string
}

// LoadPlayerRecord lit le record all-time d'une métrique via la vue append-only
// player_records_latest. Retourne un PlayerRecord non-chargé si absent/pdb invalide.
func LoadPlayerRecord(ctx context.Context, pdb *PlayerDB, metric string) (PlayerRecord, error) {
	if pdb == nil || pdb.SharedSocial == nil || pdb.XUID == "" {
		return PlayerRecord{}, nil
	}
	var v sql.NullFloat64
	var matchID sql.NullString
	// Lit la vue append-only player_records_latest (et non plus la table legacy
	// player_records) : les écritures passent désormais par AppendPlayerRecord
	// → player_records_history. period='all_time' = la clé écrite par
	// UpsertPlayerRecord (corrige le split-brain lecture/écriture historique).
	err := pdb.SharedSocial.QueryRow(ctx,
		`SELECT value, achieved_match_id FROM player_records_latest WHERE xuid = ? AND metric = ? AND period = 'all_time'`,
		pdb.XUID, metric,
	).Scan(&v, &matchID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return PlayerRecord{}, nil
	case err != nil:
		return PlayerRecord{}, err
	}
	return PlayerRecord{
		Loaded:        v.Valid,
		Value:         v.Float64,
		AchievedMatch: matchID.String,
	}, nil
}

// UpsertPlayerRecord écrit (append-only) le record all-time d'une métrique.
func UpsertPlayerRecord(ctx context.Context, pdb *PlayerDB, metric string, value float64, matchID string) error {
	if pdb == nil || pdb.SharedSocial == nil || pdb.XUID == "" {
		return fmt.Errorf("UpsertPlayerRecord: shared_social or xuid missing")
	}

	// ADR 0022 : route via Persister (append-only sur player_records_history
	// + CHECKPOINT garanti). Élimine le UPSERT problématique qui pressionnait
	// l'index ART DuckDB (bug #9277) et accumulait du WAL non-checkpointed
	// (bug #7659). La vue player_records_latest expose toujours la dernière
	// valeur pour les lecteurs.
	//
	// Path nominal : SocialPersister wired (toujours en prod via main.go).
	if pdb.SocialPersister != nil {
		var matchIDPtr *string
		if matchID != "" {
			matchIDPtr = &matchID
		}
		now := time.Now().UTC()
		return pdb.SocialPersister.AppendPlayerRecord(ctx,
			pdb.XUID, metric, "all_time", value, &now, matchIDPtr, nil, nil)
	}

	// ADR 0021 Gap 1 : en prod (RequireSocialPersister=true set par main.go),
	// refuse d'écrire silencieusement via le path legacy — l'absence de Persister
	// est un bug de wiring, pas un mode de fonctionnement nominal.
	if RequireSocialPersister {
		return ErrSocialPersisterNotWired
	}

	// Fallback (tests uniquement — RequireSocialPersister=false par défaut).
	// Append-only dans player_records_history, COHÉRENT avec LoadPlayerRecord qui
	// lit player_records_latest : plus d'ON CONFLICT sur la table legacy (qui
	// aurait été invisible pour le lecteur → split-brain). CHECKPOINT immédiat
	// pour ne pas accumuler de WAL non-checkpointed même en mode test.
	if _, err := pdb.SharedSocial.Exec(ctx, `
		INSERT INTO player_records_history
			(xuid, metric, period, value, achieved_at, achieved_match_id, written_at)
		VALUES (?, ?, 'all_time', ?, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP), ?, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))
	`, pdb.XUID, metric, value, nullableMatchID(matchID)); err != nil {
		return err
	}
	return CheckpointSharedSocial(ctx, pdb.SharedSocial)
}

func nullableMatchID(id string) any {
	if id == "" {
		return nil
	}
	return id
}
