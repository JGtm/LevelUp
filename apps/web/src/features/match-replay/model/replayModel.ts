/**
 * replayModel — LA JOINTURE DE L'ARTEFACT ET DE LA VUE MATCH, ÉCRITE UNE FOIS.
 *
 * DEUX SOURCES, DEUX RÔLES (le commentaire que la route portait, et qui reste vrai). Le FILM
 * porte ce qui s'est passé et l'identifie par XUID ; la BASE porte qui sont les gens.
 * L'artefact ne mélange pas les deux — la jointure se fait ici, sur le xuid, la seule clé qui
 * ne suppose rien (surtout pas un ordre).
 *
 * POURQUOI CE MODULE EXISTE (registre 2026-09-05, W1). La page de rejeu tenait cette jointure
 * dans une douzaine de `useMemo` empilés au sommet de sa route : chacun juste, aucun testable
 * sans monter React et un routeur, et l'ordre des dépendances entre eux (l'horloge avant la
 * fenêtre, la fenêtre avant la présence, la présence avant le fil) n'était lisible qu'en
 * suivant les tableaux de dépendances à l'oeil. La jointure est désormais une fonction PURE,
 * `buildReplayModel`, que `useReplayModel` se contente de mémoïser.
 *
 * L'ORDRE DES ÉTAGES EST UNE CONTRAINTE, PAS UNE MISE EN PAGE :
 *
 *   1. le SCOREBOARD, d'où sortent l'identité (`identity`) et les marques (`marks`) ;
 *   2. l'HORLOGE (`clock`), qui ne dépend que de l'artefact et du countdown ;
 *   3. la FENÊTRE de gameplay (`window`), qui exige l'horloge ET la durée jouable ;
 *   4. le ROSTER (`players`), jointure du film et du scoreboard ;
 *   5. le FIL (`feed`), qui exige les quatre précédents — c'est lui qui porte le recalage,
 *      ASSEMBLÉ ICI ET NULLE PART AILLEURS (cf. `killFeedLogic`, et le garde-rail de J2).
 *
 * CE QUI N'EST PAS DANS LE MODÈLE, ET POURQUOI. Le fond de carte et les zones nommées viennent
 * de trois AUTRES requêtes, pas de la vue match : ils ne relèvent pas de cette jointure-ci. Le
 * son de fin de partie et la cascade de couleur d'équipe dépendent de la LANGUE et de
 * l'affichage : ils restent à la page.
 *
 * Zéro dépendance React : testable en pur (`replayModel.test.ts`).
 */
import { collectKillEvents, type KillEvent } from '@/features/match-view/_momentum'
import { meXUIDOf, resolveXuidMeta, type XuidMeta } from '@/features/match-view/xuidMeta'
import type { MatchScoreboardRow, MatchViewResponse } from '@/lib/api/types'

import {
  buildFeedEntries,
  collectMedalEvents,
  type MedalEvent,
  type ReplayFeedEntry,
} from './killFeedLogic'
import { buildPlayerMarks, type PlayerMarkKind } from '../../../lib/replay/playerMarks'
import { mergeFeedWithPresence, presenceEntries } from './presenceFeed'
import { buildReplayMedia } from './replayMediaLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayMediaItem } from './replayTimelineTracksLogic'
import { replayWindow, type ReplayWindowBounds } from './replayWindow'
import { buildPlayers, type ReplayPlayer } from '../../../lib/replay/rosterLogic'
import { finalScoreFromHeader, type FinalScoreReading } from './victoryLogic'
import { replayClock, type ReplayClock } from './replayClock'

/**
 * Ce que le modèle lit des RÉGLAGES : la liste d'amis du compte connecté, et rien d'autre.
 * Décrit structurellement, pour que ce module ne dépende pas de la feature des réglages.
 */
export interface ReplayModelSettings {
  friend_gamertags?: string[] | null
}

