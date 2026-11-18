/**
 * playerInstantStats.ts — CE QU'UN JOUEUR A FAIT DU DÉBUT DU MATCH JUSQU'À L'IMAGE LUE, sans React.
 *
 * DEUX TEMPS. `readPlayerInstantSeries` lit une fois par joueur et par document ce qui est daté (compteurs du
 * film, frags du fil, médailles, vies, bouclier) et le range en séries triées ; `playerInstantStatsAt` répond
 * pour une image par dichotomie, sans rien relire du document. Le rendu appelle le premier dans un `useMemo`
 * et le second à chaque image.
 *
 * AUCUNE FORMULE N'EST RÉÉCRITE QUAND UNE FONCTION EXISTE : les compteurs viennent de `playerCountersAt`, le
 * FDA de `matchFda` (le net `(frags + assistances / 3 - morts)`, jamais le quotient frags / morts), les
 * médailles de `readMatchMedals` (via le fil qui les date), les vies de `trackWindow` / `isAliveAt`.
 *
 * LA RÈGLE DES FICHES VAUT ICI : une valeur que le document ne permet pas d'établir est ABSENTE (`null`),
 * jamais un zéro ni une valeur devinée. Le sens de chaque absence est écrit sur le champ de
 * `PlayerInstantStats`.
 *
 * Les BOTS comptent comme tout joueur. Leur clé de roster est `bot:<nom>` et le xuid de la feuille (celui que
 * portent le fil et les calques datés) est `board.xuid` : chaque lecture essaie les deux (`identityKeys`).
 *
 * Module pur : ni DOM, ni texte d'interface, ni couleur.
 */
import { matchFda } from '../../../lib/fda'
import { frameToMs, isAliveAt, lastIndexAt, trackWindow } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer, VitalityPresence } from '../../../lib/replay/rosterLogic'
import {
  playerCountersAt,
  playerScoreFor,
  scoreAtFrame,
  type ReplayPlayerScoreReady,
  type ReplayScoreTimelineReady,
} from '../../../lib/replay/scoreTimeline'
import type { MedalEvent, ReplayFeedEntry } from './killFeedLogic'

/**
 * Le nom de la médaille « éliminations parfaites » dans la table du film (`medals[].name`, clé de
 * `medalLabels`). Le jeu écrit une seule médaille pour ce fait ; un film d'un autre titre dont la table
 * nommerait ce fait autrement ne produirait aucune médaille de ce nom, donc un compte de zéro mesuré.
 */
export const PERFECT_MEDAL_NAME = 'Perfect'

/** Ce que la lecture rend pour une image : tout est cumulé depuis le début du match. */
export interface PlayerInstantStats {
  /**
   * Compteurs du film (frags, morts, assistances) à l'image. `null` = le calque de score ne publie pas ce
   * joueur (mode sans compteurs, joueur non apparié, document sans calque ou horloge non recalée) : ce n'est
   * PAS « à zéro ».
   */
  counters: { kills: number; deaths: number; assists: number } | null
  /** FDA du jeu sur ces compteurs (`matchFda`). `null` quand `counters` l'est. */
  fda: number | null
  /**
   * Série en cours : frags depuis la dernière mort du joueur (depuis le début du match s'il n'est pas encore
   * mort). Un frag daté à la même image qu'une mort compte avant elle. `null` quand `counters` l'est.
   */
  currentStreak: number | null
  /** Meilleure série du match jusqu'à l'image, la série en cours comprise. `null` quand `counters` l'est. */
  bestStreak: number | null
  /**
   * Tirs à la tête : `count` éliminations dont le coup fatal est mesuré à la tête, sur `measured`
   * éliminations dont le coup fatal est mesuré. Une élimination non mesurée n'entre dans aucun des deux
   * nombres. `null` quand le fil ne porte aucune élimination (contexte du match absent).
   */
  headshots: { count: number; measured: number } | null
  /**
   * Éliminations parfaites : médailles `PERFECT_MEDAL_NAME` gagnées jusqu'à l'image. `null` quand les
   * médailles du joueur ne s'établissent pas (cf. `medals`) : un document qui ne déclare pas le calque des
   * médailles ne dit pas que le joueur n'en a gagné aucune.
   */
  perfects: number | null
  /**
   * Médailles gagnées jusqu'à l'image, dans l'ordre du temps (lecture unique `readMatchMedals`). Un document
   * qui déclare le calque (`coverage.medals`) sans médaille donne une liste vide. `null` quand le document ne
   * déclare pas le calque (cuisson antérieure : le tableau normalisé est vide sans que rien ait été lu), et pour
   * un bot que le document ne nomme sur aucune médaille : le film ne date pas les médailles d'un bot, une
   * liste vide serait un zéro inventé.
   */
  medals: readonly MedalEvent[] | null
  /**
   * Réapparitions : vies commencées jusqu'à l'image, la première (naissance au début du match) exclue. Une
   * remise en place de fin de manche est une vie qui commence, donc une réapparition. `null` sans aucune vie.
   */
  respawns: number | null
  /**
   * Durée de la vie en cours, en millisecondes, depuis sa naissance (ou depuis le début du match pour la
   * première vie, quand `matchStartFrame` est donné). `null` quand le joueur n'a pas de vie à l'image : mort,
   * ou pas encore apparu.
   */
  currentLifeMs: number | null
  /**
   * Moyenne des vies terminées avant l'image, en millisecondes. `null` tant qu'aucune vie n'est terminée.
   * Une vie est la piste du joueur, de sa naissance à sa fin (mort ou fin de manche).
   */
  averageLifeMs: number | null
  /**
   * Boucliers perdus : passages du bouclier du joueur à zéro, lus aux points publiés de ses vies. Une
   * recharge puis une nouvelle chute comptent deux ; on apparaît à plein (comme `playerStateAt`). `null`
   * quand le document ne publie jamais de bouclier, ou quand le joueur n'a aucune vie.
   */
  shieldsLost: number | null
}

