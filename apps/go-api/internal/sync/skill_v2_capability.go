package sync

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

// lusrCapRegistry : registre par défaut, source de vérité des capabilities par
// titre. halo_infinite déclare CapLUSR (cf. title.NewRegistry) ; un futur titre
// qui ne la déclare pas → LUSR no-op pour ce titre, sans toucher ce code.
//
// On instancie un registre dédié plutôt que d'en injecter un dans le SyncEngine :
// les capabilities sont statiques (déclarées en dur dans NewRegistry), donc une
// copie locale est cohérente avec le registre du serveur et évite de threader
// une dépendance à travers toute la chaîne sync. Si un jour les capabilities
// deviennent dynamiques (par DB/config), il faudra injecter le registre runtime.
var lusrCapRegistry = title.NewRegistry()

// slugHasLUSR retourne true si le slug (défaut halo_infinite si vide) déclare
// CapLUSR. Slug inconnu → false (le LUSR ne tourne pas pour ce titre).
func slugHasLUSR(slug string) bool {
	if slug == "" {
		slug = title.DefaultSlug
	}
	desc := lusrCapRegistry.Get(slug)
	return desc != nil && desc.HasCapability(title.CapLUSR)
}

// titleHasLUSR : variante contexte. CLI/background (context.Background) →
// ctxkeys.TitleSlug défaut "halo_infinite" → CapLUSR OK (comportement inchangé).
func titleHasLUSR(ctx context.Context) bool {
	return slugHasLUSR(ctxkeys.TitleSlug(ctx))
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
