// Package ctxkeys fournit les clés de contexte partagées entre middleware et services.
package ctxkeys

import (
	"context"

	"levelup/go-api/internal/domain"
)

type contextKey string

const (
	titleSlugKey      contextKey = "title_slug"
	haloTokensKey     contextKey = "halo_tokens"
	haloXUIDKey       contextKey = "halo_xuid"
	viewerGamertagKey contextKey = "viewer_gamertag"
	requestIDKey      contextKey = "request_id"
	eventIDKey        contextKey = "event_id"
	localeKey         contextKey = "locale"
	dbWriterLabelKey  contextKey = "db_writer_label"
)

// WithLocale place la locale UI ("fr"/"en") dans le contexte. Utilisée par les
// services qui localisent des libellés (médailles, etc.) sans avoir à threader
// la locale dans chaque signature.
func WithLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, localeKey, locale)
}

// Locale extrait la locale depuis le contexte. Retourne "fr" si absente.
func Locale(ctx context.Context) string {
	if v, ok := ctx.Value(localeKey).(string); ok && v != "" {
		return v
	}
	return "fr"
}

// WithTitleSlug place le slug du titre dans le contexte.
func WithTitleSlug(ctx context.Context, slug string) context.Context {
	return context.WithValue(ctx, titleSlugKey, slug)
}

// TitleSlug extrait le slug du titre depuis le contexte.
// Retourne "halo_infinite" si absent (rétrocompatibilité).
func TitleSlug(ctx context.Context) string {
	if v, ok := ctx.Value(titleSlugKey).(string); ok && v != "" {
		return v
	}
	return "halo_infinite"
}

// TitleSlugIfSet retourne le slug du titre ET true SEULEMENT s'il est réellement
// présent dans le contexte — sans forcer le fallback "halo_infinite" (MT-05).
// À utiliser quand l'absence doit rester distinguable de halo_infinite (ex.
// observabilité : ne pas polluer les logs background avec un titre fantôme).
func TitleSlugIfSet(ctx context.Context) (string, bool) {
	if v, ok := ctx.Value(titleSlugKey).(string); ok && v != "" {
		return v, true
	}
	return "", false
}

// WithHaloAuth place les tokens Halo et le XUID du joueur connecté dans le contexte.
func WithHaloAuth(ctx context.Context, tokens *domain.HaloTokens, xuid string) context.Context {
	ctx = context.WithValue(ctx, haloTokensKey, tokens)
	return context.WithValue(ctx, haloXUIDKey, xuid)
}

// WithHaloXUID place uniquement le XUID dans le contexte (sans toucher aux
// tokens). Utilisé pour les requêtes non-authentifiées (démo) où aucune session
// ne porte de tokens mais où les lectures xuid-filtrées (identité Spartan) doivent
// cibler le joueur de la page.
func WithHaloXUID(ctx context.Context, xuid string) context.Context {
	return context.WithValue(ctx, haloXUIDKey, xuid)
}

// HaloTokens extrait les tokens Halo depuis le contexte. Retourne nil si absent.
func HaloTokens(ctx context.Context) *domain.HaloTokens {
	v, _ := ctx.Value(haloTokensKey).(*domain.HaloTokens)
	return v
}

// HaloXUID extrait le XUID depuis le contexte. Retourne "" si absent.
func HaloXUID(ctx context.Context) string {
	v, _ := ctx.Value(haloXUIDKey).(string)
	return v
}

