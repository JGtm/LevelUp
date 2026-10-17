package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// AnalysisProbeRepository lit l'avancement de l'analyse du film d'un LOT de matchs.
// Implémenté par platform/duckdb.AnalysisProbeRepo.
type AnalysisProbeRepository interface {
	// LoadAnalysisProbes rend, par match_id du lot, ce que le registre et la base du joueur
	// savent de l'analyse. Un match absent du registre est absent de la map. Deux lectures
	// au total, quelle que soit la taille du lot (jamais une requête par match). Erreur
	// propagée : l'appelant journalise et dégrade (aucun indicateur).
	LoadAnalysisProbes(ctx context.Context, matchIDs []string) (map[string]domain.AnalysisProbe, error)
}

// FilmDecodeObserver dit quel match un décodage de film est en train de traiter sur la machine.
// Implémenté par filmproc.DecodeObserver (lecture de la description du détenteur du verrou
// machine, sans jamais prendre le verrou).
type FilmDecodeObserver interface {
	// DecodingMatchID rend l'identifiant du match que décode un processus vivant, ou "" quand
	// aucun décodage n'est en cours (ou que le détenteur ne s'est pas décrit).
	DecodingMatchID() string
}
