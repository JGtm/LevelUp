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
import { type SemanticToken } from '@/lib/accessibility'
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

// Mapping outcome int → clé canonique : centralisé dans `@/lib/outcome`
// (`outcomeCodeToValue`, défaut null). Consommé directement par
// SessionNetScoreArea / SessionOutcomeDonut.

/** Tronque un nom (carte/mode) pour une étiquette d'axe compacte. */
function truncateAxisName(s: string, max = 10): string {
  return s.length <= max ? s : s.slice(0, max - 1) + '…'
}

/**
 * Étiquette d'axe X chronologique par match, façon page Escouade : "#N\nCarte"
 * (numéro de match 1-based + nom de carte tronqué, rendu sur 2 lignes par ECharts).
 * Fallback sur le mode (pair_name) si la carte manque, "—" si rien.
 */
export function sessionMatchAxisLabel(
  index: number,
  mapName?: string | null,
  pairName?: string | null,
): string {
  const name = mapName?.trim() || pairName?.trim() || '—'
  return `#${index + 1}\n${truncateAxisName(name)}`
}

/** Tier de performance (1=meilleur … 5=pire) depuis un score. Seuils alignés sur analysis.PerfTier (Go). */
export function perfTierFromScore(score: number): 1 | 2 | 3 | 4 | 5 {
  if (score >= 80) return 1
  if (score >= 65) return 2
  if (score >= 50) return 3
  if (score >= 35) return 4
  return 5
}

// formatRankDelta a migré vers le module formatters de rating (réutilisé hors
// session : Match View, Session Compare). Ré-exporté ici pour les importeurs existants.
export { formatRankDelta } from '@/lib/formatters/rating'

/** Token de couleur d'un delta de rang : positif=vert, négatif=rouge, nul=neutre. */
export function rankDeltaToken(delta: number): SemanticToken {
  if (delta > 0) return 'divergent-pos'
  if (delta < 0) return 'divergent-neg'
  return 'divergent-neutral'
}
