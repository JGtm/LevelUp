/**
 * playerGestureStats.ts — LES GESTES D'UN JOUEUR DU DÉBUT DU MATCH JUSQU'À L'IMAGE LUE : coups de mêlée, lancers
 * amorcés, accroupissements et sauts au bouton, sans React.
 *
 * DEUX TEMPS, comme `playerReloadStats`. `readGestureSeries` lit une fois par document, et `playerGesturesAt` répond
 * pour une image par dichotomie, sans rien relire.
 *
 * DEUX SOURCES, DEUX FAÇONS DE NOMMER LE JOUEUR.
 *  - `gestures[]` : un fait daté `{t, place, kind}` (`melee`, `throwStart`). La place est l'index de joueur du film,
 *    rattaché par le roster (`placeOwnerResolver`), un bot compris. Le film dit le coup qui part, jamais son résultat,
 *    ni la grenade qui suivra (`grenades[]`) : un client compte des gestes.
 *  - `stances[]` de genre `crouchCommand` et `jumpCommand` : le bouton tenu, un intervalle par appui, porté par une
 *    VIE (le joueur est celui qui occupe le slot à `t0`, `buildSlotOwnership`). On compte les intervalles.
 *
 * LA PORTE DE CADENCE. Les gestes, l'accroupi et le saut ne se publient que sur un film dont les commandes sont
 * enregistrées à pleine cadence, c'est-à-dire la version actuelle du jeu. Sur un autre, le décodeur compte les gestes
 * lus comme retenus (`coverage.gestures.gated`) et ne publie ni geste ni commande tenue. Une grandeur que le document
 * ne permet pas d'établir est donc ABSENTE (`null`), jamais un zéro :
 *  - mêlée et lancer : établis seulement si `coverage.gestures` existe et ne retient aucun geste (`gated` à zéro) ;
 *  - accroupi et saut : établis seulement si `coverage.stances.byKind` compte au moins un intervalle du genre.
 * Quand la grandeur est établie, « aucun geste pour ce joueur » est un zéro MESURÉ.
 *
 * UN COMPTE PEUT ÊTRE UN PLANCHER : une trame dont la liste d'événements s'arrête sur un message non lu perd ceux qui
 * la suivent (`stoppedFrames`), et un geste sans joueur n'est pas publié (`withoutPlayer`). `minimum` le dit pour la
 * mêlée et le lancer ; l'écran doit l'écrire.
 *
 * Module pur : ni DOM, ni texte d'interface, ni couleur.
 */
import { lastIndexAt } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { placeOwnerResolver } from '../../../lib/replay/placeOwner'
import { buildSlotOwnership, type ReplayPlayer } from '../../../lib/replay/rosterLogic'
import { identityKeys } from './playerInstantStats'
import { intervalsByOwner } from './playerMovementStats'

/** Genres de `gestures[].kind`. */
const GESTURE_MELEE = 'melee'
const GESTURE_THROW_START = 'throwStart'
/** Genres de `stances[].kind` : le bouton d'accroupi et le bouton de saut tenus. */
const STANCE_CROUCH_COMMAND = 'crouchCommand'
const STANCE_JUMP_COMMAND = 'jumpCommand'

/** Ce que la lecture rend pour une image : tout est cumulé depuis le début du match. */
export interface PlayerGestures {
  /** Coups de mêlée jusqu'à l'image, comprise. `0` = « aucun » mesuré. `null` = le document ne permet pas de l'établir. */
  melee: number | null
  /** Lancers amorcés, même contrat. */
  throwStarts: number | null
  /** Appuis sur le bouton d'accroupi, même contrat. */
  crouches: number | null
  /** Appuis sur le bouton de saut, même contrat (le saut LU, à distinguer de l'estimation dérivée de la vitesse). */
  jumpPresses: number | null
  /** Vrai quand le compte de mêlée et de lancers est un plancher (trame arrêtée, geste sans joueur). */
  minimum: boolean
}

/** Les gestes datés d'un joueur, consultables par `lastIndexAt` ; `null` = la grandeur n'est pas établie. */
export interface PlayerGestureSeries {
  readonly melee: readonly { t: number }[] | null
  readonly throwStarts: readonly { t: number }[] | null
  readonly crouches: readonly { t: number }[] | null
  readonly jumpPresses: readonly { t: number }[] | null
  readonly minimum: boolean
}

/** Les séries de tous les joueurs, sous leur clé de roster et sous le xuid de leur ligne de feuille. */
export type GestureSeriesIndex = ReadonlyMap<string, PlayerGestureSeries>

/** Ce que la lecture a besoin du document. */
export type GestureSeriesSource = Pick<ReplayDocumentReady, 'gestures' | 'roster' | 'coverage' | 'stances'>

