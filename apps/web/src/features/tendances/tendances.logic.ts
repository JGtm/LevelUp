/**
 * tendances.logic — logique PURE de l'onglet Tendances (aucun React, aucune requête).
 *
 * L'API sert une réponse unique à tous les horizons ; le web découpe. Ce module porte la
 * découpe d'une série par horizon, les pas proposés, le formatage d'une valeur par unité et
 * la borne des écarts.
 */
import { formatNumber, intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'

import { getTendancesText } from './i18n'

/** Horizons proposés, en jours. */
export const HORIZONS = [7, 30, 90, 365] as const
export type Horizon = (typeof HORIZONS)[number]
export const DEFAULT_HORIZON: Horizon = 90

/** Vue de la page : le joueur seul, ou une escouade choisie. */
export type TendancesView = 'solo' | 'squad'

/** Pas de temps d'une série. */
export type Step = 'match' | 'day' | 'week' | 'month'

/** Matchs requis de part et d'autre pour qu'un horizon soit comparé (miroir du seuil Go). */
export const MIN_COMPARE_MATCHES = 10

/** Borne de l'écart-type affiché : au-delà, la case garde la teinte extrême. */
export const Z_LIMIT = 2.5

const DAY_MS = 24 * 60 * 60 * 1000

/**
 * Garde les points dont l'instant `t` est dans `]asOf − days, asOf]` : un point appartient à
 * l'horizon quand le DÉBUT de son intervalle y est. Borne basse exclue, borne haute incluse.
 */
export function pointsInHorizon<T extends { t: string }>(
  points: readonly T[] | null | undefined,
  asOf: string | Date,
  days: number,
): T[] {
  const fin = new Date(asOf).getTime()
  const debut = fin - days * DAY_MS
  return (points ?? []).filter((p) => {
    const t = new Date(p.t).getTime()
    return t > debut && t <= fin
  })
}

/** Pas proposés pour un horizon (7 j : match, jour ; 365 j : semaine, mois). */
export function stepsForHorizon(days: number): Step[] {
  switch (days) {
    case 7:
      return ['match', 'day']
    case 30:
      return ['match', 'day', 'week']
    case 90:
      return ['day', 'week', 'month']
    default:
      return ['week', 'month']
  }
}

/** Pas par défaut d'un horizon : match, jour, semaine, mois. */
export function defaultStep(days: number): Step {
  switch (days) {
    case 7:
      return 'match'
    case 30:
      return 'day'
    case 90:
      return 'week'
    default:
      return 'month'
  }
}

/** Borne un écart-type à ± 2,5. */
export function clampZ(z: number): number {
  return Math.max(-Z_LIMIT, Math.min(Z_LIMIT, z))
}

/** Décimales d'un pourcentage : celles de l'API moins 2 (un ratio 0..1 en perd deux), jamais moins de 0. */
function percentDecimals(decimals: number): number {
  return Math.max(0, decimals - 2)
}

/** Suffixe de pourcentage : espace insécable en français, collé en anglais. */
function percentSuffix(locale: Locale): string {
  return locale === 'en' ? '%' : '\u00a0%'
}

/**
 * Écrit `|value × scale|` avec les séparateurs de la locale ; Intl arrondit lui-même à
 * `decimals` décimales, comme le reste de l'application. `zero` est vrai quand le texte ne
 * contient aucun chiffre de 1 à 9 : le signe suit ce texte, jamais un nombre arrondi à part.
 */
function numberParts(
  value: number,
  scale: number,
  decimals: number,
  locale: Locale,
): { text: string; zero: boolean } {
  const text = formatNumber(Math.abs(value * scale), intlLocale(locale), decimals)
  return { text, zero: !/[1-9]/.test(text) }
}

/**
 * Formate une valeur d'indicateur selon son unité : `ratio` en pourcentage, `seconds` avec
 * « s », `hours` avec « h », `number` tel quel. Séparateurs et arrondi d'Intl ; le moins
 * typographique (U+2212) précède une valeur négative, un zéro à l'affichage s'écrit sans signe.
 */
export function formatTrendValue(
  value: number,
  unit: string,
  decimals: number,
  locale: Locale,
): string {
  const t = getTendancesText(locale)
  const ratio = unit === 'ratio'
  const parts = numberParts(
    value,
    ratio ? 100 : 1,
    ratio ? percentDecimals(decimals) : decimals,
    locale,
  )
  const nombre = value < 0 && !parts.zero ? `\u2212${parts.text}` : parts.text
  if (ratio) return `${nombre}${percentSuffix(locale)}`
  switch (unit) {
    case 'seconds':
      return `${nombre}\u00a0${t.unitSeconds}`
    case 'hours':
      return `${nombre}\u00a0${t.unitHours}`
    default:
      return nombre
  }
}

/** Suffixe d'unité d'un écart : un ratio s'écrit en points de pourcentage. */
function deltaSuffix(unit: string, locale: Locale): string {
  const t = getTendancesText(locale)
  if (unit === 'ratio') return `\u00a0${t.unitPoints}`
  if (unit === 'seconds') return `\u00a0${t.unitSeconds}`
  if (unit === 'hours') return `\u00a0${t.unitHours}`
  return ''
}

/**
 * Formate l'écart signé entre deux valeurs d'un indicateur : un ratio s'écrit en points de
 * pourcentage, les autres unités comme `formatTrendValue`. Arrondi d'Intl ; le signe suit le texte écrit,
 * un zéro à l'affichage est sans signe.
 */
export function formatTrendDelta(
  delta: number,
  unit: string,
  decimals: number,
  locale: Locale,
): string {
  const echelle = unit === 'ratio' ? 100 : 1
  const d = unit === 'ratio' ? percentDecimals(decimals) : decimals
  const { text, zero } = numberParts(delta, echelle, d, locale)
  const suffixe = deltaSuffix(unit, locale)
  if (zero) return `${text}${suffixe}`
  return `${delta > 0 ? '+' : '\u2212'}${text}${suffixe}`
}

/**
 * Les clés proposées par le filtre « Type de partie » : celles de la réponse, plus le type
 * choisi s'il n'y figure pas (sans lui, le filtre afficherait « Toutes les parties » alors qu'il
 * reste appliqué).
 */
export function gameTypeOptions(keys: readonly string[], selected: string): string[] {
  return selected !== '' && !keys.includes(selected) ? [...keys, selected] : [...keys]
}
