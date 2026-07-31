package migration

// steps_shared_player_positions.go — table v3 (film) pour les POSITIONS joueurs
// keyframe (§N de .ai/RESEARCH_THEATER_RE.md).
//
// Net-new : aucune table existante touchée. Produite par
// internal/analysis/positions.DecodeKeyframePositions depuis les chunks film
// (paquet TYPE_2, combs bit-level + triplet float32-LE). 1 ligne = 1 position
// full-state décodée (granularité ~20s du snapshot type-2).
//
// LIMITE (honnête, §N) : v1 = MATCH-LEVEL — pas d'attribution xuid (la
// delta-compression bloque l'index par joueur). La colonne team est best-effort
// (-1 = inconnu) ; aucun lien xuid n'est posé.
//
// Pas de PK contraignante : l'écriture est un DELETE-then-INSERT par match (un
// même triplet (time_ms, x, y, z) peut légitimement se répéter — spawns, points
// fixes — donc une PK (match_id, time_ms, x, y, z) rejetterait des positions
// valides). team est INTEGER (0/1 ou -1). written_at trace la provenance.
//
// Écriture HORS chemin live (backfill diagnostic sérialisé, MaxOpenConns(1)) →
// pas de pression concurrente ART (cf. .ai/PLAN_WEAPON_ATTRIBUTION_V3.md §10).

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_match_player_positions_v1",
		TargetDB:    TargetShared,
		Description: "Table match_player_positions (positions keyframe match-level, v3 film)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS match_player_positions (
					match_id   VARCHAR NOT NULL,
					time_ms    INTEGER,
					x          REAL,
					y          REAL,
					z          REAL,
					team       INTEGER,
					written_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
			`)
		},
	})
}
