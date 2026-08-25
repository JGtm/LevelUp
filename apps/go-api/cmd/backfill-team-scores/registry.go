package main

// registry.go — LA JONCTION AVEC LA BASE, RÉDUITE À CE QU'ON PEUT TESTER.
//
// POURQUOI CE FICHIER EXISTE. La règle de correction (`decide.go`) est pure et couverte,
// mais elle ne protège de rien si le câblage qui l'entoure lit les mauvaises colonnes ou
// lie les valeurs à l'envers : un `SET team_0_score = ?, team_1_score = ?` dont les deux
// arguments seraient permutés produirait un journal PARFAITEMENT correct — il est construit
// depuis la décision, pas depuis ce qui part réellement vers la base — et écrirait pourtant
// l'inverse de ce qui a été décidé. Constat P1 de la revue adversariale du 2026-08-24.
//
// D'où les coutures ci-dessous. Elles ne sont pas de l'abstraction pour l'abstraction :
// chacune existe pour qu'un double de test puisse observer EXACTEMENT ce que la vraie
// implémentation enverrait à DuckDB — le texte SQL, l'ordre des arguments liés, et la
// correspondance colonne -> champ au retour.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// selectScoresSQL et updateScoresSQL sont sortis des fonctions pour être ASSERTABLES par
// les tests : l'ordre des colonnes du SELECT et celui des placeholders de l'UPDATE font
// partie du contrat, pas du détail d'implémentation.
const (
	selectScoresSQL = `SELECT team_0_score, team_1_score FROM match_registry WHERE match_id = ?`
	updateScoresSQL = `UPDATE match_registry SET team_0_score = ?, team_1_score = ? WHERE match_id = ?`
)

// execer est le minimum dont l'écriture a besoin. `*dblease.LeasedWriter` et `*sql.DB` le
// satisfont tels quels ; un double de test aussi, ce qui rend le SQL et l'ordre des
// arguments observables.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// queryRower est le pendant en lecture. `*sql.DB` le satisfait.
type queryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// rowScanner est ce que la lecture consomme réellement d'une ligne : un Scan. `*sql.Row`
// le satisfait, et un double aussi — c'est ce qui permet de tester la correspondance
// « position dans le SELECT -> champ de RegistryScores » sans base.
type rowScanner interface {
	Scan(dest ...any) error
}

// registryReader lit les scores courants d'un match. found=false si le match n'est pas
// au registre.
type registryReader interface {
	ReadScores(ctx context.Context, matchID string) (RegistryScores, bool, error)
}

// registryWriter écrit les scores d'un match.
type registryWriter interface {
	WriteScores(ctx context.Context, matchID string, team0, team1 int) error
}

// matchFetcher rend le payload GetMatchStats d'un match. Coupe le réseau pour les tests.
type matchFetcher interface {
	GetMatchStats(ctx context.Context, matchID string) (map[string]any, error)
}

// sqlRegistry est l'implémentation réelle, au-dessus d'une base DuckDB.
//
// Les deux rôles sont portés par des champs SÉPARÉS, et c'est volontaire : en phase A
// l'outil n'a qu'un lecteur (aucun droit d'écriture n'existe encore), en phase B il a les
// deux. Un `ex` nil n'est donc pas un oubli, c'est l'état normal d'une répétition à blanc,
// et `WriteScores` refuse alors d'écrire au lieu de paniquer.
type sqlRegistry struct {
	q  queryRower
	ex execer
}

// ReadScores lit la ligne du registre.
func (r sqlRegistry) ReadScores(ctx context.Context, matchID string) (RegistryScores, bool, error) {
	if r.q == nil {
		return RegistryScores{}, false, errors.New("lecture demandée sans base (bug d'appel)")
	}
	return scanRegistryScores(r.q.QueryRowContext(ctx, selectScoresSQL, matchID))
}

// scanRegistryScores traduit UNE ligne en RegistryScores.
//
// L'ordre des destinations suit celui de `selectScoresSQL` : team_0 puis team_1. Cette
// fonction est isolée pour qu'un double de rowScanner puisse prouver que la première
// colonne atterrit bien dans Team0 — une permutation ici serait invisible partout ailleurs.
func scanRegistryScores(sc rowScanner) (RegistryScores, bool, error) {
	var t0, t1 sql.NullInt64
	err := sc.Scan(&t0, &t1)
	if errors.Is(err, sql.ErrNoRows) {
		return RegistryScores{}, false, nil
	}
	if err != nil {
		return RegistryScores{}, false, err
	}
	return RegistryScores{Team0: nullIntPtr(t0), Team1: nullIntPtr(t1)}, true, nil
}

// WriteScores écrit UNE ligne.
//
// Forme volontairement row-by-row, `WHERE match_id = ?`, toutes les valeurs liées à des
// placeholders. Un `UPDATE … FROM (VALUES …)` ou un UPDATE set-based nu déclencheraient le
// bug ART #23046 sur une table indexée ; le ratchet `internal/sync/no_art_patterns_test.go`
// les interdit dans le code serveur mais EXCLUT explicitement `cmd/` de son périmètre
// (`no_art_patterns_test.go:220`). Cette forme est donc un choix conforme à la doctrine, PAS
// une forme imposée par un garde-rail existant — c'est `no_bulk_update_test.go`, local à ce
// paquet, qui la protège réellement.
func (r sqlRegistry) WriteScores(ctx context.Context, matchID string, team0, team1 int) error {
	if r.ex == nil {
		return errors.New("écriture demandée sans droit d'écriture (bug d'appel)")
	}
	res, err := r.ex.ExecContext(ctx, updateScoresSQL, team0, team1, matchID)
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

func nullIntPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}
