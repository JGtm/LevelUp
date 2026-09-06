/**
 * useZoneStates — CE QUE LE CALQUE VIVANT DES ZONES A BESOIN DE SAVOIR, résolu une fois.
 *
 * POURQUOI UN HOOK PLUTÔT QUE DEUX `useMemo` DANS LE CANVAS. `ReplayCanvas.tsx` est sous
 * plafond de taille (garde-rail `max-lines` eslint, R5), et la règle du dépôt est de ne
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
 * `usePlacementHover` porte un `useState` qui fait rendre le canvas. Les membres sont déjà
 * stables : leur enveloppe doit l'être aussi.
 *
 * LA JOINTURE EST VÉRIFIÉE ICI (revue R1-7). `zoneStates[].zoneRef` indexe la liste que
 * l'artefact avait sous les yeux à la CUISSON ; `mapObjectives` est reconstruit à la requête.
 * `coverage.zones.catalog` dit combien de zones l'artefact comptait : s'il diffère de la liste
 * servie, `joinable` est faux et le calque vivant se tait (cf. `zoneCatalogMatches`).
 *
 * LA TENUE DE LA JAUGE EN DIRECT (schéma 18) SE CONVERTIT ICI, une fois par document : déclarée
 * en temps réel (`ZONE_GAUGE_HOLD_MS`), jamais en nombre d'images — la cadence du film peut
 * changer au build sans que la lecture change (même règle que `useReplayTiming`).
 */
import { useCallback, useMemo } from 'react'

import type { MatchScoreboardRow } from '@/lib/api/types'

import type { ObjectiveElementReady } from './objectivesLayer'
import { msToFrames } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import {
  ZONE_GAUGE_HOLD_MS,
  zoneCatalogMatches,
  zoneElementsOf,
  type ZoneStatesLayerInput,
} from './zoneStatesLayer'
import { allyTeamFromScoreboard } from '../model/matchSides'

/** Ce que le canvas recopie tel quel dans ses appels de dessin. */
export interface ReplayZoneStates extends ZoneStatesLayerInput {
  /**
   * Couleur d'un index d'équipe pour le calque STATIQUE et les pulses : celle que
   * L'UTILISATEUR a réglée (`team-ally` / `team-enemy`), la MÊME que les fiches, le fil et les
   * points des joueurs — jamais le bleu et le rouge officiels du jeu (cf. `colorOfTeam` plus
   * bas). Encre neutre pour -1, et pour tout camp quand aucune ligne « moi » ne permet de
   * situer les camps. `team` est DÉJÀ arbitré côté serveur (Bastion = neutre).
   */
  colorOfTeam: (team: number) => string
}

export function useZoneStates(
  objectives: readonly ObjectiveElementReady[],
  scoreboard: MatchScoreboardRow[] | null | undefined,
  teamColorOf: (isAlly: boolean) => string,
  /**
   * L'encre du « AUCUN CAMP » : objectif neutre du catalogue, et zone que personne ne tient.
   *
   * C'EST UN TOKEN SÉMANTIQUE, pas une encre de mise en page (`useReplayInks.neutral`,
   * `divergent-neutral` — le même neutre que le fil emploie pour une mort que personne ne
   * revendique). Le calque servait `floor.edge` jusqu'au 2026-08-25, c'est-à-dire
   * `--muted-foreground` : une variable de LAYOUT employée pour dire un fait de jeu, ce que
   * `canvasInk.ts` s'interdit lui-même. Elle ne suivait pas non plus la palette
   * d'accessibilité de l'utilisateur.
   */
  neutral: string,
  /** Le document : `coverage.zones.catalog` (jointure) et sa cadence (tenue de la jauge). */
  doc: ReplayDocumentReady,
): ReplayZoneStates {
  const zoneElements = useMemo(() => zoneElementsOf(objectives), [objectives])
  const joinable = zoneCatalogMatches(doc.coverage?.zones?.catalog, zoneElements.length)
  const gaugeHoldFrames = useMemo(() => msToFrames(ZONE_GAUGE_HOLD_MS, doc), [doc])
  const allyTeamID = useMemo(() => allyTeamFromScoreboard(scoreboard), [scoreboard])
  /**
   * L'ENCRE D'UN CAMP VIENT DES RÉGLAGES DE L'UTILISATEUR, plus du référentiel du jeu
   * (retour du 2026-08-26 : « le socle de l'équipe est en bleu alors que j'utilise une
   * couleur verte »).
   *
   * CE QUI SE PASSAIT : ce calque appelait `resolveTeamColorFromID`, c'est-à-dire la table
   * `TEAM_COLORS_HALO_INFINITE` — le bleu et le rouge OFFICIELS du jeu, écrits en dur. Les
   * fiches, le fil et les points des joueurs, eux, passent tous par `teamColorOf`
   * (`team-ally` / `team-enemy`, tokens que la palette d'accessibilité de l'utilisateur
   * surcharge). Les objectifs parlaient donc seuls une autre langue que le reste de la page.
   *
   * LA PAGE PARLE D'UNE SEULE VOIX (décision D1) : le camp se dit par son RAPPORT au joueur
   * de la page — allié ou adverse — dans les couleurs qu'il a choisies. `resolveTeamColorFromID`
   * n'est donc plus lu ici.
   *
   * SANS LIGNE « MOI », AUCUN CAMP N'EST SITUABLE, et rien n'est deviné : l'encre neutre
   * sert alors, comme pour une zone que personne ne tient (même règle que `colorOfOwner`).
   */
  const colorOfTeam = useCallback(
    (team: number) => (team >= 0 && allyTeamID !== null ? teamColorOf(team === allyTeamID) : neutral),
    [allyTeamID, teamColorOf, neutral],
  )
  const style = useMemo(
    () => ({
      colorOfOwner: (team: number) =>
        allyTeamID === null ? null : teamColorOf(team === allyTeamID),
      // Le camp QUI CAPTURE une zone tenue est le camp d'en face (cf. ZoneStateStyle) : allié
      // si le propriétaire est adverse, adverse s'il est allié.
      colorOfCapturer: (owner: number) =>
        allyTeamID === null ? null : teamColorOf(owner !== allyTeamID),
      neutral,
    }),
    [allyTeamID, teamColorOf, neutral],
  )
  return useMemo(
    () => ({ zoneElements, joinable, style, colorOfTeam, gaugeHoldFrames }),
    [zoneElements, joinable, style, colorOfTeam, gaugeHoldFrames],
  )
}
