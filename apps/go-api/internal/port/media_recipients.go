package port

// media_recipients.go — les lectures qui désignent les destinataires d'une notification d'import de média :
// les matchs associés aux médias récents, puis les joueurs qui y ont participé.

import (
	"context"
	"time"
)

// MediaRecipientSource lit, dans les bases partagées, ce qu'il faut pour désigner les destinataires.
type MediaRecipientSource interface {
	// RecentMediaMatchIDs rend les identifiants de match associés à un média depuis `since` (au plus 50),
	// lus dans la base sociale partagée située à sharedSocialDBPath.
	RecentMediaMatchIDs(ctx context.Context, sharedSocialDBPath string, since time.Time) ([]string, error)
	// MatchParticipantXUIDs rend les xuid distincts (au plus 50) des participants aux matchs donnés, lus
	// dans la base de matchs partagée située à sharedMatchesDBPath.
	MatchParticipantXUIDs(ctx context.Context, sharedMatchesDBPath string, matchIDs []string) ([]string, error)
}
