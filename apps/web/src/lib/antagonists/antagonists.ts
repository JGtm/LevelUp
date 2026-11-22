/**
 * Antagonistes d'un joueur : némésis et souffre-douleur, à chaque instant d'un match.
 *
 * Fonction pure, sans React ni accès au réseau : le rejeu et la page du match la lisent
 * toutes les deux, sur les mêmes éliminations, pour ne pas diverger.
 *
 * Deux titres, comptés parmi les ADVERSAIRES du sujet seulement :
 *  - némésis (`nemesis`) : l'adversaire qui a éliminé le sujet le plus de fois ;
 *  - souffre-douleur (`victim`) : l'adversaire que le sujet a éliminé le plus de fois.
 *
 * Règle du titre : le premier adversaire qui atteint un compte > 0 le prend ; ensuite il ne
 * change de main que sur un dépassement STRICT du compte du tenant. Un adversaire qui rejoint
 * le compte du tenant sans le dépasser ne prend rien : le changement est signalé (`egalite`)
 * et la lecture à l'instant le rend dans `tiedWith`.
 *
 * Qui est adversaire n'est pas décidé ici : l'appelant fournit `isFoe`. Il fournit aussi des
 * clés de joueur déjà rapprochées (la même clé pour un même joueur, côté tueur comme côté
 * victime). Une élimination dont l'un des deux n'est pas le sujet, dont l'autre n'est pas un
 * adversaire (trahison) ou qui est une auto-élimination n'est pas comptée.
 */

/** Une élimination : clés de joueur déjà rapprochées, instant en millisecondes. */
export interface AntagonistKill {
  killer: string
  victim: string
  t: number
}

/** `nemesis` : un adversaire élimine le sujet ; `victim` : le sujet élimine un adversaire. */
export type AntagonistRole = 'nemesis' | 'victim'

/**
 * Un événement de la vie d'un titre, à l'instant `t` d'une élimination comptée.
 *
 * `holder` est l'adversaire dont le compte vient de bouger (`count`, après l'élimination).
 * `previousHolder` est le tenant du titre juste avant (`null` pour la première prise) et
 * `previousCount` son compte, inchangé par cette élimination (0 pour la première prise).
 *  - `prise` : `holder` dépasse strictement le tenant et prend le titre ;
 *  - `egalite` : `holder` rejoint le compte du tenant sans le dépasser ; le titre reste à
 *    `previousHolder` (`count === previousCount`).
 */
export interface AntagonistChange {
  t: number
  role: AntagonistRole
  kind: 'prise' | 'egalite'
  holder: string
  previousHolder: string | null
  count: number
  previousCount: number
}

/** Le titre à un instant : son tenant, son compte et les adversaires à égalité avec lui. */
export interface AntagonistReading {
  holder: string
  count: number
  /** Adversaires au même compte que le tenant, dans l'ordre où ils l'ont rejoint. */
  tiedWith: string[]
}

/** Lecture des deux titres ; `null` tant qu'aucun adversaire n'a atteint un compte > 0. */
export interface AntagonistsAt {
  nemesis: AntagonistReading | null
  victim: AntagonistReading | null
}

/** Index des antagonistes d'un joueur, construit une fois pour un match. */
export interface Antagonists {
  /** Changements de titre des deux rôles, dans l'ordre chronologique. */
  readonly changes: readonly AntagonistChange[]
  /**
   * Titres à l'instant `t` (ms), éliminations d'instant `t` comprises. Recherche
   * dichotomique : temps en O(log n) plus la copie de `tiedWith`, sans reparcourir les
   * éliminations. Les tableaux rendus sont des copies que l'appelant peut modifier.
   */
  at(t: number): AntagonistsAt
}

/** État du titre juste après une élimination comptée. */
interface Snapshot {
  t: number
  holder: string
  count: number
  tiedWith: readonly string[]
}

/** État courant d'un rôle pendant la construction. */
interface RoleState {
  counts: Map<string, number>
  holder: string | null
  tiedWith: string[]
  snapshots: Snapshot[]
}

function newRoleState(): RoleState {
  return { counts: new Map(), holder: null, tiedWith: [], snapshots: [] }
}

