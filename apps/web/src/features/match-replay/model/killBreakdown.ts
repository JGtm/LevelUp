/**
 * killBreakdown.ts — LES ÉLIMINATIONS D'UN JOUEUR JUSQU'À L'IMAGE LUE, PAR CLASSE D'ARME PUIS PAR ARME. Sans React.
 *
 * DEUX TEMPS, comme `playerInstantStats.ts` : `readKillBreakdownSeries` lit une fois par joueur et par document
 * les éliminations du fil et précalcule un instantané par élimination ; `killBreakdownAt` répond pour une image
 * par dichotomie, sans rien relire du document.
 *
 * LA CLASSE EST CELLE DE LA SYNTHÈSE, PAS UNE RÈGLE NOUVELLE. La classe d'un frag vient du registre d'armes du
 * titre (clé d'arme -> classe, le même registre que la Répartition des frags). Le document de rejeu ne porte pas
 * cette classe : l'appelant FOURNIT la table `weaponClassByKey`, et ce module n'écrit aucun classement. Une
 * élimination dont la clé d'arme est absente, ou absente de la table (mêlée générique, objet, arme inconnue),
 * tombe dans `unclassified` : jamais répartie au hasard, jamais jetée.
 *
 * INVARIANTS (à chaque image) : la somme des classes plus `unclassified` vaut `totals.atInstant` ; la somme des
 * armes d'une classe vaut le compte de la classe. Une part est `null` quand le total à l'instant vaut zéro. Le
 * périmètre (classes, armes) est celui du MATCH entier : une classe ou une arme encore à zéro à l'instant reste
 * listée, part absente.
 *
 * Les BOTS comptent comme tout joueur : leurs éliminations se reconnaissent par `identityKeys`, comme dans
 * `playerInstantStats.ts`. Absence = `null` (fil absent, ou sans aucune élimination), jamais un zéro inventé.
 *
 * Module pur : ni DOM, ni texte d'interface, ni couleur.
 */
import type { FragClassKey } from '../../../lib/accessibility/scales/fragClass'
import { frameToMs, lastIndexAt } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import type { ReplayFeedEntry } from './killFeedLogic'
import { identityKeys } from './playerInstantStats'

/** Une arme du joueur : sa clé du registre, son compte à l'instant, son compte au match, sa part à l'instant. */
export interface KillBreakdownWeapon {
  weaponKey: string
  atInstant: number
  atMatch: number
  /** Part dans les éliminations à l'instant du joueur (0 à 1). `null` quand ce total vaut zéro. */
  share: number | null
}

/** Une classe d'arme : ses comptes, sa part, et ses armes (même ordre que la liste des classes). */
export interface KillBreakdownClass {
  classKey: FragClassKey
  atInstant: number
  atMatch: number
  share: number | null
  /** Toutes les armes de la classe ayant une élimination au match, même à zéro à l'instant. */
  weapons: KillBreakdownWeapon[]
}

/** Les éliminations sans classe établie. `null` dans `KillBreakdown` quand il n'y en a aucune au match. */
export interface KillBreakdownUnclassified {
  atInstant: number
  atMatch: number
  share: number | null
}

export interface KillBreakdown {
  /** Classes ayant au moins une élimination au match, par compte au match décroissant, puis par identifiant. */
  classes: KillBreakdownClass[]
  unclassified: KillBreakdownUnclassified | null
  /** Éliminations du joueur : à l'instant et au match. Égal à la somme des classes plus `unclassified`. */
  totals: { atInstant: number; atMatch: number }
}

export interface KillBreakdownInput {
  doc: Pick<ReplayDocumentReady, 'frameIntervalMs'>
  player: ReplayPlayer
  /** Le fil du rejeu (`ReplayModel.feed`). `null` = absent. */
  feed: readonly ReplayFeedEntry[] | null
  /** Classe de chaque arme du registre du titre, par clé d'arme (`ReplayKill.weaponKey`). Fournie par l'appelant. */
  weaponClassByKey: ReadonlyMap<string, FragClassKey>
}

/** Un instantané : comptes cumulés jusqu'à une élimination comprise. `perWeapon` suit l'index de `plan.weapons`. */
interface KillSnapshot {
  t: number
  total: number
  unclassified: number
  perWeapon: readonly number[]
}

interface PlanWeapon {
  weaponKey: string
  classKey: FragClassKey
  atMatch: number
}

interface PlanClass {
  classKey: FragClassKey
  atMatch: number
  /** Index dans `plan.weapons`, dans l'ordre d'affichage. */
  weaponIndexes: number[]
}

/** La structure figée au match : classes, armes, et comptes au match. */
interface BreakdownPlan {
  classes: PlanClass[]
  weapons: PlanWeapon[]
  unclassifiedAtMatch: number
  totalAtMatch: number
}

/** Ce que `readKillBreakdownSeries` range pour `killBreakdownAt`. */
export interface KillBreakdownSeries {
  readonly doc: Pick<ReplayDocumentReady, 'frameIntervalMs'>
  readonly plan: BreakdownPlan
  readonly snapshots: readonly KillSnapshot[]
}

/** Part d'un compte dans un total ; absente quand le total vaut zéro. */
function shareOf(part: number, total: number): number | null {
  return total > 0 ? part / total : null
}

