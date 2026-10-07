/**
 * matchEmprise.logic.ts — LES MODÈLES PURS des cartes de l'Emprise de la Vue match (plan
 * PLAN_MATCHVIEW_EMPRISE_2026-10-06, M3.2) : contrôle des ressources objet par objet (D), prises par
 * joueur de l'équipe (E), lignes « non mesuré » de « Frags par ressource » (G) et « non mesurable » de
 * « Rendement par ressource » (H), une ligne d'« Isolement » par joueur (I), couverture de
 * l'intertitre, et LE prédicat de présence de chaque carte, lu par l'onglet et par chaque carte.
 *
 * Tout vient des blocs du Go (`emprise`, `lives_near_teammate`) : ce module
 * range, choisit la raison d'une absence dans une liste fermée, et ne calcule aucune mesure. Pur :
 * aucun React, aucune couleur, aucune chaîne de langue (les raisons sont des clés).
 */
import {
  buildPickupSheets,
  RESOURCE_ORDER,
  RESOURCE_POWER_WEAPON,
  RESOURCE_POWERUP,
  RESOURCE_RACK,
  RESOURCE_VEHICLE,
  type PickupSheets,
} from '@/features/squad/emprise/emprise.logic'
import { buildProductionRows, buildYieldRows, type ProductionRow, type YieldRow } from '@/features/squad/emprise/production.logic'
import { buildLivesModel, type LivesModel } from '@/features/timeseries/usages/usages.logic'
import type {
  MatchEmpriseBlock,
  MatchLivesNearTeammate,
  MatchScoreboardRow,
  SquadEmpriseCount,
  SquadEmpriseMatch,
  SquadEmpriseObject,
} from '@/lib/api/types'

const VEHICLES_MEASURED = 'measured'
/** États du journal des morts (contrat Go `MatchKillJournal*`). */
const KILL_JOURNAL_PUBLISHABLE = 'publishable'
const KILL_JOURNAL_UNAVAILABLE = 'unavailable'

/**
 * La raison d'un temps d'effet sans frag, selon l'état du journal : aucun frag (journal publiable),
 * frags non mesurés (non publiable), lecture indisponible (la lecture a échoué — jamais « non publiable »).
 */
const EFFECT_KILLS_REASON: Record<MatchEmpriseBlock['kill_journal'], ProductionReason> = {
  publishable: 'powerupNoKills',
  not_publishable: 'powerupKillsUnpublished',
  unavailable: 'powerupKillsUnavailable',
}
const SHEET_LOAD_FAILED = 'sheet_load_failed'

const total = (c: SquadEmpriseCount | null | undefined) => (c ? c.us + c.them : 0)

/** Le seul match du bloc (la Vue match publie un match). */
function theMatch(block: MatchEmpriseBlock | null | undefined): SquadEmpriseMatch | null {
  return block?.matches?.[0] ?? null
}

/** Le match se lit camp contre camp : filmé, équipe connue. */
function measured(m: SquadEmpriseMatch | null): boolean {
  return !!m && m.has_film && m.team_known
}

// ---------------------------------------------------------------------------
// D — Contrôle des ressources, par match
// ---------------------------------------------------------------------------

export interface MatchControlRow {
  key: string
  resource: string
  /** L'objet d'une ligne indentée ; absent pour la piste de la ressource. */
  object?: SquadEmpriseObject
  us: number
  them: number
  /** Bonus seulement : les socles vidés du match (somme des objets). */
  padsEmptied?: number
}

export interface MatchControl {
  rows: MatchControlRow[]
  /** Le nombre d'objets de râtelier (bouton de repli) ; 0 sans râtelier. */
  racks: number
  /** Prises sur un emplacement non identifié ; null sans aucune. */
  unclassified: SquadEmpriseCount | null
}

/**
 * buildMatchControl — par ressource du match (ordre `RESOURCE_ORDER`) qui a au moins une prise
 * attribuée : sa piste, puis une piste par objet pris (l'ordre du bloc). Une ressource sans prise
 * n'a pas de ligne (règle des pages sœurs) ; sans mesure (sans film, équipe inconnue), aucune ligne.
 */
