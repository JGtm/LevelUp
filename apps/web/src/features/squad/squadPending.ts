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
// L'identite d'un label de session (suffixe « (N) » volatil) vit dans `lib/sessions` :
// deux features la lisent depuis le 2026-09-06 (Escouade et Tactique).
import { stripSessionCountSuffix } from '@/lib/sessions/sessionLabels'

const DEFAULT_SESSIONS: SessionsInput = { picked_sessions: [], gap_minutes: 120 }
const DEFAULT_PERIOD: PeriodInput = { start_date: null, end_date: null }

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
   * une sélection manuelle ENCORE valide pour la composition est respectée —
   * SAUF si une session jamais ancrée est apparue (cf. lastAnchoredLatestSession).
   */
  followLatest: boolean
  /** Dernière session de la composition (back-end), '' si jamais jouée ensemble. */
  latestCompositionSession: string
  /** Sessions actuellement pickées (filterContext.sessions.picked_sessions). */
  pickedSessions: string[]
  /** Labels des sessions de la composition courante (validité d'une sélection manuelle). */
  compositionSessionLabels: string[]
  /**
   * Dernière session sur laquelle l'ancrage a DÉJÀ été posé (persistée dans le
   * store squad, `lastKnownLatestSessionId`), '' si aucun ancrage connu.
   *
   * Clé de détection « une nouvelle session est arrivée » — sans elle, une
   * sélection épinglée gèle la page sur une session périmée : `followLatest`
   * est faux dès que N'IMPORTE quel chemin technique a appelé setSessions /
   * setFilterContext (bouton Analyser, resync mount, réconciliation anti-zombie
   * des suffixes « (N) »), et pas seulement sur un choix délibéré.
   */
  lastAnchoredLatestSession: string
}

/**
 * Décide du ré-ancrage de session pour la composition exacte courante
 * (joueur principal + coéquipiers sélectionnés).
 *
 *  - aucun coéquipier → 'none' (l'ancrage n'est pas piloté par la composition) ;
 *  - sélection MANUELLE (followLatest=false) encore valide pour la composition ET
 *    dernière session déjà ancrée → 'none' (on respecte le choix, ex. session
 *    restaurée au reload) ;
 *  - dernière session de la composition JAMAIS ancrée (nouvelle session arrivée
 *    depuis le dernier atterrissage) → on ré-ancre, même sur sélection manuelle :
 *    c'est le sens de l'autosnap escouade (atterrir sur la dernière soirée jouée) ;
 *  - latest vide (composition jamais jouée ensemble) → 'clear' si une session est
 *    pickée (vider pour afficher l'état vide), sinon 'none' ;
 *  - déjà ancré sur la dernière (comparaison par clé SANS le suffixe « (N) »
 *    volatil) → 'none' ;
 *  - sinon → 'snap' sur la dernière session de la composition.
 */
export function decideCompositionReanchor(input: CompositionReanchorInput): CompositionReanchorAction {
  const {
    hasTeammates,
    followLatest,
    latestCompositionSession,
    pickedSessions,
    compositionSessionLabels,
    lastAnchoredLatestSession,
  } = input
  if (!hasTeammates) return { kind: 'none' }

  const stillValid =
    pickedSessions.length > 0 &&
    pickedSessions.every((p) =>
      compositionSessionLabels.some((l) => stripSessionCountSuffix(l) === stripSessionCountSuffix(p)),
    )
  // Comparaison par clé sans le suffixe « (N) » : ce compte grossit à chaque sync
  // sur une session en cours et ne dénote donc pas une session différente.
  const latestKey = stripSessionCountSuffix(latestCompositionSession)
  const latestAlreadyAnchored =
    latestKey !== '' && stripSessionCountSuffix(lastAnchoredLatestSession) === latestKey
  if (!followLatest && stillValid && latestAlreadyAnchored) return { kind: 'none' }

  if (!latestCompositionSession) {
    return pickedSessions.length > 0 ? { kind: 'clear' } : { kind: 'none' }
  }
  const alreadyOnLatest =
    pickedSessions.length === 1 && stripSessionCountSuffix(pickedSessions[0]) === latestKey
  return alreadyOnLatest ? { kind: 'none' } : { kind: 'snap', label: latestCompositionSession }
}

/**
 * Fusionne les compteurs de sessions affichés par le sélecteur de sessions.
 *
 * Règle canonique du contexte escouade : le nombre affiché est le compte
 * « commencés ensemble » servi par teammates (`composition_sessions.match_count`,
 * population du roster) — exactement la population des tableaux et graphes.
 * Les counts de `/filters/resolve` (population du joueur principal, cascade
 * seule) ne servent que de repli tant que la réponse teammates n'est pas
 * arrivée : c'est cette double source qui affichait 11/8/6/5 pour une session.
 */
export function mergeSessionCounts(
  fallback: { label: string; match_count_filtered: number }[],
  compositionSessions: { label: string; match_count?: number }[],
): Map<string, number> {
  const map = new Map<string, number>()
  for (const s of fallback) map.set(s.label, s.match_count_filtered)
  for (const s of compositionSessions) {
    // 0/undefined = producteur qui ne renseigne pas le compte : on garde le repli
    // plutôt que d'afficher une session à zéro (qui serait masquée à tort).
    if (typeof s.match_count === 'number' && s.match_count > 0) map.set(s.label, s.match_count)
  }
  return map
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
