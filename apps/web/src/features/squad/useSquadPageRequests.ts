/**
 * useSquadPageRequests — les DEUX requêtes de la page Escouade et l'ancrage qui les
 * ordonne (lot perf L4b, 2026-09-23 — `.ai/PLAN_PERF_CHARGEMENTS_2026-09-23.md` §9).
 *
 * Défaut mesuré (`.ai/ETAT_DES_LIEUX_PERF_CHARGEMENTS_2026-09-23.md` §1.2 et C3) : le
 * sélecteur de sessions et l'ancrage sur la dernière session de la composition ne
 * lisaient que la réponse LOURDE (POST `/pages/teammates`). La première requête lourde
 * partait donc sur tout l'historique, puis l'ancrage la relançait sur la bonne session :
 * deux calculs de la page pour une seule page affichée.
 *
 * L'ordre désormais :
 *  1. la lecture LÉGÈRE (`useCompositionSessions`) part dès que la composition initiale
 *     est connue (verrou du lot L4a : `teammatesReady`) ;
 *  2. sa réponse nourrit le sélecteur et `decideCompositionReanchor` : snap posé dans le
 *     store, session vidée (« aucune session commune ») ou sélection manuelle encore
 *     valide respectée ;
 *  3. la requête LOURDE n'est activée qu'une fois cette décision prise pour la
 *     composition courante : elle part DÉJÀ sur la bonne session. Un clic du rail ne
 *     change pas la composition : une seule requête lourde, sans relecture légère.
 *
 * Sans coéquipier, la décision est connue d'avance (`none` : l'ancrage n'est pas piloté
 * par la composition) : la requête lourde part sans attendre. Si l'endpoint léger
 * échoue, la page retombe sur le comportement du lot L4a (sessions et ancrage lus dans la
 * réponse lourde, activée aussitôt) : cet échec ne bloque jamais la page.
 */
import { useEffect, useMemo, useRef, useState } from 'react'

import { reconcileSquadSessionLabels, stripSessionCountSuffix } from '@/lib/sessions/sessionLabels'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSquadFilterStore } from '@/stores/squadFilterStore'
import type { TeammatesQueryRequest } from '@/lib/api/types'

import { useCompositionSessions, useTeammates } from './queries'
import {
  decideCompositionReanchor,
  pickCompositionSessionsSource,
  type CompositionSessionsSource,
} from './squadPending'

export interface SquadPageRequestsInput {
  playerSlug: string
  /** Corps de la requête lourde, construit par le layout depuis le store escouade. */
  request: TeammatesQueryRequest
  filterContextHash: string
  /** Composition confirmée (coéquipiers sélectionnés). */
  selectedGts: string[]
  exactComposition: boolean
  /** État de montage posé ET composition initiale connue (`useSquadSessionSelection`). */
  teammatesReady: boolean
  /** Sessions pickées (store escouade, source unique depuis le lot L4a). */
  pickedSquadSessionLabels: string[]
  applySessionLabels: (labels: string[]) => void
}

/**
 * Applique la décision d'ancrage au store escouade : le corps de l'effet que portait
 * `SquadLayout` avant le lot L4b, inchangé — seule la source des sessions a changé.
 */
function appliquerAncrage(
  hasTeammates: boolean,
  source: CompositionSessionsSource,
  applySessionLabels: (labels: string[]) => void,
): void {
  const {
    filterContext: fc,
    isAutoSnappingToLatest,
    lastKnownLatestSessionId,
    setLastKnownLatestSessionId,
    autoSnapToLatestSession,
  } = useSquadFilterStore.getState()
  const picked = fc.sessions?.picked_sessions ?? []
  const hasPeriod = !!(fc.period?.start_date || fc.period?.end_date)
  // « follow-latest » : pas de sélection manuelle épinglée (cf. useFollowLatestSession).
  const followLatest = isAutoSnappingToLatest || (!hasPeriod && picked.length === 0)
  const action = decideCompositionReanchor({
    hasTeammates,
    followLatest,
    latestCompositionSession: source.latest,
    pickedSessions: picked,
    compositionSessionLabels: source.sessions.map((s) => s.label),
    // Ancrage déjà posé (persisté) : distingue « l'utilisateur a épinglé cette session »
    // de « une nouvelle session est arrivée depuis ».
    lastAnchoredLatestSession: lastKnownLatestSessionId ?? '',
  })
  if (action.kind === 'clear') {
    // Composition sans session commune : on vide, la page affiche l'état vide.
    applySessionLabels([])
  } else if (action.kind === 'snap') {
    autoSnapToLatestSession({ session_id: action.label, label: action.label }, true)
  } else if (source.latest && source.latest !== lastKnownLatestSessionId) {
    // Pas de snap (déjà dessus, ou sélection délibérée respectée) : on mémorise quand même
    // la dernière session vue, sinon elle resterait « jamais ancrée » et re-déclencherait
    // un snap à chaque montage.
    setLastKnownLatestSessionId(source.latest)
  }
}

