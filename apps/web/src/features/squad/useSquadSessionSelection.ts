/**
 * useSquadSessionSelection — la COMPOSITION et la SESSION de la page Escouade, et le
 * moment où sa requête lourde (POST `/pages/teammates`) a le droit de partir.
 *
 * Extrait de `SquadLayout` le 2026-09-23 (campagne perf, lot L4a —
 * `.ai/PLAN_PERF_CHARGEMENTS_2026-09-23.md` §4). Défaut mesuré le même jour
 * (`.ai/ETAT_DES_LIEUX_PERF_CHARGEMENTS_2026-09-23.md` §1.2 et C3) : un premier
 * passage envoyait une requête SANS coéquipier (la composition arrive par
 * `GET /friends`), puis chaque snap ou clic du rail en envoyait DEUX (une
 * intermédiaire jamais affichée : 8,2 s + 26,8 s par clic). Deux règles :
 *
 *  1. SOURCE UNIQUE de la session pickée (D4.1) : `useSquadFilterStore`,
 *     `filterContext.sessions.picked_sessions`. Plus d'état local ni de miroir
 *     localStorage `squad-sessions-<slug>` : `applySessionLabels` EST `setSessions`,
 *     et la clé de query teammates n'a plus de segment « sessions » (le hash du
 *     filterContext les couvre). Un snap ou un clic du rail change donc UNE clé.
 *     L'ancienne clé locale est migrée une fois au montage (appliquée si le store
 *     est vide), puis retirée.
 *  2. PAS DE REQUÊTE À VIDE (D4.2) : `teammatesReady` n'est vrai qu'une fois
 *     (a) l'état de montage posé dans le store (lien profond de l'accueil,
 *     migration ci-dessus) et (b) la composition initiale connue — coéquipiers
 *     restaurés ou imposés par le lien profond, choix de l'utilisateur, ou liste
 *     d'amis RÉSOLUE et vide (exploration sans coéquipier : requête légitime).
 *     Une lecture de `/friends` en échec vaut liste vide (comportement historique
 *     de `useFriendGamertags`) : sans cela la page resterait sur « Chargement… ».
 */
import { useEffect, useState } from 'react'
import { useSearch } from '@tanstack/react-router'

import { useFriendGamertagsState } from '@/features/friends/queries'
import { DEFAULT_GAP_MINUTES } from '@/stores/filterDefaults'
import { useSquadFilterStore } from '@/stores/squadFilterStore'

import { MAX_SELECTION } from './colors'

/** Référence STABLE : un `?? []` neuf à chaque rendu casserait les mémos aval. */
const NO_SESSIONS: string[] = []

type TeammatesUpdate = string[] | ((prev: string[]) => string[])

export interface SquadSessionSelection {
  /** Composition sélectionnée (appliquée en direct, persistée par joueur). */
  selectedGts: string[]
  /** Setter du combobox et des presets : un CHOIX de l'utilisateur. */
  setSelectedGts: (next: TeammatesUpdate) => void
  /** Sessions pickées, lues dans le store escouade (source unique). */
  pickedSquadSessionLabels: string[]
  /** Pose les sessions pickées dans le store (= `setSessions`). */
  applySessionLabels: (labels: string[]) => void
  /** L'état de montage (lien profond, migration de l'ancienne clé) est posé dans le store. */
  mountApplied: boolean
  /** La requête teammates peut partir : montage posé ET composition initiale connue. */
  teammatesReady: boolean
}

interface SquadDeepLink {
  session: string
  teammates: string[]
}

/** Lien profond de l'accueil (card session escouade) : `?session=…&teammates=a,b`. */
function readDeepLink(search: { session?: string; teammates?: string }): SquadDeepLink | null {
  if (!search.session) return null
  return {
    session: search.session,
    teammates: (search.teammates ?? '')
      .split(',')
      .map((g) => g.trim())
      .filter(Boolean),
  }
}

function readStoredTeammates(key: string): string[] {
  try {
    const stored = localStorage.getItem(key)
    return stored ? (JSON.parse(stored) as string[]) : []
  } catch {
    return []
  }
}

function writeStoredTeammates(key: string, value: string[]): void {
  try {
    localStorage.setItem(key, JSON.stringify(value))
  } catch {
    /* ignore */
  }
}

/**
 * Le lien profond de l'accueil, capturé UNE fois au montage (état, donc lisible au
 * rendu), AVANT le redirect index → /squad/synergies qui drop la query.
 *
 * Deux lecteurs, dans le même rendu de `SquadLayout` : cette sélection (composition et
 * session posées au montage) et `useSquadPageRequests`, pour qui la session du lien ne
 * vaut que jusqu'au premier ancrage de sa composition (lot perf L9-web, 2026-09-23).
 */
export function useSquadDeepLink(): SquadDeepLink | null {
  const search = useSearch({ strict: false }) as { session?: string; teammates?: string }
  const [deepLink] = useState(() => readDeepLink(search))
  return deepLink
}

