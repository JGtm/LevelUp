// Package handlers — health_home.go : smoke endpoint /healthz/home.
//
// Sonde de santé orientée *contenu* de la home page Mission Control. Inspecte
// les 5 sections critiques (banner Spartan, peaks CSR/LUSR, playlists récentes,
// arme favorite) et rapporte celles qui sont vides sans raison.
//
// But : transformer une régression silencieuse (UI affiche "—" ou "Aucune
// partie classée" alors que la donnée existe) en alerte HTTP 503 visible.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) au point de montage
// /api/v1 et enregistre le GET via huma.Get. Logique métier inchangée
// (HomeAuthFactory + GetHomePage), seul le wrapping HTTP change. Le param
// `player` reste un query param (pas un path param) ; le 503 « section vide »
// est un corps de DIAGNOSTIC (Output{Status:503, Body}) et non une erreur Huma.
//
// Contrat :
//   - GET /api/v1/healthz/home?player=<slug>
//   - Le param `player` est obligatoire (pas de auto-pick : la home dépend
//     du joueur, l'opérateur doit choisir qui sonder).
//   - 200 OK + {ok: true, player, checks: {section: "ok"}} si tout OK
//   - 503 + {ok: false, player, checks, empty_sections: [...]} si une ou
//     plusieurs sections sont vides sans raison (= régression)
//   - 404 si player_slug inconnu
//   - 500 si GetHomePage panique
package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
)

// HealthHomeHandler enveloppe une HomeAuthFactory pour exposer le smoke check.
// Réutilise la factory existante (HomeCtxWithAuth) — aucun nouveau pool ou
// repository à provisionner.
type HealthHomeHandler struct {
	newSvc HomeAuthFactory
}

// NewHealthHomeHandler crée un HealthHomeHandler.
func NewHealthHomeHandler(newSvc HomeAuthFactory) *HealthHomeHandler {
	return &HealthHomeHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma au point de montage chi (préfixe /api/v1 +
// middleware racine hérités). Le chemin relatif /healthz/home est identique à la
// route chi d'origine.
func (h *HealthHomeHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/healthz/home", h.handleCheck, humacore.Op("getHealthzHome", "Smoke endpoint contenu home (banner, peaks CSR/LUSR, playlists, arme favorite)", "health"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// healthHomeInput : param `player` en QUERY (pas un path param) — la home dépend
// du joueur, mais le chemin /healthz/home n'en porte pas le slug.
type healthHomeInput struct {
	Player string `query:"player"`
}

// healthHomeOutput : 200 OK (tout peuplé) OU 503 (sections vides / GetHomePage en
// erreur). Le 503 est un corps de DIAGNOSTIC (pas une erreur Huma) → Status
// explicite + Body map. Le 200 doit aussi fixer Status (le champ Status override
// le défaut Huma, sinon une réponse sans Status renverrait 0/204).
type healthHomeOutput struct {
	Status int
	Body   any
}

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handleCheck répond à GET /api/v1/healthz/home?player=<slug>.
//
//nolint:funlen // checklist explicite section-par-section pour lisibilité
func (h *HealthHomeHandler) handleCheck(ctx context.Context, in *healthHomeInput) (*healthHomeOutput, error) {
	slug := in.Player
	if slug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_player",
			"Le paramètre 'player' est obligatoire. Exemple : /api/v1/healthz/home?player=jgtm")
	}

	svc, svcCtx, _, gamertag, err := h.newSvc(ctx, slug)
	if err != nil {
		slog.WarnContext(ctx, "healthz/home: newSvc failed", "slug", slug, "err", err)
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found",
			"Joueur introuvable dans db_profiles.json")
	}

	page, err := svc.GetHomePage(svcCtx, gamertag, "fr")
	if err != nil {
		slog.ErrorContext(svcCtx, "healthz/home: GetHomePage failed", "err", err, "gamertag", gamertag)
		return &healthHomeOutput{Status: http.StatusServiceUnavailable, Body: map[string]any{
			"ok":             false,
			"player":         gamertag,
			"error":          err.Error(),
			"checks":         map[string]string{"home_page": "err: " + err.Error()},
			"empty_sections": []string{"home_page"},
		}}, nil
	}

	checks := map[string]string{}
	emptySections := []string{}

	// 1. Banner Spartan identity (dynamique, depuis career_progression).
	if page.SpartanIdentity != nil && page.SpartanIdentity.BannerImageURL != nil && *page.SpartanIdentity.BannerImageURL != "" {
		checks["banner"] = "ok"
	} else {
		checks["banner"] = "missing (career_progression.banner_image_url vide)"
		emptySections = append(emptySections, "banner")
	}

	// 2. Highest CSR — peak rating OU placement avec badge unranked_N.png.
	if page.SpartanIdentity != nil && page.SpartanIdentity.HighestCSR != nil {
		checks["highest_csr"] = describePeak(page.SpartanIdentity.HighestCSR)
	} else {
		checks["highest_csr"] = "missing (match_skill_rank CSR vide + player_csr_snapshots sans alltime)"
		emptySections = append(emptySections, "highest_csr")
	}

	// 3. Highest LUSR — même contrat que CSR.
	if page.SpartanIdentity != nil && page.SpartanIdentity.HighestLUSR != nil {
		checks["highest_lusr"] = describePeak(page.SpartanIdentity.HighestLUSR)
	} else {
		checks["highest_lusr"] = "missing (match_skill_rank LUSR vide)"
		emptySections = append(emptySections, "highest_lusr")
	}

	// 4. Playlists récentes : len ≥ 1 attendu pour un joueur actif.
	if len(page.RecentPlaylistRanks) > 0 {
		checks["recent_playlists"] = "ok"
	} else {
		checks["recent_playlists"] = "missing (shared.match_registry.playlist_id vide ou Q26g trop strict)"
		emptySections = append(emptySections, "recent_playlists")
	}

	// 5. Arme favorite : nom non-vide attendu pour un joueur avec kills.
	if page.Hero.KPIs.FavoriteWeaponName != "" {
		checks["favorite_weapon"] = "ok"
	} else {
		checks["favorite_weapon"] = "missing (shared.v_weapon_kills vide ou label metadata absent)"
		emptySections = append(emptySections, "favorite_weapon")
	}

	status := http.StatusOK
	if len(emptySections) > 0 {
		status = http.StatusServiceUnavailable
	}
	return &healthHomeOutput{Status: status, Body: map[string]any{
		"ok":             len(emptySections) == 0,
		"player":         gamertag,
		"checks":         checks,
		"empty_sections": emptySections,
	}}, nil
}

// describePeak retourne une chaîne courte décrivant l'état d'un peak.
func describePeak(p *domain.HomeSkillPeakSummary) string {
	if p == nil {
		return "missing"
	}
	if p.MeasurementMatchesRemaining != nil && *p.MeasurementMatchesRemaining > 0 {
		return "ok (placement)"
	}
	return "ok"
}
