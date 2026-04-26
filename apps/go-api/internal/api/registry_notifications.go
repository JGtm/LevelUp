// Package api — registry_notifications.go : factories pour le système de
// notifications in-app (per-player).
//
// Externalisé de registry.go pour respecter la limite 500L par module
// (CLAUDE.md §14).
package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
)

// notifServicesByXUID cache les *notifications.Service par xuid.
//
// Important : la monotonicité des IDs (snowflake-like) est garantie *au sein
// d'une instance de Service*. Si deux instances coexistaient pour le même
// joueur, leurs générateurs d'ID pourraient collisionner sub-milliseconde.
// On cache donc une instance unique par xuid pour la durée de vie du process.
var notifServicesByXUID sync.Map // map[string]*notifications.Service

// Notifications retourne le *notifications.Service associé au joueur identifié
// par slug. Cache process-level par xuid (cf. notifServicesByXUID).
func (r *ServiceRegistry) Notifications(ctx context.Context, slug string) (*notifications.Service, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return notifServiceFor(pdb), nil
}

// NotificationsEmitter retourne l'interface Emitter (sous-ensemble de Service)
// pour l'injection dans les hooks d'émission (sync engine, media handler, etc.).
//
// Variante minimaliste : ce qu'attend un consommateur est l'interface Emitter,
// pas le Service complet. Le compile-time check `var _ Emitter = (*Service)(nil)`
// dans le package notifications garantit la compatibilité.
func (r *ServiceRegistry) NotificationsEmitter(ctx context.Context, slug string) (notifications.Emitter, error) {
	return r.Notifications(ctx, slug)
}

// notifServiceFor construit ou retourne l'instance cachée pour ce PlayerDB.
func notifServiceFor(pdb *duckdb.PlayerDB) *notifications.Service {
	if v, ok := notifServicesByXUID.Load(pdb.XUID); ok {
		return v.(*notifications.Service)
	}
	repo := duckdb.NewNotificationsRepo(pdb)
	svc := notifications.NewService(repo)
	// LoadOrStore évite la race : si un autre goroutine a inséré
	// entre Load et Store, on retourne celle-là.
	if existing, loaded := notifServicesByXUID.LoadOrStore(pdb.XUID, svc); loaded {
		return existing.(*notifications.Service)
	}
	return svc
}

// MediaRecipientResolver retourne la liste des player_slug à notifier après un
// upload média. Stratégie : pour chaque association créée depuis `since`, on
// récupère le match_id, puis les xuids participants ; on remappe vers les
// player_slug connus dans db_profiles.json (max 5 destinataires distincts).
//
// L'uploader est exclu (passé en `uploaderSlug`).
func (r *ServiceRegistry) MediaRecipientResolver(cfg *config.AppConfig) func(
	context.Context, string, string, string, time.Time,
) ([]string, error) {
	return func(
		ctx context.Context,
		uploaderSlug, sharedSocialDBPath, sharedMatchesDBPath string,
		since time.Time,
	) ([]string, error) {
		if sharedSocialDBPath == "" || sharedMatchesDBPath == "" {
			return nil, nil
		}
		matchIDs, err := loadRecentMediaMatchIDs(ctx, sharedSocialDBPath, since)
		if err != nil {
			return nil, fmt.Errorf("recent media associations: %w", err)
		}
		if len(matchIDs) == 0 {
			return nil, nil
		}
		xuids, err := loadParticipantXUIDs(ctx, sharedMatchesDBPath, matchIDs)
		if err != nil {
			return nil, fmt.Errorf("participants xuids: %w", err)
		}
		return xuidsToSlugs(cfg, uploaderSlug, xuids, 5), nil
	}
}

// loadRecentMediaMatchIDs : SELECT DISTINCT match_id depuis shared_social.media_match_associations
// où created_at >= since. Plafonné implicitement par le nombre d'uploads concomitants.
func loadRecentMediaMatchIDs(ctx context.Context, sharedSocialDBPath string, since time.Time) ([]string, error) {
	db, err := duckdb.OpenReadOnly(sharedSocialDBPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(ctx,
		`SELECT DISTINCT match_id FROM media_match_associations WHERE created_at >= ? LIMIT 50`,
		since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// loadParticipantXUIDs : SELECT DISTINCT xuid FROM shared.match_participants WHERE match_id IN (...).
func loadParticipantXUIDs(ctx context.Context, sharedMatchesDBPath string, matchIDs []string) ([]string, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	db, err := duckdb.OpenReadOnly(sharedMatchesDBPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	placeholders := make([]string, len(matchIDs))
	args := make([]any, len(matchIDs))
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	q := fmt.Sprintf(
		`SELECT DISTINCT xuid FROM match_participants WHERE match_id IN (%s) LIMIT 50`,
		joinComma(placeholders),
	)
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// xuidsToSlugs remappe une liste de xuids vers les player_slug connus dans
// db_profiles.json, exclut l'uploader, déduplique, plafonne à `max`.
func xuidsToSlugs(cfg *config.AppConfig, uploaderSlug string, xuids []string, max int) []string {
	xuidSet := make(map[string]struct{}, len(xuids))
	for _, x := range xuids {
		xuidSet[x] = struct{}{}
	}
	players, err := cfg.LoadPlayers()
	if err != nil {
		return nil
	}
	seen := map[string]struct{}{uploaderSlug: {}}
	out := []string{}
	for _, p := range players {
		if _, ok := xuidSet[p.XUID]; !ok {
			continue
		}
		if _, dup := seen[p.PlayerSlug]; dup {
			continue
		}
		seen[p.PlayerSlug] = struct{}{}
		out = append(out, p.PlayerSlug)
		if len(out) >= max {
			break
		}
	}
	return out
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ","
		}
		out += p
	}
	return out
}
