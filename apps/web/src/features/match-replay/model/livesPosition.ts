/**
 * livesPosition.ts — L'UNIQUE écriture du bloc « position d'un joueur à une image ».
 *
 * CE QUE CE MODULE CENTRALISE. Relire la position d'un joueur à une image donnée demande
 * quatre pièces, toutes ici : la FENÊTRE APRÈS-MORT (`KILLPOS_WINDOW_MS`, puis
 * `deathWindowFrames` qui la convertit en images), l'INDEX des vies par joueur
 * (`buildLivesByXuid`), la RELECTURE elle-même (`posOfPlayerAt`) et son assemblage sur un
 * document (`buildPlayerPosAt`). Ce bloc était copié dans quatre hooks et deux constructeurs
 * purs ; règle n° 6 du dépôt : à la troisième copie on centralise, et le garde-rail
 * (`livesPosition.guard.test.ts`) interdit la re-divergence.
 *
 * LA PRIMITIVE A ÉTÉ RAPATRIÉE DE `killFx.ts` LE 2026-09-05, ET LA « COPIE JUMELLE
 * AUTORISÉE » N'EXISTE PLUS. `posOfPlayerAt` et `KILLPOS_WINDOW_MS` vivaient dans killFx.ts,
 * ce qui forçait ce module à l'importer — et interdisait donc à killFx.ts d'importer
 * celui-ci (cycle). killFx.ts y entretenait, en conséquence, sa propre copie de l'index et
 * de la fenêtre, dûment allowlistée. En déplaçant la primitive vers son module canonique, le
 * cycle disparaît, la copie avec lui, et l'allowlist du garde-rail retombe à DEUX entrées.
 *
 * LA FENÊTRE APRÈS-MORT fait partie du bloc, pas une option : un poseur de bombe meurt souvent
 * DANS son explosion, un porteur lâche à sa mort — sans elle, l'événement le mieux daté du
 * match perdrait sa position.
 *
 * C'EST LA POSITION DU BIPÈDE, ET ELLE N'EST PLUS LE DERNIER MOT. Un joueur embarqué ne
 * réplique plus sa position monde : la relecture ci-dessous interpole alors en ligne droite à
 * travers le décor. Les lecteurs de production passent donc par `carrierPosition.ts`
 * (`useCarrierPosAt` / `buildCarrierPosAt`), qui pose le véhicule par-dessus ce repli. Ce
 * module reste la relecture DU BIPÈDE, écrite une fois — il ne connaît aucun véhicule.
 */

import { altitudeAt, isAliveAt, msToFrames, positionAt, trackWindow, type XY } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from '../../../lib/replay/replayNormalize'

/**
 * Fenêtre d'acceptation d'une DERNIÈRE position après la fin d'une vie, en ms. La victime,
 * par construction, vient de mourir — sa trace est close, sa dernière position reste vraie.
 *
 * ELLE VAUT LA FENÊTRE DEATH D'ORIGINE (1,5 s) MAIS N'EN DÉPEND PLUS : la croix de mort est
 * passée à 2,5 s le 2026-08-18, celle-ci non. Ce sont deux questions différentes — combien de
 * temps un repère reste LISIBLE d'un côté, jusqu'où une position reste VRAIE de l'autre — et
 * les lier ferait bouger la seconde à chaque réglage d'écran de la première.
 */
export const KILLPOS_WINDOW_MS = 1_500

/** La position relue, telle que `posOfPlayerAt` la rend. */
export type PlayerPosAt = (xuid: string, frame: number) => XY | null

/**
 * readAtPlayer — LA RÈGLE DE RELECTURE d'une grandeur du joueur à une frame, écrite UNE fois : une
 * vie VIVANTE à cette frame répond ; sinon la DERNIÈRE lecture d'une vie close depuis moins de
 * `deathFrames` (la victime vient de mourir, par construction). Au-delà : null, rien d'inventé.
 * La position et l'altitude s'y branchent — deux grandeurs, une seule règle de vie.
 */
function readAtPlayer<T>(
  lives: ReplayTrackReady[] | undefined,
  frame: number,
  deathFrames: number,
  atFrame: (life: ReplayTrackReady) => T | null,
  atEnd: (life: ReplayTrackReady) => T | null,
): T | null {
  if (!lives || lives.length === 0) return null
  for (const l of lives) {
    if (isAliveAt(l, frame)) {
      const v = atFrame(l)
      if (v !== null) return v
    }
  }
  let best: ReplayTrackReady | null = null
  let bestD = Number.POSITIVE_INFINITY
  for (const l of lives) {
    const d = frame - trackWindow(l).end
    if (d >= 0 && d <= deathFrames && d < bestD) {
      bestD = d
      best = l
    }
  }
  return best ? atEnd(best) : null
}

