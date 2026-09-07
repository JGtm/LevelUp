package migration

// steps_metadata_purge_weapon_families_labels.go — purge des colonnes inertes
// weapon_families.name_en / weapon_families.name_fr (plan libellés en dur, lot M5 L4,
// .ai/PLAN_LIBELLES_EN_DUR_GO_2026-09-07.md §4 L4).
//
// Contexte : `weapon_families` (games/weapons/registry.go, ApplyRegistry) porte un
// libellé EN/FR par famille d'arme (« battle_rifle » -> "Battle Rifle"/"Fusil de
// combat"…), en dur dans le slice Go `weaponRegistryFamilies`. Vérifié par grep sur tout
// le module (Go ET web) avant d'écrire cette migration : AUCUN lecteur ne sélectionne
// jamais name_en/name_fr depuis cette table — ni un handler, ni un service, ni un DTO,
// ni le web (`apps/web/src/lib/i18n/manifests/frags.toml` localise déjà les niveaux
// « classe » et « rôle » du sunburst de frags, jamais le niveau « famille » ; le nom
// d'affichage PAR ARME est une source distincte et déjà correcte, `weapon_name_labels` /
// `config/titles/{slug}/mappings/weapon_names.toml`, V72-06). Seul un test d'intégrité
// référentielle (`registry_test.go::TestWeaponRegistry_ReferentialIntegrity`) rejoint
// `weapon_families` sur sa clé, jamais sur ses libellés. `family_key` reste la SEULE
// donnée utile de la table (whitelist référentielle jointe depuis `weapons.family_key`).
//
// Pourquoi une purge et pas une migration vers un TOML : un libellé qui n'a AUCUN
// lecteur n'a pas de destination utile — le déplacer vers `assets.toml` recopierait du
// contenu mort dans un second fichier au lieu de l'éteindre (règle dépôt « 0 code mort »,
// CLAUDE.md règle 7). Suppression au lieu de migration, décision consignée dans le
// journal (.ai/thought_log.md, entrée du lot M5 L4).
//
// Pattern : rebuild CTAS-swap transactionnel, calqué EXACTEMENT sur
// `purge_weapons_name_fr_column` (steps_metadata_purge_weapons_name_fr.go, V721-05.1) —
// DuckDB refuse `ALTER TABLE ... DROP COLUMN` tant qu'un index existe (ici la PK
// `family_key`), donc on recrée la table sans les deux colonnes. `weapon_families` est un
// référentiel STATIQUE explicitement HORS périmètre du bug ART #23046 (zéro writer
// concurrent, zéro écriture per-match — cf. tête de games/weapons/registry.go) : pas de
// vue `_latest` ni de `written_at` ici, juste la PK simple recréée à l'identique.
//
// Idempotente : columnExists(name_en) fait office de garde — no-op si les colonnes sont
// déjà absentes (DB neuve post-lot, ou 2e exécution). Garde anti-perte rebuilt==before
// AVANT le DROP de l'ancienne table. Transaction intégrale (rollback complet sur toute
// erreur).

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func init() {
	Register(Migration{
		Name:        "purge_weapon_families_labels_columns",
		TargetDB:    TargetMetadata,
		Description: "Retire les colonnes inertes weapon_families.name_en / name_fr (libellés jamais lus — plan libellés en dur, lot M5 L4) via rebuild CTAS-swap : DuckDB refuse ALTER ... DROP COLUMN sous PK",
		ApplySchema: applyPurgeWeaponFamiliesLabels,
	})
}

// applyPurgeWeaponFamiliesLabels retire name_en/name_fr de weapon_families si présentes.
func applyPurgeWeaponFamiliesLabels(db *sql.DB) error {
	has, err := tableExists(db, "weapon_families")
	if err != nil {
		return fmt.Errorf("purge weapon_families labels: check table: %w", err)
	}
	if !has {
		// metadata pas encore provisionnée (add_weapon_registry pas encore joué,
		// ou provider title-owned nil dans un binaire de test) — rien à purger.
		return nil
	}
	hasCol, err := columnExists(db, "weapon_families", "name_en")
	if err != nil {
		return fmt.Errorf("purge weapon_families labels: check column: %w", err)
	}
	if !hasCol {
		// Déjà purgée (DB neuve post-lot, ou 2e exécution) — idempotent.
		return nil
	}

	ctx := bootCtx()
	cols, err := loadTableColumns(ctx, db, "weapon_families")
	if err != nil {
		return fmt.Errorf("purge weapon_families labels: enumerate columns: %w", err)
	}
	keep := make([]string, 0, len(cols))
	for _, c := range cols {
		if c != "name_en" && c != "name_fr" {
			keep = append(keep, c)
		}
	}
	if len(keep) != len(cols)-2 {
		// Défensif : columnExists vient de confirmer name_en — ne devrait jamais
		// se produire (pas de writer concurrent au boot), mais on refuse un swap
		// qui ne retirerait pas exactement les deux colonnes plutôt que de le
		// faire en silence.
		return fmt.Errorf("purge weapon_families labels: colonnes détectées mais absence incohérente de PRAGMA table_info")
	}

	var before int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM weapon_families`).Scan(&before); err != nil {
		return fmt.Errorf("purge weapon_families labels: count before: %w", err)
	}

	return swapWeaponFamiliesPurgeLabelsTx(ctx, db, strings.Join(keep, ", "), before)
}

// swapWeaponFamiliesPurgeLabelsTx effectue le swap CTAS dans une transaction unique.
// Garde anti-perte : refuse de détruire l'original si le rebuild n'a pas exactement le
// même nombre de rows. Rollback intégral sur toute erreur — `weapon_families` reste
// intacte.
func swapWeaponFamiliesPurgeLabelsTx(ctx context.Context, db *sql.DB, colList string, before int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("purge weapon_families labels: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS weapon_families__purge_labels`); err != nil {
		return fmt.Errorf("purge weapon_families labels: drop stale rebuild table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(
		`CREATE TABLE weapon_families__purge_labels AS SELECT %s FROM weapon_families`, colList)); err != nil {
		return fmt.Errorf("purge weapon_families labels: create rebuild table: %w", err)
	}
	var rebuilt int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM weapon_families__purge_labels`).Scan(&rebuilt); err != nil {
		return fmt.Errorf("purge weapon_families labels: count rebuild table: %w", err)
	}
	if rebuilt != before {
		return fmt.Errorf("purge weapon_families labels: swap abandonné, rebuilt=%d != before=%d (rollback, zéro perte)", rebuilt, before)
	}
	for _, stmt := range []string{
		`DROP TABLE weapon_families`,
		`ALTER TABLE weapon_families__purge_labels RENAME TO weapon_families`,
		`ALTER TABLE weapon_families ADD PRIMARY KEY (family_key)`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("purge weapon_families labels: swap step (%s): %w", firstWords(stmt, 3), err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("purge weapon_families labels: commit: %w", err)
	}
	committed = true
	return nil
}
