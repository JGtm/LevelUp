/**
 * Heuristiques de classification du feedback côté front.
 *
 * Helper pur (pas de React, pas de Zustand). Combine la saisie utilisateur
 * (titre + description + type choisi) avec le contexte capturé (URL, erreurs
 * console récentes) pour produire un triplet { type, severity, area }.
 *
 * Ce triplet alimente les `labels=` de l'URL GitHub Issues. La GitHub Action
 * de triage peut affiner la classification a posteriori via Claude Haiku.
 */
import type { ConsoleEntry } from '@/lib/global-capture/buffers'

export type FeedbackType = 'bug' | 'enhancement' | 'question'
export type FeedbackSeverity = 'low' | 'medium' | 'high' | 'critical'
export type FeedbackArea =
  | 'synthesis'
  | 'explorer'
  | 'squad'
  | 'sessions'
  | 'timeseries'
  | 'match_view'
  | 'palmares'
  | 'player_home'
  | 'media'
  | 'career'
  | 'notifications'
  | 'objectifs'
  | 'citations'
  | 'settings'
  | 'meta'
  | 'general'

export type UserPickedType = FeedbackType | 'auto'

export interface ClassificationInput {
  /** Type explicitement choisi via segmented control. 'auto' = inférer depuis description. */
  pickedType: UserPickedType
  description: string
}

export interface ClassificationContext {
  pathname: string
  recentConsole: ConsoleEntry[]
}

export interface Classification {
  type: FeedbackType
  severity: FeedbackSeverity
  area: FeedbackArea
}

const HIGH_SEVERITY_KEYWORDS = ['crash', 'perd ma progression', 'impossible', 'bloqué', 'bloque']
const BUG_KEYWORDS = ['bug', 'marche pas', 'erreur', 'cassé', 'casse', 'plante']
const QUESTION_KEYWORDS = ['comment', 'pourquoi', 'comment ça', 'comment se']
const ENHANCEMENT_KEYWORDS = ['j\'aimerais', 'ce serait bien', 'feature', 'idée', 'pourrait']

const FATAL_CONSOLE_PATTERNS = [/TypeError/i, /ReferenceError/i]

export function classifyFeedback(
  input: ClassificationInput,
  context: ClassificationContext,
): Classification {
  const desc = input.description.toLowerCase().trim()
  const hasFatalConsole = context.recentConsole.some((e) =>
    FATAL_CONSOLE_PATTERNS.some((rx) => rx.test(e.message)),
  )
  const hasAnyConsoleError = context.recentConsole.some((e) => e.level === 'error')
  const hasHighKeyword = HIGH_SEVERITY_KEYWORDS.some((kw) => desc.includes(kw))
  const hasBugKeyword = BUG_KEYWORDS.some((kw) => desc.includes(kw))
  const hasQuestionMark = desc.includes('?')
  const hasQuestionKeyword = QUESTION_KEYWORDS.some((kw) => desc.includes(kw))
  const hasEnhancementKeyword = ENHANCEMENT_KEYWORDS.some((kw) => desc.includes(kw))
  // hasEnhancementKeyword renforce le default 'enhancement' mais n'inverse rien
  // si un signal bug est présent → on l'utilise uniquement comme tie-breaker.
  void hasEnhancementKeyword

  let type: FeedbackType
  let severity: FeedbackSeverity

  if (input.pickedType !== 'auto') {
    type = input.pickedType
  } else if (hasFatalConsole || hasHighKeyword || hasBugKeyword || hasAnyConsoleError) {
    type = 'bug'
  } else if (hasQuestionMark || hasQuestionKeyword) {
    type = 'question'
  } else {
    type = 'enhancement'
  }

  // Severity dépend principalement des signaux d'urgence, indépendamment du type choisi.
  if (hasFatalConsole) severity = 'critical'
  else if (hasHighKeyword) severity = 'high'
  else if (hasAnyConsoleError && type === 'bug') severity = 'high'
  else if (hasBugKeyword && type === 'bug') severity = 'medium'
  else severity = 'low'

  return { type, severity, area: matchArea(context.pathname) }
}

const AREA_PATTERNS: Array<[RegExp, FeedbackArea]> = [
  [/\/players\/[^/]+\/(stats\/)?synthesis(\/|$|\?)/, 'synthesis'],
  [/\/players\/[^/]+\/explorer(\/|$|\?)/, 'explorer'],
  [/\/players\/[^/]+\/squad(\/|$|\?)/, 'squad'],
  [/\/players\/[^/]+\/stats\/sessions(\/|$|\?)/, 'sessions'],
  [/\/players\/[^/]+\/stats\/timeseries(\/|$|\?)/, 'timeseries'],
  [/\/players\/[^/]+\/matches\/[^/]+/, 'match_view'],
  [/\/players\/[^/]+\/(community|palmares)/, 'palmares'],
  [/\/players\/[^/]+\/home(\/|$|\?)/, 'player_home'],
  [/\/players\/[^/]+\/media(\/|$|\?)/, 'media'],
  [/\/players\/[^/]+\/(career\/)?citations(\/|$|\?)/, 'citations'],
  [/\/players\/[^/]+\/career(\/|$|\?)/, 'career'],
  [/\/players\/[^/]+\/notifications(\/|$|\?)/, 'notifications'],
  [/\/players\/[^/]+\/(objectifs|ascension)(\/|$|\?)/, 'objectifs'],
  [/^\/(setup|settings)(\/|$|\?)/, 'settings'],
  [/^\/(changelog|help)(\/|$|\?)/, 'meta'],
]

export function matchArea(pathname: string): FeedbackArea {
  for (const [rx, area] of AREA_PATTERNS) {
    if (rx.test(pathname)) return area
  }
  return 'general'
}
