package migrations

import (
	"database/sql"

	"levelup/go-api/internal/migration"
)

// sharedKillPositionsSteps — table kill_positions (shared-core, schéma commun
// inter-titres). Positions monde (Vec3) du tueur et de la victime par kill,
// jointes au kill par (match_id, killer_xuid, time_ms).
//
// Forme actuelle : append-only depuis G.2 (steps_appendonly_misc.go : id PK + written_at),
// arbitrée PAR PASSE depuis le lot 1.7 (steps_shared_kill_positions_pass.go : + decode_pass,
// vue kill_positions_latest = dernière passe entière par match). Le CREATE ci-dessous est le
// schéma d'ORIGINE, name-keyed et donc figé : les deux steps suivants le font évoluer.
//
// Halo 5 la remplit NATIVEMENT (KillerWorldLocation/VictimWorldLocation dans la
// timeline) ; Halo Infinite la laisse vide tant que le décodeur de film n'extrait
// pas les coordonnées monde (`not_exposed`). C'est le schéma de référence que
// Halo 5 valide et qu'Infinite remplira plus tard — append-only, INSERT-only.
func sharedKillPositionsSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:        "shared_create_kill_positions",
			TargetDB:    migration.TargetShared,
			Description: "Positions monde tueur/victime par kill (kill_positions) — Halo 5 natif, Infinite plus tard",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS kill_positions (
						match_id    VARCHAR NOT NULL,
						killer_xuid VARCHAR,
						time_ms     INTEGER,
						killer_x    DOUBLE, killer_y DOUBLE, killer_z DOUBLE,
						victim_x    DOUBLE, victim_y DOUBLE, victim_z DOUBLE
					);
					CREATE INDEX IF NOT EXISTS idx_kill_positions_match ON kill_positions(match_id);
				`)
			},
		},
	}
}