/**
 * posOfPlayerAt — position d'un joueur à une frame, relue dans ses trajectoires.
 *
 * Patron `posOfNameAt` du POC : une vie VIVANTE à cette frame rend sa position ; sinon on
 * accepte la DERNIÈRE position d'une vie close depuis moins de `deathFrames` — c'est le cas
 * de la victime, par construction. Au-delà : null, aucune position inventée.
 */
export function posOfPlayerAt(
  lives: ReplayTrackReady[] | undefined,
  frame: number,
  deathFrames: number,
): XY | null {
  return readAtPlayer(
    lives,
    frame,
    deathFrames,
    (l) => positionAt(l.points, frame),
    (l) => {
      const last = l.points[l.points.length - 1]
      return last ? { x: last.x, y: last.y } : null
    },
  )
}

/**
 * altitudeOfPlayerAt — l'altitude du joueur à une frame, sous la MÊME règle de vie que sa position.
 * Une vie dont AUCUN point ne porte de z (artefact sans altitude) rend null : `altitudeAt` rendrait 0,
 * et un 0 inventé nommerait un étage — l'absence est la donnée.
 */
export function altitudeOfPlayerAt(
  lives: ReplayTrackReady[] | undefined,
  frame: number,
  deathFrames: number,
): number | null {
  const withZ = (l: ReplayTrackReady) => l.points.some((p) => p.z !== undefined)
  return readAtPlayer(
    lives,
    frame,
    deathFrames,
    (l) => (withZ(l) ? altitudeAt(l.points, frame) : null),
    (l) => {
      const last = l.points[l.points.length - 1]
      return last && withZ(l) ? (last.z ?? 0) : null
    },
  )
}

/** buildLivesByXuid — l'index des vies par joueur, construit UNE fois par document. */
export function buildLivesByXuid(tracks: readonly ReplayTrackReady[]): Map<string, ReplayTrackReady[]> {
  const map = new Map<string, ReplayTrackReady[]>()
  for (const t of tracks) {
    if (!t.xuid) continue
    const list = map.get(t.xuid)
    if (list) list.push(t)
    else map.set(t.xuid, [t])
  }
  return map
}

/**
 * buildLivesBySlot — l'index des vies par SLOT, construit UNE fois par document.
 *
 * POURQUOI PAR SLOT ET PAS PAR XUID (registre 2026-09-05, K1). Les EFFETS attachés à un geste
 * — éclair de bouche, « ! » du tireur, câble de grappin, dash de propulseur — sont datés par
 * le film avec un numéro de SLOT, pas un xuid : c'est la vie du bipède qui tire, et un même
 * joueur en a plusieurs. Cet index était réécrit byte pour byte dans les quatre calques
 * concernés ; il n'a désormais qu'une écriture.
 */
export function buildLivesBySlot(
  tracks: readonly ReplayTrackReady[],
): Map<number, ReplayTrackReady[]> {
  const map = new Map<number, ReplayTrackReady[]>()
  for (const t of tracks) {
    const lives = map.get(t.slot)
    if (lives) lives.push(t)
    else map.set(t.slot, [t])
  }
  return map
}

/**
 * lifeOfSlotAt — LA VIE d'un slot qui couvre une image donnée, ou `undefined`.
 *
 * PRENDRE LA PREMIÈRE VENUE SERAIT UNE FAUTE, et c'est tout l'objet de cette fonction : un
 * joueur a plusieurs vies dans le film, et lire le regard ou la position de la MAUVAISE vie
 * place le geste ailleurs sur la carte. Un slot sans vie à cet instant ne rend rien — jamais
 * une vie voisine.
 */
export function lifeOfSlotAt(
  bySlot: ReadonlyMap<number, ReplayTrackReady[]>,
  slot: number,
  frame: number,
): ReplayTrackReady | undefined {
  return (bySlot.get(slot) ?? []).find((l) => isAliveAt(l, frame))
}

/** deathWindowFrames — la fenêtre après-mort en images (au moins une). */
export function deathWindowFrames(doc: ReplayDocumentReady): number {
  return Math.max(1, Math.round(msToFrames(KILLPOS_WINDOW_MS, doc)))
}

/** buildPlayerPosAt — la relecture pure, pour les constructeurs hors React. */
export function buildPlayerPosAt(doc: ReplayDocumentReady): PlayerPosAt {
  const lives = buildLivesByXuid(doc.tracks)
  const window = deathWindowFrames(doc)
  return (xuid, frame) => posOfPlayerAt(lives.get(xuid), frame, window)
}

/** L'altitude relue, telle que `altitudeOfPlayerAt` la rend (null = inconnue). */
export type PlayerAltAt = (xuid: string, frame: number) => number | null

/** buildPlayerAltAt — l'altitude du bipède, sous la même règle de vie que `buildPlayerPosAt`. */
export function buildPlayerAltAt(doc: ReplayDocumentReady): PlayerAltAt {
  const lives = buildLivesByXuid(doc.tracks)
  const window = deathWindowFrames(doc)
  return (xuid, frame) => altitudeOfPlayerAt(lives.get(xuid), frame, window)
}