export interface PlayerInstantInput {
  doc: Pick<ReplayDocumentReady, 'frameIntervalMs' | 'coverage'>
  player: ReplayPlayer
  /** Quels champs de vitalité le document publie (`vitalityPresence`) : sans bouclier publié, aucun compte. */
  presence: VitalityPresence
  /** Calque de score du film, déjà passé par la garde d'horloge (`scoreTimelineOf`). */
  scoreTimeline?: ReplayScoreTimelineReady
  /** Le fil du rejeu (`ReplayModel.feed`) : les frags du joueur et ses médailles datées. `null` = absent. */
  feed: readonly ReplayFeedEntry[] | null
  /** Première image du match (`ReplayWindowBounds.startFrame`) ; absente, les vies se mesurent depuis leur piste. */
  matchStartFrame?: number | null
}

/** Les clés sous lesquelles un joueur se lit : sa clé de roster, puis le xuid de sa ligne de feuille. */
export function identityKeys(player: ReplayPlayer): string[] {
  const keys = [player.xuid]
  const board = player.board?.xuid
  if (board && board !== player.xuid) keys.push(board)
  return keys.filter((k) => k !== '')
}

/** Un événement daté sur un axe numérique (image ou milliseconde), consultable par `lastIndexAt`. */
interface Dated {
  t: number
}

/** Les événements d'un joueur arrêtés à l'image : le nombre d'éléments de `events` datés au plus tard à `t`. */
function countUpTo(events: readonly Dated[], t: number): number {
  return lastIndexAt(events, t) + 1
}

// ---------------------------------------------------------------------------------------
// Séries
// ---------------------------------------------------------------------------------------

interface StreakStep extends Dated {
  /** Frags du joueur à l'image de cette mort (compris). */
  kills: number
  /** Meilleure série terminée par cette mort ou avant elle. */
  best: number
}

/**
 * Une étape par MORT : les paliers de `deaths.total` sont cumulatifs, un palier qui saute de n en vaut n.
 * La série terminée par une mort est la différence des frags entre deux morts consécutives.
 */
function streakSteps(score: ReplayPlayerScoreReady): StreakStep[] {
  const steps: StreakStep[] = []
  let deaths = 0
  let killsAtLastDeath = 0
  let best = 0
  for (const tick of score.deaths.total) {
    const kills = scoreAtFrame(score.kills.total, tick.t)
    for (let n = tick.v - deaths; n > 0; n--) {
      best = Math.max(best, kills - killsAtLastDeath)
      steps.push({ t: tick.t, kills, best })
      killsAtLastDeath = kills
    }
    deaths = Math.max(deaths, tick.v)
  }
  return steps
}

interface HeadshotStep extends Dated {
  /** Éliminations à la tête mesurées, cumulées. */
  count: number
  /** Éliminations dont le coup fatal est mesuré, cumulées. */
  measured: number
}

/** Les éliminations du joueur sur l'axe du fil (ms), avec leurs deux cumuls. */
function headshotSteps(feed: readonly ReplayFeedEntry[], keys: readonly string[]): HeadshotStep[] {
  const steps: HeadshotStep[] = []
  let count = 0
  let measured = 0
  for (const e of feed) {
    const k = e.kill
    if (!k || !keys.includes(k.xuid)) continue
    if (k.headshot !== undefined) measured++
    if (k.headshot === true) count++
    steps.push({ t: k.replayMs, count, measured })
  }
  return steps
}

interface MedalStep extends Dated {
  medal: MedalEvent
  /** Médailles parfaites gagnées jusqu'à celle-ci comprise. */
  perfects: number
}

