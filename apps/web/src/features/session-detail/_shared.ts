/**
 * Helpers partagés entre les sous-composants de session-detail.
 *
 * - `useSessionT` : hook i18n local (résout `sessionManifest` selon `appShellStore.locale`).
 * - `formatNumber` : alias `formatNumberFixed` (lib/formatters) — wrapper conservé pour
 *   les call-sites internes ; bascule directe à terme.
 * - `formatPercent` : valeur reçue en 0..100 (legacy). TODO P4 ADR 0006 : basculer
 *   vers `lib/formatters.formatPercent` quand l'API passera en 0..1.
 * - `formatShortDateTime` : format jj/mm HH:MM en locale FR.
 * - `parseDelta` : parse une string delta éventuellement préfixée (+/-).
 */
import type React from 'react'

import { outcomeScale } from '@/lib/accessibility/scales'
import { tokenCssVar } from '@/lib/accessibility'
import { formatNumberFixed } from '@/lib/formatters'
import { formatMessage } from '@/lib/i18n/format'
import { sessionManifest, type SessionManifestKey } from '@/lib/i18n/generated/session'
import { useAppShellStore } from '@/stores/appShellStore'

export function useSessionT() {
  const locale = useAppShellStore((s) => s.locale)
  return (key: SessionManifestKey, values?: Record<string, string | number>) =>
    formatMessage(sessionManifest, key, locale, values)
}

export const formatNumber = formatNumberFixed

export function formatPercent(value: number | null) {
  if (value == null) {
    return '—'
  }
  return `${value.toFixed(1)}%`
}

export function parseDelta(delta: string | null) {
  if (!delta) {
    return null
  }
  const parsed = Number.parseFloat(delta)
  return Number.isNaN(parsed) ? null : parsed
}

export function formatShortDateTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return new Intl.DateTimeFormat('fr-FR', {
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

const OUTCOME_INT_KEY: Record<number, string> = { 2: 'win', 1: 'draw', 3: 'loss', 4: 'dnf' }

export function matchOutcomeTone(outcome: number | null): {
  className: string
  style?: React.CSSProperties
} {
  if (outcome == null) return { className: 'text-muted-foreground' }
  const key = OUTCOME_INT_KEY[outcome]
  const token = key ? outcomeScale(key) : null
  if (!token) return { className: 'text-muted-foreground' }
  return { className: 'font-medium', style: { color: tokenCssVar(token) } }
}

/**
 * Mapping outcome int → clé canonique pour les wrappers ECharts (OutcomeSequenceTape).
 * win=2, tie=1, loss=3, dnf=4 (cf. ADR 0006).
 */
export function outcomeIntToKey(outcome: number | null): 'win' | 'loss' | 'tie' | 'dnf' | null {
  if (outcome === 2) return 'win'
  if (outcome === 3) return 'loss'
  if (outcome === 1) return 'tie'
  if (outcome === 4) return 'dnf'
  return null
}
