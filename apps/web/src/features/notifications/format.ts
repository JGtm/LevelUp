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

// enrichParams ajoute les paramètres dérivés (ex: metric_label depuis metric_key)
// résolus via i18n. Permet aux templates d'utiliser {metric_label} sans que le
// backend ait à connaître la locale.
function enrichParams(
  params: Record<string, unknown> | undefined,
  locale: NotificationsLocale,
): Record<string, unknown> | undefined {
  if (!params) return params
  const out = { ...params }
  if (typeof params.metric_key === 'string' && out.metric_label == null) {
    const t = getNotificationsText(locale)
    out.metric_label = t.metricLabel[params.metric_key] ?? params.metric_key
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