/**
 * Les médailles du joueur, datées sur l'axe du fil : une médaille rattachée à une élimination (±500 ms) prend
 * l'instant de cette élimination, comme sa ligne du fil ; une médaille seule garde le sien.
 */
function medalSteps(feed: readonly ReplayFeedEntry[], keys: readonly string[]): MedalStep[] {
  const dated: { t: number; medal: MedalEvent }[] = []
  for (const e of feed) {
    if (e.kill) {
      for (const m of e.kill.medals) if (keys.includes(m.xuid)) dated.push({ t: e.kill.replayMs, medal: m })
    } else if (e.medal && keys.includes(e.medal.xuid)) {
      dated.push({ t: e.replayMs, medal: e.medal })
    }
  }
  dated.sort((a, b) => a.t - b.t)
  let perfects = 0
  return dated.map((d) => {
    if (d.medal.name === PERFECT_MEDAL_NAME) perfects++
    return { ...d, perfects }
  })
}

interface ClosedLife extends Dated {
  /** Somme des durées des vies terminées jusqu'à celle-ci comprise, en ms. */
  totalMs: number
}

/** Début de la vie, ramené au début du match quand il est donné. */
function lifeStart(life: ReplayTrackReady, matchStartFrame: number | null | undefined): number {
  const start = trackWindow(life).start
  return matchStartFrame == null ? start : Math.max(start, matchStartFrame)
}

/**
 * Les vies terminées, datées à leur FIN (image de la dernière mesure). Une vie qui finit avant le début du
 * match, ou dont la durée mesurée est nulle, n'est pas une vie jouée : elle n'entre pas dans la moyenne.
 */
function closedLives(
  lives: readonly ReplayTrackReady[],
  doc: Pick<ReplayDocumentReady, 'frameIntervalMs' | 'coverage'>,
  matchStartFrame: number | null | undefined,
): ClosedLife[] {
  const spans = lives
    .map((l) => ({ end: trackWindow(l).end, start: lifeStart(l, matchStartFrame) }))
    .filter((s) => s.end > s.start)
    .sort((a, b) => a.end - b.end)
  let totalMs = 0
  return spans.map((s) => {
    totalMs += frameToMs(s.end - s.start, doc)
    return { t: s.end, totalMs }
  })
}

/**
 * Les images où le bouclier du joueur PASSE à zéro, sur toutes ses vies. Le bouclier est publié au
 * changement seulement : un point sans `sh` ne dit rien. On apparaît à plein ; une première mesure déjà à zéro
 * à la naissance de la vie n'est pas une chute observée (mode sans bouclier), elle ne compte pas.
 */
function shieldDrops(lives: readonly ReplayTrackReady[]): Dated[] {
  const drops: Dated[] = []
  for (const life of lives) {
    const born = trackWindow(life).start
    let up = true
    for (const p of life.points) {
      if (p.sh === undefined) continue
      if (p.sh > 0) {
        up = true
      } else {
        if (up && p.t > born) drops.push({ t: p.t })
        up = false
      }
    }
  }
  return drops.sort((a, b) => a.t - b.t)
}

/**
 * Ce que `readPlayerInstantSeries` range pour `playerInstantStatsAt`. Les champs ne se lisent pas ailleurs :
 * ils sont l'état interne d'une lecture, exposé pour le typage seulement.
 */
export interface PlayerInstantSeries {
  readonly doc: Pick<ReplayDocumentReady, 'frameIntervalMs' | 'coverage'>
  readonly timeline: ReplayScoreTimelineReady | undefined
  /** Clé sous laquelle le calque de score publie le joueur ; `null` = non publié. */
  readonly scoreKey: string | null
  readonly streaks: readonly StreakStep[] | null
  readonly headshots: readonly HeadshotStep[] | null
  readonly medals: readonly MedalStep[] | null
  readonly lifeStarts: readonly Dated[]
  readonly lives: readonly ReplayTrackReady[]
  readonly matchStartFrame: number | null | undefined
  readonly closedLives: readonly ClosedLife[]
  readonly shieldDrops: readonly Dated[] | null
}

/**
 * Les médailles du joueur s'établissent-elles ? Il faut que le document DÉCLARE le calque (`coverage.medals`,
 * posé par la cuisson quand le morceau des temps forts a été lu : le tableau normalisé vide ne distingue pas
 * « aucune médaille » de « calque non lu »). Celles d'un humain s'établissent alors toujours (aucune médaille
 * donne une liste vide) ; celles d'un bot seulement si le document le nomme au moins une fois — sinon il n'y a
 * rien à compter.
 */
