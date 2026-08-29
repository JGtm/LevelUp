package main

// registry.go — LA JONCTION AVEC LA BASE, RÉDUITE À CE QU'ON PEUT TESTER.
//
// POURQUOI CE FICHIER EXISTE. La règle (`decide.go`) est pure et couverte, mais elle ne
// protège de rien si le câblage qui l'entoure lit les mauvaises colonnes ou lie les valeurs
// à l'envers : un `SET team_0_rounds_won = ?, team_1_rounds_won = ?` dont les arguments
// seraient permutés produirait un journal PARFAITEMENT correct — il est construit depuis la
// décision, pas depuis ce qui part réellement vers la base — et écrirait pourtant l'inverse.
// Même constat que sur `cmd/backfill-team-scores` (revue adversariale du 2026-08-24), même
// parade : des coutures observables par un double de test.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// selectRoundsSQL, updateRoundsSQL et pendingIDsSQL sont sortis des fonctions pour être
// ASSERTABLES par les tests : l'ordre des colonnes du SELECT et celui des placeholders de
// l'UPDATE font partie du contrat, pas du détail d'implémentation.
const (
	selectRoundsSQL = `SELECT team_0_rounds_won, team_1_rounds_won, rounds_total FROM match_registry WHERE match_id = ?`
	updateRoundsSQL = `UPDATE match_registry SET team_0_rounds_won = ?, team_1_rounds_won = ?, rounds_total = ? WHERE match_id = ?`
	// pendingIDsSQL est la liste de travail par défaut : les matchs dont les manches sont
	// encore inconnues. `team_0_score IS NOT NULL` écarte les lignes sans aucune donnée
	// d'équipe (rien à espérer de leur payload côté camps).
	pendingIDsSQL = `SELECT match_id FROM match_registry
		WHERE rounds_total IS NULL AND team_0_score IS NOT NULL
		ORDER BY match_id`
)

// execer est le minimum dont l'écriture a besoin. `*dblease.LeasedWriter` et `*sql.DB` le
// satisfont tels quels ; un double de test aussi, ce qui rend le SQL et l'ordre des
// arguments observables.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// queryRower est le pendant en lecture d'UNE ligne. `*sql.DB` le satisfait.
type queryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// rowScanner est ce que la lecture consomme réellement d'une ligne : un Scan. `*sql.Row`
// le satisfait, et un double aussi — c'est ce qui permet de tester la correspondance
// « position dans le SELECT -> champ de RegistryRounds » sans base.
type rowScanner interface {
	Scan(dest ...any) error
}

// registryReader lit les manches courantes d'un match. found=false si le match est absent.
type registryReader interface {
	ReadRounds(ctx context.Context, matchID string) (RegistryRounds, bool, error)
}

// registryWriter écrit les manches d'un match.
type registryWriter interface {
	WriteRounds(ctx context.Context, matchID string, won0, won1, total int) error
}

// matchFetcher rend le payload GetMatchStats d'un match. Coupe le réseau pour les tests.
type matchFetcher interface {
	GetMatchStats(ctx context.Context, matchID string) (map[string]any, error)
}

// sqlRegistry est l'implémentation réelle, au-dessus d'une base DuckDB.
//
// Les deux rôles sont portés par des champs SÉPARÉS : en phase A l'outil n'a qu'un lecteur
// (aucun droit d'écriture n'existe encore), en phase B il a les deux. Un `ex` nil n'est donc
// pas un oubli, c'est l'état normal d'une répétition à blanc, et `WriteRounds` refuse alors
// d'écrire au lieu de paniquer.
type sqlRegistry struct {
	q  queryRower
	ex execer
}

// ReadRounds lit la ligne du registre.
func (r sqlRegistry) ReadRounds(ctx context.Context, matchID string) (RegistryRounds, bool, error) {
	if r.q == nil {
		return RegistryRounds{}, false, errors.New("lecture demandée sans base (bug d'appel)")
	}
	return scanRegistryRounds(r.q.QueryRowContext(ctx, selectRoundsSQL, matchID))
}

// scanRegistryRounds traduit UNE ligne en RegistryRounds.
//
// L'ordre des destinations suit celui de `selectRoundsSQL` : camp 0, camp 1, total. Cette
// fonction est isolée pour qu'un double de rowScanner puisse prouver que la première
// colonne atterrit bien dans Team0Won — une permutation ici serait invisible partout ailleurs.
func scanRegistryRounds(sc rowScanner) (RegistryRounds, bool, error) {
	var w0, w1, tot sql.NullInt64
	err := sc.Scan(&w0, &w1, &tot)
	if errors.Is(err, sql.ErrNoRows) {
		return RegistryRounds{}, false, nil
	}
	if err != nil {
		return RegistryRounds{}, false, err
	}
	return RegistryRounds{Team0Won: nullIntPtr(w0), Team1Won: nullIntPtr(w1), Total: nullIntPtr(tot)}, true, nil
}

// WriteRounds écrit UNE ligne.
//
// Forme volontairement row-by-row, `WHERE match_id = ?`, toutes les valeurs liées à des
// placeholders. Un `UPDATE … FROM (VALUES …)` ou un UPDATE set-based nu déclencheraient le
// bug ART #23046 sur une table indexée ; le ratchet `internal/sync/no_art_patterns_test.go`
// les interdit dans le code serveur mais EXCLUT explicitement `cmd/` de son périmètre.
// C'est donc `no_bulk_update_test.go`, local à ce paquet, qui protège réellement la forme.
func (r sqlRegistry) WriteRounds(ctx context.Context, matchID string, won0, won1, total int) error {
	if r.ex == nil {
		return errors.New("écriture demandée sans droit d'écriture (bug d'appel)")
	}
	res, err := r.ex.ExecContext(ctx, updateRoundsSQL, won0, won1, total, matchID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		// Pilote sans RowsAffected : l'UPDATE a réussi, on ne peut simplement pas compter.
		return nil
	}
	if n != 1 {
		return fmt.Errorf("UPDATE a touché %d lignes au lieu de 1", n)
	}
	return nil
}

// pendingMatchIDs rend la liste de travail par défaut, lue DANS LA BASE.
//
// Pas de fichier de liste ici, contrairement à `cmd/backfill-team-scores` : la population à
// traiter se définit exactement par « rounds_total IS NULL », ce que la base sait dire. La
// conséquence est heureuse — l'outil est reprenable tel quel après une interruption, et un
// second passage ne re-télécharge que ce qui manque encore.
func pendingMatchIDs(ctx context.Context, db *sql.DB, limit int) ([]string, error) {
	q := pendingIDsSQL
	if limit > 0 {
		q = fmt.Sprintf("%s LIMIT %d", q, limit)
	}
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("lecture de la liste de travail : %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan match_id : %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func nullIntPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}
