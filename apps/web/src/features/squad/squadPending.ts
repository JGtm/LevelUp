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

/** Action de ré-ancrage de session décidée quand la composition d'escouade change. */
export type CompositionReanchorAction =
  | { kind: 'none' }
  | { kind: 'clear' }
  | { kind: 'snap'; label: string }

/** Entrée de `decideCompositionReanchor` (regroupée — au-delà de 3 params utiles). */
export interface CompositionReanchorInput {
  hasTeammates: boolean
  /**
   * « suit la dernière » : pas de sélection manuelle épinglée
   * (isAutoSnappingToLatest OU ni période ni session pickée). Quand false,
   * une sélection manuelle ENCORE valide pour la composition est respectée.
   */
  followLatest: boolean
  /** Dernière session de la composition (back-end), '' si jamais jouée ensemble. */
  latestCompositionSession: string
  /** Sessions actuellement pickées (filterContext.sessions.picked_sessions). */
  pickedSessions: string[]
  /** Labels des sessions de la composition courante (validité d'une sélection manuelle). */
  compositionSessionLabels: string[]
}

/**
 * Décide du ré-ancrage de session pour la composition exacte courante
 * (joueur principal + coéquipiers sélectionnés).
 *
 *  - aucun coéquipier → 'none' (l'ancrage n'est pas piloté par la composition) ;
 *  - sélection MANUELLE (followLatest=false) encore valide pour la composition →
 *    'none' (on respecte le choix, ex. session restaurée au reload) ;
 *  - latest vide (composition jamais jouée ensemble) → 'clear' si une session est
 *    pickée (vider pour afficher l'état vide), sinon 'none' ;
 *  - déjà ancré sur la dernière (comparaison par clé SANS le suffixe « (N) »
 *    volatil) → 'none' ;
 *  - sinon → 'snap' sur la dernière session de la composition.
 */
export function decideCompositionReanchor(input: CompositionReanchorInput): CompositionReanchorAction {
  const { hasTeammates, followLatest, latestCompositionSession, pickedSessions, compositionSessionLabels } = input
  if (!hasTeammates) return { kind: 'none' }

  const stillValid =
    pickedSessions.length > 0 &&
    pickedSessions.every((p) =>
      compositionSessionLabels.some((l) => stripSessionCountSuffix(l) === stripSessionCountSuffix(p)),
    )
  if (!followLatest && stillValid) return { kind: 'none' }

  if (!latestCompositionSession) {
    return pickedSessions.length > 0 ? { kind: 'clear' } : { kind: 'none' }
  }
  const alreadyOnLatest =
    pickedSessions.length === 1 &&
    stripSessionCountSuffix(pickedSessions[0]) === stripSessionCountSuffix(latestCompositionSession)
  return alreadyOnLatest ? { kind: 'none' } : { kind: 'snap', label: latestCompositionSession }
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