export function buildMatchControl(block: MatchEmpriseBlock | null | undefined): MatchControl {
  const m = theMatch(block)
  const rows: MatchControlRow[] = []
  let racks = 0
  for (const resource of RESOURCE_ORDER) {
    const r = m?.resources?.find((x) => x.resource === resource)
    if (!m || !r || total(r.taken) <= 0) continue
    if (resource === RESOURCE_VEHICLE ? m.vehicles !== VEHICLES_MEASURED : !measured(m)) continue
    const objects = (r.objects ?? []).filter((o) => total(o.taken) > 0)
    const padsEmptied = resource === RESOURCE_POWERUP ? (r.objects ?? []).reduce((a, o) => a + (o.pads_emptied ?? 0), 0) : undefined
    rows.push({ key: resource, resource, us: r.taken.us, them: r.taken.them, padsEmptied })
    for (const o of objects) rows.push({ key: `${resource}|${o.key}`, resource, object: o, us: o.taken.us, them: o.taken.them })
    if (resource === RESOURCE_RACK) racks = objects.length
  }
  const unclassified = measured(m) && m?.unclassified_pickups && total(m.unclassified_pickups) > 0 ? m.unclassified_pickups : null
  return { rows, racks, unclassified }
}

// ---------------------------------------------------------------------------
// E — Prises par joueur
// ---------------------------------------------------------------------------

/**
 * buildMatchSheets — une fiche par joueur de l'équipe (l'ordre de `players`), SANS fiche du reste de
 * l'équipe (bots et partis : leurs prises restent comptées dans « n des m prises de l'équipe ») ;
 * une section par ressource prise par l'équipe sur le match, râteliers compris. null sans ligne.
 */
export function buildMatchSheets(block: MatchEmpriseBlock | null | undefined, nameOf: (o: SquadEmpriseObject) => string): PickupSheets | null {
  if (!block) return null
  const control = buildMatchControl(block)
  const resources = control.rows.filter((r) => !r.object && r.us > 0).map((r) => r.resource)
  if (resources.length === 0) return null
  const all = buildPickupSheets(block, nameOf, resources)
  const n = all.owners.length - 1 // la dernière fiche est le reste de l'équipe
  const cut = (xs: number[]) => xs.slice(0, n)
  return {
    owners: all.owners.slice(0, n),
    sections: all.sections.map((s) => ({
      ...s,
      totals: cut(s.totals),
      lines: s.lines.map((l) => ({ ...l, taken: cut(l.taken), kept: cut(l.kept), dropped: cut(l.dropped) })),
    })),
    dominant: all.dominant.slice(0, n),
    losses: all.losses,
  }
}

// ---------------------------------------------------------------------------
// G / H — Frags par ressource, Rendement par ressource
// ---------------------------------------------------------------------------

/** Les raisons d'une ligne sans mesure (liste fermée du plan, §3) ; les textes sont dans `MatchOwnText`. */
export type ProductionReason =
  | 'powerupKillsUnpublished'
  | 'powerupKillsUnavailable'
  | 'powerupNoEffect'
  | 'powerupNoKills'
  | 'powerZero'
  | 'sheetFailed'
  | 'vehicleUnmeasured'

export interface MatchProductionPending {
  resource: string
  reason: ProductionReason
  exposure?: ProductionRow['exposure']
}

export interface MatchProduction {
  rows: ProductionRow[]
  pending: MatchProductionPending[]
  /** Ressources dont la barre épaisse n'a pas de barre fine faute de prise mesurée (« aucune prise… »). */
  noPickupNote: string[]
}

function production(block: MatchEmpriseBlock, resource: string) {
  return (block.production ?? []).find((p) => p.resource === resource)
}

function exposureOf(block: MatchEmpriseBlock, resource: string): ProductionRow['exposure'] {
  const e = production(block, resource)?.exposure
  return e && total(e.value) > 0 ? { kind: e.kind, value: e.value } : null
}

function vehicleUnmeasured(block: MatchEmpriseBlock, m: SquadEmpriseMatch | null): boolean {
  return !!block.vehicles && !!m && m.vehicles !== VEHICLES_MEASURED
}

