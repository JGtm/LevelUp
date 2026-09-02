/**
 * presenceFeed — QUI ENTRE EN PARTIE, QUI N'EN REVIENT PLUS, dérivé des bornes de vie.
 *
 * LE FILM NE PORTE AUCUN ÉVÉNEMENT DE CONNEXION (mesuré : la liste d'événements du paquet
 * ne connaît que kill/death/medal/mode). Ce que le film porte, ce sont les VIES — datées à
 * la frame — et c'est d'elles que ce module déduit la présence (demande user 2026-09-02 :
 * « qui a quitté et qui est arrivé, tout est indiqué au niveau des timestamps ») :
 *
 *   ENTRÉE  la PREMIÈRE vie d'un joueur commence bien après le coup d'envoi — au coup
 *           d'envoi, tout le monde apparaît ; une première vie tardive est un arrivant.
 *   SORTIE  la DERNIÈRE vie d'un joueur s'achève bien avant la fin du match — le fil ne
 *           dit pas POURQUOI (départ, ou élimination définitive d'un mode à manches), et
 *           le libellé reste donc au FAIT : « ne reviendra plus ».
 *
 * SEULES LA PREMIÈRE ET LA DERNIÈRE VIE PARLENT. Un trou ENTRE deux vies n'émet rien :
 * sur un mode à élimination, rester mort 30-60 s en attendant la manche suivante est
 * NORMAL — en déduire des allers-retours spammerait le fil de faux départs.
 *
 * LES MARGES tiennent le même raisonnement : la réapparition médiane mesurée est de
 * 8,0 s (film de référence), la marge de sortie en prend plus du double ; l'entrée
 * tolère le flottement du coup d'envoi. Un joueur ANONYME (vie jamais nommée) n'émet
 * rien : on ne fait pas entrer ni sortir quelqu'un qu'on ne sait pas nommer.
 */
import type { ReplayFeedEntry } from './killFeedLogic'
import { frameToMs, trackWindow } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import type { ReplayWindowBounds } from './replayWindow'
import { playerName, type ReplayPlayer } from './rosterLogic'

/** Première vie qui commence au-delà : le joueur est un ARRIVANT (flottement du départ). */
export const PRESENCE_ARRIVE_MS = 10_000
/** Dernière vie qui s'achève plus tôt que ça avant la fin : il ne reviendra plus. */
export const PRESENCE_DEPART_MS = 20_000

/** Une ligne de présence du fil : qui, quand, dans quel sens. */
export interface PresenceEvent {
  kind: 'joined' | 'left'
  /** Clé d'identité du joueur (xuid, ou `bot:<nom>`) — pour la couleur et les marques. */
  xuid: string
  name: string
  bot: boolean
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
): ReplayFeedEntry[] {
  if (!playWindow) return []
  const out: ReplayFeedEntry[] = []
  for (const p of players) {
    if (p.lives.length === 0) continue
    const name = playerName(p)
    if (!name) continue
    const first = trackWindow(p.lives[0])
    const last = trackWindow(p.lives[p.lives.length - 1])
    const firstMs = frameToMs(first.start, doc)
    const lastMs = frameToMs(last.end, doc)
    if (firstMs - playWindow.startMs > PRESENCE_ARRIVE_MS) {
      out.push(entry('joined', p, name, firstMs))
    }
    if (playWindow.endMs - lastMs > PRESENCE_DEPART_MS) {
      out.push(entry('left', p, name, lastMs))
    }
  }
  return out
}

function entry(
  kind: PresenceEvent['kind'],
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
    presence: { kind, xuid: p.xuid, name, bot: p.bot === true },
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
