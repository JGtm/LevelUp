/**
 * playerReloadStats.ts — LES RECHARGEMENTS D'UN JOUEUR DU DÉBUT DU MATCH JUSQU'À L'IMAGE LUE, sans React.
 *
 * DEUX TEMPS, comme `playerMovementStats`. `readReloadSeries` lit une fois par document les rechargements
 * (`doc.reloads`), les attribue au joueur que l'index du message (`place` = `roster[].filmIndex`) désigne à l'instant
 * du rechargement et range, par joueur, une série triée ; `playerReloadsAt` répond pour une image par dichotomie, sans
 * rien relire.
 *
 * QUI EST CE JOUEUR. Un index désigne UN joueur quand une seule identité du roster le porte (`rosterEntryKey`). Quand
 * plusieurs identités le portent (un joueur parti puis son remplaçant, des bots qui partagent un index), l'index ne
 * suffit pas : le rechargement revient à l'identité dont la PRÉSENCE publiée couvre son image (`presence`, bornes
 * `from` à `toMax`), à elle seule. Zéro ou plusieurs identités présentes : il ne revient à personne, jamais à tous.
 *
 * CE QUE LE DOCUMENT PUBLIE. Un rechargement est un fait daté `{t, place}` : ni l'arme, ni la fin du rechargement.
 * Un client compte des rechargements par joueur, rien de plus.
 *
 * LA RÈGLE DES FICHES VAUT ICI : une grandeur que le document ne permet pas d'établir est ABSENTE (`null`), jamais un
 * zéro. Elle n'est établie que si `coverage.reloads` existe : le calque n'est publié que lorsque la marche des
 * trames a lu au moins une trame, et un film dont la lecture n'a rien rendu (versions anciennes) n'a aucune
 * couverture. Quand la grandeur est établie, « aucun rechargement pour ce joueur » est un zéro MESURÉ.
 *
 * UN COMPTE PEUT ÊTRE UN PLANCHER : une trame dont la liste d'événements s'arrête sur un message non lu perd les
 * messages qui la suivent (`coverage.reloads.stoppedFrames`), et un rechargement que le film n'a pu rattacher à aucun
 * joueur n'est pas publié (`coverage.reloads.withoutPlayer`). `minimum` le dit pour tout le document ; l'écran doit
 * l'écrire.
 *
 * Module pur : ni DOM, ni texte d'interface, ni couleur.
 */
import { lastIndexAt } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { rosterEntryKey, type ReplayPlayer } from '../../../lib/replay/rosterLogic'
import { identityKeys } from './playerInstantStats'

/** Ce que la lecture rend pour une image : tout est cumulé depuis le début du match. */
export interface PlayerReloads {
  /** Rechargements jusqu'à l'image, comprise. `0` = « aucun » mesuré. `null` = le document ne permet pas de l'établir. */
  count: number | null
  /** Vrai quand `count` est un plancher : une trame du film s'arrête sur un message non lu, ou un rechargement n'a pas de joueur. */
  minimum: boolean
}

/** Les rechargements datés d'un joueur, consultables par `lastIndexAt`. */
export interface PlayerReloadSeries {
  readonly times: readonly { t: number }[] | null
  readonly minimum: boolean
}

/** Les séries de tous les joueurs, sous leur clé de roster et sous le xuid de leur ligne de feuille. */
export type ReloadSeriesIndex = ReadonlyMap<string, PlayerReloadSeries>

/** Ce que la lecture a besoin du document. */
export type ReloadSeriesSource = Pick<ReplayDocumentReady, 'reloads' | 'roster' | 'coverage'>

/** Rien n'est établi : la réponse d'un joueur sans série, partagée (elle est immuable). */
const NOT_ESTABLISHED: PlayerReloads = { count: null, minimum: false }

/** Une identité du roster qui porte un index, avec ses présences. */
type RosterEntry = ReloadSeriesSource['roster'][number]

/** La présence publiée d'une entrée couvre-t-elle l'image ? La borne haute est `toMax`, à défaut `to`. */
function presentAt(entry: RosterEntry, frame: number): boolean {
  return entry.presence.some((p) => frame >= p.from && frame <= (p.toMax ?? p.to))
}

/**
 * ownerKeyOf rend la clé de roster du joueur qui a rechargé : celle de l'unique identité de l'index, ou à défaut de
 * l'unique identité présente à l'image. `null` : aucune, ou plusieurs.
 */
function ownerKeyOf(candidates: readonly RosterEntry[] | undefined, frame: number): string | null {
  if (candidates === undefined) return null
  const keys = new Set<string>()
  for (const entry of candidates) {
    const key = rosterEntryKey(entry)
    if (key) keys.add(key)
  }
  if (keys.size === 1) return [...keys][0]
  const present = new Set<string>()
  for (const entry of candidates) {
    const key = rosterEntryKey(entry)
    if (key && presentAt(entry, frame)) present.add(key)
  }
  return present.size === 1 ? [...present][0] : null
}

/**
 * readReloadSeries lit, une fois par document, les rechargements de chaque joueur du roster. `players` est le roster du
 * film (`buildPlayers`). Un rechargement dont l'index ne désigne aucune entrée du roster, ou que la présence ne permet
 * pas d'attribuer à un seul joueur, ne revient à personne.
 */
export function readReloadSeries(doc: ReloadSeriesSource, players: readonly ReplayPlayer[]): ReloadSeriesIndex {
  const coverage = doc.coverage?.reloads
  const index = new Map<string, PlayerReloadSeries>()
  const minimum = coverage !== undefined && (coverage.stoppedFrames > 0 || coverage.withoutPlayer > 0)
  const byOwner = new Map<string, number[]>()
  if (coverage !== undefined) {
    const entriesByPlace = new Map<number, RosterEntry[]>()
    for (const entry of doc.roster) entriesByPlace.set(entry.filmIndex, [...(entriesByPlace.get(entry.filmIndex) ?? []), entry])
    for (const r of doc.reloads) {
      const key = ownerKeyOf(entriesByPlace.get(r.place), r.t)
      if (key === null) continue
      const list = byOwner.get(key)
      if (list) list.push(r.t)
      else byOwner.set(key, [r.t])
    }
  }
  for (const player of players) {
    if (player.lives.length === 0) continue
    const times = coverage === undefined ? null : (byOwner.get(player.xuid) ?? []).sort((a, b) => a - b).map((t) => ({ t }))
    const series: PlayerReloadSeries = { times, minimum }
    for (const key of identityKeys(player)) index.set(key, series)
  }
  return index
}

/**
 * reloadSeriesOf — la série d'un joueur : sa clé de roster, puis le xuid de sa ligne de feuille. `null` = le joueur
 * n'a aucune vie dans le film : rien n'est établi pour lui.
 */
export function reloadSeriesOf(index: ReloadSeriesIndex, player: ReplayPlayer): PlayerReloadSeries | null {
  for (const key of identityKeys(player)) {
    const series = index.get(key)
    if (series) return series
  }
  return null
}

/**
 * playerReloadsAt rend les rechargements du joueur du début du match jusqu'à `frame` (compris). Le compte ne recule
 * jamais d'une image à l'autre ; un joueur mort garde ce qu'il a rechargé.
 */
export function playerReloadsAt(series: PlayerReloadSeries | null, frame: number): PlayerReloads {
  if (!series || !series.times) return NOT_ESTABLISHED
  return { count: lastIndexAt(series.times, frame) + 1, minimum: series.minimum }
}
