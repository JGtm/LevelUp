/**
 * feedFollowed — QUELLES LIGNES DU FIL CONCERNENT LE JOUEUR SUIVI PAR LA CAMÉRA : la décision pure du marquage
 * « fond légèrement teinté » des lignes du fil (colonne, fil posé sur le plan, extrait de la fiche). Pas de React :
 * le composant de ligne lit `concernsFollowed`, il ne décide de rien.
 *
 * UNE LIGNE CONCERNE LE JOUEUR SUIVI quand il en est l'AUTEUR ou la CIBLE : le tueur ou la victime d'un frag, le
 * défunt d'une mort sans tueur, le décoré d'une médaille seule, l'arrivant ou le partant d'une entrée ou sortie de
 * partie, l'acteur d'une ligne d'événement (objectif, prise, montée, crâne), le destructeur d'un véhicule ou un
 * joueur tué avec lui. Une assistance de capture n'en fait pas : le crédité n'est ni l'auteur ni la cible. Les lignes
 * sans joueur (fin de manche, colline) ne concernent personne. Aucun joueur suivi : aucune ligne.
 *
 * Ce marquage n'est PAS la ligne « choisie » (la ligne de frag cliquée, `bg-muted` et `aria-pressed`) ni le gras du
 * joueur dont la fiche est ouverte (`strongXuids`) : les trois se cumulent, aucun ne remplace un autre.
 */
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import type { ReplayFeedEntry } from './killFeedLogic'

/** Aucun joueur suivi : l'ensemble vide partagé, jamais recréé. */
const NO_KEYS: ReadonlySet<string> = new Set()

/** Les ensembles déjà bâtis, par joueur : un même joueur suivi rend le MÊME ensemble d'un rendu à l'autre. */
const KEYS_OF = new WeakMap<object, ReadonlySet<string>>()

/**
 * playerKeys rend les clés sous lesquelles le fil désigne un joueur : sa clé de roster, et le xuid de base de sa
 * ligne du tableau de score (le fil nomme un joueur par son xuid de base, le film par sa clé : les deux le désignent).
 * Absent (personne n'est suivi) : l'ensemble vide.
 */
export function playerKeys(player: Pick<ReplayPlayer, 'xuid' | 'board'> | null | undefined): ReadonlySet<string> {
  if (!player) return NO_KEYS
  const known = KEYS_OF.get(player)
  if (known) return known
  const keys = new Set([player.xuid, ...(player.board?.xuid ? [player.board.xuid] : [])])
  KEYS_OF.set(player, keys)
  return keys
}

/** Les joueurs que désigne une ligne d'événement : l'acteur, puis les joueurs tués avec un véhicule. */
function eventPlayers(event: NonNullable<ReplayFeedEntry['event']>): readonly (string | null)[] {
  const actor = 'xuid' in event ? event.xuid : null
  const killed = 'killedXuids' in event ? event.killedXuids : []
  return [actor, ...killed]
}

/** Les joueurs que désigne une ligne du fil, tous rôles confondus (auteur ou cible). */
function playersOf(entry: ReplayFeedEntry): readonly (string | null)[] {
  if (entry.event) return eventPlayers(entry.event)
  if (entry.presence) return [entry.presence.xuid]
  if (entry.death) return [entry.death.xuid]
  if (entry.kill) return [entry.kill.xuid, entry.kill.victimXuid]
  if (entry.medal) return [entry.medal.xuid]
  return []
}

/**
 * concernsFollowed dit si la ligne concerne l'un des joueurs de `followed` (les clés du joueur suivi,
 * `playerKeys`). Vrai dès que l'un des joueurs de la ligne en est ; faux pour un ensemble vide.
 */
export function concernsFollowed(entry: ReplayFeedEntry, followed: ReadonlySet<string>): boolean {
  if (followed.size === 0) return false
  return playersOf(entry).some((xuid) => xuid !== null && followed.has(xuid))
}
