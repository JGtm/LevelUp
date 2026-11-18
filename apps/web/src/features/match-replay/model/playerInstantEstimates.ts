/**
 * playerInstantEstimates.ts — LE SCORE, LE RANG ET LES ESTIMATIONS DE TIR D'UN JOUEUR À L'IMAGE LUE, sans React.
 *
 * DEUX TEMPS, comme `playerInstantStats`. `readInstantEstimatesSeries` lit une fois par document ce qui est daté
 * (points par action, tirs écrits par le film, touches, dégâts) et le range en séries triées, par joueur ;
 * `playerInstantEstimatesAt` répond pour une image par dichotomie, sans rien relire du document. Le rendu appelle
 * le premier dans un `useMemo` et le second à chaque image.
 *
 * UNE VALEUR QUE LE DOCUMENT NE PERMET PAS D'ÉTABLIR EST ABSENTE (`null`), jamais zéro, et sa cause est rendue
 * à part (`absences`) sous forme d'un petit type fermé, sans texte d'interface : le rendu n'écrit rien à la place
 * d'une valeur absente.
 *
 * SCORE. Le score personnel à l'instant est la somme des points par action (`scoreActions`) jusqu'à l'image.
 * Il n'est établi que si TROIS gardes tiennent : le document déclare le score par action complet
 * (`coverage.scoreActions.complete`), l'horloge du film est recalée (`filmClockTrusted`, la garde du calque de
 * score) et, à la fin du match, la somme du joueur ÉGALE le score personnel de sa ligne de feuille de l'API
 * (`ReplayPlayer.board.score`). Le verdict du document seul ne suffit pas : il peut être vrai alors que la somme
 * s'écarte de la feuille. La garde vaut pour tout build et tout mode ; aucune liste n'en décide.
 *
 * RANG. Le rang n'existe que si le score est établi pour TOUS les joueurs du lobby (les joueurs donnés à la
 * lecture, bots compris) : un seul score non établi rend le rang absent pour tout le monde. Égalités : rang de
 * compétition. Chaque joueur prend 1 + le nombre de joueurs dont le score est STRICTEMENT supérieur au sien ; deux
 * joueurs à égalité en tête sont donc 1 et 1, le suivant est 3.
 *
 * TIRS ET DÉGÂTS. Pour un joueur dont la lecture des tirs est jugée SUFFISANTE par le document
 * (`shotStats[].sufficient`) et pour lui seul, tout est cumulé sur les événements DATÉS : un tir écrit par le film
 * est rattaché à son tireur par la vie qui occupe le slot à l'image du tir (`buildSlotOwnership`) ; une touche et
 * une ligne de dégât le sont par xuid. Ces valeurs sont des ESTIMATIONS (`estimated: true`).
 *
 * LES TIRS ESTIMÉS. Le film n'écrit qu'une part des tirs réels : le bilan du joueur publie les tirs écrits
 * (`shotEvents`) et les tirs du jeu (`shots`, la numérotation du jeu). À l'instant, les tirs estimés sont les tirs
 * écrits à l'instant au rapport `shots / shotEvents` du bilan, arrondis à l'entier ; la précision est les touches
 * sur ces tirs estimés, jamais sur les tirs écrits (ils sous-comptent : la précision serait trop haute et
 * contredirait celle de fin de match). Un bilan sans tirs ou sans tirs écrits ne donne pas de rapport : les tirs
 * estimés sont alors les tirs écrits. Les compteurs d'un joueur ne dépassent jamais les totaux de match de
 * `shotStats` (qui comptent aussi hors de l'axe).
 *
 * Les BOTS comptent comme tout joueur pour le score et le rang (clés d'identité : `identityKeys`). Le film ne
 * publie ni tir, ni touche, ni dégât pour un bot : ses tirs sont absents.
 *
 * Module pur : ni DOM, ni texte d'interface, ni couleur.
 */
