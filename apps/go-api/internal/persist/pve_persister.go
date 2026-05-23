// Package persist — pve_persister.go : écriture INSERT-only atomique du
// sous-batch PVEBatch dans shared_pve.duckdb (stats Firefight).
//
// 1 batch = 1 row (PK match_id, xuid) — l'écriture est triviale (pas
// d'idempotence à gérer côté Persister, simplement INSERT OR IGNORE pour
// supporter le retry après crash entre COMMIT shared et ACK).

package persist

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// PVEPersister écrit le sous-batch PVE dans shared_pve.duckdb (Firefight).
type PVEPersister struct {
	db txBeginner
}

// NewPVEPersister construit un persister sur la connexion shared_pve.duckdb
// (avec write lease actif côté caller).
func NewPVEPersister(db txBeginner) *PVEPersister {
	return &PVEPersister{db: db}
}

// Persist écrit le batch.PVE en 1 transaction.
//
// Cas particuliers :
//   - batch == nil               → error.
//   - batch.PVE == nil           → no-op (match non-Firefight).
//   - batch.PVE.Stats == nil     → no-op.
//   - (match_id, xuid) déjà persisté → INSERT OR IGNORE skip silently.
func (p *PVEPersister) Persist(ctx context.Context, batch *MatchBatch) error {
	if batch == nil {
		return errors.New("persist: PVEPersister.Persist: batch nil")
	}
	if batch.PVE == nil || len(batch.PVE.Stats) == 0 {
		return nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx pve: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for i := range batch.PVE.Stats {
		if err := persistPVEStats(ctx, tx, &batch.PVE.Stats[i]); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit pve %s: %w", batch.PVE.Stats[0].MatchID, err)
	}
	return nil
}

func persistPVEStats(ctx context.Context, tx *sql.Tx, row *PVEMatchStatsInsert) error {
	if row == nil {
		return nil
	}
	// INSERT OR IGNORE car la PK (match_id, xuid) peut pré-exister sur
	// retry après crash entre COMMIT et ACK. Aucune mise à jour : la 1ʳᵉ
	// écriture est la source de vérité (Collect→Persist garantit que tous
	// les enrichments PVE sont déjà résolus à la collecte).
	_, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO pve_match_stats (
			match_id, xuid,
			waves_completed, boss_kills,
			grunt_kills, elite_kills, jackal_kills, brute_kills,
			hunter_kills, skimmer_kills, crawler_kills, soldier_kills,
			knight_kills, warden_kills, sentinel_kills, marine_kills,
			total_kills, deaths, damage_dealt, pve_bits
		) VALUES (
			?, ?,
			?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?
		)`,
		row.MatchID, row.XUID,
		row.WavesCompleted, row.BossKills,
		row.GruntKills, row.EliteKills, row.JackalKills, row.BruteKills,
		row.HunterKills, row.SkimmerKills, row.CrawlerKills, row.SoldierKills,
		row.KnightKills, row.WardenKills, row.SentinelKills, row.MarineKills,
		row.TotalKills, row.Deaths, row.DamageDealt, row.PveBits,
	)
	if err != nil {
		return fmt.Errorf("persist: INSERT pve_match_stats %s/%s: %w",
			row.MatchID, row.XUID, err)
	}
	return nil
}
