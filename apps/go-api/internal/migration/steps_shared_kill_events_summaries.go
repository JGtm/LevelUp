package migration

// steps_shared_kill_events_summaries.go — LES RÉSUMÉS PAR MATCH DU JOURNAL DES MORTS (ADR 0039) :
// `match_kill_heads` (S0), `match_kill_pairs` (S1), `match_assist_pairs` (S2).
//
// Chaque passe de `match_kill_events` écrit, dans SA transaction et avec SON horodatage, une
// génération de résumés (`persist.insertKillSummaryRows`, fonction pure
// `analysis/killsummary`). Les colonnes communes :
//
//	id               PK technique, séquence propre (append-only, ADR 0026)
//	summary_pass     la génération : une par écriture de résumés, même pour une passe relue
//	summary_rev      la révision de la règle (killsummary.Rev)
//	src_decode_pass  la passe du journal résumée
//	written_at       l'horodatage de la passe résumée
//
// ─── LA TÊTE DÉCIDE LA GÉNÉRATION ───────────────────────────────────────────────────────────
//
// `match_kill_heads_latest` retient, par match, la tête de plus grand (written_at, id) ;
// `match_kill_pairs_latest` et `match_assist_pairs_latest` se joignent à elle sur (match_id,
// summary_pass) et exposent son `publishable` — modèle `match_usage_films` /
// `match_usage_players`. Une relecture de la même passe à une révision nouvelle écrit une
// génération entière de plus : la jointure ne sert que la dernière, jamais la somme des deux.
//
// Un `SELECT *` de vue FIGE ses colonnes à la création : tout `ADD COLUMN` sur ces tables doit
// recréer les vues dans le même step (patron de `steps_shared_usage_summary.go`).
//
// ─── TITRE AGNOSTIQUE ───────────────────────────────────────────────────────────────────────
//
// Migration générique : tous les titres portent le journal des morts, donc les trois tables
// (vides pour un titre sans écrivain). [EnsureMatchKillEvents] les pose aussi, avec le journal :
// une base ou une fixture qui reçoit des passes sans être passée par les migrations doit pouvoir
// recevoir leurs résumés dans la même transaction.

import "database/sql"

func init() {
	Register(Migration{
		Name:     "shared_match_kill_summaries_v1",
		TargetDB: TargetShared,
		Description: "Tables append-only match_kill_heads (S0), match_kill_pairs (S1), " +
			"match_assist_pairs (S2) : résumés par match du journal des morts écrits avec la passe, " +
			"vues _latest décidées par la tête (ADR 0039)",
		ApplySchema: ensureMatchKillSummaries,
	})
}

// ensureMatchKillSummaries crée les trois tables, leurs index, séquences et vues. Idempotente ;
// les tables étant nettes-neuves, le piège « CREATE TABLE IF NOT EXISTS n'ajoute pas de PK »
// ne s'applique pas.
func ensureMatchKillSummaries(db *sql.DB) error {
	for _, script := range []string{
		ddlMatchKillHeads, ddlMatchKillPairs, ddlMatchAssistPairs,
		// La tête AVANT ses filles : leurs vues se définissent sur la sienne.
		ddlMatchKillHeadsLatest, ddlMatchKillPairsLatest, ddlMatchAssistPairsLatest,
	} {
		if err := execScript(db, script); err != nil {
			return err
		}
	}
	return nil
}

// ddlMatchKillHeads : S0, une ligne par match et par génération. UN SEUL INDEX, celui des
// autres tables de passe : (match_id, written_at).
const ddlMatchKillHeads = `
	CREATE SEQUENCE IF NOT EXISTS match_kill_heads_id_seq START 1;
	CREATE TABLE IF NOT EXISTS match_kill_heads (
		id               BIGINT    PRIMARY KEY DEFAULT nextval('match_kill_heads_id_seq'),
		match_id         VARCHAR   NOT NULL,
		summary_pass     VARCHAR   NOT NULL,
		summary_rev      VARCHAR   NOT NULL,
		src_decode_pass  VARCHAR   NOT NULL,
		written_at       TIMESTAMP NOT NULL,
		-- publishable : attribut de la passe (KillSourceBatch.Publishable), porté par la tête.
		publishable      BOOLEAN   NOT NULL,
		-- n_rows : morts de la passe. n_assist_known : morts à assistant connu (nommé ou absent
		-- mesuré). n_killer_xuid : tueur crédité avec xuid. n_src_killer : source mesurée ET
		-- tueur avec xuid.
		n_rows           INTEGER   NOT NULL,
		n_assist_known   INTEGER   NOT NULL,
		n_killer_xuid    INTEGER   NOT NULL,
		n_src_killer     INTEGER   NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_match_kill_heads_match
		ON match_kill_heads(match_id, written_at);
`

