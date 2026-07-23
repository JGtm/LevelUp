/**
 * Agrège le contexte technique + métier nécessaire pour pré-remplir une
 * issue GitHub depuis le drawer feedback.
 *
 * Helper pur : reçoit le contexte (stores + browser APIs) en argument,
 * pas d'import direct de Zustand pour rester testable et indépendant
 * du runtime React.
 */
import type { FilterContextInput } from '@/lib/api/types'
import type { ConsoleEntry, FailedRequest } from '@/lib/global-capture/buffers'
import type { Locale } from '@/lib/i18n/locale'

export interface BrowserEnv {
  url: string
  pathname: string
  userAgent: string
  viewportWidth: number
  viewportHeight: number
  locale: Locale
  theme: 'dark' | 'light'
  timestampIso: string
  focusedElement: string | null
}

export interface ShellSummary {
  titleSlug: string
  playerSlug: string | null
  appVersion: string | null
}

export interface FeedbackContext {
  browser: BrowserEnv
  shell: ShellSummary
  filters: FilterContextInput | null
  console: ConsoleEntry[]
  failedRequests: FailedRequest[]
}

export interface CollectInputs {
  browser: BrowserEnv
  shell: ShellSummary
  filters: FilterContextInput | null
  console: ConsoleEntry[]
  failedRequests: FailedRequest[]
}

/** Pass-through structuré : permet aux tests de stub indépendamment chaque source. */
export function collectContext(inputs: CollectInputs): FeedbackContext {
  return {
    browser: inputs.browser,
    shell: inputs.shell,
    filters: inputs.filters,
    console: inputs.console,
    failedRequests: inputs.failedRequests,
  }
}

/** Capture l'élément actuellement focus de manière defensive. Renvoie null si rien. */
export function describeFocusedElement(): string | null {
  if (typeof document === 'undefined') return null
  const el = document.activeElement
  if (!el || el === document.body) return null
  const tag = el.tagName.toLowerCase()
  const cls = (el.getAttribute('class') ?? '').trim().split(/\s+/).slice(0, 3).join('.')
  return cls ? `${tag}.${cls}` : tag
}
