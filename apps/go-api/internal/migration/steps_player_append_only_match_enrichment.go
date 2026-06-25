package migration

// steps_player_append_only_match_enrichment.go — éradication ART de
// player_match_enrichment (player DB, la table la PLUS écrite) — 2026-06-21.
//
// **Pourquoi** : PK(match_id) + 3 index ART (idx_pme_session, idx_pme_engagement_history,
// idx_pme_engagement_paces) sur des colonnes MUTÉES par des UPDATE/ON CONFLICT
// incrémentaux (perf/engagement/session/friends/bot/exclusion/psa) = vecteur DuckDB
// #23046 (crash prod sur `engagement-coefs --with-scores`). Cf. .ai/PLAN_PME_ART_HARDENING.md.
//
// **Stratégie append-only + MERGE-ON-READ PAR GROUPE** : PK technique id BIGINT
// (séquence pme_seq) + written_at + colonne `stage` discriminant l'étape d'écriture.
// Chaque writer INSÈRE une row partielle taguée de son `stage` (perf/session/friends/
// bot/exclusion/psa/engagement/teammates/dominance). Les rows existantes deviennent le
// socle stage='legacy' (elles portent toutes les colonnes de l'état mergé courant).
//
// La vue player_match_enrichment_latest reconstitue 1 ligne logique par match_id :
//   1. dédup par (match_id, stage) → la DERNIÈRE row de chaque étape (written_at, id) ;
//   2. par colonne : SI l'étape propriétaire a une row → sa valeur (NULL inclus =
//      reset légitime, ex engagement_score insufficient_history) ; SINON le socle legacy.
// → préserve les écritures partielles ET les resets à NULL (le piège du last_value
// IGNORE NULLS par-colonne, écarté). Booléens bidirectionnels (is_with_friends,
// is_excluded) : COALESCE(..., FALSE) en sortie ; les writers écrivent la valeur
// EXPLICITE (jamais NULL).
//
// **Migration TRANSACTIONNELLE** (calquée RebuildPlayerMatchEnrichmentART /
// media_files_drop_filepath_unique_v1) : swap CTAS sous BeginTx + garde anti-perte
// rebuilt==before + recoverOrphan. PME n'est PAS re-dérivable (perf/sessions/exclusions)
// → un crash mid-swap non transactionnel = perte irrécupérable au boot. INTERDIT.
//
// **Idempotence** : columnExists('id') → CREATE OR REPLACE VIEW + no-op rebuild.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

func init() {
	Register(Migration{
		Name:        "player_append_only_match_enrichment_v1",
		TargetDB:    TargetPlayer,
		Description: "Rebuild player_match_enrichment en append-only (id PK + stage + vue merge-on-read par-groupe) — élimine PK/index ART mutés (#23046, table la plus écrite)",
		ApplySchema: applyAppendOnlyMatchEnrichment,
	})
}

// pmeColumnStage : colonne → étape propriétaire (le writer qui l'écrit). Une colonne =
// exactement une étape ; le socle 'legacy' est le fallback universel pour les rows
// migrées. known_teammates_count / friends_xuids sont MORTES (aucun writer Go) → legacy.
var pmeColumnStage = []struct{ col, stage string }{
	{"performance_score", "perf"},
	{"performance_chain", "perf"},
	{"dominance_flag", "dominance"},
	{"session_id", "session"},
	{"session_label", "session"},
	{"is_with_friends", "friends"},
	{"teammates_signature", "teammates"},
	{"had_bot_teammate", "bot"},
	{"is_excluded", "exclusion"},
	{"psa_checked_at", "psa"},
	{"engagement_score", "engagement"},
	{"engagement_score_brut", "engagement"},
	{"engagement_score_confidence", "engagement"},
	{"mode_category", "engagement"},
	{"engagement_pace_player", "engagement"},
	{"engagement_pace_team", "engagement"},
	{"engagement_pace_lobby", "engagement"},
	{"engagement_player_activity", "engagement"},
	// Phase 2 — readiness marker (snapshot). Étape propriétaire dédiée 'snapshot' :
	// les writers perf/psa/dominance/etc. n'écrasent JAMAIS snapshot_ready_at via le
	// merge-on-read par stage (CASE WHEN stage='snapshot'). Posé en fin de post-sync.
	{"snapshot_ready_at", "snapshot"},
	{"partial_reasons", "snapshot"},
}

// pmeBooleanFalseDefault : colonnes booléennes à transition bidirectionnelle, COALESCE FALSE.
var pmeBooleanFalseDefault = map[string]bool{"is_with_friends": true, "is_excluded": true}

// pmeBaselineFallback retourne l'expression SQL « valeur de baseline » d'une colonne :
// la row 'live' (collect après migration) gagne, sinon la row 'legacy' (socle pré-migration).
// Le collect live (player_persister.persistEnrichment) écrit UNE row multi-colonnes
// stage='live' ; les writers post-sync owner-stage l'overrident par colonne.
func pmeBaselineFallback(col string) string {
	return fmt.Sprintf(
		"COALESCE(MAX(CASE WHEN stage='live' THEN %[1]s END), MAX(CASE WHEN stage='legacy' THEN %[1]s END))",
		col)
}

