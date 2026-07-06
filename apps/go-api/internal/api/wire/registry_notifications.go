// Package api — registry_notifications.go : factories pour le système de
// notifications in-app (per-player).
//
// Externalisé de registry.go pour respecter la limite 500L par module
// (CLAUDE.md §14).
package wire

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/dblease"
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
//
// Le service est configuré avec un WriterAcquirer qui acquiert un lease sur
// shared_social.duckdb avant chaque méthode write — sérialise les écritures
// avec le sync engine et les autres handlers (commit 4 du refactor
// leased-writer-enforcement). Si shared_social n'est pas disponible (boot
// initial), l'acquireur retourne ErrSharedSocialUnavailable que le service
// remonte tel quel — comportement préexistant.
func notifServiceFor(pdb *duckdb.PlayerDB) *notifications.Service {
	if v, ok := notifServicesByXUID.Load(pdb.XUID); ok {
		return v.(*notifications.Service)
	}
	repo := duckdb.NewNotificationsRepo(pdb)
	acquirer := func() (*dblease.LeasedWriter, error) {
		return pdb.AcquireSharedSocialWriterTimeout(dblease.SharedLeaseTimeout)
	}
	svc := notifications.NewService(repo, notifications.WithWriterAcquirer(acquirer))
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
			slog.WarnContext(ctx, "media_recipients: load match associations",
				"uploader", uploaderSlug, "since", since, "err", err)
			return nil, fmt.Errorf("recent media associations: %w", err)
		}
		if len(matchIDs) == 0 {
			slog.DebugContext(ctx, "media_recipients: no recent associations",
				"uploader", uploaderSlug, "since", since)
			return nil, nil
		}
		xuids, err := loadParticipantXUIDs(ctx, sharedMatchesDBPath, matchIDs)
		if err != nil {
			slog.WarnContext(ctx, "media_recipients: load participants",
				"match_count", len(matchIDs), "err", err)
			return nil, fmt.Errorf("participants xuids: %w", err)
		}
		recipients := xuidsToSlugs(cfg, uploaderSlug, xuids, 5)
		slog.InfoContext(ctx, "media_recipients: resolved",
			"uploader", uploaderSlug,
			"matches", len(matchIDs),
			"xuids", len(xuids),
			"recipients", len(recipients))
		return recipients, nil
	}
}

// loadRecentMediaMatchIDs : SELECT DISTINCT match_id depuis shared_social.media_match_associations
// où created_at >= since. Plafonné implicitement par le nombre d'uploads concomitants.
//
// IMPORTANT : OpenReadWriteShared (et NON OpenReadOnly) — shared_social.duckdb
// est déjà ouverte en mode RW partagé par le pool joueur (pool.go) et par
// prestige_setup au boot. Un OpenReadOnly ici créerait une 2e clé cache
// ("ro:"+path) → 2 duckdb_open() avec des access_mode différents sur le même
// fichier → erreur "Can't open a connection to same database file with a
// different configuration". Cf. fix similaire commit 9c.5 sur shared_matches_v2.
func loadRecentMediaMatchIDs(ctx context.Context, sharedSocialDBPath string, since time.Time) ([]string, error) {
	db, err := duckdb.OpenReadWriteShared(sharedSocialDBPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(ctx,
		// APPEND-ONLY : lire la vue _latest + associated_at (immuable) — pas created_at
		// sur la table legacy. Préserve la fenêtre de récence d'upload sans faux positifs.
		`SELECT DISTINCT match_id FROM media_match_associations_latest WHERE associated_at >= ? LIMIT 50`,
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
	// OpenReadForQuery réutilise le handle en cache (RO ou RW) s'il existe et ne
	// retombe sur OpenReadOnly que si rien n'est en cache → évite l'échec DuckDB
	// "different configuration" quand le shared est déjà tenu RW ailleurs
	// (incident 2026-06-01, cf. db.go). Fan-out best-effort.
	sqlDB, release, err := duckdb.OpenReadForQuery(sharedMatchesDBPath)
	if err != nil {
		return nil, err
	}
	defer release()
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
	rows, err := sqlDB.QueryContext(ctx, q, args...)
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
