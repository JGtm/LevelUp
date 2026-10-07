/**
 * usages.logic.ts — LES MODÈLES PURS de l'onglet « Usages » des Séries temporelles : l'Emprise
 * appliquée aux matchs solo du périmètre (plan PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05, maquette v4).
 *
 * Les briques de l'Escouade (`squad/emprise`, `squad/objectif`) font le gros du travail ; ce fichier
 * leur donne ce que la page solo sait (l'index des matchs depuis `match_rows`), adapte la grille par
 * carte (`emprise.maps`, calculée côté Go) à la table de l'Emprise, et bâtit les modèles des cartes
 * propres à l'onglet (Mes prises, Équipement, Mes vies). `usagesSections` est LE prédicat des blocs :
 * la page l'utilise bloc par bloc ET pour son état vide — les deux ne peuvent pas diverger. Aucun
 * branchement sur le titre : tout se lit dans les blocs (Halo 5 sans film n'a que la feuille de match).
 *
 * Pur : aucun React, aucune couleur, aucune chaîne de langue.
 */
import { gridHasFilmRows } from '@/features/squad/emprise/empriseContent'
import {
  RESOURCE_RACK,
  buildControlRows,
  buildMatchGrid,
  buildPickupSheets,
  buildResourceFil,
  type ControlRow,
  type EmpriseMatchIndex,
  type GridSection,
  type PickupLosses,
  type ResourceFil,
} from '@/features/squad/emprise/emprise.logic'
import { buildProductionRows, buildYieldRows, type ProductionRow, type YieldRow } from '@/features/squad/emprise/production.logic'
import { buildVehicleCoverage, type VehicleCoverage } from '@/features/squad/emprise/vehicles.logic'
import { objectiveMatches } from '@/features/squad/formes/model/objectives'
import type {
  EmpriseEquipmentFamily,
  EmpriseMapColumn,
  SoloEmpriseBlock,
  SquadEmpriseMatch,
  SquadEmpriseObject,
  SquadFormesBlock,
  TimeseriesLivesNearTeammate,
  TimeseriesMatchRow,
  TimeseriesPageResponse,
} from '@/lib/api/types'
import { outcomeCodeToValue } from '@/lib/outcome'

/** États publiés par le Go pour un match mesuré (contrat `domain.EmpriseTiersMeasured`, `EmpriseVehiclesMeasured`). */
const MEASURED = 'measured'
const NOT_MEASURED = 'not_measured'
/** La clé de la colonne « Autres cartes » (la colonne de repli n'a ni clé ni libellé de carte). */
const OTHER_MAPS_KEY = 'others'

// ---------------------------------------------------------------------------
// Les matchs de la page
// ---------------------------------------------------------------------------

/**
 * L'index des matchs des Séries temporelles, depuis `match_rows` : date, carte (nom FR d'abord, comme
 * les axes de la page), liste de jeu, résultat. Ni score ni drapeau de dominance : sur une période, le
 * fil ne porte pas d'encoche (D12).
 */
export function timeseriesMatchIndex(rows: TimeseriesMatchRow[]): EmpriseMatchIndex {
  return new Map(
    rows.map((r) => [
      r.match_id,
      {
        matchId: r.match_id,
        startTime: r.start_time ?? '',
        map: r.map_name_fr || r.map_name || '',
        mode: r.playlist_name ?? '',
        outcome: outcomeCodeToValue(r.outcome),
        score: null,
        dominance: undefined,
      },
    ]),
  )
}

/** Matchs filmés sur le périmètre : la légende de « au fil des matchs » et le sous-titre du bilan. */
export function empriseCoverage(block: SoloEmpriseBlock): { filmed: number; total: number } {
  const matches = block.matches ?? []
  return { filmed: matches.filter((m) => m.has_film).length, total: block.matches_total || matches.length }
}

// ---------------------------------------------------------------------------
// Contrôle des ressources, carte par carte
// ---------------------------------------------------------------------------

/** Une colonne de la grille par carte ; `name` vide et `otherMaps` > 0 pour « Autres cartes ». */
export interface MapColumnInfo {
  key: string
  name: string
  otherMaps: number
  matches: number
  filmed: number
  wins: number
  losses: number
  others: number
}

export interface MapGrid {
  columns: MapColumnInfo[]
  sections: GridSection[]
}

function mapKeyOf(c: EmpriseMapColumn): string {
  if ((c.other_maps ?? 0) > 0) return OTHER_MAPS_KEY
  return c.map_key || `label:${c.map_label ?? ''}`
}

/**
 * Une colonne lue comme un match de la grille de l'Emprise : filmée si un de ses matchs l'est, camp
 * connu si un de ses matchs est mesuré, niveaux et véhicules mesurés si un de ses matchs l'est — les
 * sommes de la colonne (Go) ne portent que sur les matchs où l'objet se lit.
 */
