package migration

// steps_player_schema_authority.go — LA CHAÎNE DE MIGRATIONS EST L'AUTORITÉ DE SCHÉMA
// UNIQUE DES PLAYER DB (décisions utilisateur du 2026-08-05).
//
// CONTEXTE. Deux autorités de schéma concurrentes cohabitaient sur les player DB :
//   1. la chaîne de migrations (RunForDB(TargetPlayer)) ;
//   2. sync.EnsurePlayerSchema (internal/sync/schema.go), rejoué à CHAQUE OpenPlayerDB
//      en ADD COLUMN / CREATE ... IF NOT EXISTS.
// Le soin SILENCIEUX de (2) a masqué la dérive de (1) pendant des mois : le bug prod
// career_progression du 2026-08-05 (« Table career_progression does not have a column
// with name last_fetch_status ») n'était visible que sur une DB FRAÎCHE, car les DB de
// prod étaient re-soignées par Ensure à chaque boot.
//
// CE FICHIER referme les deux trous constatés par l'inventaire (schéma d'une DB
// migrations-seul vs schéma après EnsurePlayerSchema) :
//   - personal_score_awards et player_csr_snapshots (+ séquences, index, vue _latest)
//     n'existaient QUE par Ensure : aucun step ne les créait. Elles reçoivent ici leurs
//     steps de CREATE idempotents ;
//   - idx_career_xuid n'était posé QUE par Ensure : il est SUPPRIMÉ partout (décision A).
//
// COMPLÉMENT 2026-08-05 (même arbitrage, appliqué à l'index jumeau) : idx_psa_xuid
// (personal_score_awards) suit idx_career_xuid — retiré des DEUX autorités qui le
// posaient (le DDL ci-dessous et le PostSwap de steps_player_append_only_personal_score_awards.go)
// et retiré des DB existantes par le step drop_psa_xuid_art_index_v1.
// Idem idx_psa_match_xuid (troisième du lot, même jour) : posé par le SEUL PostSwap
// legacy (jamais par le DDL fraîche → divergence latente entre autorités, invisible
// de l'invariant no-op) et pur préfixe d'idx_psa_gen. Retiré du PostSwap + step
// drop_psa_match_xuid_art_index_v1.
//
// Ensure GARDE ses créations en SOIN DE TRANSITION (les player DB legacy en dépendent),
// mais n'est plus une autorité : son action réelle est désormais BRUYANTE
// (schema_drift_healed, internal/sync/schemadrift) et l'invariant « Ensure est un no-op
// sur une DB fraîchement migrée » est verrouillé par
// internal/sync/schema_authority_test.go.
//
// SOURCE UNIQUE DU DDL. PlayerPersonalScoreAwardsDDL et PlayerCSRSnapshotsDDL sont
// CONSOMMÉS par les 3 écrivains du schéma player (migrations ici, sync.playerSchemaSQL,
// ops/seed_demo_synthetic_player) — avant, la même DDL vivait recopiée en 3 exemplaires
// divergents (règle « ≤ 2 copies »). Le package migration est le seul importable par les
// trois (sync et ops importent migration ; l'inverse est un cycle).

import "database/sql"

