package wire

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// Sessions retourne un SessionsService pour le joueur, branché sur la règle de découpage de
// SON titre et sur SES amis : GET /pages/sessions découpe alors comme les écrivains de
// session_id (synchro, recalcul complet).
func (r *ServiceRegistry) Sessions(ctx context.Context, slug string) (port.SessionsService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	svc := service.NewSessionsService(duckdb.NewSessionsRepo(pdb)).
		WithSessionRule(r.sessionRuleFor(pdb.TitleSlug), r.friendGamertagsResolver(pdb.XUID))
	return svc, nil
}

// sessionRuleFor rend la règle de découpage d'un titre depuis le store de réglages du
// registre. Store ou configuration absents : settings.SessionOptions applique la règle par
// défaut et le journalise.
func (r *ServiceRegistry) sessionRuleFor(titleSlug string) func(context.Context) domain.SessionComputeOptions {
	overlay := ""
	if r.cfg != nil {
		overlay = r.cfg.TitleSettingsPath(titleSlug)
	}
	return settings_platform.SessionOptionsLoader(r.settingsStore, overlay)
}
