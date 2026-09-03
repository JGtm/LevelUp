package main

// registre.go — LE COTE BASE : ce que `match_registry` sait du T0, et comment on le repare.
//
// # OUVERTURE
//
// La simulation ouvre en LECTURE SEULE par `duckdb.OpenReadForQuery` : correct meme quand le
// serveur local tient la base en ecriture (modele mono-process, ADR 0013/0016). Le `--commit`,
// lui, exige la base pour lui seul — `duckdb.OpenReadWrite` monte un pool a UNE connexion, ce
// qui rend la transaction ci-dessous single-connection par construction.
//
// # FORME DE L'ECRITURE
//
// UN `UPDATE ... WHERE match_id = ?` PAR MATCH, sequentiel, dans une transaction. C'est la
// seule forme autorisee sur `match_registry` : la forme bulk (`UPDATE ... FROM (VALUES ...)`,
// ou un UPDATE a condition large sans parametre) est le declencheur direct du bug ART #23046
// et le garde-rail `internal/sync/no_art_patterns_test.go` la refuse.

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/platform/duckdb"
)

// ligneRegistre : ce que la base sait du T0 d'un match, et rien d'autre.
type ligneRegistre struct {
	// startUTC est le start CANONIQUE (`analysis.SQLStartTimeCanonical`), jamais `start_time`
	// brut — ratchet `archlint/no_raw_start_time_literal_test.go`.
	startUTC time.Time
	// realStart est le debut de gameplay deja en base. NULL = aucun T0 stocke.
	realStart sql.NullTime
	// qualite est `t0_quality`, vide quand la colonne n'a jamais ete renseignee.
	qualite string
}

// ouvrirBase rend un handle et son release. `commit` decide du mode d'acces, et ce couplage
// est voulu : une simulation ne doit JAMAIS pouvoir ecrire, meme par accident de code.
func ouvrirBase(path string, commit bool) (*sql.DB, func(), error) {
	if !commit {
		return duckdb.OpenReadForQuery(path)
	}
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		return nil, nil, err
	}
	return db.SQLDb(), func() { _ = db.Close() }, nil
}

// chargerRegistre lit le registre entier en une requete. Une centaine d'artefacts contre
// quelques milliers de lignes : une passe unique coute moins qu'une requete par artefact, et
// elle donne au compte rendu les matchs SANS ligne comme ceux qui en ont une.
func chargerRegistre(ctx context.Context, db *sql.DB) (map[string]ligneRegistre, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT match_id,
		       `+analysis.SQLStartTimeCanonical("")+` AS start_utc,
		       real_start_time,
		       COALESCE(t0_quality, '')
		FROM match_registry`)
	if err != nil {
		return nil, fmt.Errorf("lecture match_registry: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make(map[string]ligneRegistre)
	for rows.Next() {
		var id string
		var l ligneRegistre
		if err := rows.Scan(&id, &l.startUTC, &l.realStart, &l.qualite); err != nil {
			return nil, fmt.Errorf("scan match_registry: %w", err)
		}
		l.startUTC = l.startUTC.UTC()
		out[id] = l
	}
	return out, rows.Err()
}

// ecrireReparations persiste les reparations. Rend le nombre de lignes touchees.
//
// La premiere erreur annule TOUT le lot : une reparation d'historique a demi ecrite serait
// impossible a distinguer d'une reparation complete au passage suivant.
func ecrireReparations(ctx context.Context, db *sql.DB, reps []reparation) (int, error) {
	if len(reps) == 0 {
		return 0, nil
	}
	// Migration idempotente : la colonne doit exister avant l'UPDATE (meme garde que
	// `cmd/backfill_t0`, pour une base qui n'aurait pas encore recu l'etape partagee).
	if _, err := db.ExecContext(ctx,
		`ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS t0_quality VARCHAR`); err != nil {
		return 0, fmt.Errorf("migration t0_quality: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin: %w", err)
	}
	stmt, err := tx.PrepareContext(ctx,
		`UPDATE match_registry SET real_start_time = ?, t0_quality = ? WHERE match_id = ?`)
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("prepare: %w", err)
	}
	defer func() { _ = stmt.Close() }()
	for _, r := range reps {
		if _, err := stmt.ExecContext(ctx, r.nouveau,
			string(timeline.T0QualityFilmMovement), r.matchID); err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("update %s: %w", r.matchID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return len(reps), nil
}
