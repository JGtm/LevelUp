/**
 * descriptorLabel.ts — builder de label compact pour `ContextDescriptor`.
 *
 * Extrait de `i18n.ts` (Phase 2c, 2026-05-07) pour respecter la limite
 * de 500 lignes/fichier (CLAUDE.md §5). Le builder reste localisé via
 * `MATCH_VIEW_TEXT[locale]` mais vit dans son propre module pour faciliter
 * tests unitaires + extension future (nouveaux kinds de descriptor).
 */
import type { ContextDescriptor } from '@/lib/match-nav/navContext'
import { MATCH_VIEW_TEXT, type MatchViewLocale } from './i18n'

/**
 * Format date court pour les descriptors `session` / `period`.
 *
 * FR : `07/05/26 à 21:30` · EN : `07/05/26 at 21:30`.
 * Si `withTime=false`, seule la date est rendue (descriptor `period`).
 */
function fmtShortDateTime(iso: string, locale: MatchViewLocale, withTime: boolean): string {
  const d = new Date(iso)
  if (isNaN(d.getTime())) return iso
  const intlLocale = locale === 'en' ? 'en-GB' : 'fr-FR'
  const dateStr = new Intl.DateTimeFormat(intlLocale, {
    day: '2-digit',
    month: '2-digit',
    year: '2-digit',
  }).format(d)
  if (!withTime) return dateStr
  const timeStr = new Intl.DateTimeFormat(intlLocale, {
    hour: '2-digit',
    minute: '2-digit',
  }).format(d)
  return locale === 'en' ? `${dateStr} at ${timeStr}` : `${dateStr} à ${timeStr}`
}

/**
 * buildDescriptorLabel — Phase 2c : construit le **fragment** du compteur
 * pour un `ContextDescriptor` typé. Renvoie une chaîne vide si le descriptor
 * n'a pas l'info attendue (ex : `with_player` sans gamertag).
 *
 * Le résultat est destiné à être injecté dans `matchCounterCtxFmt(label, n, total)` :
 *   `Matchs <label> X/Y` (FR) · `<Label> matches X/Y` (EN).
 */
export function buildDescriptorLabel(
  descriptor: ContextDescriptor | null | undefined,
  locale: MatchViewLocale,
): string {
  if (!descriptor) return ''
  const t = MATCH_VIEW_TEXT[locale]
  switch (descriptor.kind) {
    case 'recent':
      return t.ctxRecent
    case 'favorites':
      return t.ctxFavorites
    case 'media':
      return t.ctxMedia
    case 'top_matches':
      return t.ctxTopMatches
    case 'with_player':
      return descriptor.gamertag ? t.ctxWithPlayerFmt(descriptor.gamertag) : ''
    case 'session':
      return descriptor.startTimeUtc
        ? t.ctxSessionFmt(fmtShortDateTime(descriptor.startTimeUtc, locale, true))
        : ''
    case 'period': {
      const from = descriptor.from ? fmtShortDateTime(descriptor.from, locale, false) : ''
      const to = descriptor.to ? fmtShortDateTime(descriptor.to, locale, false) : ''
      if (from && to) return t.ctxPeriodFromToFmt(from, to)
      if (from) return t.ctxPeriodFromFmt(from)
      if (to) return t.ctxPeriodToFmt(to)
      return ''
    }
    case 'playlist':
      return descriptor.name ? t.ctxPlaylistFmt(descriptor.name) : ''
    case 'mode':
      return descriptor.category ? t.ctxModeFmt(descriptor.category) : ''
  }
}
