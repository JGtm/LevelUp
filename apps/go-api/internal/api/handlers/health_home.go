// Package handlers — health_home.go : smoke endpoint /healthz/home.
//
// Sonde de santé orientée *contenu* de la home page Mission Control. Inspecte
// les 5 sections critiques (banner Spartan, peaks CSR/LUSR, playlists récentes,
// arme favorite) et rapporte celles qui sont vides sans raison.
//
// But : transformer une régression silencieuse (UI affiche "—" ou "Aucune
// partie classée" alors que la donnée existe) en alerte HTTP 503 visible.
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
//
// Utilisation typique : appelé après chaque backfill, dans la CI smoke step,
// et optionnellement consommé par le frontend pour afficher une bannière
// d'alerte dev quand une section est vide.
package handlers

import (
	"log/slog"
	"net/http"

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

// Check répond à GET /api/v1/healthz/home?player=<slug>.
//
//nolint:funlen // checklist explicite section-par-section pour lisibilité
func (h *HealthHomeHandler) Check(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("player")
	if slug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_player",
			"Le paramètre 'player' est obligatoire. Exemple : /api/v1/healthz/home?player=jgtm")
		return
	}

	svc, ctx, _, gamertag, err := h.newSvc(r.Context(), slug)
	if err != nil {
		slog.WarnContext(r.Context(), "healthz/home: newSvc failed", "slug", slug, "err", err)
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found",
			"Joueur introuvable dans db_profiles.json")
		return
	}

	page, err := svc.GetHomePage(ctx, gamertag, "fr")
	if err != nil {
		slog.ErrorContext(ctx, "healthz/home: GetHomePage failed", "err", err, "gamertag", gamertag)
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"ok":             false,
			"player":         gamertag,
			"error":          err.Error(),
			"checks":         map[string]string{"home_page": "err: " + err.Error()},
			"empty_sections": []string{"home_page"},
		})
		return
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
	writeJSON(w, status, map[string]any{
		"ok":             len(emptySections) == 0,
		"player":         gamertag,
		"checks":         checks,
		"empty_sections": emptySections,
	})
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
