package migrations

import (
	"database/sql"

	"levelup/go-api/internal/migration"
)

// sharedCommendationsSteps — table match_commendations (shared-core, schéma commun
// inter-titres). Compteur par-match des commendations NATIVES progressées sur un
// match (Halo 5 natif, AXE B prod-gate), exactement comme medals_earned compte les
// médailles par (match_id, xuid).
//
// Halo 5 la remplit NATIVEMENT (carnage PlayerStats[].ProgressiveCommendationDeltas :
// count = Progress − PreviousProgress par commendation). Infinite la laisse vide
// (commendations dérivées via le moteur de citations, pas une donnée per-match
// native) — c'est le substrat append-only que Halo 5 valide.
//
// ART-SAFETY (#23046) : INSERT-only / INSERT OR IGNORE côté persister. La clé
// naturelle (match_id, xuid, commendation_id) n'est JAMAIS mutée (count posé une
// fois à l'INSERT). AUCUN index secondaire — la PK suffit (parité medals_earned).
// commendation_id est l'UUID natif de commendation (VARCHAR), pas un numérique.
func sharedCommendationsSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:        "shared_create_match_commendations",
			TargetDB:    migration.TargetShared,
			Description: "Compteur par-match des commendations natives (match_commendations) — Halo 5 natif, INSERT-only ART-safe",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS match_commendations (
						match_id        VARCHAR NOT NULL,
						xuid            VARCHAR NOT NULL,
						commendation_id VARCHAR NOT NULL,
						count           INTEGER,
						created_at      TIMESTAMP,
						PRIMARY KEY (match_id, xuid, commendation_id)
					);
				`)
			},
		},
	}
}
