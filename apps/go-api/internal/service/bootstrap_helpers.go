// Package service — bootstrap_helpers.go : état d'authentification, identité liée, réglages et titres
// disponibles de /bootstrap.
package service

import (
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
)

// resolveUsername retourne le username de la session (ou nil).
func resolveUsername(sess *domain.SessionData) *string {
	if sess == nil {
		return nil
	}
	return sess.Username
}

// ResolveAuthState déduit l'état d'authentification depuis la session.
func ResolveAuthState(sess *domain.SessionData) string {
	if sess == nil || !sess.AuthReady {
		return bootstrapAuthMissing
	}
	if sess.LinkedHaloIdentity == nil || sess.LinkedHaloIdentity.Gamertag == "" {
		return "partial"
	}
	return bootstrapSetupReady
}

// ResolveLinkedIdentity extrait l'identité Halo liée si présente dans la session.
func ResolveLinkedIdentity(sess *domain.SessionData) *domain.HaloIdentitySummary {
	if sess == nil || sess.LinkedHaloIdentity == nil {
		return nil
	}
	return &domain.HaloIdentitySummary{
		Gamertag: sess.LinkedHaloIdentity.Gamertag,
		XUID:     sess.LinkedHaloIdentity.XUID,
	}
}

func getBoolSetting(settings map[string]interface{}, key string, def bool) bool {
	if v, ok := settings[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

func getStringSetting(settings map[string]interface{}, key, def string) string {
	if v, ok := settings[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return def
}

// BuildAvailableTitles construit la liste des titres depuis le registre PARTAGÉ
// (MT-16 / day-one 2e titre : DefaultRegistry inclut les titres découverts en
// config). Sprint 49 : exportée pour réutilisation par le handler session_context.
func BuildAvailableTitles() []domain.TitleSummary {
	return buildAvailableTitlesFrom(titlePkg.DefaultRegistry())
}

// buildAvailableTitlesFrom projette les titres servables d'un registre donné.
// Extrait pour testabilité (registre injectable avec coming_soon/archived).
func buildAvailableTitlesFrom(reg *titlePkg.Registry) []domain.TitleSummary {
	// MT-22 (PMT-8) : le switcher liste les titres jouables (active) ET ceux
	// « bientôt disponibles » (coming_soon), en conservant leur Status pour que
	// le front affiche l'état. Exclus : les retirés (archived) ET les internes
	// (fixtures de test comme synthetic_title_b — cf. IsInternal). Tous deux
	// restent inspectables côté admin (/admin/titles via reg.All/NonArchived).
	all := reg.PublicTitles()
	out := make([]domain.TitleSummary, 0, len(all))
	for _, t := range all {
		caps := make([]string, len(t.Capabilities))
		for i, c := range t.Capabilities {
			caps[i] = string(c)
		}
		out = append(out, domain.TitleSummary{
			Slug:                    t.Slug,
			Name:                    t.Name,
			IconURL:                 t.IconURL,
			Status:                  string(t.Status),
			Capabilities:            caps,
			IsDefault:               t.IsDefault,
			EffectiveHpToKill:       games.EffectiveHpToKill(t.Slug),
			OffensiveConversionP80:  games.OffensiveConversionP80(t.Slug),
			DefensiveResistanceP80:  games.DefensiveResistanceP80(t.Slug),
			ProvidesDamageTaken:     games.ProvidesDamageTaken(t.Slug),
			ProvidesTeamMMR:         games.ProvidesTeamMMR(t.Slug),
			ProvidesMaxKillingSpree: games.ProvidesMaxKillingSpree(t.Slug),
		})
	}
	return out
}
