/**
 * format.ts — LES FORMATS DE LA RENCONTRE, en un seul point.
 *
 * Deux vues affichent les MÊMES chiffres de rencontre entre deux joueurs : le tableau de la
 * vue match (`features/match-view/MatchEncountersTable.tsx`) et la carte de briefing de
 * l'explorateur (`features/explorer/ExplorerEncounterBriefing.tsx`). Les quatre formats
 * ci-dessous y vivaient EN DOUBLE, au caractère près — l'en-tête du second le disait même
 * (« copiés à l'identique de MatchEncountersTable.tsx »). Deux copies d'un tiret cadratin,
 * d'un seuil de semaines ou d'un « hier » finissent par diverger sans que rien ne le dise :
 * la règle « ≤ 2 copies » (CLAUDE.md n° 6) veut le helper ET le garde-rail, d'où
 * `encountersFormat.guard.test.ts` à côté.
 *
 * AUCUNE DÉCISION N'A CHANGÉ en centralisant : mêmes bornes, mêmes libellés, même tiret
 * cadratin d'absence. Ce module est un déplacement, pas une réécriture.
 */
import type { Locale } from '@/lib/i18n/locale'

/** Le tiret cadratin de l'absence de mesure — le MÊME dans les quatre formats. */
const ABSENT = '—'

/**
 * formatKDCross — frags infligés / morts subies face à ce joueur, en croisé.
 *
 * `0` est une MESURE, pas une absence : un seul des deux côtés renseigné suffit à afficher
 * la paire (l'autre vaut 0). Les DEUX absents seulement rendent le tiret.
 */
export function formatKDCross(
  kills: number | null | undefined,
  deaths: number | null | undefined,
): string {
  if (kills == null && deaths == null) return ABSENT
  return `${kills ?? 0}/${deaths ?? 0}`
}

/**
 * formatKDRatio — frags ÷ morts, à deux décimales.
 *
 * Zéro mort : `∞` quand il y a des frags (le ratio le plus favorable possible), le tiret
 * quand il n'y en a pas non plus — 0/0 n'est pas un ratio infini, c'est une absence.
 */
export function formatKDRatio(
  kills: number | null | undefined,
  deaths: number | null | undefined,
): string {
  if (kills == null || deaths == null) return ABSENT
  if (deaths === 0) return kills > 0 ? '∞' : ABSENT
  return (kills / deaths).toFixed(2)
}

/** Ancienneté relative, en français. Une date illisible rend le tiret, jamais « Invalid Date ». */
export function formatRelativeFR(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ABSENT
  const diffMs = Date.now() - date.getTime()
  const minutes = Math.round(diffMs / 60_000)
  if (minutes < 1) return "à l'instant"
  if (minutes < 60) return `il y a ${minutes} min`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return hours <= 1 ? 'il y a 1 h' : `il y a ${hours} h`
  const days = Math.round(hours / 24)
  if (days === 1) return 'hier'
  if (days < 7) return `il y a ${days} j`
  const weeks = Math.round(days / 7)
  if (weeks < 5) return weeks <= 1 ? 'il y a 1 sem.' : `il y a ${weeks} sem.`
  const months = Math.round(days / 30)
  if (months < 12) return months <= 1 ? 'il y a 1 mois' : `il y a ${months} mois`
  const years = Math.round(days / 365)
  return years <= 1 ? 'il y a 1 an' : `il y a ${years} ans`
}

/** Ancienneté relative, en anglais. Mêmes bornes que [formatRelativeFR], au mot près. */
export function formatRelativeEN(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ABSENT
  const diffMs = Date.now() - date.getTime()
  const minutes = Math.round(diffMs / 60_000)
  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes} min ago`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours} h ago`
  const days = Math.round(hours / 24)
  if (days === 1) return 'yesterday'
  if (days < 7) return `${days} d ago`
  const weeks = Math.round(days / 7)
  if (weeks < 5) return `${weeks} w ago`
  const months = Math.round(days / 30)
  if (months < 12) return `${months} mo ago`
  const years = Math.round(days / 365)
  return years <= 1 ? '1 y ago' : `${years} y ago`
}

/**
 * formatRelativeFor — le formateur d'ancienneté de la langue du lecteur.
 *
 * Les deux vues écrivaient le MÊME ternaire (`locale === 'en' ? …EN : …FR`) : il vit ici,
 * pour qu'une troisième langue n'ait qu'un seul endroit à toucher.
 */
export function formatRelativeFor(locale: Locale): (iso: string) => string {
  return locale === 'en' ? formatRelativeEN : formatRelativeFR
}
