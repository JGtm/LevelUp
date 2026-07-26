// Package ctxkeys fournit les clés de contexte partagées entre middleware et services.
package ctxkeys

import (
	"context"

	"levelup/go-api/internal/domain"
)

type contextKey string

const (
	titleSlugKey          contextKey = "title_slug"
	haloTokensKey         contextKey = "halo_tokens"
	haloXUIDKey           contextKey = "halo_xuid"
	tokensOwnerXUIDKey    contextKey = "tokens_owner_xuid"
	requestIDKey          contextKey = "request_id"
	eventIDKey            contextKey = "event_id"
	localeKey             contextKey = "locale"
	dbWriterLabelKey      contextKey = "db_writer_label"
	gamertagLiveSearchKey contextKey = "gamertag_live_search"
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

// WithHaloAuth place les tokens Halo et le XUID de leur PORTEUR dans le contexte.
//
// Le `xuid` passé est le PORTEUR RÉEL des tokens (compte connecté / joueur pour
// lequel ils ont été mintés). Il sert à deux titres, posés ici de façon atomique :
//
//   - haloXUIDKey : SUJET par défaut des lectures xuid-filtrées (identité Spartan,
//     etc.). Ce sujet peut être ré-écrit ensuite par WithHaloXUID (forçage page)
//     SANS toucher au porteur — voir tokensOwnerXUIDKey.
//   - tokensOwnerXUIDKey : PORTEUR des tokens, source d'attribution du budget API
//     (ratebudget.ForXUID). Immuable après pose : un forçage de sujet (page ≠
//     compte) ne doit pas dévier le débit vers le mauvais bucket (finding ID3,
//     revue 2026-07 : le quota se débite au porteur réel, pas au xuid de page).
//
// WithHaloAuth est le SEUL point de pose des tokens (haloTokensKey n'est écrit
// que par cette fonction) : poser le porteur ici garantit qu'aucun contexte
// portant des tokens n'existe sans porteur associé.
func WithHaloAuth(ctx context.Context, tokens *domain.HaloTokens, xuid string) context.Context {
	ctx = context.WithValue(ctx, haloTokensKey, tokens)
	ctx = context.WithValue(ctx, haloXUIDKey, xuid)
	return context.WithValue(ctx, tokensOwnerXUIDKey, xuid)
}

// WithHaloXUID place uniquement le SUJET (XUID) dans le contexte (sans toucher aux
// tokens NI au porteur tokensOwnerXUID). Utilisé pour les requêtes non-authentifiées
// (démo) où aucune session ne porte de tokens mais où les lectures xuid-filtrées
// (identité Spartan) doivent cibler le joueur de la page, ET pour le forçage
// d'identité de page (forcePageIdentityXUID) qui aligne le SUJET sur la page sans
// jamais réattribuer le budget API du porteur (finding ID3).
func WithHaloXUID(ctx context.Context, xuid string) context.Context {
	return context.WithValue(ctx, haloXUIDKey, xuid)
}

// WithTokensOwnerXUID place le PORTEUR des tokens sans toucher au sujet (haloXUIDKey).
// Nécessaire quand le porteur doit être préservé indépendamment du sujet posé par
// WithHaloAuth — cas du refresh background carrière, détaché du contexte requête :
// il ré-injecte les mêmes tokens (donc le même porteur) mais cible le sujet de la
// page. Sans cette correction, l'attribution du budget dévierait vers le sujet.
func WithTokensOwnerXUID(ctx context.Context, xuid string) context.Context {
	return context.WithValue(ctx, tokensOwnerXUIDKey, xuid)
}

// HaloTokens extrait les tokens Halo depuis le contexte. Retourne nil si absent.
func HaloTokens(ctx context.Context) *domain.HaloTokens {
	v, _ := ctx.Value(haloTokensKey).(*domain.HaloTokens)
	return v
}

// HaloXUID extrait le SUJET (XUID) depuis le contexte. Retourne "" si absent.
func HaloXUID(ctx context.Context) string {
	v, _ := ctx.Value(haloXUIDKey).(string)
	return v
}

// TokensOwnerXUID extrait le XUID du PORTEUR des tokens (attribution du budget API).
// Retourne "" si absent (contexte sans tokens, ou porteur inconnu). Distinct de
// HaloXUID : le sujet d'une requête peut être le joueur de la PAGE tandis que les
// tokens appartiennent au compte connecté.
func TokensOwnerXUID(ctx context.Context) string {
	v, _ := ctx.Value(tokensOwnerXUIDKey).(string)
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

// WithDBWriterLabelIfAbsent pose le label UNIQUEMENT si le contexte n'en porte
// pas déjà un. Destiné aux closures d'ACQUISITION intermédiaires (ex. le
// SharedDBAcquirer du post-sync V2) : elles doivent labelliser les call-sites
// nus, mais jamais écraser le label plus fin déjà posé par leur appelant
// (SharedAccess.Write pose "sync_v2_postsync/weapons"). Avec WithDBWriterLabel
// nu à cet endroit, TOUTE la ventilation par étape retombait sur le label
// grossier — l'attribution prod du 2026-07-26 ne distinguait plus weapons
// d'events dans les WARN watchdog.
func WithDBWriterLabelIfAbsent(ctx context.Context, label string) context.Context {
	if DBWriterLabel(ctx) != DBWriterLabelUnlabeled {
		return ctx
	}
	return WithDBWriterLabel(ctx, label)
}

// DBWriterLabel extrait le label de détenteur du writer DB partagé.
// Retourne DBWriterLabelUnlabeled si absent (call-site non instrumenté).
func DBWriterLabel(ctx context.Context) string {
	if v, ok := ctx.Value(dbWriterLabelKey).(string); ok && v != "" {
		return v
	}
	return DBWriterLabelUnlabeled
}

// WithGamertagLiveSearch arme (ou non) le REPLI LIVE de la recherche de gamertag
// (résolution Xbox d'un joueur jamais croisé). Posé par le handler à partir du
// paramètre ?live= de l'endpoint. Défaut (absent) = false : recherche LOCALE seule,
// rapide — le repli live (2-3 s bloquants) n'est armé que sur intention explicite de
// l'utilisateur (bouton « Rechercher sur Xbox »). Challenge V72-24 (latence typeahead).
func WithGamertagLiveSearch(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, gamertagLiveSearchKey, enabled)
}

// GamertagLiveSearch indique si le repli live de la recherche de gamertag est armé
// pour cette requête. Retourne false si absent (défaut : local seul).
func GamertagLiveSearch(ctx context.Context) bool {
	v, _ := ctx.Value(gamertagLiveSearchKey).(bool)
	return v
}