import { lastIndexAt } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { buildSlotOwnership, type ReplayPlayer } from '../../../lib/replay/rosterLogic'
import { filmClockTrusted, scoreAtFrame } from '../../../lib/replay/scoreTimeline'
import type { ReplayScoreTick } from '@/lib/api/types'

import { identityKeys } from './playerInstantStats'

// ---------------------------------------------------------------------------------------
// Ce que la lecture rend
// ---------------------------------------------------------------------------------------

/**
 * Pourquoi le score est absent :
 *  - `score-incomplet` : le document ne déclare pas le score par action complet ;
 *  - `horloge-non-recalee` : l'horloge du film n'est pas recalée, les points seraient datés à un écart inconnu ;
 *  - `feuille-absente` : le joueur n'a pas de score sur la feuille de l'API, rien ne permet de vérifier la somme ;
 *  - `somme-differente` : à la fin du match, la somme des points par action n'égale pas le score de la feuille.
 */
export type ScoreAbsence = 'score-incomplet' | 'horloge-non-recalee' | 'feuille-absente' | 'somme-differente'

/** Pourquoi le rang est absent : le score d'au moins un joueur du lobby n'est pas établi (le sien compris). */
export type RankAbsence = 'lobby-incomplet'

/**
 * Pourquoi les tirs sont absents :
 *  - `non-publie` : le document ne publie aucun bilan de tirs pour ce joueur (un bot, un film sans dégâts lus) ;
 *  - `lecture-insuffisante` : le bilan existe et le document juge la lecture des tirs insuffisante pour ce joueur.
 */
export type ShotsAbsence = 'non-publie' | 'lecture-insuffisante'

/** L'ensemble fermé des causes d'absence. */
export type EstimateAbsence = ScoreAbsence | RankAbsence | ShotsAbsence

/** La place d'un joueur dans le lobby : `place` (1 = meilleur score, rang de compétition) sur `of` joueurs. */
export interface PlayerRank {
  place: number
  of: number
}

/** Les tirs et les dégâts d'une arme, jusqu'à l'image. `weapon` est la clé d'arme du document ; `null` = arme non résolue. */
export interface WeaponEstimate {
  weapon: string | null
  /** Tirs écrits par le film avec cette arme. */
  fired: number
  /** Tirs estimés avec cette arme : les tirs écrits au rapport du bilan du joueur, arrondis. */
  firedEstimated: number
  /** Points de dégât infligés avec cette arme (au dixième près). */
  damage: number
}

/**
 * Les tirs et les dégâts d'un joueur à lecture suffisante, jusqu'à l'image. Tout est estimé (`estimated`).
 * Les totaux sont les sommes de `byWeapon` : une arme non résolue y figure sous `weapon: null`. Les tirs
 * estimés par arme sont répartis au plus grand reste : ils somment à `firedEstimated`.
 */
export interface PlayerShotEstimates {
  /** Toujours vrai : ces valeurs sont des estimations, le rendu l'écrit. */
  estimated: true
  /** Tirs écrits par le film, toutes armes. */
  fired: number
  /** Tirs estimés, toutes armes : les tirs écrits au rapport `shots / shotEvents` du bilan du joueur, arrondis. */
  firedEstimated: number
  /** Touches estimées (une par tireur, unité touchée et image). */
  hit: number
  /**
   * `hit / firedEstimated`, entre 0 et 1. `null` quand aucun tir n'est estimé, et quand les touches dépassent les
   * tirs estimés (une touche d'arme à zone n'est pas un tir : la précision n'est pas établie, jamais au-dessus de 100 %).
   */
  accuracy: number | null
  /** Points de dégât infligés aux autres joueurs et aux bots, hors soi-même (au dixième près). */
  damageDealt: number
  /** Points de dégât subis, hors ceux qu'on s'inflige (au dixième près). */
  damageTaken: number
  /** Une entrée par arme ayant tiré ou blessé : tirs décroissants, puis dégâts décroissants, puis clé ; `null` en dernier. */
  byWeapon: readonly WeaponEstimate[]
}

