package port

import (
	"context"

	"levelup/go-api/internal/analysis/leadshare"
)

// LeadShareRepository — la MAINMISE de l'équipe d'un joueur sur un lot de matchs : UN chargement
// par requête, borné par la liste des matchs du périmètre (ADR 0036), sur la vue
// `match_lead_share_latest` seulement. `playerXUID` est le joueur de la page : son camp dans chaque
// match est lu de `match_participants`.
//
// Le résultat ne porte que les matchs MESURÉS : un match absent de la carte n'est pas mesuré
// (mode à peu de points, score non couvert, film absent...) et ne se lit jamais comme zéro.
//
// Implémenté par internal/platform/duckdb.LeadShareRepo, câblé sous `film.lead_share`. Table
// absente : games.ErrCapabilityNotSupported.
type LeadShareRepository interface {
	LoadLeadShare(ctx context.Context, matchIDs []string, playerXUID string) (map[string]leadshare.PlayerShare, error)
}