function asGridMatch(c: EmpriseMapColumn): SquadEmpriseMatch {
  return {
    match_id: mapKeyOf(c),
    has_film: c.matches_filmed > 0,
    team_known: c.matches_measured > 0,
    tiers: c.matches_tiers > 0 ? MEASURED : NOT_MEASURED,
    vehicles: c.vehicles_measured > 0 ? MEASURED : NOT_MEASURED,
    resources: c.resources,
    power_weapon_kills: c.power_weapon_kills,
  }
}

/**
 * buildMapGrid — la grille « carte par carte » : les colonnes du Go (la plus jouée d'abord, puis
 * « Autres cartes »), les lignes et les cases de la grille de l'Emprise (mêmes règles : sans film,
 * camp inconnu, non classé, rien à prendre, qui chez moi).
 */
export function buildMapGrid(block: SoloEmpriseBlock): MapGrid {
  const cols = block.maps ?? []
  const { sections } = buildMatchGrid({ ...block, matches: cols.map(asGridMatch) }, new Map())
  const columns = cols.map((c) => ({
    key: mapKeyOf(c),
    name: c.map_label ?? '',
    otherMaps: c.other_maps ?? 0,
    matches: c.matches,
    filmed: c.matches_filmed,
    wins: c.wins,
    losses: c.losses,
    others: c.others,
  }))
  return { columns, sections }
}

// ---------------------------------------------------------------------------
// Mes prises dans mon camp
// ---------------------------------------------------------------------------

export interface MineRow {
  object: SquadEmpriseObject
  me: number
  rest: number
  /** Les prises de mon camp sur l'objet (moi + reste). */
  camp: number
}

export interface MineGroup {
  resource: string
  /** Les armes de râtelier, repliées derrière leur intertitre. */
  folded: boolean
  rows: MineRow[]
}

export interface MinePickups {
  groups: MineGroup[]
  /** L'échelle commune des barres : les prises de mon camp sur l'objet le plus pris. */
  max: number
  losses: PickupLosses | null
}

/**
 * buildMinePickups — par ressource du bilan, les objets pris par mon camp, triés par volume de mon camp
 * (l'ordre de `buildPickupSheets`) ; moi = ma fiche, reste = le camp moins moi. Sans prise : null.
 */
export function buildMinePickups(block: SoloEmpriseBlock, nameOf: (o: SquadEmpriseObject) => string): MinePickups | null {
  const sheets = buildPickupSheets(block, nameOf)
  const groups = sheets.sections
    .map((s) => ({
      resource: s.resource,
      folded: s.resource === RESOURCE_RACK,
      rows: s.lines.map((l) => ({ object: l.object, me: l.taken[0] ?? 0, rest: l.camp - (l.taken[0] ?? 0), camp: l.camp })),
    }))
    .filter((g) => g.rows.length > 0)
  if (groups.length === 0) return null
  const max = Math.max(...groups.flatMap((g) => g.rows.map((r) => r.camp)))
  return { groups, max, losses: sheets.losses }
}

/** Mes prises et celles de mon camp sur une ressource (vue compacte de « Mes prises »). */
export interface MineResource {
  resource: string
  me: number
  camp: number
}

/** mineByResource — par ressource, la somme de ses objets (moi, mon camp), dans l'ordre du bilan. */
export function mineByResource(mine: MinePickups): MineResource[] {
  return mine.groups.map((g) => ({
    resource: g.resource,
    me: g.rows.reduce((a, r) => a + r.me, 0),
    camp: g.rows.reduce((a, r) => a + r.camp, 0),
  }))
}

// ---------------------------------------------------------------------------
// Équipement pris, et ce que j'en ai fait
// ---------------------------------------------------------------------------

/** [servi, gardé, lâché]. */
export type EquipmentParts = [number, number, number]

export type EquipmentRow =
  | { family: string; measured: true; me: EquipmentParts; rest: EquipmentParts; takenMe: number }
  | { family: string; measured: false; droppedMe: number }

/**
 * La famille a-t-elle été tenue par au moins un joueur du lobby (mon camp, l'adversaire, les joueurs
 * sans camp connu) ? Mesurée : servi + gardé + lâché du lobby ; non mesurée : lâchers du lobby.
 */
function heldInLobby(f: EmpriseEquipmentFamily): boolean {
  if (!f.measured) return (f.dropped_lobby ?? 0) > 0
  const l = f.lobby
  return (l?.used ?? 0) + (l?.kept ?? 0) + (l?.dropped ?? 0) > 0
}

function equipmentRow(f: EmpriseEquipmentFamily): EquipmentRow {
  if (!f.measured) return { family: f.family, measured: false, droppedMe: f.dropped_me ?? 0 }
  const parts = (o: EmpriseEquipmentFamily['me']): EquipmentParts => [o?.used ?? 0, o?.kept ?? 0, o?.dropped ?? 0]
  return { family: f.family, measured: true, me: parts(f.me), rest: parts(f.rest), takenMe: f.me?.taken ?? 0 }
}

/**
 * buildEquipmentRows — une ligne par famille tenue dans le lobby, dans l'ordre du Go (D4) ; une
 * famille que personne n'a tenue n'a pas de ligne, même quand d'autres en ont (aucune ligne : la
 * carte se retire).
 */