/**
 * Ce que la lecture rend pour un joueur et une image. Invariant : une valeur est `null` si et seulement si sa
 * cause est non nulle dans `absences`.
 */
export interface PlayerInstantEstimates {
  /** Score personnel à l'instant (somme des points par action). `null` : cf. `absences.score`. */
  score: number | null
  /** Place dans le lobby à l'instant. `null` : cf. `absences.rank`. */
  rank: PlayerRank | null
  /** Tirs et dégâts à l'instant. `null` : cf. `absences.shots`. */
  shots: PlayerShotEstimates | null
  absences: {
    score: ScoreAbsence | null
    rank: RankAbsence | null
    shots: ShotsAbsence | null
  }
}

// ---------------------------------------------------------------------------------------
// Entrée et séries
// ---------------------------------------------------------------------------------------

export interface InstantEstimatesInput {
  doc: Pick<ReplayDocumentReady, 'originMs' | 'coverage' | 'scoreActions' | 'shotStats' | 'shots' | 'hits' | 'damage'>
  /** Le lobby : tous les joueurs du rejeu (`buildPlayers`), bots compris. Le rang se calcule sur eux. */
  players: readonly ReplayPlayer[]
}

/** Un événement daté sur la grille d'images du document. */
interface Dated {
  t: number
}

/** Un cumul de dégâts en dixièmes de point, valable à partir de l'image `t`. */
interface TenthsStep extends Dated {
  v: number
}

interface WeaponSeries {
  fired: Dated[]
  damage: TenthsStep[]
}

interface ShotSeries {
  /** Le rapport tirs estimés sur tirs écrits du bilan du joueur ; 1 quand le bilan n'en donne pas. */
  scale: number
  hit: Dated[]
  taken: TenthsStep[]
  weapons: Map<string | null, WeaponSeries>
}

type ScoreReading = { ticks: ReplayScoreTick[]; absence: null } | { ticks: null; absence: ScoreAbsence }
type ShotsReading = { series: ShotSeries; absence: null } | { series: null; absence: ShotsAbsence }

interface PlayerEstimateSeries {
  score: ScoreReading
  shots: ShotsReading
}

/** Ce que `readInstantEstimatesSeries` range pour `playerInstantEstimatesAt` : état interne d'une lecture. */
export interface InstantEstimatesSeries {
  readonly lobbySize: number
  readonly players: ReadonlyMap<string, PlayerEstimateSeries>
}

// ---------------------------------------------------------------------------------------
// Score
// ---------------------------------------------------------------------------------------

/** La cause d'absence qui vaut pour tout le document, ou `null`. */
function documentScoreAbsence(doc: InstantEstimatesInput['doc']): ScoreAbsence | null {
  if (doc.coverage?.scoreActions?.complete !== true) return 'score-incomplet'
  if (!filmClockTrusted(doc)) return 'horloge-non-recalee'
  return null
}

/** Les points par action du document, rangés par destinataire (xuid décimal), dans l'ordre du temps. */
function actionsByRecipient(doc: InstantEstimatesInput['doc']): Map<string, ReplayDocumentReady['scoreActions']> {
  const out = new Map<string, ReplayDocumentReady['scoreActions']>()
  for (const a of doc.scoreActions) {
    const list = out.get(a.xuid)
    if (list) list.push(a)
    else out.set(a.xuid, [a])
  }
  return out
}

/** Le score d'un joueur : les paliers cumulés de ses points, si la somme finale égale sa feuille. */
function readScore(
  player: ReplayPlayer,
  byRecipient: ReadonlyMap<string, ReplayDocumentReady['scoreActions']>,
  docAbsence: ScoreAbsence | null,
): ScoreReading {
  if (docAbsence) return { ticks: null, absence: docAbsence }
  const sheet = player.board?.score
  if (sheet === undefined || sheet === null) return { ticks: null, absence: 'feuille-absente' }
  const actions = identityKeys(player)
    .flatMap((k) => byRecipient.get(k) ?? [])
    .sort((a, b) => a.t - b.t)
  const ticks: ReplayScoreTick[] = []
  let total = 0
  for (const a of actions) {
    total += a.points
    ticks.push({ t: a.t, v: total })
  }
  return total === sheet ? { ticks, absence: null } : { ticks: null, absence: 'somme-differente' }
}

