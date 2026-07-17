/**
 * Résolution des libellés title/body à partir des clés i18n + params reçus du backend.
 *
 * Le serveur ne renvoie que `title_key`, `body_key` et `params` — la résolution
 * FR/EN se fait ici à partir du dictionnaire `templates` de i18n.ts.
 *
 * Interpolation simple `{name}` → params.name. Pas de plurals ICU pour le MVP
 * (certains templates contiennent "match(s)" littéralement, suffisant pour l'instant).
 */
import type { Notification } from './types'
import { subTierRoman } from '@/lib/skillTiers'
import { getNotificationsText, type NotificationsLocale } from './i18n'

export function resolveTitle(
  notif: Pick<Notification, 'title_key' | 'params'>,
  locale: NotificationsLocale,
): string {
  return resolveTemplate(notif.title_key, notif.params, locale)
}

export function resolveBody(
  notif: Pick<Notification, 'body_key' | 'params'>,
  locale: NotificationsLocale,
): string {
  if (!notif.body_key) return ''
  return resolveTemplate(notif.body_key, notif.params, locale)
}

function resolveTemplate(
  key: string,
  params: Record<string, unknown> | undefined,
  locale: NotificationsLocale,
): string {
  const t = getNotificationsText(locale)
  const template = t.templates[key]
  if (!template) {
    // Fallback : retourne la clé brute (utile pour repérer les clés manquantes)
    return key
  }
  return interpolate(template, enrichParams(params, locale))
}

// subTierToRoman garde le contrat « unknown → '' si non-numérique ou ≤0 », puis
// délègue la conversion romaine à la source unique (lib/skillTiers.subTierRoman).
function subTierToRoman(v: unknown): string {
  if (typeof v !== 'number' || v <= 0) return ''
  return subTierRoman(v)
}

// enrichParams ajoute les paramètres dérivés (ex: metric_label depuis metric_key)
// résolus via i18n. Permet aux templates d'utiliser {metric_label} sans que le
// backend ait à connaître la locale.
//
// 2026-05-18 — Progression V2 : le coach generator passe `metric` (pas
// `metric_key`) dans ses params. Fallback sur `metric` + nouvel enrichissement
// `period` → `period_label` pour les templates near_miss.
// 2026-05-22 — sub_tier / previous_sub_tier convertis en chiffres romains.
function enrichParams(
  params: Record<string, unknown> | undefined,
  locale: NotificationsLocale,
): Record<string, unknown> | undefined {
  if (!params) return params
  const out = { ...params }
  const t = getNotificationsText(locale)
  if (out.metric_label == null) {
    const metricKey =
      (typeof params.metric_key === 'string' ? params.metric_key : null) ??
      (typeof params.metric === 'string' ? params.metric : null)
    if (metricKey) {
      out.metric_label = t.metricLabel[metricKey] ?? metricKey
    }
  }
  if (out.period_label == null && typeof params.period === 'string') {
    out.period_label = t.periodLabel[params.period] ?? params.period
  }
  // Normalise sub_tier entier → chiffre romain (couvre notifications déjà stockées).
  for (const key of ['sub_tier', 'previous_sub_tier'] as const) {
    if (typeof out[key] === 'number') {
      out[key] = subTierToRoman(out[key])
    }
  }
  // Arrondit les valeurs de métriques flottantes à 2 décimales. Le backend
  // envoie des float64 bruts (ex: pspm = 6000/11 = 545.4545…) ; sans ça les
  // templates affichaient toute la mantisse. Math.round préserve les entiers
  // (5 → 5, pas "5.00"). Couvre aussi les notifs déjà stockées.
  for (const key of ['value', 'target', 'previous_value'] as const) {
    if (typeof out[key] === 'number') {
      out[key] = Math.round((out[key] as number) * 100) / 100
    }
  }
  return out
}

function interpolate(template: string, params?: Record<string, unknown>): string {
  if (!params) return template
  return template.replace(/\{(\w+)\}/g, (_, name: string) => {
    const v = params[name]
    return v == null ? `{${name}}` : String(v)
  })
}
