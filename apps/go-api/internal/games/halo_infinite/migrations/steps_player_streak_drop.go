package migrations

// steps_player_streak_drop.go — drop_streak_legacy_v1 : retrait de la table héritée `streak`.
//
// `streak` a été remplacée par `streak_history` (INSERT pur) et la vue `streak_latest` ; la
// recopie des lignes est faite par create_streak_history_append_only, qui précède ce step dans
// canonicalOrder. Une table `streak` qui subsiste fait croire au réparateur de clés déclarées
// (declared_keys_repair.go) que la table est append-only (la vue streak_latest porte son nom de
// base) et lui fait refuser chaque démarrage une réparation qu'il ne peut pas faire.
//
// Aucune ligne n'est perdue : la table n'est supprimée que si chacun de ses id existe déjà dans
// streak_history. Sinon elle reste, intacte, et le journal le dit.

import (
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/migration"
)

// dropLegacyStreakTable supprime `streak` quand streak_history a reçu la recopie.
func dropLegacyStreakTable(db *sql.DB) error {
	hasLegacy, err := migration.TableExists(db, "streak")
	if err != nil {
		return fmt.Errorf("drop_streak_legacy: check streak: %w", err)
	}
	if !hasLegacy {
		return nil
	}
	hasHistory, err := migration.TableExists(db, "streak_history")
	if err != nil {
		return fmt.Errorf("drop_streak_legacy: check streak_history: %w", err)
	}
	if !hasHistory {
		slog.WarnContext(migration.BootCtx(), "streak héritée conservée : streak_history absente, rien n'a été recopié")
		return nil
	}
	var missing int
	if err := db.QueryRowContext(migration.BootCtx(),
		`SELECT COUNT(*) FROM streak s WHERE NOT EXISTS (SELECT 1 FROM streak_history h WHERE h.id = s.id)`,
	).Scan(&missing); err != nil {
		return fmt.Errorf("drop_streak_legacy: count rows not copied: %w", err)
	}
	if missing > 0 {
		slog.WarnContext(migration.BootCtx(), "streak héritée conservée : des lignes ne sont pas dans streak_history",
			"lignes_non_recopiees", missing)
		return nil
	}
	// Les index d'abord : un DROP TABLE refuse une table dont des index dépendent.
	return migration.ExecScript(db, `
		DROP INDEX IF EXISTS idx_streak_user_title_type;
		DROP INDEX IF EXISTS idx_streak_user_status;
		DROP TABLE streak;
	`)
}
