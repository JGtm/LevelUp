package migration

// steps_shared_objective_events.go — tables v3 (film) pour les events objectif.
//
// Net-new : aucune table existante touchée (highlight_events ne porte que kill/death/medal/mode).
// Mode-agnostique : objective_type (parent: flag|zone|hill|skull|bomb) + event_type (action) ; le mode se dérive
// de match_registry.game_variant_name. source/confidence tracent la provenance + la précision (CTF ms-exact vs
// Strongholds/KOTH/Oddball ~5-20s). details (JSON-as-VARCHAR) = échappatoire pour les champs non encore modélisés.
//
// Écriture : DELETE-then-INSERT par match (comme weapon_kills) depuis le backfill diagnostic v3 (sérialisé,
// MaxOpenConns(1)), HORS chemin live → pas de pression concurrente ART. Voir .ai/PLAN_WEAPON_ATTRIBUTION_V3.md §10.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_objective_events_v1",
		TargetDB:    TargetShared,
		Description: "Tables match_objective_events + match_objective_event_players (v3 film, mode-agnostique)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS match_objective_events (
					match_id       VARCHAR NOT NULL,
					seq            INTEGER NOT NULL,
					time_ms        INTEGER,
					objective_type VARCHAR,
					event_type     VARCHAR,
					team_id        INTEGER,
					objective_id   INTEGER,
					value          INTEGER,
					source         VARCHAR,
					confidence     VARCHAR,
					details        VARCHAR,
					written_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (match_id, seq)
				);
				CREATE TABLE IF NOT EXISTS match_objective_event_players (
					match_id   VARCHAR NOT NULL,
					seq        INTEGER NOT NULL,
					xuid       VARCHAR NOT NULL,
					role       VARCHAR,
					written_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (match_id, seq, xuid)
				);
			`)
		},
	})
}
