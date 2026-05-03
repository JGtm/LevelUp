/**
 * squadPending — dérivation pure de FilterContextInput pour le preview Escouade.
 *
 * SquadLayout maintient deux états parallèles :
 *  - `pending` (FilterContextInput) : période + cascade, commité via Analyser.
 *  - `pickedSquadSessionLabels` (string[]) : multi-sélection sessions, persistée
 *    en localStorage et appliquée immédiatement (sans Analyser).
 *
 * Le compteur sticky et le calcul des available_options proviennent du POST
 * `filters/resolve`, qui ne consomme que `FilterContextInput`. Sans cette
 * fonction, le multi-select de sessions vit en orbite : ni le compteur ni les
 * counts cascade ne reflètent les sessions cochées.
 *
 * Quand `pickedSquadSessionLabels` est non vide on bascule en
 * `filter_mode='sessions'` et on injecte les labels dans `picked_sessions`.
 * Période et sessions étant mutuellement exclusives côté backend
 * (filters_service.go applique l'une OU l'autre), la période courante est
 * silencieusement ignorée pour le preview tant qu'une session est sélectionnée
 * — le `pending` d'origine reste intact pour le commit Analyser.
 */
import type { FilterContextInput, SessionsInput, PeriodInput } from '@/lib/api/types'

const DEFAULT_SESSIONS: SessionsInput = { picked_sessions: [], gap_minutes: 120 }
const DEFAULT_PERIOD: PeriodInput = { start_date: null, end_date: null }

export function deriveSquadPending(
  pending: FilterContextInput,
  pickedSquadSessionLabels: string[],
): FilterContextInput {
  const base: FilterContextInput = { ...pending, match_context: 'squad' }
  if (pickedSquadSessionLabels.length === 0) return base
  return {
    ...base,
    filter_mode: 'sessions',
    sessions: {
      ...(pending.sessions ?? DEFAULT_SESSIONS),
      picked_sessions: pickedSquadSessionLabels,
    },
    period: DEFAULT_PERIOD,
  }
}
