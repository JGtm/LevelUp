/**
 * presenceFeed — QUI A REJOINT, QUI A QUITTÉ. L'API d'abord, le film en repli.
 *
 * DEUX SOURCES, UNE HIÉRARCHIE (retour user 2026-09-02 : « l'API nous dit précisément
 * quand le joueur a rejoint et quand il est parti ») :
 *
 *   API   `PlayerParticipationInfo` du match — `joined_in_progress` / `left_in_progress`
 *         et leurs horodatages ABSOLUS (`first_joined_time` / `last_leave_time`, RFC3339
 *         UTC, bug TZ des matchs anciens corrigé en base le 2026-05-29). C'est la source
 *         PRÉCISE et AFFIRMATIVE : les drapeaux disent le fait, pas une déduction — un
 *         drapeau à `false` fait TAIRE ce joueur, ce n'est pas une absence de donnée.
 *         Le recalage vers l'axe du rejeu est CELUI DES MÉDIAS (replayMediaLogic) :
 *         replayMs = (absolu − header.start_time) − origine du film — une seule doctrine,
 *         portée par l'horloge de la page (`model/replayClock`).
 *
 *   FILM  le repli, quand les drapeaux manquent (matchs d'avant les colonnes, en-tête
 *         absent) : les bornes de vie. Première vie bien après le coup d'envoi = entrée ;
 *         dernière vie bien avant la fin = « ne reviendra plus » — le film ne distingue
 *         pas un départ d'une élimination définitive, le libellé du repli reste au FAIT
 *         et l'infobulle porte la réserve. Les marges (10 s / 20 s, >2x la réapparition
 *         médiane mesurée 8,0 s) n'existent QUE sur ce chemin.
 *
 * SEULES LA PREMIÈRE ET LA DERNIÈRE VIE PARLENT côté film : un trou ENTRE deux vies
 * n'émet rien (rester mort entre deux manches d'élimination est normal). L'API, elle,
 * n'a pas ce problème — ses drapeaux sont par match.
 */
import { stripBotSuffix } from '@/lib/players/displayName'

import type { ReplayFeedEntry } from './killFeedLogic'
import { replayClock, type ReplayClock } from './replayClock'
import { frameToMs, trackWindow } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import type { ReplayWindowBounds } from './replayWindow'
import { playerName, type ReplayPlayer } from './rosterLogic'

/** REPLI FILM : première vie qui commence au-delà = un arrivant (flottement du départ). */
export const PRESENCE_ARRIVE_MS = 10_000
/** REPLI FILM : dernière vie qui s'achève plus tôt que ça avant la fin = ne reviendra plus. */
export const PRESENCE_DEPART_MS = 20_000

/** Ce que la présence lit de l'en-tête : l'origine ABSOLUE de l'axe du match. */
export interface PresenceHeader {
  start_time?: string | null
}

/** Une ligne de présence du fil : qui, dans quel sens, et par quelle source. */
export interface PresenceEvent {
  kind: 'joined' | 'left'
  /** Clé d'identité du joueur (xuid, ou `bot:<nom>`) — pour la couleur et les marques. */
  xuid: string
  name: string
  bot: boolean
  /** `api` = drapeaux de participation (précis) ; `film` = dérivé des bornes de vie. */
  source: 'api' | 'film'
}

/**
 * presenceEntries — les lignes d'entrée/sortie, prêtes à fusionner dans le fil.
 *
 * Rendues comme des `ReplayFeedEntry` (kill/medal/death à null, `presence` posé) : le fil
 * les trie et les fenêtre exactement comme les autres lignes, aucun second axe de temps.
 */
export function presenceEntries(
  players: readonly ReplayPlayer[],
  playWindow: ReplayWindowBounds | null,
  doc: ReplayDocumentReady,
  header?: PresenceHeader | null,
): ReplayFeedEntry[] {
  // LE MÊME VERDICT QUE LE RESTE DE LA PAGE (`model/replayClock`) : sans horloge établie, une
  // ligne d'entrée serait posée à un instant faux de 3,6 à 50,8 s. La fenêtre de gameplay
  // n'existe d'ailleurs pas non plus dans ce cas — les deux portes disent la même chose.
  const clock = replayClock(doc)
  if (!playWindow || !clock) return []
  const matchStartMs = parseInstant(header?.start_time)
  const out: ReplayFeedEntry[] = []
  for (const p of players) {
    const rawName = playerName(p)
    if (!rawName) continue
    // Suffixe « [bot] » = marqueur de donnée killsource (schéma 36), retiré avant
    // d'entrer dans la ligne de présence (« a rejoint » / « a quitté »).
    const name = stripBotSuffix(rawName)
    // LE CHEMIN API NE DEMANDE AUCUNE VIE : un remplaçant dont le pont n'a pas pu nommer
    // les vies (bot du match témoin 1b2d9e08) a quand même rejoint la partie — la base le
    // date, la ligne s'affiche. Seul le REPLI film exige des vies : il n'a rien d'autre.
    const api = apiPresence(p, name, matchStartMs, clock, playWindow)
    if (api) {
      out.push(...api)
      continue
    }
    if (p.lives.length > 0) out.push(...filmPresence(p, name, playWindow, doc))
  }
  return out
}

