/**
 * useZoneStates — CE QUE LE CALQUE VIVANT DES ZONES A BESOIN DE SAVOIR, résolu une fois.
 *
 * POURQUOI UN HOOK PLUTÔT QUE DEUX `useMemo` DANS LE CANVAS. `ReplayCanvas.tsx` est sous
 * plafond de taille (garde-rail `placementFamily.guard.test.ts`), et la règle du dépôt est de ne
 * pas accroître cette dette : chaque calque y garde UNE ligne, son détail vit à côté de sa
 * donnée. Même partage que `useReplayWeaponPads` et `useReplayStaticLayers`.
 *
 * LE CAMP ALLIÉ EST UN NUMÉRO ICI, PAS UN XUID, et c'est ce qui rend ce hook nécessaire. Le
 * propriétaire d'une zone vient du film (`team_id` du registre, valeur du canal de propriété) ;
 * le point de vue de la page, lui, se lit sur la ligne « moi » du tableau de bord, dont le
 * `team_side` est écrit `t{N}`. Sans cette ligne, AUCUN camp n'est allié — et la zone garde son
 * encre neutre plutôt qu'une couleur devinée (même règle que `xuidMeta`).
 *
 * LE RETOUR EST MÉMOÏSÉ, ET CE N'EST PAS DU CONFORT (revue R1, 2026-08-18). L'objet entre dans
 * les dépendances de `draw` chez l'appelant ; un littéral neuf à chaque rendu recuisait donc le
 * `useCallback` du tracé — c'est-à-dire TOUTE la scène — à chaque mouvement de pointeur, puisque
 * `usePlacementHover` porte un `useState` qui fait rendre le canvas. Les trois membres sont déjà
 * stables : leur enveloppe doit l'être aussi.
 */
import { useCallback, useMemo } from 'react'

import type { MatchScoreboardRow } from '@/lib/api/types'
import { parseTeamSideID, resolveTeamColorFromID } from '@/lib/halo/teamNames'

import type { ObjectiveElementReady } from './objectivesLayer'
import { zoneElementsOf, type ZoneStateStyle } from './zoneStatesLayer'

/** Ce que le canvas recopie tel quel dans ses appels de dessin. */
export interface ReplayZoneStates {
  /** Les zones SURFACIQUES dans l'ordre servi : celui que `zoneStates[].zoneRef` indexe. */
  zoneElements: ObjectiveElementReady[]
  style: ZoneStateStyle
  /**
   * Couleur d'un index d'équipe pour le calque STATIQUE et les pulses : le référentiel
   * d'identité du jeu (donnée de domaine, pas un choix d'UI), encre neutre du thème pour -1 ou
   * un index hors référentiel. `team` y est DÉJÀ arbitré côté serveur (Bastion = neutre).
   */
  colorOfTeam: (team: number) => string
}

export function useZoneStates(
  objectives: readonly ObjectiveElementReady[],
  scoreboard: MatchScoreboardRow[] | null | undefined,
  teamColorOf: (isAlly: boolean) => string,
  neutral: string,
): ReplayZoneStates {
  const zoneElements = useMemo(() => zoneElementsOf(objectives), [objectives])
  const colorOfTeam = useCallback(
    (team: number) => (team >= 0 ? resolveTeamColorFromID(team) : null) ?? neutral,
    [neutral],
  )
  const allyTeamID = useMemo(
    () => parseTeamSideID(scoreboard?.find((r) => r.is_me)?.team_side ?? null),
    [scoreboard],
  )
  const style = useMemo(
    () => ({
      colorOfOwner: (team: number) =>
        allyTeamID === null ? null : teamColorOf(team === allyTeamID),
      neutral,
    }),
    [allyTeamID, teamColorOf, neutral],
  )
  return useMemo(
    () => ({ zoneElements, style, colorOfTeam }),
    [zoneElements, style, colorOfTeam],
  )
}