/**
 * Compte une élimination de `foe` pour ce rôle à l'instant `t` ; rend le changement de titre
 * produit, ou `null` quand le titre ne bouge pas (le tenant continue, ou un suiveur reste
 * sous lui).
 */
function countFor(state: RoleState, role: AntagonistRole, foe: string, t: number): AntagonistChange | null {
  const count = (state.counts.get(foe) ?? 0) + 1
  state.counts.set(foe, count)

  let change: AntagonistChange | null = null
  const tenant = state.holder
  if (tenant === null) {
    state.holder = foe
    change = { t, role, kind: 'prise', holder: foe, previousHolder: null, count, previousCount: 0 }
  } else if (foe === tenant) {
    // Le tenant creuse l'écart : plus personne n'est à égalité avec lui.
    state.tiedWith = []
  } else {
    const tenantCount = state.counts.get(tenant) ?? 0
    if (count > tenantCount) {
      // Dépassement strict : le nouveau tenant est seul en tête.
      state.holder = foe
      state.tiedWith = []
      change = { t, role, kind: 'prise', holder: foe, previousHolder: tenant, count, previousCount: tenantCount }
    } else if (count === tenantCount) {
      state.tiedWith = [...state.tiedWith, foe]
      change = { t, role, kind: 'egalite', holder: foe, previousHolder: tenant, count, previousCount: tenantCount }
    }
  }

  const holder = state.holder as string
  state.snapshots.push({
    t,
    holder,
    count: state.counts.get(holder) ?? 0,
    tiedWith: state.tiedWith,
  })
  return change
}

/** Ordre total (instant, tueur, victime) : le résultat ne dépend pas de l'ordre d'entrée. */
function compareKills(a: AntagonistKill, b: AntagonistKill): number {
  if (a.t !== b.t) return a.t - b.t
  if (a.killer !== b.killer) return a.killer < b.killer ? -1 : 1
  if (a.victim !== b.victim) return a.victim < b.victim ? -1 : 1
  return 0
}

/** Éliminations triées, doublons (même tueur, même victime, même instant) écartés. */
function sortedUniqueKills(kills: readonly AntagonistKill[]): AntagonistKill[] {
  const valid = kills.filter((kill) => Number.isFinite(kill.t))
  valid.sort(compareKills)
  return valid.filter((kill, i) => i === 0 || compareKills(valid[i - 1], kill) !== 0)
}

/** Dernier instantané d'instant <= `t`, `null` s'il n'y en a pas (recherche dichotomique). */
function readAt(snapshots: readonly Snapshot[], t: number): AntagonistReading | null {
  let lo = 0
  let hi = snapshots.length
  while (lo < hi) {
    const mid = (lo + hi) >>> 1
    if (snapshots[mid].t <= t) lo = mid + 1
    else hi = mid
  }
  if (lo === 0) return null
  const snap = snapshots[lo - 1]
  return { holder: snap.holder, count: snap.count, tiedWith: [...snap.tiedWith] }
}

/**
 * Construit l'index des antagonistes de `subject` : un seul parcours des éliminations
 * (triées sur une copie, l'entrée n'est pas modifiée). À égalité d'instant, l'ordre de
 * traitement est celui des clés (tueur puis victime), pour un résultat reproductible.
 * Un instant non fini est écarté.
 */
export function buildAntagonists(
  kills: readonly AntagonistKill[],
  subject: string,
  isFoe: (other: string) => boolean,
): Antagonists {
  const nemesis = newRoleState()
  const victim = newRoleState()
  const changes: AntagonistChange[] = []

  for (const kill of sortedUniqueKills(kills)) {
    if (kill.killer === kill.victim) continue
    let change: AntagonistChange | null = null
    if (kill.victim === subject && isFoe(kill.killer)) {
      change = countFor(nemesis, 'nemesis', kill.killer, kill.t)
    } else if (kill.killer === subject && isFoe(kill.victim)) {
      change = countFor(victim, 'victim', kill.victim, kill.t)
    }
    if (change !== null) changes.push(change)
  }

  return {
    changes,
    at: (t) => ({ nemesis: readAt(nemesis.snapshots, t), victim: readAt(victim.snapshots, t) }),
  }
}