// buildPMELatestViewSQL génère la vue merge-on-read par-groupe.
func buildPMELatestViewSQL() string {
	var sb strings.Builder
	sb.WriteString("CREATE OR REPLACE VIEW player_match_enrichment_latest AS\n")
	sb.WriteString("WITH ls AS (\n")
	sb.WriteString("  SELECT * FROM player_match_enrichment\n")
	sb.WriteString("  QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, stage ORDER BY written_at DESC, id DESC) = 1\n")
	sb.WriteString(")\nSELECT\n  match_id")
	for _, cs := range pmeColumnStage {
		// SI l'étape propriétaire a une row → sa valeur (NULL préservé = reset
		// légitime) ; SINON baseline (live > legacy).
		expr := fmt.Sprintf(
			"CASE WHEN MAX(CASE WHEN stage='%[2]s' THEN 1 ELSE 0 END)=1 "+
				"THEN MAX(CASE WHEN stage='%[2]s' THEN %[1]s END) "+
				"ELSE %[3]s END",
			cs.col, cs.stage, pmeBaselineFallback(cs.col))
		if pmeBooleanFalseDefault[cs.col] {
			expr = "COALESCE(" + expr + ", FALSE)"
		}
		sb.WriteString(",\n  " + expr + " AS " + cs.col)
	}
	// known_teammates_count / friends_xuids : baseline uniquement (colonnes mortes,
	// aucun writer owner-stage).
	sb.WriteString(",\n  " + pmeBaselineFallback("known_teammates_count") + " AS known_teammates_count")
	sb.WriteString(",\n  " + pmeBaselineFallback("friends_xuids") + " AS friends_xuids")
	sb.WriteString(",\n  MIN(created_at) AS created_at")
	sb.WriteString(",\n  MAX(updated_at) AS updated_at")
	sb.WriteString("\nFROM ls\nGROUP BY match_id")
	return sb.String()
}

// EnsurePlayerMatchEnrichmentAppendOnly convertit player_match_enrichment en
// append-only (id PK + stage + written_at) et (re)crée la vue _latest + l'index
// lookup. Idempotent. Exposé pour les fixtures de test qui construisent une player
// DB à la main (sans RunForDB) : un seul appel après la DDL legacy suffit —
// ensurePMEColumns ajoute toutes les colonnes manquantes référencées par la vue.
func EnsurePlayerMatchEnrichmentAppendOnly(db *sql.DB) error {
	return applyAppendOnlyMatchEnrichment(db)
}

func applyAppendOnlyMatchEnrichment(db *sql.DB) error {
	ctx := bootCtx()

	if err := recoverOrphanMatchEnrichment(ctx, db); err != nil {
		return err
	}

	hasTable, err := tableExists(db, "player_match_enrichment")
	if err != nil {
		return fmt.Errorf("append-only pme: check table: %w", err)
	}
	if !hasTable {
		return nil
	}

	// Garantir que toutes les colonnes référencées par la vue existent (les colonnes
	// engagement/dominance/psa sont ADD COLUMN par d'autres migrations ; on ne dépend
	// pas de l'ordre).
	if err := ensurePMEColumns(db); err != nil {
		return err
	}

	hasID, err := columnExists(db, "player_match_enrichment", "id")
	if err != nil {
		return fmt.Errorf("append-only pme: check id column: %w", err)
	}
	if hasID {
		// Déjà append-only : (ré)assurer la vue + l'index lookup (idempotent).
		if _, err := db.ExecContext(ctx, buildPMELatestViewSQL()); err != nil {
			return fmt.Errorf("append-only pme: refresh view: %w", err)
		}
		return nil
	}

	if err := swapMatchEnrichmentAppendOnlyTx(ctx, db); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`CREATE INDEX IF NOT EXISTS idx_pme_match_lookup ON player_match_enrichment(match_id, written_at)`); err != nil {
		return fmt.Errorf("append-only pme: create idx_pme_match_lookup: %w", err)
	}
	if _, err := db.ExecContext(ctx, buildPMELatestViewSQL()); err != nil {
		return fmt.Errorf("append-only pme: create view: %w", err)
	}
	return nil
}

