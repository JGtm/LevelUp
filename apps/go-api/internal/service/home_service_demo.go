// Package service — home_service_demo.go : fixtures de contenu DÉMO injectées au
// niveau LECTURE (pas au seed), guardées par s.demoMode.
//
// Pourquoi read-time et pas une table seedée : en démo il n'y a pas d'API Halo
// live, et le cache `challenge_snapshots` a un TTL de 24h (battlePassCacheTTLFallback)
// — une fixture seedée disparaîtrait 24h après chaque reseed. Une fixture embarquée
// servie à la lecture est stable (survit reseed + déploiement, pas d'ageing).
package service

import (
	"context"
	_ "embed"
	"encoding/json"
	"log/slog"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

//go:embed demo_fixtures/challenges.json
var demoChallengesJSON []byte

//go:embed demo_fixtures/challenges.en.json
var demoChallengesENJSON []byte

// demoChallenges retourne la fixture défis démo (embarquée), dans la locale de la
// requête (ctxkeys.Locale). Bypass live + cache. snapshot_at = maintenant (fraîcheur
// ok côté UI, pas d'indicateur "périmé"). Parité FR/EN 1:1 (challenges.json /
// challenges.en.json) : sans sélection par locale, la démo affiche les défis en FR
// même quand l'UI passe en anglais.
func demoChallenges(ctx context.Context) domain.ChallengesResponse {
	fixture := demoChallengesJSON
	if ctxkeys.Locale(ctx) == "en" {
		fixture = demoChallengesENJSON
	}
	var resp domain.ChallengesResponse
	if err := json.Unmarshal(fixture, &resp); err != nil {
		slog.WarnContext(ctx, "demo challenges fixture parse failed", "err", err)
		return domain.ChallengesResponse{Available: false}
	}
	resp.Available = true
	resp.FromCache = false
	now := time.Now().UTC().Format(time.RFC3339)
	resp.SnapshotAt = &now
	return resp
}