// ---------------------------------------------------------------------------------------
// Tirs, touches et dégâts
// ---------------------------------------------------------------------------------------

/** Le bilan de tirs que le document publie pour ce joueur, sous l'une de ses clés d'identité. */
function shotStatOf(doc: InstantEstimatesInput['doc'], player: ReplayPlayer) {
  const keys = identityKeys(player)
  return doc.shotStats.find((s) => keys.includes(s.xuid))
}

function weaponSeries(acc: ShotSeries, weapon: string | null): WeaponSeries {
  let w = acc.weapons.get(weapon)
  if (!w) {
    w = { fired: [], damage: [] }
    acc.weapons.set(weapon, w)
  }
  return w
}

/** Des dégâts bruts (par ligne, en dixièmes) aux paliers cumulés, triés dans le temps. */
function cumulate(lines: readonly { t: number; d: number }[]): TenthsStep[] {
  let v = 0
  return [...lines]
    .sort((a, b) => a.t - b.t)
    .map((l) => {
      v += l.d
      return { t: l.t, v }
    })
}

/** Les lignes brutes d'un joueur avant cumul : ses dégâts par arme et ses dégâts subis. */
interface RawDamage {
  dealt: Map<string | null, { t: number; d: number }[]>
  taken: { t: number; d: number }[]
}

/** Les accumulateurs des joueurs à lecture suffisante, par clé de joueur. */
function openAccumulators(
  doc: InstantEstimatesInput['doc'],
  players: readonly ReplayPlayer[],
): { acc: Map<string, ShotSeries>; raw: Map<string, RawDamage>; owner: Map<string, string> } {
  const acc = new Map<string, ShotSeries>()
  const raw = new Map<string, RawDamage>()
  const owner = new Map<string, string>()
  for (const p of players) {
    if (shotStatOf(doc, p)?.sufficient !== true) continue
    acc.set(p.xuid, { scale: estimatedShotsScale(shotStatOf(doc, p)), hit: [], taken: [], weapons: new Map() })
    raw.set(p.xuid, { dealt: new Map(), taken: [] })
    for (const k of identityKeys(p)) owner.set(k, p.xuid)
  }
  return { acc, raw, owner }
}

/** Le rapport tirs estimés sur tirs écrits d'un bilan ; 1 quand l'un des deux manque (aucun rapport à tirer). */
function estimatedShotsScale(stat: { shots: number; shotEvents: number } | undefined): number {
  return stat && stat.shots > 0 && stat.shotEvents > 0 ? stat.shots / stat.shotEvents : 1
}

/** Un tireur ou une victime : le joueur à lecture suffisante que désigne ce xuid de film, ou `undefined`. */
function ownerOf(owner: ReadonlyMap<string, string>, xuid: string | undefined): string | undefined {
  return xuid ? owner.get(xuid) : undefined
}

/** Les points retirés au bouclier et au corps par une ligne de dégât, en dixièmes de point. */
const tenthsOf = (shield: number, body: number): number => Math.round((shield + body) * 10)

/**
 * readShotSeries range, pour chaque joueur à lecture suffisante, ses tirs, ses touches et ses dégâts datés.
 * Les autres joueurs n'ont aucune série : le rendu n'écrit rien pour eux.
 */
