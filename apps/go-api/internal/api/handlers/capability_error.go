// Package handlers — capability_error.go : mapping HTTP central pour
// games.ErrCapabilityNotSupported (Lot B, audit robustesse + CLAUDE.md règle #6).
//
// Un titre qui n'expose pas une capability (ADR 0011) doit répondre 503 propre,
// jamais 500. Ce helper est la SOURCE UNIQUE de cette traduction côté handlers
// Huma ; le garde-rail no_capability_error_dup_test.go interdit toute copie.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability"
)

// MapCapabilityError traduit games.ErrCapabilityNotSupported en une erreur HTTP
// 503 (log Warn + compteur titre-aware), et retourne (nil, false) sinon — le
// caller mappe alors le reste (typiquement un 500 générique). `probe` identifie
// la surface pour le diagnostic (ex. "match.detail.events", "squad.page").
func MapCapabilityError(ctx context.Context, err error, probe string) (error, bool) {
	if errors.Is(err, games.ErrCapabilityNotSupported) {
		slog.WarnContext(ctx, "capability non supportée par le titre",
			"probe", probe, "titleSlug", ctxkeys.TitleSlug(ctx), "err", err)
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "http_capability_not_supported_total")
		return humacore.NewError(http.StatusServiceUnavailable, "capability_not_supported", err.Error()), true
	}
	return nil, false
}