function medalsEstablished(
  doc: Pick<ReplayDocumentReady, 'coverage'>,
  player: ReplayPlayer,
  steps: readonly MedalStep[],
): boolean {
  if (doc.coverage?.medals === undefined) return false
  return !player.bot || steps.length > 0
}

/** readPlayerInstantSeries lit, une fois, tout ce qui est daté pour ce joueur. */
export function readPlayerInstantSeries(input: PlayerInstantInput): PlayerInstantSeries {
  const { doc, player, presence, scoreTimeline, feed, matchStartFrame } = input
  const keys = identityKeys(player)
  const scoreKey = keys.find((k) => playerScoreFor(scoreTimeline, k) !== null) ?? null
  const score = scoreKey === null ? null : playerScoreFor(scoreTimeline, scoreKey)
  const hasKills = feed !== null && feed.some((e) => e.kill)
  const medals = feed === null ? null : medalSteps(feed, keys)
  const lives = player.lives
  return {
    doc,
    timeline: scoreTimeline,
    scoreKey,
    streaks: score ? streakSteps(score) : null,
    headshots: feed !== null && hasKills ? headshotSteps(feed, keys) : null,
    medals: medals && medalsEstablished(doc, player, medals) ? medals : null,
    lifeStarts: lives.map((l) => ({ t: trackWindow(l).start })).sort((a, b) => a.t - b.t),
    lives,
    matchStartFrame,
    closedLives: closedLives(lives, doc, matchStartFrame),
    shieldDrops: presence.shield && lives.length > 0 ? shieldDrops(lives) : null,
  }
}

// ---------------------------------------------------------------------------------------
// Lecture à l'image
// ---------------------------------------------------------------------------------------

/** Les compteurs, le FDA et les séries d'élimination à l'image : tout vient des paliers du film. */
function countersAt(s: PlayerInstantSeries, frame: number): Pick<
  PlayerInstantStats,
  'counters' | 'fda' | 'currentStreak' | 'bestStreak'
> {
  const read = s.scoreKey === null ? null : playerCountersAt(s.timeline, s.scoreKey, frame)
  if (!read || !s.streaks) return { counters: null, fda: null, currentStreak: null, bestStreak: null }
  const counters = { kills: read.kills, deaths: read.deaths, assists: read.assists }
  const i = lastIndexAt(s.streaks, frame)
  const sinceDeath = read.kills - (i >= 0 ? s.streaks[i].kills : 0)
  return {
    counters,
    fda: matchFda(counters),
    currentStreak: sinceDeath,
    bestStreak: Math.max(i >= 0 ? s.streaks[i].best : 0, sinceDeath),
  }
}

/** Les éliminations à la tête, les médailles et les éliminations parfaites à l'image. */
function feedReadingsAt(
  s: PlayerInstantSeries,
  frame: number,
): Pick<PlayerInstantStats, 'headshots' | 'perfects' | 'medals'> {
  const nowMs = frameToMs(frame, s.doc)
  let headshots: PlayerInstantStats['headshots'] = null
  if (s.headshots) {
    const h = lastIndexAt(s.headshots, nowMs)
    headshots = h >= 0 ? { count: s.headshots[h].count, measured: s.headshots[h].measured } : { count: 0, measured: 0 }
  }
  if (!s.medals) return { headshots, perfects: null, medals: null }
  const m = lastIndexAt(s.medals, nowMs)
  return {
    headshots,
    perfects: m >= 0 ? s.medals[m].perfects : 0,
    medals: s.medals.slice(0, m + 1).map((x) => x.medal),
  }
}

/** Les vies à l'image : réapparitions, vie en cours et moyenne des vies terminées. */
function livesAt(
  s: PlayerInstantSeries,
  frame: number,
): Pick<PlayerInstantStats, 'respawns' | 'currentLifeMs' | 'averageLifeMs'> {
  if (s.lives.length === 0) return { respawns: null, currentLifeMs: null, averageLifeMs: null }
  const current = s.lives.find((l) => isAliveAt(l, frame))
  const closed = lastIndexAt(s.closedLives, frame - 1)
  return {
    respawns: Math.max(0, countUpTo(s.lifeStarts, frame) - 1),
    currentLifeMs: current ? frameToMs(Math.max(0, frame - lifeStart(current, s.matchStartFrame)), s.doc) : null,
    averageLifeMs: closed >= 0 ? s.closedLives[closed].totalMs / (closed + 1) : null,
  }
}

/** playerInstantStatsAt rend ce que le joueur a fait du début du match jusqu'à `frame` (compris). */
export function playerInstantStatsAt(s: PlayerInstantSeries, frame: number): PlayerInstantStats {
  return {
    ...countersAt(s, frame),
    ...feedReadingsAt(s, frame),
    ...livesAt(s, frame),
    shieldsLost: s.shieldDrops ? countUpTo(s.shieldDrops, frame) : null,
  }
}