function readShotSeries(
  doc: InstantEstimatesInput['doc'],
  players: readonly ReplayPlayer[],
): Map<string, ShotSeries> {
  const { acc, raw, owner } = openAccumulators(doc, players)
  const slots = buildSlotOwnership(players)
  for (const s of doc.shots) {
    const p = slots.ownerAtFrame(s.slot, s.t)
    const mine = p && acc.get(p.xuid)
    if (mine) weaponSeries(mine, s.w ?? null).fired.push({ t: s.t })
  }
  for (const h of doc.hits) {
    const mine = acc.get(ownerOf(owner, h.shooter) ?? '')
    if (mine && h.shooter !== h.victim) mine.hit.push({ t: h.t })
  }
  for (const g of doc.damage) {
    if (g.attacker === g.victim) continue
    const line = { t: g.t, d: tenthsOf(g.shield, g.body) }
    const dealer = raw.get(ownerOf(owner, g.attacker) ?? '')
    if (dealer) {
      const key = g.w ?? null
      const list = dealer.dealt.get(key)
      if (list) list.push(line)
      else dealer.dealt.set(key, [line])
    }
    raw.get(ownerOf(owner, g.victim) ?? '')?.taken.push(line)
  }
  for (const [key, series] of acc) {
    const r = raw.get(key)
    if (!r) continue
    series.hit.sort((a, b) => a.t - b.t)
    series.taken = cumulate(r.taken)
    for (const [weapon, lines] of r.dealt) weaponSeries(series, weapon).damage = cumulate(lines)
    for (const w of series.weapons.values()) w.fired.sort((a, b) => a.t - b.t)
  }
  return acc
}

function readShots(
  doc: InstantEstimatesInput['doc'],
  player: ReplayPlayer,
  series: ReadonlyMap<string, ShotSeries>,
): ShotsReading {
  const stat = shotStatOf(doc, player)
  if (!stat) return { series: null, absence: 'non-publie' }
  const mine = series.get(player.xuid)
  if (!stat.sufficient || !mine) return { series: null, absence: 'lecture-insuffisante' }
  return { series: mine, absence: null }
}

// ---------------------------------------------------------------------------------------
// Lecture d'un document
// ---------------------------------------------------------------------------------------

/** readInstantEstimatesSeries lit, une fois par document, tout ce qui est daté pour chaque joueur du lobby. */
export function readInstantEstimatesSeries(input: InstantEstimatesInput): InstantEstimatesSeries {
  const { doc, players } = input
  const docAbsence = documentScoreAbsence(doc)
  const byRecipient = docAbsence ? new Map<string, ReplayDocumentReady['scoreActions']>() : actionsByRecipient(doc)
  const shotSeries = readShotSeries(doc, players)
  const out = new Map<string, PlayerEstimateSeries>()
  for (const p of players) {
    out.set(p.xuid, { score: readScore(p, byRecipient, docAbsence), shots: readShots(doc, p, shotSeries) })
  }
  return { lobbySize: players.length, players: out }
}

// ---------------------------------------------------------------------------------------
// Lecture à l'image
// ---------------------------------------------------------------------------------------

/** Le nombre d'événements datés au plus tard à `frame`. */
const countUpTo = (events: readonly Dated[], frame: number): number => lastIndexAt(events, frame) + 1

/** Le cumul en dixièmes à `frame` (0 avant le premier palier). */
function tenthsAt(steps: readonly TenthsStep[], frame: number): number {
  const i = lastIndexAt(steps, frame)
  return i >= 0 ? steps[i].v : 0
}

/** Deux parties fractionnaires plus proches que cela sont égales (le bruit des flottants ne décide pas d'un rang). */
const FRACTION_EPSILON = 1e-9

/**
 * apportionEstimates répartit `total` entre les armes au plus grand reste : chaque arme reçoit la partie entière
 * de ses tirs écrits au rapport `scale`, puis le reste va une unité à la fois aux plus grandes parties
 * fractionnaires. À égalité, l'arme vue la première par le film passe devant (ordre stable). Les parts
 * somment à `total`.
 */