export function useSquadPageRequests(input: SquadPageRequestsInput) {
  const { playerSlug, request, filterContextHash, selectedGts, exactComposition, teammatesReady } = input
  const { pickedSquadSessionLabels, applySessionLabels } = input
  const hasTeammates = selectedGts.length > 0
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const light = useCompositionSessions(playerSlug, selectedGts, exactComposition, teammatesReady)

  // La composition dont la décision d'ancrage est prise : la requête lourde n'a le droit
  // de partir que pour elle (ou sans coéquipier, ou quand l'endpoint léger a échoué).
  const compositionKey = [titleSlug, playerSlug, [...selectedGts].sort().join(','), exactComposition].join('|')
  const [decidedFor, setDecidedFor] = useState<string | null>(null)
  const heavyEnabled = teammatesReady && (!hasTeammates || light.isError || decidedFor === compositionKey)
  const heavy = useTeammates(playerSlug, request, filterContextHash, selectedGts, heavyEnabled)

  const source = useMemo(
    () =>
      pickCompositionSessionsSource(
        { data: light.data, isError: light.isError, isPlaceholderData: light.isPlaceholderData },
        { data: heavy.data, isError: heavy.isError, isPlaceholderData: heavy.isPlaceholderData },
        hasTeammates,
      ),
    [light.data, light.isError, light.isPlaceholderData, heavy.data, heavy.isError, heavy.isPlaceholderData, hasTeammates],
  )

  // Réconciliation anti-zombie des sessions pickées (suffixe « (N) » volatil, cf.
  // buildSessionLabel côté Go) : chaque label pické est remappé vers sa forme courante,
  // doublons retirés. Si TOUS sont des zombies pour la composition courante, rien : le
  // ré-ancrage ci-dessous reprend la main.
  useEffect(() => {
    if (source.sessions.length === 0 || pickedSquadSessionLabels.length === 0) return
    const reconciled = reconcileSquadSessionLabels(pickedSquadSessionLabels, source.sessions)
    if (reconciled.length === 0) return
    const unchanged =
      reconciled.length === pickedSquadSessionLabels.length &&
      reconciled.every((l, i) => l === pickedSquadSessionLabels[i])
    if (!unchanged) applySessionLabels(reconciled)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [source.sessions])

  // Ré-ancrage composition-aware : UNE fois par couple (composition, dernière session) —
  // clé sans le suffixe « (N) » volatil, pour qu'une nouvelle soirée arrivée pendant que
  // la page est montée rouvre la décision — et seulement sur une donnée FRAÎCHE (jamais le
  // placeholder d'une composition précédente). La navigation du rail et les filtres ne
  // changent ni l'une ni l'autre : pas de conflit avec une sélection délibérée.
  const lastAnchoredRef = useRef<string | null>(null)
  useEffect(() => {
    if (!source.fresh) return
    const anchorKey = `${hasTeammates ? [...selectedGts].sort().join(',') : ''}|${stripSessionCountSuffix(source.latest)}`
    if (anchorKey !== lastAnchoredRef.current) {
      lastAnchoredRef.current = anchorKey
      appliquerAncrage(hasTeammates, source, applySessionLabels)
    }
    if (source.origin === 'light') {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- feu vert de la requête lourde, posé APRÈS l'écriture de l'ancrage dans le store externe : sans cet ordre elle partirait sur la session d'avant (lot perf L4b, 2026-09-23)
      setDecidedFor(compositionKey)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [source, compositionKey])

  return { teammates: heavy, compositionSessions: source.sessions }
}
