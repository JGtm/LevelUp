package migrations

// steps_medal_watch_the_throne.go — la médaille VIP « Watch the Throne » (1210968206).
//
// Comme « Clash of Kings » (steps.go, seed_clash_of_kings_medal), elle est ABSENTE du catalogue
// officiel GameCMS (hi/Waypoint/file/medals/metadata.json : 151 médailles, aucune médaille VIP,
// relu le 2026-10-17 par `refresh-metadata medal-images --dump-raw`). Sa seule source est
// SpartanRecord src/Objects/Helpers/AllMedals.tsx : nom « Watch the Throne », description
// « Save your VIP's life », rareté Normal, catégorie VIP. Le nom est confirmé indépendamment par
// la table de noms du film (games/halo_infinite/film/medalname/table.go, couple 50/160).
//
// La ligne héritée de l'ère Python portait la description d'une médaille de Roi de la colline
// (« Control the hill from start to finish during a complete cycle. » / « Contrôlez la colline
// du début à la fin lors d'un cycle complet. »). La correction est INCONDITIONNELLE sur
// l'identifiant, gardée par `description_en <> 'Save your VIP''s life'` pour rester idempotente.
//
// Libellés FR : aucune source officielle n'en existe. Le nom « Garde du trône » (déjà servi) et
// la description « Sauvez la vie de votre VIP. » sont une traduction LevelUp, validée par
// l'utilisateur le 2026-10-17.
//
// medal_translations est purgée pour cet identifiant : la chaîne FR de lecture commence par
// `mt_loc.name` (platform/duckdb/medal_label_resolve.go), une traduction héritée y masquerait
// medal_definitions. INSERT … WHERE NOT EXISTS et non ON CONFLICT : medal_definitions n'a pas de
// clé primaire sur les bases héritées (cf. TestMedalDefinitions_PasDOnConflict).

import (
	"database/sql"

	"levelup/go-api/internal/migration"
)

func medalWatchTheThroneSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:        "fix_watch_the_throne_medal",
			TargetDB:    migration.TargetMetadata,
			Description: "medal_definitions : médaille VIP Watch the Throne (1210968206), description de Roi de la colline remplacée",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					INSERT INTO medal_definitions
						(medal_name_id, name_fr, name_en, description_fr, description_en,
						 is_custom, difficulty_index, type_index, difficulty, medal_type, personal_score)
					SELECT 1210968206, 'Garde du trône', 'Watch the Throne',
					       'Sauvez la vie de votre VIP.', 'Save your VIP''s life',
					       FALSE, 0, 1, 'Normal', 'mode', 0
					WHERE NOT EXISTS (
						SELECT 1 FROM medal_definitions WHERE medal_name_id = 1210968206
					);

					UPDATE medal_definitions
					SET name_en        = 'Watch the Throne',
					    name_fr        = 'Garde du trône',
					    description_en = 'Save your VIP''s life',
					    description_fr = 'Sauvez la vie de votre VIP.'
					WHERE medal_name_id = 1210968206
					  AND COALESCE(description_en, '') <> 'Save your VIP''s life';

					DELETE FROM medal_translations WHERE medal_name_id = 1210968206;
				`)
			},
		},
	}
}
