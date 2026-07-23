/**
 * appearanceDiagDisplay — logique PURE d'affichage du diagnostic apparence
 * Spartan ID (Lot G du plan .ai/PLAN_DIAG_APPARENCE_ADMIN_2026-07.md).
 *
 * Mappe les clés STABLES du DTO Go (verdict / detail / component / served_from /
 * last_fetch_status) vers la sémantique UI : statut du badge canonique
 * (StatusBadge → token de couleur) + clés i18n `admin.appearance.*`. Aucun rendu
 * React ici — testable sans DOM (canon admin : *Display.ts + *Display.test.ts).
 *
 * Sémantique verdict → token (via AdminStatus) : ok=success, transient=warning,
 * auth_required=destructive, upstream_missing/not_supported=neutre. upstream_missing
 * n'est PAS une erreur (« rien à faire, servi par design ») ; not_supported n'est
 * JAMAIS un faux « cassé ».
 */
import type { AdminManifestKey } from '@/lib/i18n/generated/admin'
import type { AdminStatus } from '../statusDisplay'

/** Verdicts du DTO — enum FERMÉ (5 valeurs, cf. haloclient.Verdict). */
export type AppearanceVerdict =
  | 'ok'
  | 'upstream_missing'
  | 'transient'
  | 'auth_required'
  | 'not_supported'

/** Composants diagnostiqués (clés stables domain.AppearanceComponent*). */
export type AppearanceComponentKey = 'banner' | 'emblem' | 'backdrop' | 'service_tag'

/** Nature de l'action à mener (pilote le CTA côté composant). */
export type AppearanceActionKind = 'none' | 'wait' | 'reauth'

/**
 * Statut de badge (StatusBadge) pour un verdict. Le StatusBadge dérive le token
 * de couleur du statut : ok→success, warning→warning, error→destructive,
 * idle→neutre (muted). upstream_missing et not_supported → neutre (idle) : ni une
 * erreur, ni un faux « cassé ». Ils ne co-occurrent jamais dans un même
 * diagnostic (not_supported = titre entier H5 ; upstream_missing = bannière
 * Infinite), leur libellé les distingue.
 */
export function verdictBadgeStatus(verdict: string): AdminStatus {
  switch (verdict) {
    case 'ok':
      return 'ok'
    case 'transient':
      return 'warning'
    case 'auth_required':
      return 'error'
    case 'upstream_missing':
    case 'not_supported':
      return 'idle'
    default:
      return 'idle'
  }
}

/** Libellé du badge de verdict. */
export function verdictLabelKey(verdict: string): AdminManifestKey {
  switch (verdict) {
    case 'ok':
      return 'admin.appearance.verdict.ok'
    case 'upstream_missing':
      return 'admin.appearance.verdict.upstream_missing'
    case 'transient':
      return 'admin.appearance.verdict.transient'
    case 'auth_required':
      return 'admin.appearance.verdict.auth_required'
    case 'not_supported':
      return 'admin.appearance.verdict.not_supported'
    default:
      return 'admin.appearance.verdict.unknown'
  }
}

/** Nature de l'action « quoi faire » : rien / attendre / réauthentifier. */
export function verdictActionKind(verdict: string): AppearanceActionKind {
  switch (verdict) {
    case 'transient':
      return 'wait'
    case 'auth_required':
      return 'reauth'
    default:
      // ok / upstream_missing / not_supported / inconnu → rien à faire.
      return 'none'
  }
}

/** Texte « quoi faire » associé au verdict. */
export function verdictActionKey(verdict: string): AdminManifestKey {
  switch (verdict) {
    case 'ok':
      return 'admin.appearance.action.ok'
    case 'upstream_missing':
      return 'admin.appearance.action.upstream_missing'
    case 'transient':
      return 'admin.appearance.action.transient'
    case 'auth_required':
      return 'admin.appearance.action.auth_required'
    case 'not_supported':
      return 'admin.appearance.action.not_supported'
    default:
      return 'admin.appearance.action.unknown'
  }
}

/** Libellé humain d'un composant du Spartan ID. */
export function componentLabelKey(component: string): AdminManifestKey {
  switch (component) {
    case 'banner':
      return 'admin.appearance.component.banner'
    case 'emblem':
      return 'admin.appearance.component.emblem'
    case 'backdrop':
      return 'admin.appearance.component.backdrop'
    case 'service_tag':
      return 'admin.appearance.component.service_tag'
    default:
      return 'admin.appearance.component.unknown'
  }
}

/** banner/emblem/backdrop = URL d'image ; service_tag = texte. */
export function isImageComponent(component: string): boolean {
  return component === 'banner' || component === 'emblem' || component === 'backdrop'
}

/** Provenance de la valeur servie : résolue en direct vs dernière valeur connue. */
export function servedFromKey(servedFrom: string): AdminManifestKey {
  return servedFrom === 'live'
    ? 'admin.appearance.served_from.live'
    : 'admin.appearance.served_from.carry'
}

// Details techniques connus (cf. haloclient.Detail) → explication humaine.
const DETAIL_KEYS: Record<string, AdminManifestKey> = {
  mapping_hit: 'admin.appearance.detail.mapping_hit',
  mapping_miss: 'admin.appearance.detail.mapping_miss',
  image_resolved: 'admin.appearance.detail.image_resolved',
  service_tag_present: 'admin.appearance.detail.service_tag_present',
  no_positive_cfg: 'admin.appearance.detail.no_positive_cfg',
  cms_http_error: 'admin.appearance.detail.cms_http_error',
  image_unresolved: 'admin.appearance.detail.image_unresolved',
  no_service_tag: 'admin.appearance.detail.no_service_tag',
  no_emblem_path: 'admin.appearance.detail.no_emblem_path',
  non_emblem_path: 'admin.appearance.detail.non_emblem_path',
  no_banner_field: 'admin.appearance.detail.no_banner_field',
}

// Repli par verdict quand le detail est vide (chemins uniformes auth_required /
// not_supported → detail "") ou non reconnu.
const VERDICT_FALLBACK_DETAIL: Record<string, AdminManifestKey> = {
  ok: 'admin.appearance.detail.fallback_ok',
  upstream_missing: 'admin.appearance.detail.fallback_upstream_missing',
  transient: 'admin.appearance.detail.fallback_transient',
  auth_required: 'admin.appearance.detail.fallback_auth_required',
  not_supported: 'admin.appearance.detail.fallback_not_supported',
}

/**
 * Le POURQUOI : detail technique connu → explication dédiée ; sinon repli par
 * verdict ; sinon repli générique. Ne lève jamais (clé toujours valide).
 */
export function detailExplanationKey(verdict: string, detail: string): AdminManifestKey {
  return (
    DETAIL_KEYS[detail] ??
    VERDICT_FALLBACK_DETAIL[verdict] ??
    'admin.appearance.detail.fallback_unknown'
  )
}

// Dernier statut de fetch live persisté (career_progression.last_fetch_status).
const FETCH_STATUS_KEYS: Record<string, AdminManifestKey> = {
  ok: 'admin.appearance.fetch_status.ok',
  api_empty: 'admin.appearance.fetch_status.api_empty',
  forbidden_403: 'admin.appearance.fetch_status.forbidden_403',
  auth_missing: 'admin.appearance.fetch_status.auth_missing',
  failed: 'admin.appearance.fetch_status.failed',
}

/** Libellé du dernier statut de fetch live ("" / inconnu → « Jamais tenté »). */
export function fetchStatusKey(status: string): AdminManifestKey {
  return FETCH_STATUS_KEYS[status] ?? 'admin.appearance.fetch_status.none'
}
