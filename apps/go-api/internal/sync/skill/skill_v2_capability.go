package skill

// skill_v2_capability.go — Sprint 3.C : gate du LUSR (v1 + v2) sur la capability
// title.CapLUSR au lieu d'un couplage implicite à halo_infinite.
//
// Règle projet (skill arch-rules) : aucune feature ne doit brancher sur
// `slug == "halo_infinite"`. Le LUSR est une capacité titre déclarée dans le
// registry (internal/domain/title) ; tout site qui calcule/écrit un rating LUSR
// vérifie la capability AVANT de tourner et dégrade silencieusement (pas de
// panic, pas d'erreur) si le titre courant ne la déclare pas.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain/title"
)

// SlugHasLUSR retourne true si le slug (défaut halo_infinite si vide) déclare
// CapLUSR dans le registre PARTAGÉ runtime (title.DefaultRegistry), peuplé au boot
// depuis la config via SetDefaultRegistry. C'est désormais title-aware pour TOUS
// les titres actifs : Halo 5 déclare la coarse cap "lusr" (title.toml), donc le
// LUSR v2 (données basiques) tourne pour lui. Auparavant on consultait une copie
// STATIQUE Infinite-only (NewRegistry) → tout titre non-Infinite était skippé même
// avec CapLUSR. Slug inconnu → false (le LUSR ne tourne pas pour ce titre).
func SlugHasLUSR(slug string) bool {
	if slug == "" {
		slug = title.DefaultSlug
	}
	desc := title.DefaultRegistry().Get(slug)
	return desc != nil && desc.HasCapability(title.CapLUSR)
}

// titleHasLUSR : variante contexte. CLI/background (context.Background) →
// ctxkeys.TitleSlug défaut "halo_infinite" → CapLUSR OK (comportement inchangé).
func titleHasLUSR(ctx context.Context) bool {
	return SlugHasLUSR(ctxkeys.TitleSlug(ctx))
}

// skipIfNoLUSRCapability logge un DEBUG et retourne true si le titre courant ne
// supporte pas le LUSR — à appeler en garde au début de tout chemin LUSR.
func skipIfNoLUSRCapability(ctx context.Context, caller string) bool {
	if titleHasLUSR(ctx) {
		return false
	}
	slog.DebugContext(ctx, "LUSR skip — capability absente",
		"title_slug", ctxkeys.TitleSlug(ctx), "caller", caller)
	return true
}
