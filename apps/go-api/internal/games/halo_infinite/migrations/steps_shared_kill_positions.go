package migrations

import (
	"database/sql"

	"levelup/go-api/internal/migration"
)

// sharedKillPositionsSteps — table kill_positions (shared-core, schéma commun
// inter-titres). Positions monde (Vec3) du tueur et de la victime par kill,
// jointes au kill par (match_id, killer_xuid, time_ms).
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
