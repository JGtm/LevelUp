/**
 * squadPending — dérivation pure de FilterContextInput pour le preview Escouade.
 *
 * La barre Escouade combine deux états :
 *  - `pending` (FilterContextInput, `useSquadFilterBarState`) : période + cascade,
 *    commité via Analyser.
 *  - `pickedSquadSessionLabels` (string[]) : multi-sélection sessions, appliquée
 *    immédiatement (sans Analyser) dans le store escouade — sa source UNIQUE depuis
 *    le 2026-09-23 (lot perf L4a, D4.1 : plus d'état local ni de miroir localStorage).
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
import type {
  CompositionSessionsResponse,
  FilterContextInput,
  PeriodInput,
  SessionLabelEntry,
  SessionsInput,
  TeammatesPageResponse,
} from '@/lib/api/types'
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

/** Ce qu'une requête TanStack Query expose et dont la source des sessions a besoin. */
interface QueryView<T> {
  data?: T
  isError: boolean
  isPlaceholderData: boolean
}

/**
 * Les sessions de la composition et la dernière d'entre elles, d'où qu'elles viennent :
 * de quoi nourrir le sélecteur de sessions ET `decideCompositionReanchor`.
 */
export interface CompositionSessionsSource {
  /** 'light' : GET `/pages/teammates/sessions` ; 'heavy' : repli sur POST `/pages/teammates`. */
  origin: 'light' | 'heavy'
  sessions: SessionLabelEntry[]
  /** Dernière session de la composition, '' si jamais jouée ensemble (ou sans coéquipier). */
  latest: string
  /** Donnée de la composition COURANTE : ni absente, ni placeholder d'une clé précédente. */
  fresh: boolean
  /** Identité de la donnée lue : l'ancrage se rejoue quand elle change. */
  data: unknown
}

/**
 * Choisit la source des sessions (lot perf L4b, 2026-09-23) : la réponse LÉGÈRE dès
 * qu'elle a une donnée (placeholder compris : il garde la sélection visible pendant le
 * chargement de la composition suivante, comme la réponse lourde le faisait), la
 * réponse LOURDE sinon — endpoint léger en échec (repli : exactement la lecture du lot
 * L4a) ou pas encore arrivé. Les champs sont les mêmes des deux côtés (parité testée
 * côté serveur), à une nuance près, héritée : sans coéquipier, la réponse lourde se lit
 * dans `session_labels.squad`, la légère dans `composition_sessions`.
 */
export function pickCompositionSessionsSource(
  light: QueryView<CompositionSessionsResponse>,
  heavy: QueryView<TeammatesPageResponse>,
  hasTeammates: boolean,
): CompositionSessionsSource {
  if (light.data !== undefined && !light.isError) {
    return {
      origin: 'light',
      sessions: light.data.composition_sessions ?? [],
      latest: light.data.latest_composition_session ?? '',
      fresh: !light.isPlaceholderData,
      data: light.data,
    }
  }
  const data = heavy.data
  return {
    origin: 'heavy',
    sessions: (hasTeammates ? data?.composition_sessions : data?.session_labels?.squad) ?? [],
    latest: data?.latest_composition_session ?? '',
    fresh: data !== undefined && !heavy.isPlaceholderData,
    data,
  }
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