/** La page de rejeu, jointe. Tous les champs sont dérivés — rien n'est mutable ici. */
export interface ReplayModel {
  /** Le tableau de score du match, jamais nul : `[]` quand la vue match n'est pas là. */
  scoreboard: MatchScoreboardRow[]
  /** Camp et gamertag de chaque xuid — la cascade « allié = même camp que moi ». */
  identity: XuidMeta
  /** Marques d'identité (moi, ami) par xuid : forme du point sur la carte, glyphe au fil. */
  marks: ReadonlyMap<string, PlayerMarkKind>
  /**
   * Le roster du film joint au scoreboard. Consommé ici par les lignes d'entrée/sortie ;
   * `ReplayTeams` en rebâtit encore un pour son propre usage — même fonction pure, mêmes
   * entrées, donc même résultat (mesuré, V-WEB-1 constat 2) : sa migration suit le lot des
   * canoniques.
   */
  players: ReplayPlayer[]
  /** L'horloge de la page, ou `null` : un seul verdict pour toutes les surfaces (P0-5). */
  clock: ReplayClock | null
  /** La fenêtre de gameplay sur l'axe du film, ou `null` (film entier, horloge du film). */
  window: ReplayWindowBounds | null
  /** Le fil recalé, lignes de présence fusionnées, trié sur l'axe du rejeu. */
  feed: ReplayFeedEntry[]
  /** Les captures du match posées sur l'axe du rejeu ; vide sans horloge établie. */
  media: ReplayMediaItem[]
  /** Le score FINAL tel que la vue match l'affiche, ou `null` — jamais déduit du film. */
  score: FinalScoreReading | null
  /** Le countdown d'avant-match, en ms (0 = inconnu) : l'ancre de `event_time_ms`. */
  t0Ms: number
}

/** Le modèle d'une page sans donnée : toutes les portes fermées, aucune valeur inventée. */
const VIDE: ReplayModel = {
  scoreboard: [],
  identity: new Map(),
  marks: new Map(),
  players: [],
  clock: null,
  window: null,
  feed: [],
  media: [],
  score: null,
  t0Ms: 0,
}

/**
 * buildReplayModel joint l'artefact et la vue match.
 *
 * SANS ARTEFACT, LE MODÈLE EST VIDE et non partiel : la page ne monte alors ni carte, ni fil,
 * ni fiches — elle dit « pas de rejeu » — et servir un demi-modèle inviterait un futur
 * consommateur à le lire quand même. La vue match, elle, peut manquer SEULE : le film reste
 * lisible sans elle, simplement sans noms, sans camps et sans fil.
 */
export function buildReplayModel(
  doc: ReplayDocumentReady | null | undefined,
  matchView: MatchViewResponse | null | undefined,
  settings?: ReplayModelSettings | null,
): ReplayModel {
  if (!doc) return VIDE
  const header = matchView?.header
  const scoreboard = matchView?.team_tab.scoreboard ?? []
  const identity = resolveXuidMeta(scoreboard, meXUIDOf(scoreboard))
  const marks = buildPlayerMarks(scoreboard, settings?.friend_gamertags ?? [])

  // LES DEUX HORLOGES NE COÏNCIDENT PAS : cf. `killFeedLogic` et `header.t0_ms`.
  const t0Ms = header?.t0_ms ?? 0
  const clock = replayClock(doc, header)
  // LE CADRAGE SUR LE MATCH RÉEL : le film déborde le match du countdown et d'une queue de
  // 5-6 s, et les deux bornes demandent l'artefact ET l'en-tête.
  const window = replayWindow(doc, header)
  const players = buildPlayers(doc, scoreboard)

  // LE KILL FEED VIENT DE LA BASE, PAS DU FILM : le rejeu ne porte pas les kills ; la vue
  // match, elle, les sert déjà résolus (auteur, équipe, ARME du kill avec son icône).
  const kills: KillEvent[] = collectKillEvents(matchView?.combat_tab.highlight_events, identity)
  const medals: MedalEvent[] = collectMedalEvents(matchView?.combat_tab.highlight_events)

  return {
    scoreboard,
    identity,
    marks,
    players,
    clock,
    window,
    // LES LIGNES D'ENTRÉE/SORTIE se fusionnent au fil, sur le même axe que le reste.
    feed: mergeFeedWithPresence(
      buildFeedEntries(kills, medals, t0Ms, doc),
      presenceEntries(players, window, doc, header),
    ),
    // Les médias portent des instants ABSOLUS : la soustraction se fait une fois, sur la
    // même horloge que le fil, la frise et les sièges.
    media: buildReplayMedia(matchView?.media_tab, header, clock),
    // LE SCORE FINAL QUAND IL NE SE DÉDUIT PAS DU FILM : sur un mode à manches, le calque
    // rendrait les points de la dernière manche au lieu du résultat.
    score: finalScoreFromHeader(header),
    t0Ms,
  }
}