/** Les éliminations du joueur (ses clés d'identité), triées par instant, avec la clé de leur arme. */
function killsOf(
  feed: readonly ReplayFeedEntry[],
  keys: readonly string[],
): { t: number; weaponKey: string }[] {
  const out: { t: number; weaponKey: string }[] = []
  for (const e of feed) {
    const k = e.kill
    if (k && keys.includes(k.xuid)) out.push({ t: k.replayMs, weaponKey: k.weaponKey })
  }
  return out.sort((a, b) => a.t - b.t)
}

/** Ordre d'affichage : compte au match décroissant, puis identifiant croissant (stable). */
function byMatchThenKey<T extends { atMatch: number }>(keyOf: (x: T) => string): (a: T, b: T) => number {
  return (a: T, b: T) => b.atMatch - a.atMatch || keyOf(a).localeCompare(keyOf(b))
}

/** Compte au match par arme établie (classe connue), puis le plan des classes et des armes. */
function buildPlan(
  kills: readonly { weaponKey: string }[],
  weaponClassByKey: ReadonlyMap<string, FragClassKey>,
): BreakdownPlan {
  const counts = new Map<string, { classKey: FragClassKey; atMatch: number }>()
  let unclassifiedAtMatch = 0
  for (const k of kills) {
    const classKey = weaponClassByKey.get(k.weaponKey)
    if (classKey === undefined) {
      unclassifiedAtMatch++
      continue
    }
    const c = counts.get(k.weaponKey) ?? { classKey, atMatch: 0 }
    c.atMatch++
    counts.set(k.weaponKey, c)
  }
  const weapons: PlanWeapon[] = [...counts.entries()]
    .map(([weaponKey, c]) => ({ weaponKey, classKey: c.classKey, atMatch: c.atMatch }))
    .sort(byMatchThenKey<PlanWeapon>((w) => w.weaponKey))
  const byClass = new Map<FragClassKey, PlanClass>()
  weapons.forEach((w, index) => {
    const cls = byClass.get(w.classKey) ?? { classKey: w.classKey, atMatch: 0, weaponIndexes: [] }
    cls.atMatch += w.atMatch
    cls.weaponIndexes.push(index)
    byClass.set(w.classKey, cls)
  })
  const classes = [...byClass.values()].sort(byMatchThenKey<PlanClass>((c) => c.classKey))
  return {
    classes,
    weapons,
    unclassifiedAtMatch,
    totalAtMatch: kills.length,
  }
}

/** Un instantané par élimination : comptes cumulés, arme par arme selon l'index du plan. */
function buildSnapshots(
  kills: readonly { t: number; weaponKey: string }[],
  plan: BreakdownPlan,
): KillSnapshot[] {
  const indexOf = new Map(plan.weapons.map((w, i): [string, number] => [w.weaponKey, i]))
  const perWeapon = plan.weapons.map(() => 0)
  let total = 0
  let unclassified = 0
  return kills.map((k) => {
    total++
    const index = indexOf.get(k.weaponKey)
    if (index === undefined) unclassified++
    else perWeapon[index]++
    return { t: k.t, total, unclassified, perWeapon: [...perWeapon] }
  })
}

/** readKillBreakdownSeries lit, une fois, les éliminations du joueur et précalcule ses instantanés. */
export function readKillBreakdownSeries(input: KillBreakdownInput): KillBreakdownSeries | null {
  const { doc, player, feed, weaponClassByKey } = input
  if (feed === null || !feed.some((e) => e.kill)) return null
  const kills = killsOf(feed, identityKeys(player))
  const plan = buildPlan(kills, weaponClassByKey)
  return { doc, plan, snapshots: buildSnapshots(kills, plan) }
}

/** Le détail d'une classe à l'instant : somme de ses armes, parts rapportées au total à l'instant. */
function classAt(
  cls: PlanClass,
  plan: BreakdownPlan,
  snap: KillSnapshot | null,
  totalAtInstant: number,
): KillBreakdownClass {
  const weapons: KillBreakdownWeapon[] = cls.weaponIndexes.map((index) => {
    const w = plan.weapons[index]
    const atInstant = snap ? snap.perWeapon[index] : 0
    return { weaponKey: w.weaponKey, atInstant, atMatch: w.atMatch, share: shareOf(atInstant, totalAtInstant) }
  })
  const atInstant = weapons.reduce((sum, w) => sum + w.atInstant, 0)
  return {
    classKey: cls.classKey,
    atInstant,
    atMatch: cls.atMatch,
    share: shareOf(atInstant, totalAtInstant),
    weapons,
  }
}

/** killBreakdownAt rend la répartition des éliminations du joueur jusqu'à `frame` (compris). `null` = absent. */
export function killBreakdownAt(series: KillBreakdownSeries | null, frame: number): KillBreakdown | null {
  if (series === null) return null
  const i = lastIndexAt(series.snapshots, frameToMs(frame, series.doc))
  const snap = i >= 0 ? series.snapshots[i] : null
  const { plan } = series
  const totalAtInstant = snap ? snap.total : 0
  const unclassifiedAtInstant = snap ? snap.unclassified : 0
  return {
    classes: plan.classes.map((cls) => classAt(cls, plan, snap, totalAtInstant)),
    unclassified:
      plan.unclassifiedAtMatch > 0
        ? {
            atInstant: unclassifiedAtInstant,
            atMatch: plan.unclassifiedAtMatch,
            share: shareOf(unclassifiedAtInstant, totalAtInstant),
          }
        : null,
    totals: { atInstant: totalAtInstant, atMatch: plan.totalAtMatch },
  }
}