// PlayerPersonalScoreAwardsDDL — schéma canonique de personal_score_awards (player DB).
//
// APPEND-ONLY (campagne ART #23046, Phase 2, ADR 0026) : plus de DELETE+INSERT sur les
// index. Chaque écriture INSÈRE pur avec UN generation_id (séquence psa_generation_seq,
// partagé par le batch) + written_at + is_tombstone. Lecture via la vue
// personal_score_awards_latest (DENSE_RANK, génération MAX, tombstones exclus) — créée
// par applyAppendOnlyPersonalScoreAwards (steps_player_append_only_personal_score_awards.go).
const PlayerPersonalScoreAwardsDDL = `
CREATE SEQUENCE IF NOT EXISTS personal_score_awards_id_seq;
CREATE SEQUENCE IF NOT EXISTS psa_generation_seq START 1;
CREATE TABLE IF NOT EXISTS personal_score_awards (
    id         INTEGER   PRIMARY KEY DEFAULT nextval('personal_score_awards_id_seq'),
    match_id   VARCHAR   NOT NULL,
    xuid       VARCHAR   NOT NULL,
    award_name VARCHAR   NOT NULL,
    award_category VARCHAR,
    award_count INTEGER  DEFAULT 1,
    award_score INTEGER  DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    generation_id BIGINT NOT NULL DEFAULT 0,
    written_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    is_tombstone BOOLEAN DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_psa_match    ON personal_score_awards(match_id);
-- idx_psa_xuid : SUPPRIMÉ (décision 2026-08-05, miroir exact d'idx_career_xuid). Dans une
-- player DB, xuid est QUASI CONSTANT (une DB = un joueur) → sélectivité nulle, l'index
-- n'accélère aucun filtre. Sa surface ART #23046 est déjà éteinte par construction
-- (personal_score_awards est append-only INSERT-only, ADR 0026), mais un index sans
-- lecteur ni gain est du coût d'écriture pur + une surface à rallumer au premier DELETE
-- de régression. La convergence des DB EXISTANTES est assurée par le step
-- drop_psa_xuid_art_index_v1 (ci-dessous).
CREATE INDEX IF NOT EXISTS idx_psa_category ON personal_score_awards(award_category);
CREATE INDEX IF NOT EXISTS idx_psa_gen      ON personal_score_awards(match_id, xuid, generation_id);
`

// PlayerCSRSnapshotsDDL — schéma canonique de player_csr_snapshots (player DB).
//
// APPEND-ONLY (Phase 2.G du refactor ART, ADR 0026) : PK technique id + written_at,
// N versions par (playlist_id, season_id), lecture via player_csr_snapshots_latest
// UNIQUEMENT (une lecture brute sert des lignes périmées).
const PlayerCSRSnapshotsDDL = `
CREATE SEQUENCE IF NOT EXISTS pcs_seq START 1;
CREATE TABLE IF NOT EXISTS player_csr_snapshots (
    id                               BIGINT DEFAULT nextval('pcs_seq') PRIMARY KEY,
    playlist_id                      VARCHAR NOT NULL,
    playlist_name                    VARCHAR,
    queue                            VARCHAR,
    input                            VARCHAR,
    season_id                        VARCHAR NOT NULL,
    current_value                    FLOAT,
    current_tier                     VARCHAR,
    current_sub_tier                 SMALLINT,
    current_measurement_remaining    INTEGER,
    season_value                     FLOAT,
    season_tier                      VARCHAR,
    season_sub_tier                  SMALLINT,
    alltime_value                    FLOAT,
    alltime_tier                     VARCHAR,
    alltime_sub_tier                 SMALLINT,
    fetched_at                       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    written_at                       TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
);
CREATE INDEX IF NOT EXISTS idx_pcs_lookup ON player_csr_snapshots(playlist_id, season_id, written_at);
CREATE OR REPLACE VIEW player_csr_snapshots_latest AS
    SELECT * FROM player_csr_snapshots
    QUALIFY ROW_NUMBER() OVER (PARTITION BY playlist_id, season_id ORDER BY written_at DESC, id DESC) = 1;
`

