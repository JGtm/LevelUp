/**
 * aliveBalanceLogic.ts — LA SUPÉRIORITÉ NUMÉRIQUE : combien de joueurs en vie de chaque côté,
 * et l'écart, tout au long du match.
 *
 * POURQUOI DANS `lib/`. Le graphe qui en tire parti vit dans la vue du match, mais les vies
 * viennent du document de rejeu et l'équipe de son roster : la logique partagée entre ces deux
 * surfaces se range ici, jamais dans l'une que l'autre irait importer (ratchet P8.5).
 *
 * RIEN N'EST REDÉFINI. « En vie » est `isAliveAt` (la fenêtre de la vie qui couvre l'image),
 * « équipe » est le désignateur que le film écrit au roster (`ReplayPlayer.team`, ADR 0034
 * D-9) : ce module ne fait que compter. Les BOTS COMPTENT (règle du produit) : `buildPlayers`
 * les range par leur clé `bot:<nom>` et leur équipe vient du roster comme celle d'un humain.
 * Une vie dont le film ne dit pas l'équipe n'entre dans aucun camp, et celle d'un joueur sans
 * propriétaire (caméra, spectateur) n'est dans aucune liste de joueurs : rien n'est deviné.
 *
 * L'ÉCART EST CELUI DE L'ÉQUIPE DU JOUEUR (positif = elle est en supériorité). Face à plus de
 * deux camps, « en face » est la MEILLEURE équipe adverse à cet instant (`multiTeam` le dit,
 * pour que l'écran l'écrive) : la même règle que l'écart de score, et la seule qui ne nomme
 * pas arbitrairement un adversaire parmi plusieurs.
 *
 * ÉCHANTILLONNÉ TOUTES LES SECONDES (`ALIVE_BALANCE_STEP_MS`), ET INDÉPENDANT DU PAS DU
 * DOCUMENT. Le rapport de force change au rythme des morts et des réapparitions, quelques fois
 * par minute et par joueur : un point toutes les 10 s (la maquette) en ratait la moitié. La
 * seconde est aussi la résolution du chrono que le joueur lit (M:SS), donc le survol ne promet
 * pas plus fin que ce qui s'affiche. Le coût est borné et se paie une fois par match (le
 * résultat est mémoïsé par l'appelant) : au plus ~1 500 instants (25 minutes) x 16 joueurs x
 * une quinzaine de vies, soit quelques centaines de milliers de comparaisons d'entiers.
 *
 * L'AXE EST CELUI DU GAMEPLAY (`lib/replay/matchClock`) : `ms` compte depuis le coup d'envoi, comme
 * « Frags cumulés » et « Score dans le temps » ; sans horloge établie il n'y a pas de graphe.
 *
 * Tout ce fichier est PUR : ni React, ni canvas, ni DOM.
 */
import type { MatchClock } from './matchClock'
import { isAliveAt } from './replayLogic'
import type { ReplayPlayer } from './rosterLogic'

/** Le pas d'échantillonnage, en millisecondes de gameplay (cf. l'en-tête). */
export const ALIVE_BALANCE_STEP_MS = 1000

/** Un instant : les joueurs en vie de l'équipe du joueur et de la meilleure équipe adverse. */
export interface AliveBalancePoint {
  /** Millisecondes depuis le COUP D'ENVOI. */
  ms: number
  /** Joueurs en vie de l'équipe du joueur, bots compris. */
  ally: number
  /** Joueurs en vie de l'équipe d'en face (la mieux fournie s'il y en a plusieurs). */
  opponent: number
}

export interface AliveBalance {
  /** Un point par pas, de 0 à `endMs` inclus ; la valeur tient jusqu'au point suivant. */
  points: AliveBalancePoint[]
  /** Fin de l'axe : la dernière image du film, lue sur l'horloge du gameplay. */
  endMs: number
  /** Plus d'un camp en face : l'écart se lit contre la meilleure équipe adverse. */
  multiTeam: boolean
}

