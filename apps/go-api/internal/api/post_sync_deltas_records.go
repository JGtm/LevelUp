// Package api — post_sync_deltas_records.go : persistance des records personnels
// (player_records). Extrait de post_sync_deltas.go (refactor god-file).
package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"levelup/go-api/internal/platform/duckdb"
)

type playerRecord struct {
	Loaded        bool
	Value         float64
	AchievedMatch string
}

func loadPlayerRecord(ctx context.Context, pdb *duckdb.PlayerDB, metric string) (playerRecord, error) {
	if pdb == nil || pdb.SharedSocial == nil || pdb.XUID == "" {
		return playerRecord{}, nil
	}
	var v sql.NullFloat64
	var matchID sql.NullString
	// Lit la vue append-only player_records_latest (et non plus la table legacy
	// player_records) : les écritures passent désormais par AppendPlayerRecord
	// → player_records_history. period='all_time' = la clé écrite par
	// upsertPlayerRecord (corrige le split-brain lecture/écriture historique).
	err := pdb.SharedSocial.QueryRow(ctx,
		`SELECT value, achieved_match_id FROM player_records_latest WHERE xuid = ? AND metric = ? AND period = 'all_time'`,
		pdb.XUID, metric,
	).Scan(&v, &matchID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return playerRecord{}, nil
	case err != nil:
		return playerRecord{}, err
	}
	return playerRecord{
		Loaded:        v.Valid,
		Value:         v.Float64,
		AchievedMatch: matchID.String,
	}, nil
}

func upsertPlayerRecord(ctx context.Context, pdb *duckdb.PlayerDB, metric string, value float64, matchID string) error {
	if pdb == nil || pdb.SharedSocial == nil || pdb.XUID == "" {
		return fmt.Errorf("upsertPlayerRecord: shared_social or xuid missing")
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
	if duckdb.RequireSocialPersister {
		return duckdb.ErrSocialPersisterNotWired
	}

	// Fallback (tests uniquement — RequireSocialPersister=false par défaut).
	// Append-only dans player_records_history, COHÉRENT avec loadPlayerRecord qui
	// lit player_records_latest : plus d'ON CONFLICT sur la table legacy (qui
	// aurait été invisible pour le lecteur → split-brain). CHECKPOINT immédiat
	// pour ne pas accumuler de WAL non-checkpointed même en mode test.
	if _, err := pdb.SharedSocial.Exec(ctx, `
		INSERT INTO player_records_history
			(xuid, metric, period, value, achieved_at, achieved_match_id, written_at)
		VALUES (?, ?, 'all_time', ?, NOW(), ?, NOW())
	`, pdb.XUID, metric, value, nullableMatchID(matchID)); err != nil {
		return err
	}
	return duckdb.CheckpointSharedSocial(ctx, pdb.SharedSocial)
}

func nullableMatchID(id string) any {
	if id == "" {
		return nil
	}
	return id
}

// newCitationsServiceForPDB construit un CitationsService scopé sur le joueur.
// Retourne nil si pdb est invalide — SnapshotPlayerState saute alors la lecture
// citations.