/**
 * Lit PUIS retire l'ancienne clé `squad-sessions-<slug>` (miroir local des sessions
 * pickées, supprimé par D4.1). Retirée dans tous les cas : le store est désormais la
 * seule source, la clé ne doit plus jamais être relue.
 */
function takeLegacySessionLabels(playerSlug: string): string[] {
  const key = `squad-sessions-${playerSlug}`
  try {
    const stored = localStorage.getItem(key)
    if (stored === null) return []
    localStorage.removeItem(key)
    const parsed: unknown = JSON.parse(stored)
    return Array.isArray(parsed) ? parsed.filter((l): l is string => typeof l === 'string') : []
  } catch {
    return []
  }
}

/**
 * État de montage, posé dans le store AVANT toute requête : lien profond (session
 * épinglée) puis, à défaut, migration unique de l'ancienne clé locale. Ne rend
 * vrai qu'après ces écritures : la première requête (teammates comme résolu
 * escouade) part donc sur le contexte final.
 */
function useSquadMountState(
  playerSlug: string,
  deepLink: SquadDeepLink | null,
  teammatesKey: string,
): boolean {
  const [mountApplied, setMountApplied] = useState(false)
  useEffect(() => {
    const legacy = takeLegacySessionLabels(playerSlug)
    const { filterContext, setSessions } = useSquadFilterStore.getState()
    const gapMinutes = filterContext.sessions?.gap_minutes ?? DEFAULT_GAP_MINUTES
    if (deepLink) {
      writeStoredTeammates(teammatesKey, deepLink.teammates)
      setSessions({ picked_sessions: [deepLink.session], gap_minutes: gapMinutes })
    } else if (legacy.length > 0 && (filterContext.sessions?.picked_sessions?.length ?? 0) === 0) {
      setSessions({ picked_sessions: legacy, gap_minutes: gapMinutes })
    }
    // eslint-disable-next-line react-hooks/set-state-in-effect -- verrou de montage : ouvre les requêtes APRÈS l'écriture du lien profond et de la migration dans le store externe, sans quoi la première partirait sur un contexte aussitôt remplacé (2026-09-23)
    setMountApplied(true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []) // mount-only
  return mountApplied
}

export function useSquadSessionSelection(playerSlug: string): SquadSessionSelection {
  const deepLink = useSquadDeepLink()
  const teammatesKey = `squad-teammates-${playerSlug}`

  // ── Composition ────────────────────────────────────────────────────────
  // Le lien profond impose la composition DÈS le premier rendu (pas d'effet) :
  // aucune requête ne peut partir avec la composition restaurée qu'il remplace.
  const [selectedGts, setSelectedGtsRaw] = useState<string[]>(() =>
    deepLink ? deepLink.teammates : readStoredTeammates(teammatesKey),
  )
  const [userChoseTeammates, setUserChoseTeammates] = useState(false)
  const commitTeammates = (next: TeammatesUpdate) => {
    setSelectedGtsRaw((prev) => {
      const value = typeof next === 'function' ? next(prev) : next
      writeStoredTeammates(teammatesKey, value)
      return value
    })
  }
  const setSelectedGts = (next: TeammatesUpdate) => {
    // Vider la composition est un choix : la requête « sans coéquipier » est alors
    // légitime même si le joueur a des amis (sinon la page resterait figée).
    setUserChoseTeammates(true)
    commitTeammates(next)
  }

  // Init depuis la liste d'amis, neutralisée en arrivée par lien profond (la
  // composition est alors imposée par la session, pas par les amis du joueur).
  const friends = useFriendGamertagsState(playerSlug)
  useEffect(() => {
    if (deepLink) return
    if (friends.gamertags.length && selectedGts.length === 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- init de la composition à l'arrivée async de la liste d'amis (garde deep-link + sélection vide) (2026-07-22)
      commitTeammates(friends.gamertags.slice(0, MAX_SELECTION))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [friends.gamertags])

  // ── Session pickée : le store escouade, et lui seul (D4.1) ──────────────
  const storedPicked = useSquadFilterStore((s) => s.filterContext.sessions?.picked_sessions)
  const pickedSquadSessionLabels = storedPicked ?? NO_SESSIONS
  const applySessionLabels = (labels: string[]) => {
    const { filterContext, setSessions } = useSquadFilterStore.getState()
    setSessions({
      picked_sessions: labels,
      gap_minutes: filterContext.sessions?.gap_minutes ?? DEFAULT_GAP_MINUTES,
    })
  }

  const mountApplied = useSquadMountState(playerSlug, deepLink, teammatesKey)

  const friendsSettledEmpty =
    (friends.isSuccess || friends.isError) && friends.gamertags.length === 0
  const compositionKnown =
    deepLink !== null || selectedGts.length > 0 || userChoseTeammates || friendsSettledEmpty

  return {
    selectedGts,
    setSelectedGts,
    pickedSquadSessionLabels,
    applySessionLabels,
    mountApplied,
    teammatesReady: mountApplied && compositionKnown,
  }
}
