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
// (games/halo_infinite/migrations/steps_appendonly_misc.go : id PK + written_at), et arbitree
// PAR PASSE depuis le lot 1.7 (steps_shared_kill_positions_pass.go : + decode_pass, vue
// kill_positions_latest = DERNIERE PASSE ENTIERE par match). « Remplacer » une passe consiste
// a en ecrire une nouvelle ; ce persister n emet aucun UPDATE/DELETE/ON CONFLICT et n a donc
// rien a faire figurer dans l allowlist de `internal/sync/no_art_patterns_test.go`.
//
// L UNITE DE GENERATION EST LA PASSE, PAS LA LIGNE (lot 1.7, 2026-09-09). La vue arbitrait
// jusque-la par CLE — derniere ligne par (match_id, killer_xuid, time_ms) — et cet arbitrage
// ne savait pas RETRACTER : `replay.BuildKillPositions` n ecrit AUCUNE ligne pour une mort dont
// ni le tueur ni la victime n ont pu etre localises (bornes de trajectoire, joueur non resolu,
// film re-telecharge plus court), donc un re-decodage qui ne retrouve plus une position ne la
// reecrit pas — il l omet. La vue par cle continuait alors de servir A JAMAIS la ligne de la
// passe precedente, melangee aux nouvelles. Toutes les lignes d une passe portent desormais le
// MEME `decode_pass`, tire une seule fois par [newDecodePassID] : meme doctrine que
// kill_events_persister.go et kill_opening_persister.go.
//
// LA BORNE, ASSUMEE : une passe VIDE n ecrit ni ligne ni generation, donc la vue continue de
// servir la passe precedente ENTIERE (epingle par
// TestKillPositionPersistPass_PasseVideNeRetractePas). L alternative serait une ligne
// sentinelle dont aucun lecteur n a l usage. Meme arbitrage que kill_openings.
//
// PAS DE `decoder_rev` SUR CETTE TABLE, ET C EST VOULU (cf. steps_shared_kill_positions.go) :
// `decode_pass` distingue deja deux generations, et la revision du decodeur ne se lit nulle
// part sur cette table-ci (contrairement a `match_kill_events`, ou elle sert au diagnostic).
//
// PRE-REQUIS : le caller doit tenir le lease RW sur shared_matches_v2.duckdb (comme tous les
// persisters shared). `txBeginner` accepte aussi bien *sql.DB qu un LeasedWriter.
package persist

import (
	"context"
	"database/sql"
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

// PersistPass ecrit les positions d UN match en 1 transaction, en INSERT purs, TOUTES sous le
// meme `decode_pass` : la passe est l unite de generation, la vue `_latest` la retient entiere.
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

	// Le tirage AVANT la transaction : un decode_pass non distinguable ferait rendre a la vue
	// _latest un MELANGE de passes, sans aucun symptome visible (cf. newDecodePassID).
	pass, err := newDecodePassID()
	if err != nil {
		return fmt.Errorf("persist: kill_positions %s: %w", matchID, err)
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx kill_positions %s: %w", matchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit

	if err := persistKillPositions(ctx, tx, pass, rows); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit kill_positions %s: %w", matchID, err)
	}
	return nil
}

// persistKillPositionsPass tire une generation et ecrit la passe — le point d entree du CHEMIN
// BUILDER (`Shared.KillPositions`, Halo 5 a l insertion du match : `ingest.MapKillPositions`
// rend TOUTES les positions du match d un coup, c est donc bien une passe entiere).
//
// LE TIRAGE VIT ICI, ET PAS DANS LE BATCH. La valeur de passe a une source unique dans le
// depot — [newDecodePassID] — et elle est tiree AU MOMENT DE L ECRITURE, exactement comme pour
// `kill_openings` et `match_kill_events`. La porter dans `MatchBatch` l aurait figee dans le
// WAL sans aucun lecteur : le chemin builder est un no-op des que le match existe dans
// `match_registry` (cf. SharedPersister.Persist), donc un rejeu de WAL ne peut pas ecrire deux
// passes pour un meme match, et rien n a besoin de relire la generation apres coup.
func persistKillPositionsPass(ctx context.Context, tx *sql.Tx, rows []KillPositionInsert) error {
	if len(rows) == 0 {
		return nil
	}
	pass, err := newDecodePassID()
	if err != nil {
		return fmt.Errorf("persist: kill_positions %s: %w", rows[0].MatchID, err)
	}
	return persistKillPositions(ctx, tx, pass, rows)
}