// L'ordre de Register() dans cet init() est CONTRAINT : il doit reproduire l'ordre de ces
// 3 steps dans canonicalOrder (order.go) — cf. TestSortByCanonicalIsNoOpOnCurrentRegistry.
// Ils y occupent les positions qui suivent immédiatement repair_match_citations_primary_key
// (fin du bloc player).
func init() {
	Register(Migration{
		Name:     "create_personal_score_awards_player_v1",
		TargetDB: TargetPlayer,
		Description: "Crée personal_score_awards (append-only generation_id + vue _latest) — " +
			"la table n'existait que par sync.EnsurePlayerSchema",
		ApplySchema: applyCreatePersonalScoreAwards,
	})
	Register(Migration{
		Name:     "create_player_csr_snapshots_player_v1",
		TargetDB: TargetPlayer,
		Description: "Crée player_csr_snapshots (append-only written_at + vue _latest) — " +
			"la table n'existait que par sync.EnsurePlayerSchema",
		ApplySchema: applyCreatePlayerCSRSnapshots,
	})
	Register(Migration{
		Name:     "drop_career_xuid_art_index_v1",
		TargetDB: TargetPlayer,
		Description: "Retire idx_career_xuid (xuid quasi constant dans une player DB : " +
			"sélectivité nulle, surface ART #23046 pure perte)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `DROP INDEX IF EXISTS idx_career_xuid;`)
		},
	})
	Register(Migration{
		Name:     "drop_psa_xuid_art_index_v1",
		TargetDB: TargetPlayer,
		Description: "Retire idx_psa_xuid (xuid quasi constant dans une player DB : " +
			"sélectivité nulle, coût d'écriture pur — miroir d'idx_career_xuid)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `DROP INDEX IF EXISTS idx_psa_xuid;`)
		},
	})
	Register(Migration{
		Name:     "drop_psa_match_xuid_art_index_v1",
		TargetDB: TargetPlayer,
		Description: "Retire idx_psa_match_xuid (préfixe redondant d'idx_psa_gen ; n'existait " +
			"que sur les DB converties par le PostSwap legacy — divergence fraîche/convertie)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `DROP INDEX IF EXISTS idx_psa_match_xuid;`)
		},
	})
}

// applyCreatePersonalScoreAwards : réparer — créer — garantir la vue.
//
//  1. conversion append-only idempotente AVANT le DDL. L'ORDRE EST CRITIQUE : le DDL
//     indexe generation_id (idx_psa_gen) ; sur une table legacy résiduelle qui ne porte
//     pas encore cette colonne, CREATE TABLE IF NOT EXISTS no-operait et le CREATE INDEX
//     casserait le boot. No-op sur DB vierge (table absente) et sur table déjà convertie.
//  2. le DDL crée la table DIRECTEMENT en forme append-only si elle est absente.
//  3. second passage de la conversion : sur DB vierge, l'étape 1 n'avait rien à faire —
//     c'est ce passage qui pose la vue personal_score_awards_latest sur la table qui vient
//     de naître. Idempotent (marqueur generation_id présent → simple refresh de la vue).
//
// La vue N'EST PAS dans PlayerPersonalScoreAwardsDDL à dessein : ce DDL est aussi rejoué
// par sync.EnsurePlayerSchema à chaque OpenPlayerDB, où le bind de la vue échouerait sur
// une player DB legacy non encore convertie.
func applyCreatePersonalScoreAwards(db *sql.DB) error {
	if err := applyAppendOnlyPersonalScoreAwards(db); err != nil {
		return err
	}
	if err := execScript(db, PlayerPersonalScoreAwardsDDL); err != nil {
		return err
	}
	return applyAppendOnlyPersonalScoreAwards(db)
}

// applyCreatePlayerCSRSnapshots répare D'ABORD un éventuel schéma legacy pré-2026-05-24
// (PK(playlist_id, season_id) SANS id/written_at) puis émet le schéma canonique.
//
// L'ORDRE EST CRITIQUE : l'index idx_pcs_lookup et la vue _latest référencent written_at
// et id ; sur un schéma legacy leur BIND échouerait et la migration casserait le boot.
// EnsurePlayerCSRSnapshotsAppendOnly est idempotent et no-ope sur DB vierge (table
// absente) comme sur DB déjà convertie (marqueur id présent).
func applyCreatePlayerCSRSnapshots(db *sql.DB) error {
	if err := EnsurePlayerCSRSnapshotsAppendOnly(db); err != nil {
		return err
	}
	return execScript(db, PlayerCSRSnapshotsDDL)
}
