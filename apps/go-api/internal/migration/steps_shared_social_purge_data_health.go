package migration

// steps_shared_social_purge_data_health.go — purge des notifs
// `data_health_warning` historiques.
//
// Contexte : la catégorie `data_health_warning` (audit DB périodique) émettait
// du jargon dev ("76 bits menteurs", "12 XUIDs orphelins") sans valeur pour
// un end user lambda sur une app de stats. Décision 2026-05-20 : retirer
// l'émission côté `scheduler/data_health_check.go`. Mais les notifs déjà
// persistées dans `player_notifications` continuent d'apparaître dans la
// cloche tant qu'elles ne sont pas supprimées.
//
// Cette migration les purge une fois pour toutes, tous joueurs confondus.
// Idempotente : si elles ont déjà été supprimées (DB neuve ou ré-run),
// l'opération est un no-op.
//
// Stratégie « rebuild via swap » :
// Un simple `DELETE` rate à cause d'une corruption d'index ART connue sur
// `player_notifications` (cf. thought_log 2026-05-20 « index ART corrompu
// sur shared.match_participants »). DuckDB rapporte : « Failed to delete
// all rows from index. Only deleted 0 out of N rows. » et l'erreur est
// fatale → la connection est invalidée pour le reste du process, ce qui
// casse les premières requêtes home du cold-start.
//
// Workaround robuste : table swap. On crée une nouvelle table avec les
// lignes à conserver (toutes catégories sauf `data_health_warning`), on
// drop l'ancienne, on renomme. Les index ART sont reconstruits depuis zéro
// et la corruption disparaît. Idempotent : si zéro ligne à supprimer, on
// court-circuite (pas de swap inutile).

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func init() {
	Register(Migration{
		Name:        "purge_data_health_warning_notifs",
		TargetDB:    TargetSharedSocial,
		Description: "Supprime les notifs persistées de catégorie `data_health_warning` (catégorie retirée le 2026-05-20).",
		ApplySchema: func(db *sql.DB) error {
			// Si la table n'existe pas encore (DB neuve où la migration
			// originale n'a pas tourné), no-op.
			hasTable, err := tableExists(db, "player_notifications")
			if err != nil {
				return fmt.Errorf("purge_data_health: check table: %w", err)
			}
			if !hasTable {
				return nil
			}

			// Court-circuit : si zéro ligne à purger, pas la peine de payer
			// le swap (idempotent ↔ DB neuve ou ré-run après cleanup OK).
			var toDelete int
			if err := db.QueryRowContext(bootCtx(), `
				SELECT COUNT(*) FROM player_notifications
				WHERE category = 'data_health_warning'
			`).Scan(&toDelete); err != nil {
				return fmt.Errorf("purge_data_health: count: %w", err)
			}
			if toDelete == 0 {
				return nil
			}

			// Rebuild via swap pour contourner la corruption d'index ART.
			// On recrée le PRIMARY KEY + 3 indexes après le rename pour
			// préserver les perfs des lectures notifications (cf.
			// steps_player_notifications.go pour la définition canonique).
			stmts := []string{
				// 1. Nouvelle table avec uniquement les lignes à conserver.
				//    Schéma copié à l'identique de player_notifications.
				`CREATE TABLE player_notifications__purged (
					xuid          VARCHAR NOT NULL,
					id            BIGINT NOT NULL,
					category      VARCHAR NOT NULL,
					severity      VARCHAR NOT NULL DEFAULT 'info',
					title_key     VARCHAR NOT NULL,
					body_key      VARCHAR,
					params        VARCHAR,
					target_route  VARCHAR,
					target_search VARCHAR,
					actor_xuid    VARCHAR,
					actor_name    VARCHAR,
					source        VARCHAR NOT NULL,
					created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					read_at       TIMESTAMP,
					PRIMARY KEY (xuid, id)
				)`,
				`INSERT INTO player_notifications__purged
				 SELECT * FROM player_notifications
				 WHERE category <> 'data_health_warning'`,
				`DROP TABLE player_notifications`,
				`ALTER TABLE player_notifications__purged RENAME TO player_notifications`,
				// Recréer les indexes secondaires SÛRS uniquement (cf.
				// steps_player_notifications.go). NE PAS recréer idx_pn_xuid_unread :
				// read_at est muté par MarkNotifications* → index ART corrupteur
				// (régression : il avait été droppé par drop_idx_pn_xuid_unread le
				// 2026-05-15, ce rebuild le réarmait). created_at/category jamais mutés.
				`CREATE INDEX IF NOT EXISTS idx_pn_xuid_created_desc ON player_notifications(xuid, created_at DESC)`,
				`CREATE INDEX IF NOT EXISTS idx_pn_xuid_category     ON player_notifications(xuid, category)`,
			}
			for _, sqlStmt := range stmts {
				if _, err := db.ExecContext(bootCtx(), sqlStmt); err != nil {
					return fmt.Errorf("purge_data_health: swap step (%s): %w",
						firstWords(sqlStmt, 3), err)
				}
			}

			slog.Info("migration purge_data_health_warning_notifs: lignes purgées (rebuild via swap)",
				"deleted", toDelete)
			return nil
		},
	})
}

// firstWords retourne les n premiers mots d'une SQL statement pour des
// messages d'erreur lisibles sans dumper la requête complète.
func firstWords(s string, n int) string {
	out := make([]byte, 0, 32)
	wordCount := 0
	inWord := false
	for i := 0; i < len(s) && wordCount < n; i++ {
		c := s[i]
		isSpace := c == ' ' || c == '\n' || c == '\t' || c == '\r'
		if isSpace {
			if inWord {
				wordCount++
				if wordCount < n {
					out = append(out, ' ')
				}
				inWord = false
			}
			continue
		}
		out = append(out, c)
		inWord = true
	}
	return string(out)
}
