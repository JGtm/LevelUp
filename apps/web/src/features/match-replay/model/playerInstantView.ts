/**
 * playerInstantView.ts — CE QUE LA FICHE D'UN JOUEUR ÉCRIT À L'IMAGE LUE, section par section. Sans React.
 *
 * Il ne calcule rien de nouveau : chaque lecture vient de son module (`playerInstantStats`, `killBreakdown`,
 * `playerInstantEstimates`, `playerThrowsPickups`, `playerEquipmentStats`, `playerMovementStats`, `playerReloadStats`,
 * `matchMedals`), lu UNE FOIS par document et par joueur (`playerInstantSources.ts`) puis interrogé par image. Ce
 * module les range en sections, et décide ce qui existe : une valeur que son module rend `null` n'a pas de
 * ligne ; une section dont aucune ligne n'existe est `null`, et le rendu n'écrit alors ni titre ni texte de
 * remplacement. Un zéro mesuré s'écrit pour les compteurs et les tableaux ; pour les listes de gestes (lancers,
 * ramassages, sprint, déplacement, sauts, rechargements) un zéro n'a pas de ligne, il n'y a rien à lister.
 *
 * Les noms des armes et des grenades viennent des tables du document dans la langue demandée (`catalogText`) ;
 * une arme sans nom n'a pas de ligne, jamais une clé écrite à sa place.
 */
import type { FragClassKey } from '../../../lib/accessibility/scales/fragClass'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { catalogText } from '../i18n/catalogLabel'
import type { ReplayLocale } from '../i18n/i18n'
import type { KillBreakdown } from './killBreakdown'
import { killBreakdownAt } from './killBreakdown'
import { medalGroupsOf, type MedalGroup } from './playerInstantMedals'
import type { PlayerShotEstimates, PlayerRank } from './playerInstantEstimates'
import { playerInstantEstimatesAt } from './playerInstantEstimates'
import type { PlayerInstantSources } from './playerInstantSources'
import { playerInstantStatsAt, type PlayerInstantStats } from './playerInstantStats'
import { playerEquipmentStatsAt, type EquipmentFamilyStats } from './playerEquipmentStats'
import { playerMovementStatsAt } from './playerMovementStats'
import { playerReloadsAt } from './playerReloadStats'
import { playerThrowsPickupsAt } from './playerThrowsPickups'

/** Frags, morts, assistances de l'identité : ceux du film à l'instant, sinon les totaux de la feuille (`fromFilm` faux). */
export interface IdentityCounters {
  kills: number
  deaths: number
  assists: number
  fromFilm: boolean
}

/** « Depuis le début du match » : une valeur par ligne, `null` = pas de ligne. */
export interface SinceView {
  fda: number | null
  currentStreak: number | null
  bestStreak: number | null
  /** Les tirs à la tête sur les éliminations dont le coup fatal est lu ; `null` tant qu'aucune n'est lue. */
  headshots: { count: number; measured: number } | null
  perfects: number | null
  respawns: number | null
  currentLifeMs: number | null
  averageLifeMs: number | null
  shieldsLost: number | null
  score: number | null
  rank: PlayerRank | null
}

/** Une ligne d'arme du tableau des éliminations ; `name: null` = arme sans nom connu. */
export interface KillsWeaponView {
  name: string | null
  atInstant: number
  share: number | null
  atMatch: number
}

/** Une classe d'arme du tableau des éliminations, et ses armes. */
export interface KillsClassView {
  classKey: FragClassKey
  atInstant: number
  share: number | null
  atMatch: number
  weapons: KillsWeaponView[]
}

/**
 * Le tableau « Éliminations par arme » et sa barre. `bar` : les parts à l'instant, par classe puis les
 * éliminations non classées (clé `unattributed`) ; vide tant que le joueur n'a aucune élimination à l'instant.
 */
export interface KillsView {
  classes: KillsClassView[]
  unclassified: { atInstant: number; share: number | null; atMatch: number } | null
  bar: { classKey: FragClassKey; share: number }[]
}

/** Les tirs et les dégâts estimés : une ligne par arme dont le document donne le nom. */
export interface ShotsView {
  /** Précision (0 à 1), touches et tirs estimés ; `null` tant qu'aucun tir n'est estimé. */
  accuracy: { ratio: number; hit: number; fired: number } | null
  damageDealt: number
  damageTaken: number
  weapons: { name: string; fired: number; damage: number }[]
}

/** Les lancers de grenade par type et les ramassages ; les zéros n'ont pas de ligne. */
export interface PickupsView {
  weapons: number | null
  grenades: number | null
}

/**
 * La section des gestes : une valeur par ligne, `null` = pas de ligne (zéro, ou grandeur que le document ne permet pas
 * d'établir). `reloadsMinimum` : le compte de rechargements est un plancher (le film perd des messages sur certaines
 * trames), l'écran le dit.
 */
export interface GesturesView {
  sprintMs: number | null
  moveMs: number | null
  jumps: number | null
  reloads: number | null
  reloadsMinimum: boolean
}

/** La fiche à une image : chaque section est `null` (ou vide) quand elle n'existe pas. */
export interface PlayerInstantView {
  identity: IdentityCounters | null
  since: SinceView | null
  kills: KillsView | null
  shots: ShotsView | null
  grenadesThrown: { name: string; count: number }[]
  pickups: PickupsView | null
  equipment: readonly EquipmentFamilyStats[]
  medals: MedalGroup[]
  gestures: GesturesView | null
}

const isNumber = (v: number | null | undefined): v is number => typeof v === 'number'

