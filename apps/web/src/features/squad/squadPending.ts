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
import type { FilterContextInput, SessionsInput, PeriodInput, SessionLabelEntry } from '@/lib/api/types'

const DEFAULT_SESSIONS: SessionsInput = { picked_sessions: [], gap_minutes: 120 }
const DEFAULT_PERIOD: PeriodInput = { start_date: null, end_date: null }

/**
 * Retire le suffixe " (N)" (match-count figé au sync, cf. buildSessionLabel
 * côté Go) d'un label de session pour obtenir une clé d'identité stable.
 */
export function stripSessionCountSuffix(label: string): string {
  return label.replace(/\s*\(\d+\)\s*$/, '').trim()
}

/**
 * Réconcilie les labels de sessions pickés contre la liste de sessions courante.
 *
 * Les labels backend embarquent un suffixe " (N)" qui change au gré des syncs.
 * Un label persisté en localStorage avec un ancien compte devient un "zombie" :
 * compté par le rail mais sans case à cocher correspondante (donc indécochable)
 * et filtré à 0 match côté backend. On remappe chaque label pické vers sa forme
 * courante (matching sur la clé sans suffixe) et on droppe les zombies
 * introuvables + les doublons. L'ordre des labels valides est préservé.
 */
export function reconcileSquadSessionLabels(
  picked: string[],
  sessions: SessionLabelEntry[],
): string[] {
  if (picked.length === 0 || sessions.length === 0) return picked
  const currentByKey = new Map(sessions.map((s) => [stripSessionCountSuffix(s.label), s.label]))
  const reconciled: string[] = []
  for (const label of picked) {
    const current = currentByKey.get(stripSessionCountSuffix(label))
    if (current && !reconciled.includes(current)) reconciled.push(current)
  }
  return reconciled
}

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
