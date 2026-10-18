package watcher

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
)

// reportDetected capte l'instant où le watcher lance le sync des matchs d'une fin de partie :
// c'est le « détecté » du délai du débrief (cf. domain.MatchStepDetected).
func reportDetected(req MatchRequest) {
	slug := req.TitleSlug
	if slug == "" {
		slug = title.DefaultSlug
	}
	for _, id := range req.MatchIDs {
		observability.ReportMatchStep(slug, id, domain.MatchStepDetected, "")
	}
}