export interface AliveBalanceInput {
  /** Les joueurs du film (`buildPlayers`), bots compris. */
  players: readonly ReplayPlayer[]
  /** L'équipe du film du joueur de la page (`FilmAllegiance.allyTeam`) ; `null` = inconnue. */
  allyTeam: number | null
  /** L'horloge du match ; `null` = origine du film non établie, donc pas de graphe. */
  clock: MatchClock | null | undefined
  /** Pas d'échantillonnage en ms ; défaut `ALIVE_BALANCE_STEP_MS`. */
  stepMs?: number
}

/**
 * buildAliveBalance rend la supériorité numérique du match, ou `null` — donc RIEN À L'ÉCRAN —
 * quand elle ne peut pas être dite : pas d'horloge, équipe du joueur inconnue, aucune équipe en
 * face, ou un côté dont le film ne publie aucune vie (« zéro en vie » serait une mesure
 * inventée, pas une lecture).
 */
export function buildAliveBalance(input: AliveBalanceInput): AliveBalance | null {
  const { players, allyTeam, clock } = input
  const stepMs = input.stepMs ?? ALIVE_BALANCE_STEP_MS
  if (!clock || allyTeam === null || !(stepMs > 0)) return null
  const endMs = clock.gameplayMsOfFrame(clock.frameCount - 1)
  if (endMs <= 0) return null

  const camps = new Map<number, ReplayPlayer[]>()
  for (const p of players) {
    if (p.team === undefined || p.team < 0 || p.lives.length === 0) continue
    const membres = camps.get(p.team)
    if (membres) membres.push(p)
    else camps.set(p.team, [p])
  }
  const allies = camps.get(allyTeam)
  const adverses = [...camps].filter(([team]) => team !== allyTeam).map(([, membres]) => membres)
  if (!allies || adverses.length === 0) return null

  const points: AliveBalancePoint[] = []
  for (let ms = 0; ms < endMs; ms += stepMs) points.push(pointAt(ms, allies, adverses, clock))
  points.push(pointAt(endMs, allies, adverses, clock))
  return { points, endMs, multiTeam: adverses.length > 1 }
}

function pointAt(
  ms: number,
  allies: readonly ReplayPlayer[],
  adverses: ReadonlyArray<readonly ReplayPlayer[]>,
  clock: MatchClock,
): AliveBalancePoint {
  // gameplay -> match (+ t0) -> film (- origine) -> image : la réciproque de `gameplayMsOfFrame`.
  const frame = clock.frameOfFilmMs(clock.filmMsOfMatchMs(ms + clock.t0Ms))
  let opponent = 0
  for (const camp of adverses) opponent = Math.max(opponent, aliveCount(camp, frame))
  return { ms, ally: aliveCount(allies, frame), opponent }
}

/** Joueurs d'un camp dont au moins une vie couvre l'image : un joueur compte une fois. */
function aliveCount(camp: readonly ReplayPlayer[], frame: number): number {
  let n = 0
  for (const p of camp) if (p.lives.some((life) => isAliveAt(life, frame))) n++
  return n
}

/**
 * aliveBalanceAt rend le point d'échantillonnage TENU à `ms` : le dernier au plus tard à cet
 * instant, le premier avant le coup d'envoi, le dernier après la fin. L'effectif ne change
 * qu'aux morts et aux réapparitions : il attend entre deux points. Dichotomie : l'infobulle suit
 * le curseur au pixel.
 */
export function aliveBalanceAt(balance: AliveBalance, ms: number): AliveBalancePoint {
  const pts = balance.points
  let lo = 0
  let hi = pts.length - 1
  while (lo < hi) {
    const mid = (lo + hi + 1) >> 1
    if (pts[mid].ms <= ms) lo = mid
    else hi = mid - 1
  }
  return pts[lo]
}
