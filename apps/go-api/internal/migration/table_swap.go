package migration

// table_swap.go — LE CŒUR COMMUN des reconstructions de table par swap (ADR 0026).
//
// Deux familles l'utilisent, et c'est pourquoi il vit à part :
//   - la conversion append-only (append_only_rebuild.go) : legacy -> table à `id` + vue `_latest` ;
//   - la compaction des passes supersédées (compaction.go) : table append-only -> la même table,
//     réduite aux lignes que sa vue `_latest` retient.
//
// Les deux ont la même exigence de sûreté, écrite UNE fois ici :
//   - JAMAIS de DELETE ni d'UPDATE (bug DuckDB ART #23645) : la table neuve est construite à côté
//     (`<table><suffixe>`), puis l'ancienne est DROP et la neuve renommée ;
//   - tout dans UNE transaction : une erreur à n'importe quelle étape rend la base intacte ;
//   - garde de cardinalité AVANT le DROP : la table neuve doit porter exactement le nombre de
//     lignes attendu, lu dans la même transaction ;
//   - `recoverOrphanTable` répare l'état « principale absente + `<table><suffixe>` présente »
//     qu'un swap non transactionnel aurait pu laisser (défense en profondeur : le swap ci-dessous
//     ne peut pas le produire, une transaction DuckDB non committée ne laisse rien).
//
// Ordre imposé par DuckDB, mesuré : une table qui porte un index ne se RENOMME pas (« Cannot
// alter entry … because there are entries that depend on it »). Les index se posent donc APRÈS le
// RENAME (`PostRename`), dans la transaction.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// tableSwap décrit l'échange transactionnel `<Table>` <- `<Table><Suffix>`.
type tableSwap struct {
	// Table : nom logique, identique avant et après l'échange.
	Table string
	// Suffix : suffixe de la table de construction (`__appendonly`, `__compact`).
	Suffix string
	// Expected : requête qui rend la cardinalité ATTENDUE de la table neuve, lue dans la
	// transaction AVANT Build (même instantané que la construction).
	Expected string
	// Build : statements qui créent et remplissent `<Table><Suffix>`, joués dans la transaction
	// après le DROP d'une éventuelle table de construction périmée.
	Build []string
	// PostRename : statements joués après le RENAME (clé primaire, défauts, index).
	PostRename []string
	// Verify : contrôle optionnel dans la transaction, après PostRename, avant le COMMIT. Une
	// erreur annule tout l'échange.
	Verify func(ctx context.Context, tx *sql.Tx) error
}

// swapCounts : ce que la garde de cardinalité a vu.
type swapCounts struct {
	Expected int64
	Rebuilt  int64
}

func (s tableSwap) scratch() string { return s.Table + s.Suffix }

// swapTableTx joue l'échange décrit par `s` dans une transaction unique. Rollback intégral sur la
// moindre erreur, y compris une garde de cardinalité ou un Verify qui échoue.
func swapTableTx(ctx context.Context, db *sql.DB, s tableSwap) (swapCounts, error) {
	var n swapCounts
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return n, fmt.Errorf("swap %s: begin tx: %w", s.Table, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback() // no-op après un Commit réussi ; l'erreur d'origine est déjà rendue
		}
	}()

	if err := tx.QueryRowContext(ctx, s.Expected).Scan(&n.Expected); err != nil {
		return n, fmt.Errorf("swap %s: cardinalité attendue: %w", s.Table, err)
	}
	build := append([]string{fmt.Sprintf(`DROP TABLE IF EXISTS %s`, s.scratch())}, s.Build...)
	if err := execAll(ctx, tx, s.Table, build); err != nil {
		return n, err
	}
	if err := tx.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM %s`, s.scratch())).Scan(&n.Rebuilt); err != nil {
		return n, fmt.Errorf("swap %s: count %s: %w", s.Table, s.scratch(), err)
	}
	if n.Rebuilt != n.Expected {
		return n, fmt.Errorf("swap %s: abandonné, rebuilt=%d != attendu=%d (rollback, zéro perte)",
			s.Table, n.Rebuilt, n.Expected)
	}

	swap := append([]string{
		fmt.Sprintf(`DROP TABLE %s`, s.Table),
		fmt.Sprintf(`ALTER TABLE %s RENAME TO %s`, s.scratch(), s.Table),
	}, s.PostRename...)
	if err := execAll(ctx, tx, s.Table, swap); err != nil {
		return n, err
	}
	if s.Verify != nil {
		if err := s.Verify(ctx, tx); err != nil {
			return n, fmt.Errorf("swap %s: vérification avant commit: %w", s.Table, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return n, fmt.Errorf("swap %s: commit: %w", s.Table, err)
	}
	committed = true
	return n, nil
}

func execAll(ctx context.Context, tx *sql.Tx, table string, stmts []string) error {
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("swap %s: étape (%s): %w", table, firstWords(stmt, 3), err)
		}
	}
	return nil
}

// recoverOrphanTable répare l'état laissé par un crash AU MILIEU d'un swap non transactionnel :
// table principale absente + `<table><suffix>` présente. On renomme la seconde en principale.
// Idempotent (no-op si la principale existe ou si la table de construction est absente).
func recoverOrphanTable(ctx context.Context, db *sql.DB, table, suffix string) error {
	hasMain, err := tableExists(db, table)
	if err != nil {
		return fmt.Errorf("swap %s: check main: %w", table, err)
	}
	if hasMain {
		return nil
	}
	orphan := table + suffix
	hasOrphan, err := tableExists(db, orphan)
	if err != nil {
		return fmt.Errorf("swap %s: check %s: %w", table, orphan, err)
	}
	if !hasOrphan {
		return nil
	}
	slog.WarnContext(ctx, "swap: table de construction orpheline (crash mid-swap) — récupération",
		"table", table, "action", "RENAME "+orphan+" -> "+table)
	if _, err := db.ExecContext(ctx, fmt.Sprintf(
		`ALTER TABLE %s RENAME TO %s`, orphan, table)); err != nil {
		return fmt.Errorf("swap %s: recover orphan: %w", table, err)
	}
	return nil
}