export function buildEquipmentRows(block: SoloEmpriseBlock): EquipmentRow[] {
  return (block.equipment?.families ?? []).filter(heldInLobby).map(equipmentRow)
}

// ---------------------------------------------------------------------------
// Mes vies : près d'un coéquipier ou seul
// ---------------------------------------------------------------------------

export interface LivesModel {
  near: { lives: number; kills: number }
  alone: { lives: number; kills: number }
  lives: number
  livesNearShare: number
  kills: number
  /** Part des frags tombés pendant les vies « près » ; null sans frag. */
  killsNearShare: number | null
  /** Frags par vie de chaque côté ; null quand le côté n'a aucune vie. */
  perLifeNear: number | null
  perLifeAlone: number | null
  excludedUnlocated: number
  excludedNoRadar: number
  /** Vies d'un match dont le journal des morts n'est pas publiable (aucun frag lu). */
  excludedUnpublishable: number
}

/** buildLivesModel — la carte « Mes vies » ; null sans vie rangée (bloc absent, ou toutes écartées). */
export function buildLivesModel(b: TimeseriesLivesNearTeammate | null | undefined): LivesModel | null {
  if (!b) return null
  const lives = b.near.lives + b.alone.lives
  if (lives <= 0) return null
  const kills = b.near.kills + b.alone.kills
  return {
    near: b.near,
    alone: b.alone,
    lives,
    livesNearShare: b.near.lives / lives,
    kills,
    killsNearShare: kills > 0 ? b.near.kills / kills : null,
    perLifeNear: b.near.lives > 0 ? b.near.kills / b.near.lives : null,
    perLifeAlone: b.alone.lives > 0 ? b.alone.kills / b.alone.lives : null,
    excludedUnlocated: b.excluded_unlocated,
    excludedNoRadar: b.excluded_no_radar,
    excludedUnpublishable: b.excluded_unpublishable,
  }
}

// ---------------------------------------------------------------------------
// Les modèles de l'onglet, et le prédicat des blocs
// ---------------------------------------------------------------------------

export interface UsagesModels {
  block: SoloEmpriseBlock | null
  coverage: { filmed: number; total: number }
  controlRows: ControlRow[]
  fil: ResourceFil | null
  mapGrid: MapGrid | null
  mine: MinePickups | null
  production: ProductionRow[]
  yieldRows: YieldRow[]
  vehicleCoverage: VehicleCoverage | null
  lives: LivesModel | null
  equipment: EquipmentRow[]
  /** Le bloc d'objectif, seulement quand le périmètre a au moins un match à objectif. */
  objective: SquadFormesBlock | null
}

/** buildUsagesModels — tous les modèles de l'onglet depuis la réponse de page (aucune requête). */
export function buildUsagesModels(data: TimeseriesPageResponse, nameOf: (o: SquadEmpriseObject) => string): UsagesModels {
  const block = data.emprise ?? null
  const formes = data.formes_retenues
  return {
    block,
    coverage: block ? empriseCoverage(block) : { filmed: 0, total: 0 },
    controlRows: block ? buildControlRows(block) : [],
    fil: block ? buildResourceFil(block, timeseriesMatchIndex(data.match_rows ?? [])) : null,
    mapGrid: block ? buildMapGrid(block) : null,
    mine: block ? buildMinePickups(block, nameOf) : null,
    production: block ? buildProductionRows(block) : [],
    yieldRows: block ? buildYieldRows(block) : [],
    vehicleCoverage: block ? buildVehicleCoverage(block) : null,
    lives: buildLivesModel(data.lives_near_teammate),
    equipment: block ? buildEquipmentRows(block) : [],
    objective: formes && objectiveMatches(formes).length > 0 ? formes : null,
  }
}

export interface UsagesSections {
  range: boolean
  bilan: boolean
  carte: boolean
  mine: boolean
  prendre: boolean
  lives: boolean
  objectif: boolean
  equipment: boolean
}

/**
 * usagesSections — les blocs que l'onglet rend, dans l'ordre de la page. `hasWeaponRange` = la
 * capability produit `weapon_range` du titre (Halo 5 ne la déclare pas). Les modèles déjà bâtis par
 * la page se passent en dernier argument (sinon ils sont bâtis ici).
 */
export function usagesSections(
  data: TimeseriesPageResponse,
  hasWeaponRange: boolean,
  nameOf: (o: SquadEmpriseObject) => string,
  models: UsagesModels = buildUsagesModels(data, nameOf),
): UsagesSections {
  return {
    range: hasWeaponRange && data.weapon_range != null,
    bilan: models.controlRows.length > 0,
    carte: gridHasFilmRows(models.mapGrid),
    mine: models.mine != null,
    prendre: models.production.length > 0 || models.yieldRows.length > 0,
    lives: models.lives != null,
    objectif: models.objective != null,
    equipment: models.equipment.length > 0,
  }
}
