// Package persist — kill_opening_persister.go : ecriture INSERT-ONLY d une PASSE D ENTAMES
// dans `shared.kill_openings`, pour un match DEJA insere.
//
// POURQUOI UN PERSISTER DEDIE, ET PAS `SharedPersister` : meme raison que
// [KillPositionPersister] (cf. kill_position_persister.go) — `SharedPersister.Persist` est un
// no-op des que le match existe deja dans `match_registry`, et une passe de decodage de film
// arrive TOUJOURS sur un match deja insere. Le chemin builder (`Shared.KillOpenings`) reste
// disponible pour un titre dont les positions seraient natives et connues des le sync primaire.
//
// CE QU IL ECRIT, ET CE QU IL N ECRIT PAS. Une ligne d entame porte le `time_ms` DU COUP FATAL
// (la cle du frag, celle par laquelle le kill-feed se joint) et des coordonnees prises un
// temps-pour-tuer PLUS TOT (`replay.OpeningLeadMS`). Une mort dont l instant decale tombe avant
// l origine du film n a AUCUNE ligne — pas une ligne a coordonnees nulles : la position de
// reapparition n est pas une entame, et l absence est le resultat correct.
//
// ANTI-ART (ADR 0019/0026) : INSERT purs — la table est append-only depuis sa creation
// (games/halo_infinite/migrations/steps_shared_kill_openings.go : id PK + written_at + vue
// kill_openings_latest). « Remplacer » une passe consiste a en ecrire une nouvelle ; c est la
// vue `_latest` qui ne rend que la DERNIERE ligne par (match_id, killer_xuid, time_ms). Ce
// persister n emet aucun UPDATE/DELETE/ON CONFLICT et n a donc rien a faire figurer dans
// l allowlist de `internal/sync/no_art_patterns_test.go`.
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

// KillOpeningPersister ecrit une passe d entames dans shared_matches_v2.duckdb.
type KillOpeningPersister struct {
	db txBeginner
}

// NewKillOpeningPersister construit un persister. `db` doit tenir le lease RW sur shared.
func NewKillOpeningPersister(db txBeginner) *KillOpeningPersister {
	return &KillOpeningPersister{db: db}
}

// PersistPass ecrit les entames d UN match en 1 transaction, en INSERT purs.
//
// Une passe VIDE n est pas une erreur mais n est pas anodine : elle est ignoree (aucune ligne
// ecrite, la vue `_latest` continue de servir la passe precedente si elle existe) et LOGGUEE —
// meme doctrine que [KillPositionPersister.PersistPass]. Sur cette table-ci le cas est MOINS
// exceptionnel que pour les positions (un film dont aucune mort n a d echantillon 1,5 s plus
// tot), raison de plus pour qu il se voie dans les journaux plutot que de se deviner.
func (p *KillOpeningPersister) PersistPass(ctx context.Context, matchID string, rows []KillOpeningInsert) error {
	if matchID == "" {
		return errors.New("persist: KillOpeningPersister.PersistPass: matchID vide")
	}
	if len(rows) == 0 {
		slog.WarnContext(ctx, "persist: passe entames vide, aucune ligne ecrite", "match_id", matchID)
		return nil
	}
	if err := validerLignesEntame(matchID, rows); err != nil {
		return err
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx kill_openings %s: %w", matchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit

	if err := persistKillOpenings(ctx, tx, rows); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit kill_openings %s: %w", matchID, err)
	}
	return nil
}

// validerLignesEntame refuse une passe incoherente AVANT d ouvrir la transaction : un match_id
// etranger (deux matchs melanges dans une meme passe) ou un tueur non identifie (une position
// sans tueur n est pas une ligne de cette table — elle ne se joindrait a aucun kill).
func validerLignesEntame(matchID string, rows []KillOpeningInsert) error {
	for i := range rows {
		if rows[i].MatchID != matchID {
			return fmt.Errorf("persist: KillOpeningPersister.PersistPass %s: ligne #%d porte match_id %q",
				matchID, i, rows[i].MatchID)
		}
		if rows[i].KillerXUID == "" {
			return fmt.Errorf("persist: KillOpeningPersister.PersistPass %s: ligne #%d sans killer_xuid "+
				"(une entame sans tueur identifie ne se joint a aucune mort)", matchID, i)
		}
	}
	return nil
}
