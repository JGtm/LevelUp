// Package migration — match_steps_schema.go : schéma des INSTANTS MESURÉS du chemin d'un match
// (domain.MatchStep), source du rapport « délai du débrief ».
//
// Dans la base monitoring GLOBALE, comme la file de construction : même fichier, même writer
// unique (le process serveur), même lease KindMonitoring. Aucune base par match n'est touchée.
//
// APPEND-ONLY : un instant mesuré est un FAIT, jamais corrigé. Il n'y a pas d'état courant à
// servir (pas de vue `_latest`) : le rapport agrège les faits d'un match (première occurrence
// de chaque étape, citations par joueur) — cf. analysis.DebriefDelay.
package migration

// matchStepsDDL : instructions idempotentes, posées par EnsureMonitoringSchema.
var matchStepsDDL = []string{
	`CREATE SEQUENCE IF NOT EXISTS match_step_events_seq START 1`,
	// step_at : l'instant de l'étape (capté à l'événement, UTC) ; written_at : celui de l'écriture,
	// plus tardif (vidage périodique du collecteur). Aucun index secondaire : la table est bornée
	// par la rétention (DELETE), et chaque index ART de plus y serait exposé au bug DuckDB #23645.
	`CREATE TABLE IF NOT EXISTS match_step_events (
		id         BIGINT PRIMARY KEY DEFAULT nextval('match_step_events_seq'),
		title_slug VARCHAR,
		match_id   VARCHAR NOT NULL,
		step       VARCHAR NOT NULL,
		step_at    TIMESTAMP NOT NULL,
		detail     VARCHAR,
		written_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
	)`,
}