// WithViewerGamertag place le GAMERTAG du joueur de la page (le "viewer") dans le
// contexte. Indispensable aux titres GAMERTAG-keyés (Halo 5) dont l'API match ne
// porte aucun xuid (Player.Xuid toujours null) : pour résoudre le MODE de la
// carnage + les refs header (map/playlist/startTime/isRanked) d'un match, l'adapter
// doit retrouver l'entrée de l'historique du joueur via GetPlayerMatches(gamertag).
// Or les signatures canoniques ID-keyées (LoadMatchDetail(ctx, matchID)) ne portent
// pas de viewer ; le contexte est donc le seul canal propre, à l'image du
// SpartanToken (ctxkeys.HaloTokens).
//
// Câblage : le registry/handler de la route /players/{slug}/... connaît le
// gamertag du joueur (pdb.Gamertag) et le pose ici au moment de construire le
// service. Title-agnostic : un titre xuid-keyé (Halo Infinite) ignore cette clé.
func WithViewerGamertag(ctx context.Context, gamertag string) context.Context {
	return context.WithValue(ctx, viewerGamertagKey, gamertag)
}

// ViewerGamertag extrait le gamertag du joueur de la page depuis le contexte.
// Retourne "" si absent (les titres xuid-keyés n'en ont pas besoin ; les titres
// gamertag-keyés dégradent gracieusement quand il manque).
func ViewerGamertag(ctx context.Context) string {
	v, _ := ctx.Value(viewerGamertagKey).(string)
	return v
}

// WithRequestID place l'identifiant de requête dans le contexte.
// P6.4 (revue 2026-04-29 axe 8 BLOQUANT) : permet de corréler une ligne
// d'accès middleware (`request_id` log) avec les `slog.*Context` émis par
// les services pour la même requête. Sans ça, debug prod cassé en multi-user.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID extrait l'identifiant de requête depuis le contexte.
// Retourne "" si absent (cas non-HTTP : background jobs, sync tasks, tests).
func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

// WithEventID place un identifiant d'événement multi-module dans le contexte.
// Sprint B1 commit 16 : permet de corréler un événement business (sync,
// swap RW, backfill, recompute) à travers les fichiers logs/{module}.log.
//
// Contrairement à request_id (par requête HTTP), event_id couvre des
// opérations background longues qui n'ont pas de request HTTP associée.
// Les deux peuvent coexister : une requête HTTP qui déclenche un sync
// aura request_id (court) + event_id (sync.RunDelta:abc123).
//
// Préférer logging.WithEvent(ctx, prefix) qui génère l'id automatiquement.
func WithEventID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, eventIDKey, id)
}

// EventID extrait l'identifiant d'événement depuis le contexte.
// Retourne "" si absent (logs hors d'une opération taguée WithEvent).
func EventID(ctx context.Context) string {
	v, _ := ctx.Value(eventIDKey).(string)
	return v
}

// DBWriterLabelUnlabeled est le label par défaut d'une acquisition de writer
// DB partagée dont le caller n'a pas posé de label — sa présence dans les
// métriques signale un call-site à instrumenter, pas une catégorie métier.
const DBWriterLabelUnlabeled = "unlabeled"

// WithDBWriterLabel place le label du DÉTENTEUR d'un writer DB partagé dans le
// contexte (attribution de la fenêtre RW, étape 0 contention). Posé par chaque
// sous-système à l'ENTRÉE de son acquisition (sync_v1_run, sync_v2_postsync,
// persist_worker, world_leaderboard_snapshot, backfill_*, …), lu par
// sharedprovider.AcquireWriter pour ventiler shared_provider_rw_window_ms par
// détenteur et étiqueter le watchdog. Même classe de donnée cross-cutting
// qu'event_id : capturée à l'acquisition, consommée au Release.
//
// Labels = constantes compile-time UNIQUEMENT (cardinalité bornée, ADR 0009 —
// jamais de gamertag/match_id/chemin dans un label).
func WithDBWriterLabel(ctx context.Context, label string) context.Context {
	if label == "" {
		return ctx
	}
	return context.WithValue(ctx, dbWriterLabelKey, label)
}

// DBWriterLabel extrait le label de détenteur du writer DB partagé.
// Retourne DBWriterLabelUnlabeled si absent (call-site non instrumenté).
func DBWriterLabel(ctx context.Context) string {
	if v, ok := ctx.Value(dbWriterLabelKey).(string); ok && v != "" {
		return v
	}
	return DBWriterLabelUnlabeled
}
