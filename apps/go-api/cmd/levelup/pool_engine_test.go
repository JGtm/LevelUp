package main

import (
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	auth_pool "levelup/go-api/internal/platform/auth/pool"
)

// poolFactice satisfait auth_pool.Pool sans jamais servir : le test ne fait que construire le
// moteur, aucun appel réseau n'a lieu.
type poolFactice struct{ auth_pool.Pool }

// TestNewPooledEngine_BrancheLaResolutionDesNoms : le moteur poolé de la CLI (sync-delta,
// sync-full, mono-joueur ou --all) résout les noms d'assets par le pool, comme le
// planificateur. Sans ce branchement, les matchs d'un asset non encore traduit entrent au
// registre avec l'identifiant pour nom.
func TestNewPooledEngine_BrancheLaResolutionDesNoms(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	player := domain.PlayerSummary{Gamertag: "Joueur", XUID: "2533274000000777"}

	engine := newPooledEngine(cfg, nil, poolFactice{}, player)
	if !engine.ResolvesAssetNames() {
		t.Fatal("newPooledEngine : résolution des noms d'assets non branchée (WithAssetNameResolution manquant)")
	}
}