// ensurePMEColumns ajoute (si manquantes) les colonnes additives référencées par la vue,
// pour découpler la création de la vue de l'ordre des migrations engagement/dominance/psa.
func ensurePMEColumns(db *sql.DB) error {
	// TOUTES les colonnes référencées par buildPMELatestViewSQL doivent exister avant
	// de créer la vue — y compris les colonnes « de base » (session_label, etc.) qu'une
	// table legacy MINIMALE (certaines fixtures de test, ou un vieux schéma partiel)
	// pourrait ne pas avoir. addColumnIfMissing est un no-op si la colonne existe déjà
	// (et ne change pas le type — ex. session_id INTEGER d'une fixture est préservé).
	const boolDefaultFalse = colBoolean + " DEFAULT FALSE"
	cols := []struct{ name, typ string }{
		{"performance_score", colFloat},
		{"performance_chain", colVarchar},
		{"dominance_flag", colTinyint},
		{"session_id", colVarchar},
		{"session_label", colVarchar},
		{"is_with_friends", boolDefaultFalse},
		{"teammates_signature", colVarchar},
		{"had_bot_teammate", colBoolean},
		{"is_excluded", boolDefaultFalse},
		{"psa_checked_at", colTimestamp},
		{"engagement_score", colDouble},
		{"engagement_score_brut", colDouble},
		{"engagement_score_confidence", colVarchar},
		{"mode_category", colVarchar},
		{"engagement_pace_player", colDouble},
		{"engagement_pace_team", colDouble},
		{"engagement_pace_lobby", colDouble},
		{"engagement_player_activity", colInteger},
		{"known_teammates_count", colSmallInt},
		{"friends_xuids", colVarchar},
		{"snapshot_ready_at", colTimestamp},
		{"partial_reasons", colVarchar},
		{"created_at", colTimestamp},
		{"updated_at", colTimestamp},
	}
	for _, c := range cols {
		if err := addColumnIfMissing(db, "player_match_enrichment", c.name, c.typ); err != nil {
			return fmt.Errorf("append-only pme: ensure column %s: %w", c.name, err)
		}
	}
	return nil
}

// recoverOrphanMatchEnrichment répare un crash mid-swap antérieur (table absente +
// __appendonly présent). No-op sinon.
func recoverOrphanMatchEnrichment(ctx context.Context, db *sql.DB) error {
	hasMain, err := tableExists(db, "player_match_enrichment")
	if err != nil {
		return fmt.Errorf("append-only pme: check main: %w", err)
	}
	if hasMain {
		return nil
	}
	hasRebuild, err := tableExists(db, "player_match_enrichment__appendonly")
	if err != nil {
		return fmt.Errorf("append-only pme: check __appendonly: %w", err)
	}
	if !hasRebuild {
		return nil
	}
	slog.WarnContext(ctx, "append-only pme: __appendonly orphelin (crash mid-swap) — récupération",
		"action", "RENAME player_match_enrichment__appendonly -> player_match_enrichment")
	if _, err := db.ExecContext(ctx,
		`ALTER TABLE player_match_enrichment__appendonly RENAME TO player_match_enrichment`); err != nil {
		return fmt.Errorf("append-only pme: recover orphan: %w", err)
	}
	return nil
}

// swapMatchEnrichmentAppendOnlyTx : swap CTAS transactionnel (id seq + written_at +
// stage='legacy' pour les rows existantes). Garde anti-perte rebuilt==before. Rollback
// intégral sur erreur/crash → table intacte.
func swapMatchEnrichmentAppendOnlyTx(ctx context.Context, db *sql.DB) error {
	cols, err := loadTableColumns(ctx, db, "player_match_enrichment")
	if err != nil {
		return fmt.Errorf("append-only pme: enumerate columns: %w", err)
	}
	if len(cols) == 0 {
		return nil
	}
	colList := strings.Join(cols, ", ")

	var before int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM player_match_enrichment`).Scan(&before); err != nil {
		return fmt.Errorf("append-only pme: count before: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("append-only pme: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `CREATE SEQUENCE IF NOT EXISTS pme_seq START 1`); err != nil {
		return fmt.Errorf("append-only pme: create sequence: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS player_match_enrichment__appendonly`); err != nil {
		return fmt.Errorf("append-only pme: drop stale __appendonly: %w", err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE player_match_enrichment__appendonly AS
		SELECT nextval('pme_seq') AS id, %s, CURRENT_TIMESTAMP AS written_at, 'legacy' AS stage
		FROM player_match_enrichment`, colList)); err != nil {
		return fmt.Errorf("append-only pme: create __appendonly: %w", err)
	}
	var rebuilt int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM player_match_enrichment__appendonly`).Scan(&rebuilt); err != nil {
		return fmt.Errorf("append-only pme: count __appendonly: %w", err)
	}
	if rebuilt != before {
		return fmt.Errorf("append-only pme: swap abandonné, rebuilt=%d != before=%d (rollback, zéro perte)", rebuilt, before)
	}
	for _, stmt := range []string{
		`DROP TABLE player_match_enrichment`,
		`ALTER TABLE player_match_enrichment__appendonly RENAME TO player_match_enrichment`,
		`ALTER TABLE player_match_enrichment ADD PRIMARY KEY (id)`,
		`ALTER TABLE player_match_enrichment ALTER COLUMN id SET DEFAULT nextval('pme_seq')`,
		`ALTER TABLE player_match_enrichment ALTER COLUMN written_at SET DEFAULT now()`,
		`ALTER TABLE player_match_enrichment ALTER COLUMN stage SET DEFAULT 'legacy'`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("append-only pme: swap step (%s): %w", firstWords(stmt, 3), err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("append-only pme: commit swap: %w", err)
	}
	committed = true

	slog.InfoContext(ctx, "append-only player_match_enrichment: migration appliquée (ART éradiqué)",
		"rows", before, "columns_preserved", len(cols))
	return nil
}
