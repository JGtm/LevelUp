package migrations

// steps_shared_kill_openings.go — table `kill_openings` : OÙ LES DEUX JOUEURS ÉTAIENT UN
// TEMPS-POUR-TUER AVANT LE COUP FATAL (plan .ai/PLAN_DUELS_PORTEE_2026-09-06.md, D5).
//
// ─── POURQUOI UNE TABLE SŒUR, ET PAS SIX COLONNES DE PLUS SUR kill_positions ──────────────
//
// La couverture des deux mesures N'EST PAS LA MÊME et ne le sera jamais : les positions du
// coup fatal existent pour tout match décodé, l'entame n'existe que là où les deux
// trajectoires portent un échantillon 1,5 s plus tôt (une mort dans les premières secondes
// d'un match n'a pas d'entame lisible, et c'est le résultat CORRECT — mieux vaut pas
// d'entame qu'une position de réapparition présentée comme une entame). Les fusionner
// obligerait à écrire des NULL dans une ligne existante, donc à RÉÉCRIRE une ligne
// append-only : exactement ce que la doctrine ART interdit. Deux tables, deux passes
// d'INSERT, deux couvertures qui se lisent séparément.
//
// ─── `time_ms` EST L'INSTANT DU KILL, PAS L'INSTANT MESURÉ ────────────────────────────────
//
// C'est le point le plus facile à se tromper de toute la table. La ligne porte le `time_ms`
// DU COUP FATAL — la clé du frag, celle par laquelle `match_kill_events_latest` se joint —
// et des COORDONNÉES prises à `time_ms - replay.OpeningLeadMS`. Écrire l'instant décalé
// rendrait la jointure du lecteur impossible (aucun kill à cet instant-là) et obligerait à
// une seconde jointure décalée, donc à un second lecteur qui divergerait du premier.
//
// ─── FORME APPEND-ONLY (ADR 0026), CRÉÉE DIRECTEMENT ──────────────────────────────────────
//
// Table NET-NEUVE : `id` PK adossée à une séquence + `written_at`, et lecture par la vue
// `kill_openings_latest` UNIQUEMENT — jamais `ApplyAppendOnlyRebuild`, qui est la recette de
// CONVERSION d'une table mutable existante (kill_positions y est passée en G.2). ART-safe
// par construction (#23046) : écriture = INSERT pur (`persist.KillOpeningPersister`), un
// re-décodage écrit une NOUVELLE passe que la vue fait gagner, il ne réécrit rien.
//
// Clé fonctionnelle de la vue : `(match_id, killer_xuid, time_ms)` — LA MÊME que
// kill_positions, parce que c'est la même population de frags vue à un autre instant.
//
// ─── CONSÉQUENCE ASSUMÉE — la table existe sur tous les titres qui héritent de ce schéma ──
//
// Comme sa sœur `kill_positions` (cf. steps_shared_kill_positions.go), la table est créée
// dans le shared de tout titre qui hérite des migrations Halo Infinite. Sur un titre dont le
// film ne porte pas de trajectoires elle reste VIDE, et le branchement produit se fait sur
// capability, jamais sur le slug.

import (
	"database/sql"

	"levelup/go-api/internal/migration"
)

// sharedKillOpeningsSteps — la table d'entame et sa vue `_latest`.
func sharedKillOpeningsSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:     "shared_create_kill_openings",
			TargetDB: migration.TargetShared,
			Description: "Table append-only kill_openings (positions monde tueur/victime un temps-pour-tuer " +
				"avant le coup fatal, proxy d entame D5) + index de jointure + vue kill_openings_latest",
			ApplySchema: applyKillOpenings,
		},
	}
}

// applyKillOpenings crée la séquence, la table, son index et sa vue. Idempotente
// (`IF NOT EXISTS` / `OR REPLACE`). La table étant NET-NEUVE, le piège « CREATE TABLE IF NOT
// EXISTS n'ajoute jamais une PK à une table existante » ne s'applique pas.
//
// AUCUN commentaire SQL dans le script : le splitter de `ExecScript` coupe naïvement sur
// `;`, et un `;` glissé dans un commentaire `--` casserait la migration (piège ADR 0026).
// L'explication vit donc dans l'en-tête de ce fichier.
//
// TOUT ÉLARGISSEMENT FUTUR passe par un step au NOM NEUF qui RECRÉE la vue dans le même
// step : les migrations sont name-keyed (une base déjà migrée ne rejoue jamais celle-ci) et
// DuckDB FIGE la liste de colonnes d'un `SELECT *` à la création de la vue.
func applyKillOpenings(db *sql.DB) error {
	return migration.ExecScript(db, `
		CREATE SEQUENCE IF NOT EXISTS kill_openings_id_seq START 1;
		CREATE TABLE IF NOT EXISTS kill_openings (
			id          BIGINT    PRIMARY KEY DEFAULT nextval('kill_openings_id_seq'),
			match_id    VARCHAR   NOT NULL,
			killer_xuid VARCHAR   NOT NULL,
			time_ms     INTEGER   NOT NULL,
			killer_x    DOUBLE, killer_y DOUBLE, killer_z DOUBLE,
			victim_x    DOUBLE, victim_y DOUBLE, victim_z DOUBLE,
			written_at  TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE INDEX IF NOT EXISTS idx_kill_openings_lookup
			ON kill_openings(match_id, killer_xuid, time_ms, written_at);
		CREATE OR REPLACE VIEW kill_openings_latest AS
		SELECT * FROM kill_openings
		QUALIFY ROW_NUMBER() OVER (
			PARTITION BY match_id, killer_xuid, time_ms
			ORDER BY written_at DESC, id DESC
		) = 1;
	`)
}
