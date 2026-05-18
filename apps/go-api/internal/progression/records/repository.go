package records

import "context"

// repository.go — interfaces de persistance pour les PB et leur historique.
//
// Deux DB distinctes (cf. §2bis et §7 du plan) :
//   - PBRepo  → player_records dans shared_social.duckdb (xuid en PK)
//   - HistoryRepo → record_history dans stats.duckdb (user_id en PK joueur)
//
// Le detector reçoit les deux et orchestre les écritures simultanément.

// PBRepo gère les records personnels courants (un par triplet xuid×metric×period).
type PBRepo interface {
	// Get retourne le PB courant pour (xuid, metric, period), ou nil si aucun.
	Get(ctx context.Context, xuid, metric string, period RecordPeriod) (*PersonalRecord, error)

	// Upsert crée ou remplace le PB. La PK est (xuid, metric, period).
	Upsert(ctx context.Context, r PersonalRecord) error

	// ListByXUID retourne tous les PB d'un joueur, triés par period puis metric.
	ListByXUID(ctx context.Context, xuid string) ([]PersonalRecord, error)
}

// HistoryRepo gère l'historique append-only des PB battus.
type HistoryRepo interface {
	// Append insère une entrée d'historique (PB battu à un instant t).
	Append(ctx context.Context, h RecordHistory) error

	// ListRecent retourne les N dernières entrées d'historique pour un joueur,
	// triées par achieved_at DESC. Si limit <= 0, default raisonnable.
	ListRecent(ctx context.Context, userID, titleSlug string, limit int) ([]RecordHistory, error)
}
