// Package migration — steps_shared_match_lives_mates.go : la table des COEQUIPIERS A LA MORT,
// quatrieme projection de la passe de positions du collecteur (plan
// `.ai/archive/V7.5/PLAN_TACTIQUE_COEQUIPIERS_2026-10-14.md`, lot L1).
//
// ─── CE QU'ELLE AJOUTE A `match_death_context` ────────────────────────────────────────────
//
// `match_death_context` ne garde que des COMPTES par etat et la distance du plus proche. Les
// positions que `replay.ContextesDesMorts` calcule pour chaque coequipier etaient jetees. Une
// ligne ici = un coequipier de la victime a l'instant d'une mort du journal : son etat et, pour
// le seul etat `visible`, ou il se tenait (x, y) et a quelle distance horizontale de la victime.
//
// ─── APPEND-ONLY, LA PASSE COMME UNITE DE GENERATION (ADR 0019/0026/0030) ─────────────────
//
// PK technique `id`, `written_at`, INSERT purs. Elle s'ecrit dans la MEME transaction et sous le
// MEME `decode_pass` que `match_lives` et `match_death_context` : la vue `_latest` retient LA
// DERNIERE PASSE ENTIERE par match, si bien que les comptes de `match_death_context_latest` et
// les lignes de cette table decrivent toujours le meme decodage.
//
// ─── UNE TABLE NEUVE, PAS DES COLONNES DE `match_death_context` ───────────────────────────
//
// Une mort a autant de lignes que de coequipiers : l'aplatir en colonnes aurait fige un nombre
// de joueurs par camp.
//
// UN SEUL INDEX, celui qui sert la vue `_latest` et la lecture par mort : chaque index elargit
// la surface ART #23645 le jour ou quelqu'un ecrirait un DELETE.
package migration

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_match_death_mates_v1",
		TargetDB:    TargetShared,
		Description: "Table append-only match_death_mates (etat et position de chaque coequipier a l'instant d'une mort) + vue _latest par passe de decodage — plan Tactique coequipiers, lot L1",
		ApplySchema: applyMatchDeathMates,
	})
}

// applyMatchDeathMates cree la table, son index et sa vue. Idempotente (`IF NOT EXISTS` /
// `OR REPLACE`) : la table est NET-NEUVE, le piege « CREATE TABLE IF NOT EXISTS n'ajoute jamais
// une PK a une table existante » ne s'applique pas. Comme les autres tables du film, elle est
// creee VIDE dans le shared des autres titres.
func applyMatchDeathMates(db *sql.DB) error {
	for _, ddl := range []string{ddlMatchDeathMates, ddlMatchDeathMatesLatest} {
		if err := execScript(db, ddl); err != nil {
			return err
		}
	}
	return nil
}

// ddlMatchDeathMates : une ligne par (mort du journal, coequipier de la victime).
//
// `state` : `visible` | `waiting` | `out_of_sight` (les constantes de `persist`, tenues egales a
// celles de `film/replay` par `archlint/no_life_cause_divergence_test.go`). `x`, `y` et
// `distance_m` (metres monde, deux decimales) sont NULL ensemble hors de l'etat `visible` :
// seule une position repliquee autorise une coordonnee.
const ddlMatchDeathMates = `
	CREATE SEQUENCE IF NOT EXISTS match_death_mates_id_seq START 1;
	CREATE TABLE IF NOT EXISTS match_death_mates (
		id          BIGINT PRIMARY KEY DEFAULT nextval('match_death_mates_id_seq'),
		match_id    VARCHAR   NOT NULL,
		-- decode_pass : identifiant d UNE passe, le meme que match_lives et match_death_context.
		decode_pass VARCHAR   NOT NULL,
		decoder_rev VARCHAR   NOT NULL,
		written_at  TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),

		-- (match_id, victim_xuid, time_ms) : la cle de la mort dans match_kill_events.
		victim_xuid VARCHAR   NOT NULL,
		time_ms     BIGINT    NOT NULL,
		mate_xuid   VARCHAR   NOT NULL,
		state       VARCHAR   NOT NULL,
		x           DOUBLE,
		y           DOUBLE,
		distance_m  DOUBLE
	);
	CREATE INDEX IF NOT EXISTS idx_match_death_mates_lookup
		ON match_death_mates(match_id, victim_xuid, time_ms, written_at);
`

const ddlMatchDeathMatesLatest = `
	CREATE OR REPLACE VIEW match_death_mates_latest AS
	SELECT m.*
	FROM match_death_mates AS m
	QUALIFY m.decode_pass = FIRST_VALUE(m.decode_pass) OVER (
		PARTITION BY m.match_id ORDER BY m.written_at DESC, m.id DESC
	);
`
