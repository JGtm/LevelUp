// Package migration — steps_shared_match_lives_placement.go : la table du PLACEMENT DES VIES,
// ecrite AU SYNC par le collecteur de kills (plan `.ai/V7.5/PLAN_EMPRISE_VIES_2026-09-28.md`, lot V2).
//
// ─── UNE TROISIEME PROJECTION DU MEME MATERIAU ───────────────────────────────────────────
//
// Les vies (`match_lives`) et le contexte des morts (`match_death_context`) sortent de la passe
// de positions du collecteur ; le placement d'une vie en sort aussi, sur le MEME registre et les
// MEMES positions (`replay.PlacementDesVies`). Une ligne ici = une ligne de `match_lives`, meme
// cle `(match_id, xuid, start_ms)`, meme horloge du MATCH.
//
// ─── APPEND-ONLY, LA PASSE COMME UNITE DE GENERATION (ADR 0026/0030) ─────────────────────
//
// PK technique `id`, `written_at`, INSERT purs. La vue `_latest` retient LA DERNIERE PASSE
// ENTIERE PAR MATCH (`decode_pass`), jamais la derniere ligne par cle — meme doctrine que
// `match_lives_latest` : une passe qui rendrait moins de vies ne doit laisser survivre aucune
// ligne de la precedente.
//
// ─── UNE TABLE NEUVE, PAS DES COLONNES DE `match_lives` ──────────────────────────────────
//
// Le contenu de `match_lives` ne change pas, et sa revision (`IsolationDecoderRev`) non plus.
// Des colonnes ajoutees la-bas auraient exige de redecoder tout le corpus pour les remplir et
// auraient lie deux revisions qui evoluent separement. Le placement porte la sienne
// (`PlacementRev`, dans `decoder_rev`).
package migration

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_match_life_placement_v1",
		TargetDB:    TargetShared,
		Description: "Table append-only match_life_placement (placement et rendement de chaque vie nommee) + vue _latest par passe de decodage — plan Emprise vies, lot V2",
		ApplySchema: applyMatchLifePlacement,
	})
}

// applyMatchLifePlacement cree la table, son index et sa vue. Idempotente (`IF NOT EXISTS` /
// `OR REPLACE`) : la table est NET-NEUVE, le piege « CREATE TABLE IF NOT EXISTS n'ajoute jamais
// une PK a une table existante » ne s'applique pas.
//
// Relocation : comme `shared_match_lives_v1`, ce step appartient fonctionnellement a Halo
// Infinite (le film) ; la table est creee VIDE dans le shared des autres titres.
func applyMatchLifePlacement(db *sql.DB) error {
	for _, ddl := range []string{ddlMatchLifePlacement, ddlMatchLifePlacementLatest} {
		if err := execScript(db, ddl); err != nil {
			return err
		}
	}
	return nil
}

// ddlMatchLifePlacement : une ligne par VIE NOMMEE mesuree (decision V3 du plan).
//
// ─── LES CINQ CUMULS D'UNE VIE ────────────────────────────────────────────────────────────
//
// La vie est parcourue sur une grille de 100 ms, bornes incluses ; chaque instant recoit UNE
// cause, dans cet ordre : `carrier` (le joueur porte l'objectif), `team_down` (aucun coequipier
// vivant), `unplaced` (le joueur n'a pas de position de moins d'une seconde — vehicule ou
// lecture manquee), `teammate_unplaced` (un coequipier vivant n'est pas situe), puis MESURE.
// Chaque cause est cumulee en ms ; seuls les instants mesures (`measured_ms`) entrent dans la
// mediane et dans `beyond_ms`. La grille etant fermee, la somme des cinq cumuls depasse
// `duration_ms` d'au plus un pas.
//
// `median_m` NULL = « vie non mesuree » (moins de 2 000 ms mesurees). `radar_m` et `beyond_ms`
// NULL = la variante n'a pas de portee de radar connue : aucune part hors radar n'est calculee.
//
// UN SEUL INDEX, celui qui sert la vue `_latest` et la lecture par match : chaque index elargit
// la surface ART #23645 le jour ou quelqu'un ecrirait un DELETE.
const ddlMatchLifePlacement = `
	CREATE SEQUENCE IF NOT EXISTS match_life_placement_id_seq START 1;
	CREATE TABLE IF NOT EXISTS match_life_placement (
		id                   BIGINT PRIMARY KEY DEFAULT nextval('match_life_placement_id_seq'),
		match_id             VARCHAR   NOT NULL,
		-- decode_pass : identifiant d UNE passe. La vue _latest retient une passe ENTIERE.
		decode_pass          VARCHAR   NOT NULL,
		-- decoder_rev : PlacementRev du collecteur, distincte de celle des vies.
		decoder_rev          VARCHAR   NOT NULL,
		written_at           TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),

		-- (match_id, xuid, start_ms) : la cle de la vie dans match_lives.
		xuid                 VARCHAR   NOT NULL,
		start_ms             BIGINT    NOT NULL,
		end_ms               BIGINT    NOT NULL,
		duration_ms          BIGINT    NOT NULL,
		measured_ms          BIGINT    NOT NULL,
		-- median_m : NULL = vie non mesuree. Metres, deux decimales.
		median_m             DOUBLE,
		-- beyond_ms / radar_m : NULL = variante sans portee de radar connue.
		beyond_ms            BIGINT,
		radar_m              DOUBLE,
		carrier_ms           BIGINT    NOT NULL,
		team_down_ms         BIGINT    NOT NULL,
		unplaced_ms          BIGINT    NOT NULL,
		teammate_unplaced_ms BIGINT    NOT NULL,
		-- kills : frags publiables contre l autre camp rattaches a la vie.
		kills                INTEGER   NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_match_life_placement_lookup
		ON match_life_placement(match_id, xuid, start_ms, written_at);
`

const ddlMatchLifePlacementLatest = `
	CREATE OR REPLACE VIEW match_life_placement_latest AS
	SELECT p.*
	FROM match_life_placement AS p
	QUALIFY p.decode_pass = FIRST_VALUE(p.decode_pass) OVER (
		PARTITION BY p.match_id ORDER BY p.written_at DESC, p.id DESC
	);
`