/** Rien n'est établi : la réponse d'un joueur sans série, partagée (elle est immuable). */
const NOT_ESTABLISHED: PlayerGestures = { melee: null, throwStarts: null, crouches: null, jumpPresses: null, minimum: false }

/** Les gestes de ce genre rangés par joueur (clé de roster), datés et triés. */
function gesturesByOwner(doc: GestureSeriesSource, kind: string): Map<string, { t: number }[]> {
  const ownerOf = placeOwnerResolver(doc.roster)
  const byOwner = new Map<string, { t: number }[]>()
  for (const g of doc.gestures) {
    if (g.kind !== kind) continue
    const key = ownerOf(g.place, g.t)
    if (key === null) continue
    const own = byOwner.get(key)
    if (own) own.push({ t: g.t })
    else byOwner.set(key, [{ t: g.t }])
  }
  for (const own of byOwner.values()) own.sort((a, b) => a.t - b.t)
  return byOwner
}

/** Le film publie-t-il ce genre de commande tenue ? Au moins un intervalle au décompte par genre de la couverture. */
function stanceKindPublished(doc: GestureSeriesSource, kind: string): boolean {
  const cov = doc.coverage?.stances
  return cov?.scanned === true && (cov.byKind?.[kind] ?? 0) > 0
}

/** Les accroupissements ou sauts commandés de chaque joueur : un instant par intervalle, à son début. */
function stanceTimesByOwner(
  doc: GestureSeriesSource,
  kind: string,
  ownerAt: (slot: number, frame: number) => ReplayPlayer | null,
): Map<string, { t: number }[]> | null {
  if (!stanceKindPublished(doc, kind)) return null
  const intervals = intervalsByOwner(doc, kind, ownerAt)
  return new Map([...intervals].map(([key, list]) => [key, list.map((iv) => ({ t: iv.t0 }))]))
}

/**
 * readGestureSeries lit, une fois par document, les gestes de chaque joueur qui a au moins une vie. `players` est le
 * roster du film (`buildPlayers`). Un geste dont la place ne désigne aucune identité, ou qu'une identité seule ne
 * revendique pas, ne revient à personne.
 */
export function readGestureSeries(doc: GestureSeriesSource, players: readonly ReplayPlayer[]): GestureSeriesIndex {
  const coverage = doc.coverage?.gestures
  const carried = coverage !== undefined && coverage.gated === 0
  const minimum = carried && (coverage.stoppedFrames > 0 || coverage.withoutPlayer > 0)
  const melee = carried ? gesturesByOwner(doc, GESTURE_MELEE) : null
  const throwStarts = carried ? gesturesByOwner(doc, GESTURE_THROW_START) : null
  const ownerAt = buildSlotOwnership(players).ownerAtFrame
  const crouches = stanceTimesByOwner(doc, STANCE_CROUCH_COMMAND, ownerAt)
  const jumpPresses = stanceTimesByOwner(doc, STANCE_JUMP_COMMAND, ownerAt)
  const index = new Map<string, PlayerGestureSeries>()
  for (const player of players) {
    if (player.lives.length === 0) continue
    const series: PlayerGestureSeries = {
      melee: melee ? (melee.get(player.xuid) ?? []) : null,
      throwStarts: throwStarts ? (throwStarts.get(player.xuid) ?? []) : null,
      crouches: crouches ? (crouches.get(player.xuid) ?? []) : null,
      jumpPresses: jumpPresses ? (jumpPresses.get(player.xuid) ?? []) : null,
      minimum,
    }
    for (const key of identityKeys(player)) index.set(key, series)
  }
  return index
}

/**
 * gestureSeriesOf — la série d'un joueur : sa clé de roster, puis le xuid de sa ligne de feuille. `null` = le joueur
 * n'a aucune vie dans le film : rien n'est établi pour lui.
 */
export function gestureSeriesOf(index: GestureSeriesIndex, player: ReplayPlayer): PlayerGestureSeries | null {
  for (const key of identityKeys(player)) {
    const series = index.get(key)
    if (series) return series
  }
  return null
}

const countAt = (times: readonly { t: number }[] | null, frame: number): number | null =>
  times ? lastIndexAt(times, frame) + 1 : null

/**
 * playerGesturesAt rend les gestes du joueur du début du match jusqu'à `frame` (compris). Les comptes ne reculent
 * jamais d'une image à l'autre ; un joueur mort garde ce qu'il a fait.
 */
export function playerGesturesAt(series: PlayerGestureSeries | null, frame: number): PlayerGestures {
  if (!series) return NOT_ESTABLISHED
  return {
    melee: countAt(series.melee, frame),
    throwStarts: countAt(series.throwStarts, frame),
    crouches: countAt(series.crouches, frame),
    jumpPresses: countAt(series.jumpPresses, frame),
    minimum: series.minimum,
  }
}
