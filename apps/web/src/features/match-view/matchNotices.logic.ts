/**
 * Logique de la zone de messages de la vue match : QUELS messages, dans QUEL ordre, avec QUELS
 * libellés. Le composant (`MatchNotices.tsx`) ne décide rien.
 *
 * Ordre, du plus grave au moins grave :
 *  1. lecture en échec (`unavailable`, ton erreur) — prime sur « données absentes » ;
 *  2. données absentes (`partial`, ton avertissement) ;
 *  3. analyse du film non finie (`analysis`, ton info) ;
 *  4. confidentialité (`privacy`, composant et sémantique propres).
 */
import type { MatchPrivacyWarning, MatchViewResponse } from '@/lib/api/types'
import {
  ANALYSIS_PART,
  isAnalysisInProgress,
  type AnalysisPendingPart,
} from '@/lib/analysis-pending/analysisPending.logic'
import type { Locale } from '@/lib/i18n/locale'
import type { MatchViewText } from './i18n'
import { selectMatchPartialBanner } from './matchPartial.logic'

export type MatchNotice =
  | { kind: 'unavailable' }
  | { kind: 'partial'; reasons: string[] }
  | { kind: 'analysis'; inProgress: boolean; parts: AnalysisPendingPart[] }
  | { kind: 'privacy'; warning: MatchPrivacyWarning }

export type MatchNoticesInput = Pick<MatchViewResponse, 'is_partial' | 'partial_reasons' | 'privacy_warning'> &
  Partial<Pick<MatchViewResponse, 'header'>>

const KNOWN_PARTS: readonly string[] = Object.values(ANALYSIS_PART)

function isKnownPart(part: string): part is AnalysisPendingPart {
  return KNOWN_PARTS.includes(part)
}

/** Les messages à afficher, dans l'ordre. Liste vide : la zone n'existe pas. */
export function selectMatchNotices(data: MatchNoticesInput): MatchNotice[] {
  const notices: MatchNotice[] = []
  const partial = selectMatchPartialBanner(data)
  if (partial) notices.push(partial)

  const header = data.header
  if (header?.analysis_pending === true) {
    notices.push({
      kind: 'analysis',
      inProgress: isAnalysisInProgress(header),
      parts: (header.analysis_pending_parts ?? []).filter(isKnownPart),
    })
  }

  const privacy = data.privacy_warning
  if (privacy && privacy.level !== 'none') notices.push({ kind: 'privacy', warning: privacy })
  return notices
}

/** Titre du message d'analyse : « en cours » seulement si un décodage porte sur ce match. */
export function analysisNoticeTitle(inProgress: boolean, t: MatchViewText): string {
  return inProgress ? t.analysisPendingTitleInProgress : t.analysisPendingTitleQueued
}

/** Libellé de l'emplacement d'attente d'un onglet, selon le même état. */
export function analysisSlotLabel(inProgress: boolean, t: MatchViewText): string {
  return inProgress ? t.analysisPendingSlotInProgress : t.analysisPendingSlotQueued
}

/** Corps du message d'analyse : les seules parties en attente, noms courts. */
export function analysisNoticeBody(parts: readonly AnalysisPendingPart[], t: MatchViewText): string {
  return parts.map((part) => t.analysisPendingParts[part]).join(' · ')
}

/** Impact end-user de chaque code stable de partial_reason, par langue. */
const PARTIAL_REASON_TEXT = {
  fr: {
    scoreboard_empty: 'Le tableau des scores et les stats individuelles sont indisponibles',
    events_empty:
      'Les graphes de combat sont vides — Cadence, Dominance et Frags différentiel ne peuvent pas être tracés',
    player_stats_empty: 'Certaines statistiques personnelles sont absentes — les données peuvent être incomplètes',
    medals_empty: 'Les médailles et citations ne sont pas disponibles',
  },
  en: {
    scoreboard_empty: 'Scoreboard and individual player stats are unavailable',
    events_empty: 'Combat charts cannot be displayed — Cadence, Dominance and Frag diff are empty',
    player_stats_empty: 'Some personal stats are missing — datas may be incomplete',
    medals_empty: 'Medals and commendations are unavailable',
  },
} satisfies Record<Locale, Record<string, string>>

/**
 * Traduit un code stable de partial_reason en impact end-user concret.
 * Décrit ce que l'utilisateur ne peut PAS voir, pas la raison technique.
 * Code inconnu : renvoyé tel quel.
 */
export function translatePartialReason(code: string, locale: Locale): string {
  const texts: Record<string, string> = PARTIAL_REASON_TEXT[locale]
  return Object.hasOwn(texts, code) ? texts[code] : code
}