// ddlMatchKillPairs : S1, grain (match, tueur xuid, victime xuid, source). NULL = groupe gardé :
// tueur absent, victime bot, source non mesurée. Aucun gamertag : aucun lecteur agrégat n'en lit.
const ddlMatchKillPairs = `
	CREATE SEQUENCE IF NOT EXISTS match_kill_pairs_id_seq START 1;
	CREATE TABLE IF NOT EXISTS match_kill_pairs (
		id               BIGINT    PRIMARY KEY DEFAULT nextval('match_kill_pairs_id_seq'),
		match_id         VARCHAR   NOT NULL,
		summary_pass     VARCHAR   NOT NULL,
		summary_rev      VARCHAR   NOT NULL,
		src_decode_pass  VARCHAR   NOT NULL,
		written_at       TIMESTAMP NOT NULL,
		feed_killer_xuid VARCHAR,
		victim_xuid      VARCHAR,
		source_tag       UINTEGER,
		kills            INTEGER   NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_match_kill_pairs_match
		ON match_kill_pairs(match_id, written_at);
`

// ddlMatchAssistPairs : S2, seulement les morts d'une passe publiable à assistant connu. Grain
// (match, assistant xuid + gamertag, tueur xuid + gamertag, victime xuid quand assistant ET
// tueur sont sans xuid). Une moyenne de part se lit sum_assist_pct / n_assist_pct, NULL quand
// n_assist_pct vaut 0. Tranches : n_low < 25, n_mid 25..50, n_high > 50 (bornes du paquet
// domain, liées à summary_rev).
const ddlMatchAssistPairs = `
	CREATE SEQUENCE IF NOT EXISTS match_assist_pairs_id_seq START 1;
	CREATE TABLE IF NOT EXISTS match_assist_pairs (
		id                   BIGINT    PRIMARY KEY DEFAULT nextval('match_assist_pairs_id_seq'),
		match_id             VARCHAR   NOT NULL,
		summary_pass         VARCHAR   NOT NULL,
		summary_rev          VARCHAR   NOT NULL,
		src_decode_pass      VARCHAR   NOT NULL,
		written_at           TIMESTAMP NOT NULL,
		assist_xuid          VARCHAR,
		assist_gamertag      VARCHAR,
		feed_killer_xuid     VARCHAR,
		feed_killer_gamertag VARCHAR,
		victim_xuid          VARCHAR,
		n                    INTEGER   NOT NULL,
		n_stolen             INTEGER   NOT NULL,
		sum_assist_pct       INTEGER   NOT NULL,
		n_assist_pct         INTEGER   NOT NULL,
		n_low                INTEGER   NOT NULL,
		n_mid                INTEGER   NOT NULL,
		n_high               INTEGER   NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_match_assist_pairs_match
		ON match_assist_pairs(match_id, written_at);
`

// ddlMatchKillHeadsLatest : la dernière génération de tête par match. Règle « dernière passe »
// du registre de compaction (dernierePasse, colonne summary_pass) : le SQL doit garder cette forme.
const ddlMatchKillHeadsLatest = `
	CREATE OR REPLACE VIEW match_kill_heads_latest AS
	SELECT h.*
	FROM match_kill_heads AS h
	QUALIFY h.summary_pass = FIRST_VALUE(h.summary_pass) OVER (
		PARTITION BY h.match_id ORDER BY h.written_at DESC, h.id DESC
	);
`

// ddlMatchKillPairsLatest / ddlMatchAssistPairsLatest : les lignes de la génération que retient
// la tête, avec son publishable. Règle passeDeLaVueParente du registre de compaction.
const ddlMatchKillPairsLatest = `
	CREATE OR REPLACE VIEW match_kill_pairs_latest AS
	SELECT p.*, h.publishable
	FROM match_kill_pairs AS p
	JOIN match_kill_heads_latest AS h
	  ON h.match_id = p.match_id AND h.summary_pass = p.summary_pass;
`

const ddlMatchAssistPairsLatest = `
	CREATE OR REPLACE VIEW match_assist_pairs_latest AS
	SELECT a.*, h.publishable
	FROM match_assist_pairs AS a
	JOIN match_kill_heads_latest AS h
	  ON h.match_id = a.match_id AND h.summary_pass = a.summary_pass;
`
