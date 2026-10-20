// Package migration — steps_shared_kill_events_sans_killfeed.go : la revision de decodeur sous
// laquelle le film d'un match a ete lu SANS AUCUN KILL.
//
// ─── CE QUE LA COLONNE DIT ───────────────────────────────────────────────────────────────
//
// `match_registry.killsource_sans_killfeed_rev` porte la revision de la sortie killsource
// (`decfilm.Rev`) sous laquelle le film complet du match a ete decode sans qu'aucun evenement
// `kill` n'y soit lu. Le backlog de l'etape 1.57 (`conditionBacklog`) et la selection de
// `levelup backfill-killsource` tiennent le match pour A JOUR quand elle vaut la revision
// courante : sans elle, un tel film n'ecrit aucune ligne dans `match_kill_events`, reste candidat
// a vie et occupe la liste de travail a chaque cycle. NULL = jamais constate.
//
// UNE REVISION ET NON UN BIT DE `backfill_completed` : un bit serait terminal, or une revision
// de decodeur peut lire ce que la precedente ne lisait pas. Quand `decfilm.Rev` change, la
// valeur ne correspond plus et le match redevient candidat sans autre ecriture.
//
// ─── ECRITURE ────────────────────────────────────────────────────────────────────────────
//
// `match_registry` n'est pas append-only : la colonne s'ecrit par un `UPDATE ... WHERE
// match_id = ?` par match, sous le writer exclusif (`persist.KillSourceSansKillFeedPersister`),
// la forme autorisee par `no_art_patterns_test.go`. La colonne ne porte aucun index.
package migration

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "shared_registry_killsource_sans_killfeed_rev_v1",
		TargetDB:    TargetShared,
		Description: "Colonne killsource_sans_killfeed_rev sur match_registry (revision de decodeur sous laquelle le film a ete lu sans kill — sort le match du backlog killsource pour cette revision)",
		ApplySchema: applyKillSourceSansKillFeedRev,
	})
}

// applyKillSourceSansKillFeedRev ajoute la colonne. Idempotente (`IF NOT EXISTS`). Garde
// `match_registry` absente : la table est creee par `create_base_shared_schema`, ordonne avant.
func applyKillSourceSansKillFeedRev(db *sql.DB) error {
	exists, err := tableExists(db, "match_registry")
	if err != nil {
		return fmt.Errorf("killsource_sans_killfeed_rev: check table: %w", err)
	}
	if !exists {
		return nil
	}
	return execScript(db, `
		ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS killsource_sans_killfeed_rev VARCHAR;
	`)
}