/** Les compteurs de l'identité : ceux du film à l'instant, à défaut les trois totaux de la feuille. */
function identityOf(stats: PlayerInstantStats, board: MatchScoreboardRow | undefined): IdentityCounters | null {
  if (stats.counters) return { ...stats.counters, fromFilm: true }
  if (!board) return null
  const { kills, deaths, assists } = board
  return isNumber(kills) && isNumber(deaths) && isNumber(assists) ? { kills, deaths, assists, fromFilm: false } : null
}

function sinceOf(
  stats: PlayerInstantStats,
  score: number | null,
  rank: PlayerRank | null,
): SinceView | null {
  const headshots = stats.headshots && stats.headshots.measured > 0 ? stats.headshots : null
  const since: SinceView = {
    fda: stats.fda,
    currentStreak: stats.currentStreak,
    bestStreak: stats.bestStreak,
    headshots,
    perfects: stats.perfects,
    respawns: stats.respawns,
    currentLifeMs: stats.currentLifeMs,
    averageLifeMs: stats.averageLifeMs,
    shieldsLost: stats.shieldsLost,
    score,
    rank: score === null ? null : rank,
  }
  return Object.values(since).some((v) => v !== null) ? since : null
}

/** La classe d'arme à l'instant, ses armes nommées ; la seule arme d'une classe sans nom n'a pas de ligne. */
function killsClassOf(
  cls: KillBreakdown['classes'][number],
  nameOf: (weaponKey: string) => string | null,
): KillsClassView {
  const weapons = cls.weapons.map((w) => ({
    name: nameOf(w.weaponKey),
    atInstant: w.atInstant,
    share: w.share,
    atMatch: w.atMatch,
  }))
  const lone = weapons.length === 1 && weapons[0].name === null
  return { classKey: cls.classKey, atInstant: cls.atInstant, share: cls.share, atMatch: cls.atMatch, weapons: lone ? [] : weapons }
}

function killsViewOf(b: KillBreakdown | null, nameOf: (weaponKey: string) => string | null): KillsView | null {
  if (b === null || b.totals.atMatch === 0) return null
  const classes = b.classes.map((c) => killsClassOf(c, nameOf))
  const bar = classes.flatMap((c) => (c.share === null || c.atInstant === 0 ? [] : [{ classKey: c.classKey, share: c.share }]))
  const u = b.unclassified
  if (u && u.share !== null && u.atInstant > 0) bar.push({ classKey: 'unattributed', share: u.share })
  return { classes, unclassified: u, bar }
}

function shotsViewOf(shots: PlayerShotEstimates | null, doc: PlayerInstantSources['doc'], locale: ReplayLocale): ShotsView | null {
  if (shots === null) return null
  const weapons = shots.byWeapon.flatMap((w) => {
    const name = w.weapon === null ? undefined : catalogText(doc.weaponLabels?.[w.weapon], locale)
    return name ? [{ name, fired: w.firedEstimated, damage: w.damage }] : []
  })
  return {
    accuracy:
      shots.accuracy === null ? null : { ratio: shots.accuracy, hit: shots.hit, fired: shots.firedEstimated },
    damageDealt: shots.damageDealt,
    damageTaken: shots.damageTaken,
    weapons,
  }
}

/** Un compte de ramassages, absent quand il est nul ou non publié. */
const countOrNull = (items: readonly { count: number }[] | null): number | null => {
  const total = items?.reduce((n, i) => n + i.count, 0) ?? 0
  return total > 0 ? total : null
}

/** Le temps ou le compte d'un geste, absent quand il est nul ou non établi. */
const positiveOrNull = (v: number | null): number | null => (v !== null && v > 0 ? v : null)

/**
 * playerInstantViewAt rend la fiche du joueur à `frame` : les lectures par image, rangées en sections, sans rien
 * relire du document. Les séries viennent de `readPlayerInstantSources`, lues une fois.
 */
export function playerInstantViewAt(s: PlayerInstantSources, frame: number): PlayerInstantView {
  const stats = playerInstantStatsAt(s.stats, frame)
  const estimates = playerInstantEstimatesAt(s.estimates, s.player, frame)
  const throwsPickups = playerThrowsPickupsAt(s.throws, frame)
  const movement = playerMovementStatsAt(s.movement, frame)
  const weaponsPicked = countOrNull(throwsPickups.weaponPickups)
  const grenadesPicked = countOrNull(throwsPickups.grenadePickups)
  const sprintMs = positiveOrNull(movement.sprintMs)
  const jumps = positiveOrNull(movement.jumps)
  const moveMs = positiveOrNull(movement.moveMs)
  const reloaded = playerReloadsAt(s.reloads, frame)
  const reloads = positiveOrNull(reloaded.count)
  return {
    identity: identityOf(stats, s.player.board),
    since: sinceOf(stats, estimates.score, estimates.rank),
    kills: killsViewOf(killBreakdownAt(s.kills, frame), s.weaponName),
    shots: shotsViewOf(estimates.shots, s.doc, s.locale),
    grenadesThrown: throwsPickups.grenadesThrown.flatMap((g) => {
      const name = catalogText(s.doc.grenadeLabels?.[g.rank], s.locale)
      return name ? [{ name, count: g.count }] : []
    }),
    pickups: weaponsPicked === null && grenadesPicked === null ? null : { weapons: weaponsPicked, grenades: grenadesPicked },
    equipment: playerEquipmentStatsAt(s.equipment, frame)?.families ?? [],
    medals: medalGroupsOf(stats.medals, s.doc),
    gestures:
      sprintMs === null && moveMs === null && jumps === null && reloads === null
        ? null
        : { sprintMs, moveMs, jumps, reloads, reloadsMinimum: reloads !== null && reloaded.minimum },
  }
}
