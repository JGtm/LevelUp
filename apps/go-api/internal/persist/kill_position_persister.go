// Package persist — kill_position_persister.go : ecriture INSERT-ONLY d une PASSE DE POSITIONS
// dans `shared.kill_positions`, pour un match DEJA insere.
//
// POURQUOI UN PERSISTER DEDIE, ET PAS `SharedPersister` : meme raison que [KillSourcePersister]
// (cf. kill_events_persister.go) — `SharedPersister.Persist` est un no-op des que le match existe
// deja dans `match_registry`, et une passe de decodage de film arrive TOUJOURS sur un match deja
// insere (le film n est pas pret au sync primaire). Le chemin builder (`Shared.KillPositions`,
// utilise par Halo 5 a l insertion du match) reste le bon pour un titre dont les positions sont
// natives et disponibles des le sync primaire — ce qui n est pas le cas d Infinite.
//
// ANTI-ART (ADR 0019/0026) : INSERT purs — la table est append-only depuis G.2
// (games/halo_infinite/migrations/steps_appendonly_misc.go : id PK + written_at + vue
// kill_positions_latest). « Remplacer » une passe consiste a en ecrire une nouvelle ; c est la
// vue `_latest` qui ne rend que la DERNIERE ligne par (match_id, killer_xuid, time_ms). Ce
// persister n emet aucun UPDATE/DELETE/ON CONFLICT et n a donc rien a faire figurer dans
// l allowlist de `internal/sync/no_art_patterns_test.go`.
//
// PAS DE `decoder_rev` SUR CETTE TABLE, ET C EST VOULU (cf. steps_shared_kill_positions.go) : la
// cle fonctionnelle (match_id, killer_xuid, time_ms) n a jamais eu besoin d une colonne
// supplementaire pour distinguer deux versions d une ligne — `written_at` suffit, exactement
// comme pour `match_csrs`/`pve_match_stats`.
//
// PRE-REQUIS : le caller doit tenir le lease RW sur shared_matches_v2.duckdb (comme tous les
// persisters shared). `txBeginner` accepte aussi bien *sql.DB qu un LeasedWriter.
package persist

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// KillPositionPersister ecrit une passe de positions dans shared_matches_v2.duckdb.
type KillPositionPersister struct {
	db txBeginner
}

// NewKillPositionPersister construit un persister. `db` doit tenir le lease RW sur shared.
func NewKillPositionPersister(db txBeginner) *KillPositionPersister {
	return &KillPositionPersister{db: db}
}

// PersistPass ecrit les positions d UN match en 1 transaction, en INSERT purs.
//
// Une passe VIDE n est pas une erreur mais n est pas non plus anodine : elle est ignoree (aucune
// ligne ecrite, la vue `_latest` continue de servir la passe precedente si elle existe) et
// LOGGUEE — meme doctrine que [KillSourcePersister.PersistPass] : ecrire zero ligne en silence
// serait indistinguable d un match sans position localisable.
func (p *KillPositionPersister) PersistPass(ctx context.Context, matchID string, rows []KillPositionInsert) error {
	if matchID == "" {
		return errors.New("persist: KillPositionPersister.PersistPass: matchID vide")
	}
	if len(rows) == 0 {
		slog.WarnContext(ctx, "persist: passe positions vide, aucune ligne ecrite", "match_id", matchID)
		return nil
	}
	for i := range rows {
		if rows[i].MatchID != matchID {
			return fmt.Errorf("persist: KillPositionPersister.PersistPass %s: ligne #%d porte match_id %q",
				matchID, i, rows[i].MatchID)
		}
		if rows[i].KillerXUID == "" {
			return fmt.Errorf("persist: KillPositionPersister.PersistPass %s: ligne #%d sans killer_xuid "+
				"(une position sans tueur identifie n est pas une ligne de kill_positions)", matchID, i)
		}
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx kill_positions %s: %w", matchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit

	if err := persistKillPositions(ctx, tx, rows); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit kill_positions %s: %w", matchID, err)
	}
	return nil
}
