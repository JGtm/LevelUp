package migrations

// steps_medal_hard_target.go — la médaille custom « Cible difficile » (9000000002, Hard Target).
//
// Elle récompense une recharge complète du bouclier après qu'il a été brisé (une médaille par
// recharge). Elle n'existe pas dans le catalogue officiel : LevelUp la dérive du film (colonne
// `match_usage_players.shield_recharges`, vue `v_medals_earned`) comme « Vengeur » (9000000001,
// seed_custom_vengeur_medal), dont elle reprend la catégorie (skill) et la rareté (Normal). Le nom
// français est le nom officiel de la médaille de Halo 5.
//
// CE STEP SUIT metadata_definition_texts_to_translations_v1 : les colonnes `name_fr` / `name_en` de
// `medal_definitions` n'existent plus après lui, les textes s'écrivent donc par langue dans
// `medal_translations` (migration.UpsertMedalTexts). INSERT … WHERE NOT EXISTS et non ON CONFLICT :
// medal_definitions n'a pas de clé primaire sur les bases héritées.

import (
	"database/sql"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/migration"
)

// idMedailleCibleDifficile est l'identifiant de la médaille custom Cible difficile.
const idMedailleCibleDifficile int64 = 9000000002

func medalHardTargetSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:        "seed_custom_hard_target_medal",
			TargetDB:    migration.TargetMetadata,
			Description: "medal_definitions + medal_translations : seed médaille custom Cible difficile (9000000002), recharge complète du bouclier",
			ApplySchema: func(db *sql.DB) error {
				if err := migration.ExecScript(db, `
					INSERT INTO medal_definitions
						(medal_name_id, is_custom, difficulty_index, type_index, difficulty, medal_type, personal_score)
					SELECT 9000000002, TRUE, 0, 4, 'Normal', 'skill', 0
					WHERE NOT EXISTS (
						SELECT 1 FROM medal_definitions WHERE medal_name_id = 9000000002
					);
				`); err != nil {
					return err
				}
				return migration.UpsertMedalTexts(migration.BootCtx(), db, idMedailleCibleDifficile,
					i18n.FromColumns("Hard Target", "Cible difficile"),
					i18n.FromColumns("Fully recharge your shield after it was broken.",
						"Rechargez complètement votre bouclier après qu'il a été brisé."))
			},
		},
	}
}
