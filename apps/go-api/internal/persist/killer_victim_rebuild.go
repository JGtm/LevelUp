package persist

import (
	"context"
	"fmt"
	"time"
)

// RebuildKillerVictimPairs reecrit les couples `killer_victim_pairs` d UN match (DELETE-then-INSERT :
// la table n a ni PK ni index, rien d ART a corrompre — meme regle que la completion combat).
// Sert l entretien `repair-highlight-phantoms`, qui recalcule les couples apres avoir retire les
// lignes fantomes de `highlight_events`. N ecrit PAS `match_kill_events` : la base credit se
// reconstruit par `backfill-killsource`, qui lit les noms dans l annuaire.
func (p *EventsCompletionPersister) RebuildKillerVictimPairs(ctx context.Context, matchID string, pairs []KVPairCompletion) error {
	if matchID == "" {
		return fmt.Errorf("persist: RebuildKillerVictimPairs: MatchID vide")
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: RebuildKillerVictimPairs BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit

	if _, err := tx.ExecContext(ctx, `DELETE FROM killer_victim_pairs WHERE match_id = ?`, matchID); err != nil {
		return fmt.Errorf("persist: RebuildKillerVictimPairs delete %s: %w", matchID, err)
	}
	now := time.Now().UTC()
	for _, kv := range pairs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO killer_victim_pairs
				(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, time_ms, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			matchID, kv.KillerXUID, kv.KillerGamertag, kv.VictimXUID, kv.VictimGamertag, kv.TimeMS, now); err != nil {
			return fmt.Errorf("persist: RebuildKillerVictimPairs insert %s: %w", matchID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: RebuildKillerVictimPairs Commit %s: %w", matchID, err)
	}
	return nil
}
