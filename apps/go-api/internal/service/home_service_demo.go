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
	"embed"
	"encoding/json"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

// demoFixturesFS porte une fixture de défis par langue : `demo_fixtures/challenges.<code>.json`.
// Une langue sans fixture sert celle de la langue de repli (compté, voir demoChallengesFixture).
//
//go:embed demo_fixtures/challenges.*.json
var demoFixturesFS embed.FS

// demoChallengesFixture lit la fixture de la langue `l`, sinon celle de i18n.Fallback.
func demoChallengesFixture(ctx context.Context, l i18n.Locale) ([]byte, error) {
	raw, err := demoFixturesFS.ReadFile("demo_fixtures/challenges." + string(l) + ".json")
	if err == nil {
		return raw, nil
	}
	i18n.CountDocumentFallback()
	slog.DebugContext(ctx, "demo challenges : pas de fixture dans cette langue, repli", "locale", l, "fallback", i18n.Fallback)
	return demoFixturesFS.ReadFile("demo_fixtures/challenges." + string(i18n.Fallback) + ".json")
}

// demoChallenges retourne la fixture défis démo (embarquée), dans la locale de la
// requête (i18n.FromContext). Bypass live + cache. snapshot_at = maintenant (fraîcheur
// ok côté UI, pas d'indicateur "périmé"). Une fixture par langue (challenges.<code>.json) :
// sans sélection par locale, la démo afficherait les défis dans une seule langue.
func demoChallenges(ctx context.Context) domain.ChallengesResponse {
	fixture, err := demoChallengesFixture(ctx, i18n.FromContext(ctx))
	if err != nil {
		slog.WarnContext(ctx, "demo challenges fixture read failed", "err", err)
		return domain.ChallengesResponse{Available: false}
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