/**
 * apiPresence — le chemin PRÉCIS. Rend `null` quand la participation n'est pas décidable
 * (ligne de scoreboard absente, drapeaux NULL, ou en-tête sans heure de début — sans elle
 * un horodatage absolu ne se pose pas sur l'axe) : le repli film prend alors la main.
 * Rend `[]` quand l'API DIT que le joueur n'a ni rejoint ni quitté en cours : silence
 * AFFIRMATIF, le film n'a pas à le contredire.
 */
function apiPresence(
  p: ReplayPlayer,
  name: string,
  matchStartMs: number | null,
  clock: ReplayClock,
  playWindow: ReplayWindowBounds,
): ReplayFeedEntry[] | null {
  const b = p.board
  if (!b || matchStartMs === null) return null
  if (b.joined_in_progress == null && b.left_in_progress == null) return null
  const out: ReplayFeedEntry[] = []
  const toReplayMs = (iso: string | null | undefined): number | null => {
    const abs = parseInstant(iso)
    return abs === null ? null : clock.filmMsOfMatchMs(abs - matchStartMs)
  }
  if (b.joined_in_progress) {
    const ms = toReplayMs(b.first_joined_time)
    // Un drapeau vrai sans horodatage lisible ne pose rien : une entrée à un instant
    // inventé serait pire que son absence.
    if (ms !== null) out.push(entry('joined', 'api', p, name, clamp(ms, playWindow)))
  }
  if (b.left_in_progress) {
    const ms = toReplayMs(b.last_leave_time)
    if (ms !== null) out.push(entry('left', 'api', p, name, clamp(ms, playWindow)))
  }
  return out
}

/** filmPresence — le REPLI : les bornes de vie, avec leurs marges (cf. en-tête). */
function filmPresence(
  p: ReplayPlayer,
  name: string,
  playWindow: ReplayWindowBounds,
  doc: ReplayDocumentReady,
): ReplayFeedEntry[] {
  const out: ReplayFeedEntry[] = []
  const first = trackWindow(p.lives[0])
  const last = trackWindow(p.lives[p.lives.length - 1])
  const firstMs = frameToMs(first.start, doc)
  const lastMs = frameToMs(last.end, doc)
  if (firstMs - playWindow.startMs > PRESENCE_ARRIVE_MS) {
    out.push(entry('joined', 'film', p, name, firstMs))
  }
  if (playWindow.endMs - lastMs > PRESENCE_DEPART_MS) {
    out.push(entry('left', 'film', p, name, lastMs))
  }
  return out
}

/** L'instant API, borné à la fenêtre de gameplay : une ligne hors axe ne se lirait pas. */
function clamp(ms: number, w: ReplayWindowBounds): number {
  return Math.min(Math.max(ms, w.startMs), w.endMs)
}

function parseInstant(iso: string | null | undefined): number | null {
  if (!iso) return null
  const ms = Date.parse(iso)
  return Number.isFinite(ms) ? ms : null
}

function entry(
  kind: PresenceEvent['kind'],
  source: PresenceEvent['source'],
  p: ReplayPlayer,
  name: string,
  replayMs: number,
): ReplayFeedEntry {
  return {
    key: `p-${kind}-${p.xuid}-${replayMs}`,
    replayMs,
    kill: null,
    medal: null,
    death: null,
    presence: { kind, xuid: p.xuid, name, bot: p.bot === true, source },
  }
}

/** Fusionne le fil et les lignes de présence, retriés sur l'axe commun. */
export function mergeFeedWithPresence(
  feed: readonly ReplayFeedEntry[],
  presence: readonly ReplayFeedEntry[],
): ReplayFeedEntry[] {
  if (presence.length === 0) return [...feed]
  return [...feed, ...presence].sort((a, b) => a.replayMs - b.replayMs)
}
