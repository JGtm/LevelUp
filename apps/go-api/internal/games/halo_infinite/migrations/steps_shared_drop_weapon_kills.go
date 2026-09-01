package migrations

// steps_shared_drop_weapon_kills.go — LA TABLE `weapon_kills` DISPARAÎT DU FICHIER
// HALO INFINITE, ET ELLE SEULE.
//
// # POURQUOI UNE MIGRATION TITLE-OWNED, ET PAS UNE ÉTAPE PARTAGÉE
//
// Chaque titre a SON `shared_matches_v2.duckdb` (ADR 0008), mais les deux partagent le
// SCHÉMA : une étape du registre global s'exécute sur les DEUX fichiers. Or Halo 5 tient
// 550 926 lignes de `weapon_kills` avec `confidence = 'native'`, issues de la timeline de
// son API — donnée AUTORITAIRE, sans le moindre rapport avec la corrélation de tirs qui
// alimentait Halo Infinite. La supprimer pour tous les titres aveuglerait le second.
// D'où une étape *title-owned* : elle ne tourne que sur le fichier `halo_infinite`.
//
// # CE QU'ELLE EMPORTE, ET CE QU'ELLE N'ARCHIVE PAS
//
// Les 112 139 lignes Halo Infinite partent EN TOTALITÉ. Aucune table de sauvegarde, aucune
// copie « au cas où » : décision explicite de l'utilisateur du 2026-09-01. Ces lignes sont
// le produit d'une corrélation défaillante (une épée, un marteau, un faisceau n'émettent
// pas de tirs : leurs kills étaient perdus ou recollés sur l'arme à feu tenue, ce qui
// gonflait l'AR, le BR et le Sidekick). L'arme d'un kill vient désormais de la SOURCE DU
// DÉGÂT, lue à la volée et jamais stockée.
//
// # LA VUE EXISTE EN DEUX EXEMPLAIRES, ET UN SEUL `DROP` NU N'EN VOIT QU'UN
//
// Mesure du 2026-09-01 sur la base de production (`duckdb_views()`) :
//
//	schema_name  view_name
//	main         v_weapon_kills      la vue COURANTE (dedup par generation_id)
//	shared       v_weapon_kills      une vue LEGACY, d'avant l'append-only
//
// `DROP VIEW IF EXISTS v_weapon_kills` ne résout que sur le `search_path`, donc ne tombe
// que sur l'une des deux et le nom SURVIT dans l'autre schéma — un lecteur pourrait le
// retrouver. Les deux sont donc qualifiées explicitement. La TABLE, elle, n'est PAS dans
// ce cas (vérifié : `main` seulement), pas plus que `weapon_kills_v3` ni la séquence.
//
// Aucune autre vue ne lit `v_weapon_kills` (vérifié sur `duckdb_views().sql` : le seul
// autre résultat, `mv_player_matches`, matchait sur la COLONNE `power_weapon_kills`).
// Le piège DuckDB du `SELECT *` figé à la création ne s'applique donc à personne.

import (
	"database/sql"
	"fmt"

	"levelup/go-api/internal/migration"
)

// dropWeaponKillsHINF supprime la vue (dans ses DEUX schémas), la table, la table morte
// `weapon_kills_v3` et la séquence de génération, puis force un CHECKPOINT.
//
// Le schéma `shared` peut ne pas exister (base neuve) : la vue y est donc cherchée dans
// `duckdb_views()` avant d'être droppée, plutôt que qualifiée en aveugle — un `DROP … IF
// EXISTS` sur un schéma absent est une erreur, pas un no-op.
func dropWeaponKillsHINF(db *sql.DB) error {
	schemas, err := schemasPortantLaVue(db, "v_weapon_kills")
	if err != nil {
		return err
	}
	for _, s := range schemas {
		// Nom de schéma issu du catalogue DuckDB, jamais d'une entrée utilisateur.
		if _, err := db.ExecContext(migration.BootCtx(), fmt.Sprintf(`DROP VIEW IF EXISTS "%s".v_weapon_kills`, s)); err != nil {
			return fmt.Errorf("drop %s.v_weapon_kills: %w", s, err)
		}
	}
	return migration.ExecScript(db, `
		DROP TABLE IF EXISTS weapon_kills;
		DROP TABLE IF EXISTS weapon_kills_v3;
		DROP SEQUENCE IF EXISTS weapon_kills_generation_seq;
		CHECKPOINT;
	`)
}

// schemasPortantLaVue rend les schémas où la vue existe RÉELLEMENT.
func schemasPortantLaVue(db *sql.DB, view string) ([]string, error) {
	rows, err := db.QueryContext(migration.BootCtx(), `SELECT schema_name FROM duckdb_views() WHERE view_name = ?`, view)
	if err != nil {
		return nil, fmt.Errorf("duckdb_views(%s): %w", view, err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("duckdb_views(%s) scan: %w", view, err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdb_views(%s) rows: %w", view, err)
	}
	return out, nil
}

// stepDropWeaponKills — l'étape, référencée depuis Steps().
//
// ⚠ `OnlyTitles` N'EST PAS DÉCORATIF, ET « TITLE-OWNED » NE SUFFISAIT PAS. Vérifié sur
// pièces le 2026-09-01 : `TitleMigrationSet.OwnsTarget` de Halo 5 ne possède QUE
// `metadata` ; pour `shared` il retombe DÉLIBÉRÉMENT sur le fallback, c'est-à-dire sur le
// registre global PLUS le provider title-owned de Halo Infinite (« hérite du schéma
// uniforme », cf. l'en-tête de title_set.go). Sans cette restriction, ce DROP se serait
// exécuté sur `data/titles/halo_5/warehouse/shared_matches_v2.duckdb` et aurait effacé ses
// 550 926 lignes NATIVES. Un step additif s'accommode de l'héritage ; un step destructif
// jamais.
func stepDropWeaponKills() migration.Migration {
	return migration.Migration{
		Name:        "shared_drop_weapon_kills_v1",
		TargetDB:    migration.TargetShared,
		Description: "Halo Infinite : suppression de weapon_kills, de ses deux vues v_weapon_kills (main + shared), de weapon_kills_v3 et de la séquence de génération",
		OnlyTitles:  []string{migration.DefaultSlug},
		ApplySchema: dropWeaponKillsHINF,
	}
}
