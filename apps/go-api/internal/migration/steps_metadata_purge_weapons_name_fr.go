package migration

// steps_metadata_purge_weapons_name_fr.go — purge de la colonne inerte
// weapons.name_fr (item V721-05.1).
//
// Contexte (V72-06, commit cebe2fed9) : weapon_name_labels (keyée par weapon_key,
// alimentée par config/titles/{slug}/mappings/weapon_names.toml) est devenue la
// SOURCE UNIQUE du nom d'affichage d'une arme, pour Halo Infinite ET Halo 5. La
// colonne weapons.name_fr — un repli « inventé » jamais fiable — a été retirée du
// CREATE TABLE (weapon_registry.go, applyWeaponRegistry) à cette occasion : toute
// metadata.duckdb CRÉÉE APRÈS V72-06 n'a plus cette colonne. Mais une metadata.duckdb
// DÉJÀ MIGRÉE (prod, dev) la conserve : DuckDB refuse `ALTER TABLE ... DROP COLUMN`
// tant qu'un index existe sur la table (ici la PK title_slug+weapon_key) ; c'est ce
// renoncement volontaire, documenté dans weapon_registry.go, que cette migration lève.
//
// Aucun lecteur ne subsiste sur weapons.name_fr (vérifié par grep sur tout le module
// avant d'écrire cette migration) : la colonne est purement inerte, purge cosmétique
// sans impact fonctionnel. Ne PAS confondre avec weapon_labels.name_fr (AUTRE table,
// VIVANTE — weapon_resolver.go, 3e maillon du COALESCE pour les weapon_id hors
// registre) : hors périmètre, non touchée ici.
//
// Pattern : rebuild CTAS-swap transactionnel, calqué sur
// rebuild_match_participants_defeat_art_corruption (steps_shared_rebuild_match_participants.go)
// et rebuild_catalog_fetch_queue_drop_art_indexes
// (steps_metadata_rebuild_catalog_fetch_queue_no_art.go) — DuckDB ne sait pas DROP une
// colonne indexée via ALTER, donc on recrée la table sans elle. `weapons` est un
// référentiel STATIQUE explicitement HORS périmètre du bug ART #23046 (zéro writer
// concurrent, zéro écriture per-match — cf. tête de weapon_registry.go) : ce n'est PAS
// une table append-only, donc PAS de vue `_latest` ni de `written_at` ici — juste la PK
// composite recréée à l'identique.
//
// Idempotente : columnExists(name_fr) fait office de garde — no-op si la colonne est
// déjà absente (DB neuve post-V72-06, ou 2e exécution de cette migration). Sans perte :
// garde anti-perte rebuilt==before AVANT le DROP de l'ancienne table (comme
// append_only_rebuild.go), transaction intégrale (rollback complet sur toute erreur).
// Colonnes énumérées dynamiquement (loadTableColumns) plutôt qu'en dur : robuste à un
// futur ALTER TABLE ADD COLUMN sur weapons.

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func init() {
	Register(Migration{
		Name:        "purge_weapons_name_fr_column",
		TargetDB:    TargetMetadata,
		Description: "Retire la colonne inerte weapons.name_fr (repli de nom abandonné V72-06 — source unique désormais weapon_name_labels) via rebuild CTAS-swap : DuckDB refuse ALTER ... DROP COLUMN sous PK/index",
		ApplySchema: applyPurgeWeaponsNameFR,
	})
}

// applyPurgeWeaponsNameFR retire weapons.name_fr si elle est présente.
func applyPurgeWeaponsNameFR(db *sql.DB) error {
	has, err := tableExists(db, "weapons")
	if err != nil {
		return fmt.Errorf("purge weapons.name_fr: check table: %w", err)
	}
	if !has {
		// metadata pas encore provisionnée (add_weapon_registry pas encore joué,
		// ou provider title-owned nil dans un binaire de test) — rien à purger.
		return nil
	}
	hasCol, err := columnExists(db, "weapons", "name_fr")
	if err != nil {
		return fmt.Errorf("purge weapons.name_fr: check column: %w", err)
	}
	if !hasCol {
		// Déjà purgée (DB neuve post-V72-06, ou 2e exécution) — idempotent.
		return nil
	}

	ctx := bootCtx()
	cols, err := loadTableColumns(ctx, db, "weapons")
	if err != nil {
		return fmt.Errorf("purge weapons.name_fr: enumerate columns: %w", err)
	}
	keep := make([]string, 0, len(cols))
	for _, c := range cols {
		if c != "name_fr" {
			keep = append(keep, c)
		}
	}
	if len(keep) == len(cols) {
		// Défensif : columnExists vient de confirmer sa présence — ne devrait
		// jamais se produire (pas de writer concurrent au boot), mais on refuse
		// un swap qui ne retirerait rien plutôt que de le faire en silence.
		return fmt.Errorf("purge weapons.name_fr: colonne détectée mais absente de PRAGMA table_info (incohérence)")
	}

	var before int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM weapons`).Scan(&before); err != nil {
		return fmt.Errorf("purge weapons.name_fr: count before: %w", err)
	}

	return swapWeaponsPurgeNameFRTx(ctx, db, strings.Join(keep, ", "), before)
}

// swapWeaponsPurgeNameFRTx effectue le swap CTAS dans une transaction unique. Garde
// anti-perte : refuse de détruire l'original si le rebuild n'a pas exactement le
// même nombre de rows. Rollback intégral sur toute erreur — `weapons` reste intacte.
func swapWeaponsPurgeNameFRTx(ctx context.Context, db *sql.DB, colList string, before int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("purge weapons.name_fr: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS weapons__purge_name_fr`); err != nil {
		return fmt.Errorf("purge weapons.name_fr: drop stale rebuild table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(
		`CREATE TABLE weapons__purge_name_fr AS SELECT %s FROM weapons`, colList)); err != nil {
		return fmt.Errorf("purge weapons.name_fr: create rebuild table: %w", err)
	}
	var rebuilt int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM weapons__purge_name_fr`).Scan(&rebuilt); err != nil {
		return fmt.Errorf("purge weapons.name_fr: count rebuild table: %w", err)
	}
	if rebuilt != before {
		return fmt.Errorf("purge weapons.name_fr: swap abandonné, rebuilt=%d != before=%d (rollback, zéro perte)", rebuilt, before)
	}
	for _, stmt := range []string{
		`DROP TABLE weapons`,
		`ALTER TABLE weapons__purge_name_fr RENAME TO weapons`,
		`ALTER TABLE weapons ADD PRIMARY KEY (title_slug, weapon_key)`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("purge weapons.name_fr: swap step (%s): %w", firstWords(stmt, 3), err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("purge weapons.name_fr: commit: %w", err)
	}
	committed = true
	return nil
}