/**
 * buildMatchProduction — les lignes de « Frags par ressource » (le modèle des pages sœurs) et, pour
 * une ressource sans ligne, la raison :
 *   - bonus (match mesuré) : temps d'effet mais journal non publiable → frags non mesurés (barre
 *     fine gardée) ; temps d'effet, journal publiable, aucun frag → aucun frag ; ni l'un ni l'autre →
 *     aucun temps d'effet ;
 *   - armes spéciales : feuille illisible → non mesuré ; feuille lue sans frag → 0 frag ;
 *   - véhicules : titre qui les mesure et match non mesuré → non mesuré.
 */
export function buildMatchProduction(block: MatchEmpriseBlock | null | undefined): MatchProduction {
  if (!block) return { rows: [], pending: [], noPickupNote: [] }
  const rows = buildProductionRows(block)
  const has = (r: string) => rows.some((x) => x.resource === r)
  const m = theMatch(block)
  const pending: MatchProductionPending[] = []
  if (!has(RESOURCE_POWERUP) && measured(m)) {
    const exposure = exposureOf(block, RESOURCE_POWERUP)
    if (exposure) pending.push({ resource: RESOURCE_POWERUP, reason: EFFECT_KILLS_REASON[block.kill_journal], exposure })
    else pending.push({ resource: RESOURCE_POWERUP, reason: 'powerupNoEffect' })
  }
  if (!has(RESOURCE_POWER_WEAPON)) {
    if (block.sheet_unavailable === SHEET_LOAD_FAILED) pending.push({ resource: RESOURCE_POWER_WEAPON, reason: 'sheetFailed' })
    else if (production(block, RESOURCE_POWER_WEAPON)) {
      pending.push({ resource: RESOURCE_POWER_WEAPON, reason: 'powerZero', exposure: exposureOf(block, RESOURCE_POWER_WEAPON) ?? undefined })
    }
  }
  if (!has(RESOURCE_VEHICLE) && vehicleUnmeasured(block, m)) pending.push({ resource: RESOURCE_VEHICLE, reason: 'vehicleUnmeasured' })
  const noPickupNote = rows.filter((r) => r.resource === RESOURCE_POWER_WEAPON && !r.exposure && measured(m)).map((r) => r.resource)
  return { rows, pending, noPickupNote }
}

export type YieldPendingReason =
  | { kind: 'powerupUnpublished' }
  | { kind: 'powerupUnavailable' }
  | { kind: 'noEffect'; team: boolean; teamEffectMs: number; teamKills: number }
  | { kind: 'noPickup'; team: boolean; us: number; them: number }
  | { kind: 'vehicleUnmeasured' }

export interface MatchYieldPending {
  resource: string
  reason: YieldPendingReason
}

export interface MatchYield {
  rows: YieldRow[]
  pending: MatchYieldPending[]
}

/**
 * buildMatchYield — les rendements calculés (le modèle des pages sœurs) et, sur un match mesuré, la
 * raison de chaque rendement qui ne se calcule pas : journal non publiable (bonus), un camp sans
 * temps d'effet (bonus) ou sans prise (armes spéciales) — dite au lieu d'un ratio —, véhicules non
 * mesurés.
 */
export function buildMatchYield(block: MatchEmpriseBlock | null | undefined): MatchYield {
  if (!block) return { rows: [], pending: [] }
  const rows = buildYieldRows(block)
  const m = theMatch(block)
  if (!measured(m)) return { rows, pending: [] }
  const has = (r: string) => rows.some((x) => x.resource === r)
  const pending: MatchYieldPending[] = []
  const bonus = production(block, RESOURCE_POWERUP)?.exposure
  if (!has(RESOURCE_POWERUP) && bonus && total(bonus.value) > 0) {
    if (block.kill_journal === KILL_JOURNAL_UNAVAILABLE) pending.push({ resource: RESOURCE_POWERUP, reason: { kind: 'powerupUnavailable' } })
    else if (block.kill_journal !== KILL_JOURNAL_PUBLISHABLE) pending.push({ resource: RESOURCE_POWERUP, reason: { kind: 'powerupUnpublished' } })
    else if (bonus.value.us === 0 || bonus.value.them === 0) {
      pending.push({
        resource: RESOURCE_POWERUP,
        reason: { kind: 'noEffect', team: bonus.value.us === 0, teamEffectMs: bonus.value.us, teamKills: bonus.kills.us },
      })
    }
  }
  const armes = production(block, RESOURCE_POWER_WEAPON)?.exposure
  if (!has(RESOURCE_POWER_WEAPON) && armes && (armes.value.us === 0 || armes.value.them === 0)) {
    pending.push({ resource: RESOURCE_POWER_WEAPON, reason: { kind: 'noPickup', team: armes.value.us === 0, us: armes.value.us, them: armes.value.them } })
  }
  if (!has(RESOURCE_VEHICLE) && vehicleUnmeasured(block, m)) pending.push({ resource: RESOURCE_VEHICLE, reason: { kind: 'vehicleUnmeasured' } })
  return { rows, pending }
}

