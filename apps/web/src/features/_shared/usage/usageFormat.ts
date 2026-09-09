/**
 * usageFormat.ts — LE FORMATAGE du bloc « usages d'équipement, armes spéciales et objectifs »
 * (parts, cadences, comptes, durées).
 *
 * Extrait de `session-detail/usageLogic.ts` le 2026-09-09 (étape E5.1bis, scission de taille —
 * CLAUDE.md n°5, 680 L avant scission) au moment du déménagement du bloc vers
 * `features/_shared/usage/` (étape E5.1, PLAN_EQUIPEMENT_GACHIS_2026-09-09.md) : ce fichier ne
 * porte QUE le formatage, sans classification ni forme de rendu, pour que chaque fichier
 * résultant reste sous le seuil.
 *
 * TOUT AXE EST NORMALISÉ (doctrine du handoff §1) : parts en %, cadences par dix minutes. Les
 * comptes bruts ne sortent d'ici QUE comme textes d'honnêteté, jamais comme valeur d'axe.
 *
 * Pur : aucun React, aucune couleur en dur, aucune lecture de store.
 */
import type { Locale } from '@/lib/i18n/locale'

/** Nombre à `digits` décimales, virgule en FR — jamais de séparateur de milliers. */
export function formatUsageDecimal(v: number, locale: Locale, digits = 1): string {
  const s = v.toFixed(digits)
  return locale === 'fr' ? s.replace('.', ',') : s
}

/** Une part en pourcentage : « 45,6 % » / "45.6%". `null`/absent → tiret. */
export function formatUsagePct(v: number | null | undefined, locale: Locale): string {
  if (v == null) return '—'
  const s = formatUsageDecimal(v, locale)
  return locale === 'fr' ? `${s} %` : `${s}%`
}

/** Une cadence par dix minutes : une décimale. `null`/absent → tiret. */
export function formatUsageRate(v: number | null | undefined, locale: Locale): string {
  if (v == null) return '—'
  return formatUsageDecimal(v, locale)
}

/**
 * Un TOTAL brut (dénominateur d'honnêteté). Entier écrit tel quel ; une grandeur en
 * durée (rôle « tenir ») s'écrit m:ss — et un 0 MESURÉ s'écrit « 0:00 », jamais un
 * tiret (le tiret est réservé au non-mesuré ; cf. l'en-tête de
 * `lib/formatters/duration.ts`, dont le repli sur 0 ne convient pas ici).
 */
export function formatUsageCount(
  v: number | null | undefined,
  locale: Locale,
  isDuration = false,
): string {
  if (v == null) return '—'
  if (isDuration) {
    const total = Math.max(0, Math.round(v))
    const m = Math.floor(total / 60)
    const s = total % 60
    return `${m}:${s.toString().padStart(2, '0')}`
  }
  return Number.isInteger(v) ? String(v) : formatUsageDecimal(v, locale)
}
