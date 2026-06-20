package livesync

import (
	"context"
	"log/slog"
)

// XUIDResolver résout un gamertag Xbox en xuid (impl de prod :
// worldenrich.CachingResolver sur PeopleHub — Xbox-global, title-agnostic, donc un
// gamertag h5 résout le MÊME xuid réel que côté Infinite). Interface locale
// (duck-typing) pour ne pas importer worldenrich/service depuis livesync.
type XUIDResolver interface {
	ResolveXUID(ctx context.Context, gamertag string) (string, error)
}

// ResolveXUIDClosure adapte un XUIDResolver en func(gamertag) string consommé par
// la capture (ingest.CollectMatchBatch). Échec de résolution (gamertag inconnu,
// 429, réseau) → "" loggé : l'identité reste portée par le GAMERTAG (display via
// v_gamertag_lookup, kill-feed), on ne FABRIQUE jamais d'xuid. res nil → tout "".
//
// ⚠ Le "" est toléré pour les lignes kill-feed/médailles (colonne gamertag présente)
// mais PAS pour des lignes d'agrégat/participant en masse (collision de PK xuid="") :
// le câblage viewer doit résoudre l'xuid du self AVANT (sinon échec du couple).
func ResolveXUIDClosure(ctx context.Context, res XUIDResolver, logger *slog.Logger) func(gamertag string) string {
	if res == nil {
		return func(string) string { return "" }
	}
	if logger == nil {
		logger = slog.Default()
	}
	return func(gt string) string {
		xuid, err := res.ResolveXUID(ctx, gt)
		if err != nil {
			logger.WarnContext(ctx, "h5 sync: xuid non résolu (identité gamertag conservée)",
				"gamertag", gt, "err", err)
			return ""
		}
		return xuid
	}
}
