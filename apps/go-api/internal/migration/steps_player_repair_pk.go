package migration

// steps_player_repair_pk.go — migrations correctives boot-time qui posent la
// PRIMARY KEY manquante sur les player DB legacy.
//
// Contexte (2026-06-04) : `create_base_player_schema` et `fix_match_citations_schema`
// utilisent CREATE TABLE IF NOT EXISTS (...PRIMARY KEY...). Sur une player DB
// créée par l'ancien pipeline Python (ou tout clone antérieur à l'ajout de la PK),
// la table préexiste → IF NOT EXISTS saute la recréation → la PK n'est JAMAIS
// appliquée. Les writes PK-dépendants échouent alors en Binder Error :
//   - ensurePlayerEnrichmentRows : `INSERT OR IGNORE INTO player_match_enrichment`
//   - writeCitations : `INSERT ... ON CONFLICT (match_id, citation_name_norm)`
// Symptôme observé : Chocoboflor (player_match_enrichment + match_citations +
// match_skill_rank tous SANS contrainte ; les 3 autres joueurs OK). Conséquence :
// perfs/citations/sessions/weapon orphelins jamais enrichis.
//
// Fix : si la PK manque, reconstruire la table en la préservant (colonnes
// dynamiques + dédup défensif) puis ADD PRIMARY KEY. Idempotent (no-op si la PK
// est déjà présente → DB saines non touchées). Calqué sur le précédent prouvé
// `repair_engagement_coefficients_primary_key` (steps_engagement.go).
//
// NB : match_skill_rank n'est PAS réparée ici — elle est append-only (PK
// technique gérée par player_append_only_match_skill_rank_v1, idempotent via
// columnExists('id')) ; appliquer une PK (match_id) y rouvrirait le bug ART.

import (
	"database/sql"
	"fmt"
	"strings"
)

func init() {
	Register(Migration{
		Name:        "repair_player_match_enrichment_primary_key",
		TargetDB:    TargetPlayer,
		Description: "Reconstruit player_match_enrichment avec PRIMARY KEY (match_id) quand elle manque (player DB legacy CREATE TABLE IF NOT EXISTS)",
		ApplySchema: repairPlayerMatchEnrichmentPK,
	})

	Register(Migration{
		Name:        "repair_match_citations_primary_key",
		TargetDB:    TargetPlayer,
		Description: "Reconstruit match_citations avec PRIMARY KEY (match_id, citation_name_norm) quand elle manque (player DB legacy)",
		ApplySchema: repairMatchCitationsPK,
	})
}

// repairPlayerMatchEnrichmentPK pose la PK (match_id) si absente, en réutilisant
// le rebuild CTAS dynamique existant (préserve les colonnes additives).
func repairPlayerMatchEnrichmentPK(db *sql.DB) error {
	exists, err := tableExists(db, "player_match_enrichment")
	if err != nil {
		return fmt.Errorf("repair pme PK: check table: %w", err)
	}
	if !exists {
		return nil
	}
	hasPK, err := hasPrimaryKey(db, "player_match_enrichment")
	if err != nil {
		return fmt.Errorf("repair pme PK: check PK: %w", err)
	}
	if hasPK {
		return nil
	}
	return RebuildPlayerMatchEnrichmentART(bootCtx(), db)
}

// repairMatchCitationsPK pose la PK composite (match_id, citation_name_norm) si
// absente : CTAS dynamique (préserve toutes les colonnes, y compris un éventuel
// `citation` legacy), dédup par la clé en gardant la ligne la plus récente, et
// écarte les lignes à clé NULL (anciennes lignes du schéma pré-migration, non
// lisibles par l'app et re-dérivables via le pipeline citations).
func repairMatchCitationsPK(db *sql.DB) error {
	exists, err := tableExists(db, "match_citations")
	if err != nil {
		return fmt.Errorf("repair citations PK: check table: %w", err)
	}
	if !exists {
		return nil
	}
	hasPK, err := hasPrimaryKey(db, "match_citations")
	if err != nil {
		return fmt.Errorf("repair citations PK: check PK: %w", err)
	}
	if hasPK {
		return nil
	}
	// Garantit les colonnes-clés au cas où la legacy a l'ancien schéma citation/varchar.
	if err := addColumnIfMissing(db, "match_citations", "citation_name_norm", colVarchar); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "match_citations", "value", colInteger+" DEFAULT 1"); err != nil {
		return err
	}
	cols, err := loadTableColumns(bootCtx(), db, "match_citations")
	if err != nil {
		return fmt.Errorf("repair citations PK: columns: %w", err)
	}
	if len(cols) == 0 {
		return nil
	}
	orderBy := "created_at DESC NULLS LAST"
	if ok, _ := columnExists(db, "match_citations", "created_at"); !ok {
		orderBy = "match_id"
	}
	script := fmt.Sprintf(`
		CREATE TABLE match_citations__pkfix AS
			SELECT %s FROM (
				SELECT *, ROW_NUMBER() OVER (
					PARTITION BY match_id, citation_name_norm ORDER BY %s
				) AS __rn
				FROM match_citations
				WHERE match_id IS NOT NULL AND citation_name_norm IS NOT NULL
			) WHERE __rn = 1;
		DROP TABLE match_citations;
		ALTER TABLE match_citations__pkfix RENAME TO match_citations;
		ALTER TABLE match_citations ADD PRIMARY KEY (match_id, citation_name_norm);
	`, strings.Join(cols, ", "), orderBy)
	return execScript(db, script)
}