// ---------------------------------------------------------------------------
// I — Isolement, par joueur
// ---------------------------------------------------------------------------

export interface MatchLivesRow {
  xuid: string
  gamertag: string
  /** null : aucune vie rangée (toutes écartées, ou aucune vie terminée par une mort). */
  model: LivesModel | null
}

export interface MatchLives {
  rows: MatchLivesRow[]
  excludedUnlocated: number
  excludedNoRadar: number
  excludedUnpublishable: number
}

/** buildMatchLives — une ligne par joueur de l'équipe, dans l'ordre des fiches ; null sans aucune vie rangée. */
export function buildMatchLives(
  lives: MatchLivesNearTeammate | null | undefined,
  players: readonly { xuid: string; gamertag: string }[],
): MatchLives | null {
  const byXuid = new Map((lives?.players ?? []).map((p) => [p.xuid, p]))
  const rows = players.map((p) => ({ xuid: p.xuid, gamertag: p.gamertag, model: buildLivesModel(byXuid.get(p.xuid)) }))
  if (!rows.some((r) => r.model)) return null
  const sum = (k: 'excluded_unlocated' | 'excluded_no_radar' | 'excluded_unpublishable') =>
    (lives?.players ?? []).reduce((a, p) => a + (p[k] ?? 0), 0)
  return {
    rows,
    excludedUnlocated: sum('excluded_unlocated'),
    excludedNoRadar: sum('excluded_no_radar'),
    excludedUnpublishable: sum('excluded_unpublishable'),
  }
}

// ---------------------------------------------------------------------------
// Couverture et présence
// ---------------------------------------------------------------------------

/** L'intertitre « Équipement et terrain » : film décodé ou non, joueurs présents à la fin (bots compris). */
export function matchCoverage(block: MatchEmpriseBlock | null | undefined, scoreboard: readonly MatchScoreboardRow[]): { filmed: boolean; present: number } | null {
  if (!block) return null
  return { filmed: !!theMatch(block)?.has_film, present: scoreboard.filter((r) => r.left_in_progress !== true).length }
}

export interface MatchEmpriseModels {
  control: MatchControl
  sheets: PickupSheets | null
  production: MatchProduction
  yield: MatchYield
  lives: MatchLives | null
}

export function buildMatchEmpriseModels(
  block: MatchEmpriseBlock | null | undefined,
  lives: MatchLivesNearTeammate | null | undefined,
  nameOf: (o: SquadEmpriseObject) => string,
): MatchEmpriseModels {
  return {
    control: buildMatchControl(block),
    sheets: buildMatchSheets(block, nameOf),
    production: buildMatchProduction(block),
    yield: buildMatchYield(block),
    lives: buildMatchLives(lives, block?.players ?? []),
  }
}

export interface MatchEmpriseCards {
  control: boolean
  sheets: boolean
  production: boolean
  yield: boolean
  lives: boolean
}

/**
 * LE prédicat de présence des cartes de l'Emprise de « Équipement et terrain » : l'onglet pose l'intertitre,
 * chaque carte se monte. « Outils de destruction » a le sien (`blockPredicates.hasWeaponTools`).
 */
export function matchEmpriseCards(models: MatchEmpriseModels): MatchEmpriseCards {
  return {
    control: models.control.rows.length > 0,
    sheets: models.sheets != null,
    production: models.production.rows.length + models.production.pending.length > 0,
    yield: models.yield.rows.length + models.yield.pending.length > 0,
    lives: models.lives != null,
  }
}

/** Au moins une carte de l'Emprise dans la section « Équipement et terrain ». */
export function hasEquipmentEmpriseCard(cards: MatchEmpriseCards): boolean {
  return cards.control || cards.sheets || cards.production || cards.yield || cards.lives
}