function apportionEstimates(fired: readonly number[], scale: number, total: number): number[] {
  const quotas = fired.map((f) => f * scale)
  const shares = quotas.map((q) => Math.floor(q + FRACTION_EPSILON))
  let remainder = total - shares.reduce((n, v) => n + v, 0)
  const byRest = fired
    .map((f, i) => ({ i, f, rest: quotas[i] - shares[i] }))
    .filter((e) => e.f > 0)
    .sort((a, b) => (Math.abs(a.rest - b.rest) < FRACTION_EPSILON ? a.i - b.i : b.rest - a.rest))
  for (const e of byRest) {
    if (remainder <= 0) break
    shares[e.i]++
    remainder--
  }
  return shares
}

/** Tri des armes : tirs décroissants, dégâts décroissants, clé croissante, arme non résolue en dernier. */
function byWeaponOrder(a: WeaponEstimate, b: WeaponEstimate): number {
  if (a.fired !== b.fired) return b.fired - a.fired
  if (a.damage !== b.damage) return b.damage - a.damage
  if (a.weapon === b.weapon) return 0
  if (a.weapon === null) return 1
  if (b.weapon === null) return -1
  return a.weapon < b.weapon ? -1 : 1
}

function shotsAt(s: ShotSeries, frame: number): PlayerShotEstimates {
  const byWeapon: WeaponEstimate[] = []
  let fired = 0
  let dealtTenths = 0
  for (const [weapon, w] of s.weapons) {
    const f = countUpTo(w.fired, frame)
    const d = tenthsAt(w.damage, frame)
    if (f === 0 && d === 0) continue
    byWeapon.push({ weapon, fired: f, firedEstimated: 0, damage: d / 10 })
    fired += f
    dealtTenths += d
  }
  const hit = countUpTo(s.hit, frame)
  const firedEstimated = Math.round(fired * s.scale)
  apportionEstimates(byWeapon.map((w) => w.fired), s.scale, firedEstimated).forEach((n, i) => {
    byWeapon[i] = { ...byWeapon[i], firedEstimated: n }
  })
  return {
    estimated: true,
    fired,
    firedEstimated,
    hit,
    accuracy: firedEstimated > 0 && hit <= firedEstimated ? hit / firedEstimated : null,
    damageDealt: dealtTenths / 10,
    damageTaken: tenthsAt(s.taken, frame) / 10,
    byWeapon: byWeapon.sort(byWeaponOrder),
  }
}

/** Le rang de `mine` parmi les scores du lobby à l'image, ou `null` si l'un d'eux n'est pas établi. */
function rankAt(series: InstantEstimatesSeries, mine: ScoreReading, frame: number): PlayerRank | null {
  if (!mine.ticks) return null
  const mineNow = scoreAtFrame(mine.ticks, frame)
  let ahead = 0
  for (const p of series.players.values()) {
    if (!p.score.ticks) return null
    if (scoreAtFrame(p.score.ticks, frame) > mineNow) ahead++
  }
  return { place: ahead + 1, of: series.lobbySize }
}

/** Un joueur étranger au lobby lu : rien n'est établi pour lui. */
const FOREIGN_PLAYER: PlayerInstantEstimates = {
  score: null,
  rank: null,
  shots: null,
  absences: { score: 'feuille-absente', rank: 'lobby-incomplet', shots: 'non-publie' },
}

/**
 * playerInstantEstimatesAt rend le score, le rang et les estimations de tir de `player` à `frame` (compris).
 * Un joueur qui n'était pas dans le lobby donné à la lecture est lu entièrement absent.
 */
export function playerInstantEstimatesAt(
  series: InstantEstimatesSeries,
  player: ReplayPlayer,
  frame: number,
): PlayerInstantEstimates {
  const mine = series.players.get(player.xuid)
  if (!mine) return FOREIGN_PLAYER
  const rank = rankAt(series, mine.score, frame)
  return {
    score: mine.score.ticks ? scoreAtFrame(mine.score.ticks, frame) : null,
    rank,
    shots: mine.shots.series ? shotsAt(mine.shots.series, frame) : null,
    absences: {
      score: mine.score.absence,
      rank: rank ? null : 'lobby-incomplet',
      shots: mine.shots.absence,
    },
  }
}
