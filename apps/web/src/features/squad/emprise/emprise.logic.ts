/**
 * emprise.logic.ts — LES MODÈLES PURS des cartes de l'onglet « Emprise » de l'Escouade (lot L5
 * du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 ; maquette
 * `.ai/V7.5/MAQUETTE_ONGLET_TACTIQUE_ESCOUADE_2026-09-26.html`, bloc « Proposition »).
 *
 * Toutes les cartes lisent le bloc `squad_emprise` (périmètre D2 : composition exacte ∩
 * filtres), calculé côté Go (lot L4) ; le résultat, le score, la dominance, l'heure, la carte et
 * le mode de chaque match se joignent depuis l'historique de la page (`match_history`) par
 * `match_id`, comme au lot L3.2 — un match sans ligne d'historique garde sa place, sans résultat.
 *
 * LES RESSOURCES SONT UNE LISTE : chaque carte parcourt les ressources que le bloc publie, dans
 * l'ordre de `RESOURCE_ORDER` ; une ressource absente n'a ni ligne, ni courbe, ni section. Une
 * ressource que le web ne sait pas encore nommer (les véhicules, lot L7) n'est pas rendue tant
 * qu'elle n'a pas son entrée ici et dans les textes : l'ajouter ne demande aucune refonte.
 *
 * Pur : aucun React, aucune couleur, aucune chaîne de langue.
 */
import { asDominance, type DominanceValue, type OutcomeValue } from '@/components/charts/outcomeSequence'
import type {
  SquadEmpriseBlock,
  SquadEmpriseCount,
  SquadEmpriseMatch,
  SquadEmpriseObject,
  SquadMatchHistoryRow,
} from '@/lib/api/types'
import { outcomeCodeToValue } from '@/lib/outcome'

/** Clés des ressources (contrat Go `domain.EmpriseResource*`). */
export const RESOURCE_POWERUP = 'powerup'
export const RESOURCE_POWER_WEAPON = 'power_weapon'
export const RESOURCE_RACK = 'rack'

/** L'ordre d'affichage des ressources que l'onglet sait rendre. */
export const RESOURCE_ORDER: readonly string[] = [RESOURCE_POWERUP, RESOURCE_POWER_WEAPON, RESOURCE_RACK]

/** État des niveaux de socle d'un match mesuré (contrat Go `domain.EmpriseTiersMeasured`). */
const TIERS_MEASURED = 'measured'

/** Les ressources connues d'une liste, sans doublon, dans l'ordre d'affichage. */
function orderedResources(list: Iterable<string>): string[] {
  const seen = new Set<string>()
  for (const r of list) if (RESOURCE_ORDER.includes(r)) seen.add(r)
  return RESOURCE_ORDER.filter((r) => seen.has(r))
}

const total = (c: SquadEmpriseCount | null | undefined) => (c ? c.us + c.them : 0)

// ---------------------------------------------------------------------------
// Contrôle des ressources (bilan de la soirée)
// ---------------------------------------------------------------------------

/** Une piste : notre camp contre l'adversaire sur une ressource ; `share` = notre part (0..1). */
export interface ControlRow {
  resource: string
  us: number
  them: number
  share: number
}

/** Une piste par ressource du bilan qui a au moins une prise attribuée. */
export function buildControlRows(block: SquadEmpriseBlock): ControlRow[] {
  const byResource = new Map((block.resources ?? []).map((r) => [r.resource, r]))
  return orderedResources(byResource.keys()).flatMap((resource) => {
    const taken = byResource.get(resource)!.taken
    const n = total(taken)
    return n > 0 ? [{ resource, us: taken.us, them: taken.them, share: taken.us / n }] : []
  })
}

// ---------------------------------------------------------------------------
// Le match, joint à l'historique de la page
// ---------------------------------------------------------------------------

export interface EmpriseMatchInfo {
  matchId: string
  /** Heure de début (ISO), carte et mode : de l'historique ; vides sans ligne d'historique. */
  startTime: string
  map: string
  mode: string
  outcome: OutcomeValue | null
  score: string | null
  dominance: DominanceValue | undefined
}

function matchInfo(m: SquadEmpriseMatch, byId: Map<string, SquadMatchHistoryRow>): EmpriseMatchInfo {
  const h = byId.get(m.match_id)
  return {
    matchId: m.match_id,
    startTime: h?.start_time ?? '',
    map: h?.map_ui ?? '',
    mode: h?.mode_ui ?? '',
    outcome: h ? outcomeCodeToValue(h.outcome) : null,
    score: h?.score_label || null,
    dominance: asDominance(h?.dominance_flag),
  }
}

function historyIndex(history: SquadMatchHistoryRow[]): Map<string, SquadMatchHistoryRow> {
  return new Map(history.map((h) => [h.match_id, h]))
}

function matchResource(m: SquadEmpriseMatch, resource: string) {
  return (m.resources ?? []).find((r) => r.resource === resource)
}

// ---------------------------------------------------------------------------
// Contrôle des ressources au fil de la session
// ---------------------------------------------------------------------------

/** La part d'un match sur une ressource, et le cumul de la soirée jusqu'à lui. */
export interface FilPoint {
  us: number
  them: number
  share: number
  cumUs: number
  cumTotal: number
  cumulative: number
}

export interface ResourceFilMatch extends EmpriseMatchInfo {
  /** Par ressource : le point du match, ou null (rien à prendre sur la carte, sans film, camp inconnu). */
  points: Record<string, FilPoint | null>
}

export interface ResourceFil {
  /** Les ressources tracées : celles du bilan. */
  resources: string[]
  matches: ResourceFilMatch[]
}

/**
 * buildResourceFil — les matchs du périmètre dans l'ordre de la soirée ; pour chaque ressource
 * du bilan, la part du match et notre part CUMULÉE des prises depuis le premier match (somme de
 * nos prises / somme des prises). Un match sans la ressource n'a pas de point : la courbe file
 * jusqu'au suivant. Un match sans donnée (sans film, ou filmé au camp inconnu) n'en a aucun.
 */
export function buildResourceFil(block: SquadEmpriseBlock, history: SquadMatchHistoryRow[]): ResourceFil {
  const resources = buildControlRows(block).map((r) => r.resource)
  const byId = historyIndex(history)
  const cum = new Map(resources.map((r) => [r, { us: 0, total: 0 }]))
  const matches = (block.matches ?? []).map((m) => {
    const points: Record<string, FilPoint | null> = {}
    for (const resource of resources) {
      const taken = m.has_film && m.team_known ? matchResource(m, resource)?.taken : undefined
      const n = total(taken)
      if (!taken || n <= 0) {
        points[resource] = null
        continue
      }
      const c = cum.get(resource)!
      c.us += taken.us
      c.total += n
      points[resource] = {
        us: taken.us,
        them: taken.them,
        share: taken.us / n,
        cumUs: c.us,
        cumTotal: c.total,
        cumulative: c.us / c.total,
      }
    }
    return { ...matchInfo(m, byId), points }
  })
  return { resources, matches }
}

// ---------------------------------------------------------------------------
// Répartition des prises dans l'escouade
// ---------------------------------------------------------------------------

/** Une fiche : un joueur de l'escouade, ou le reste du camp (`xuid` null). */
export interface PickupOwner {
  xuid: string | null
  gamertag: string
}

/** Un objet de la soirée, une valeur par fiche (dans l'ordre des fiches). */
export interface PickupLine {
  object: SquadEmpriseObject
  taken: number[]
  /** Bonus seulement : gardés sans être activés / lâchés en mourant (0 pour une arme). */
  kept: number[]
  dropped: number[]
  /** Les prises de notre camp sur cet objet. */
  camp: number
}

export interface PickupSection {
  resource: string
  lines: PickupLine[]
  /** Par fiche : ses prises de la ressource. */
  totals: number[]
  /** Les prises de notre camp sur la ressource. */
  camp: number
}

/** Bonus perdus (gardés + lâchés) sur bonus pris, pour chaque camp. */
export interface PickupLosses {
  us: { lost: number; taken: number }
  them: { lost: number; taken: number }
}

export interface PickupSheets {
  owners: PickupOwner[]
  sections: PickupSection[]
  /** Par fiche : la ressource où elle pèse le plus dans les prises de notre camp (null : aucune). */
  dominant: (string | null)[]
  losses: PickupLosses | null
}

/**
 * buildPickupSheets — une fiche par joueur de l'escouade (l'ordre de `players`), puis le reste
 * du camp. Une section par ressource du bilan ; dedans, les objets que notre camp a pris, rangés
 * par prises décroissantes puis par nom (`nameOf`) — MÊMES lignes dans le même ordre sur toutes
 * les fiches, un zéro reste une ligne.
 */
export function buildPickupSheets(
  block: SquadEmpriseBlock,
  nameOf: (o: SquadEmpriseObject) => string,
): PickupSheets {
  const players = block.players ?? []
  const owners: PickupOwner[] = [...players.map((p) => ({ xuid: p.xuid, gamertag: p.gamertag })), { xuid: null, gamertag: '' }]
  const slot = (xuid: string | undefined) => {
    if (!xuid) return owners.length - 1
    const i = players.findIndex((p) => p.xuid === xuid)
    return i >= 0 ? i : owners.length - 1
  }
  const sections = buildControlRows(block).map(({ resource }) => {
    const lines = (block.objects ?? [])
      .filter((o) => o.resource === resource && o.taken.us > 0)
      .map((object) => {
        const zeros = () => owners.map(() => 0)
        const line: PickupLine = { object, taken: zeros(), kept: zeros(), dropped: zeros(), camp: object.taken.us }
        for (const s of object.squad ?? []) {
          const i = slot(s.xuid)
          line.taken[i] += s.taken
          line.kept[i] += s.kept ?? 0
          line.dropped[i] += s.dropped ?? 0
        }
        return line
      })
      .sort((a, b) => b.camp - a.camp || nameOf(a.object).localeCompare(nameOf(b.object)))
    const totals = owners.map((_, i) => lines.reduce((acc, l) => acc + l.taken[i], 0))
    return { resource, lines, totals, camp: totals.reduce((a, v) => a + v, 0) }
  })
  const dominant = owners.map((_, i) => dominantResource(sections, i))
  return { owners, sections, dominant, losses: bonusLosses(block) }
}

/** La ressource où la fiche pèse le plus dans notre camp (part du camp), null sans prise. */
function dominantResource(sections: PickupSection[], owner: number): string | null {
  let best: string | null = null
  let bestShare = 0
  for (const s of sections) {
    const share = s.camp > 0 ? s.totals[owner] / s.camp : 0
    if (share > bestShare) {
      bestShare = share
      best = s.resource
    }
  }
  return best
}

function bonusLosses(block: SquadEmpriseBlock): PickupLosses | null {
  const outcomes = (block.resources ?? []).find((r) => r.resource === RESOURCE_POWERUP)?.outcomes
  if (!outcomes) return null
  return {
    us: { lost: outcomes.us.kept + outcomes.us.dropped, taken: outcomes.us.taken },
    them: { lost: outcomes.them.kept + outcomes.them.dropped, taken: outcomes.them.taken },
  }
}

// ---------------------------------------------------------------------------
// Contrôle des ressources, match par match
// ---------------------------------------------------------------------------

/** Qui chez nous a pris l'objet sur ce match (fiche : xuid, null = reste du camp). */
export interface GridWho {
  xuid: string | null
  taken: number
}

export type GridCell =
  | { kind: 'value'; us: number; them: number; share: number; who: GridWho[]; padsEmptied?: number }
  /** Rien à prendre : l'objet (ou la ressource) n'était pas sur cette carte, ou personne ne l'a pris. */
  | { kind: 'none' }
  /** Film non décodé : rien à lire pour une ligne qui vient du film. */
  | { kind: 'nofilm' }
  /** Film décodé, mais notre camp inconnu (chacun pour soi, camp absent) : rien ne se partage. */
  | { kind: 'noteam' }
  /** Film décodé, mais niveaux de socle non établis : armes spéciales et râteliers ne se séparent pas. */
  | { kind: 'untiered'; tiers: string }

export interface GridRow {
  /** Objet de la soirée (lignes d'objet) ; absent pour les lignes de synthèse et de frags. */
  object?: SquadEmpriseObject
  cells: GridCell[]
}

export interface GridSection {
  resource: string
  /** Ligne de synthèse (prises de la ressource) ; null quand aucune prise n'est mesurée. */
  summary: GridRow | null
  items: GridRow[]
  /** Armes spéciales seulement : les frags obtenus avec (feuille de match, sans film requis). */
  kills: GridRow | null
}

export interface MatchGrid {
  columns: EmpriseMatchInfo[]
  sections: GridSection[]
}

function whoOf(o: SquadEmpriseObject): GridWho[] {
  return (o.squad ?? []).filter((s) => s.taken > 0).map((s) => ({ xuid: s.xuid || null, taken: s.taken }))
}

function valueCell(taken: SquadEmpriseCount | undefined, who: GridWho[], padsEmptied?: number): GridCell {
  const n = total(taken)
  if (!taken || n <= 0) return { kind: 'none' }
  return { kind: 'value', us: taken.us, them: taken.them, share: taken.us / n, who, padsEmptied }
}

/**
 * La case d'une ligne qui vient du film, avant toute valeur : sans film, camp inconnu, niveaux
 * non établis. Filmé au camp inconnu, un match n'a aucun compte camp contre camp : « camp
 * inconnu », jamais « rien à prendre ».
 */
function filmGate(m: SquadEmpriseMatch, resource: string): GridCell | null {
  if (!m.has_film) return { kind: 'nofilm' }
  if (!m.team_known) return { kind: 'noteam' }
  if (resource !== RESOURCE_POWERUP && m.tiers !== TIERS_MEASURED) return { kind: 'untiered', tiers: m.tiers ?? '' }
  return null
}

/**
 * buildMatchGrid — une colonne par match (ordre de la soirée) ; par ressource, la ligne de
 * synthèse puis une ligne par objet de la soirée (l'ordre du bloc : prises de notre camp
 * décroissantes) ; sous les armes spéciales, la ligne des frags obtenus avec.
 */
export function buildMatchGrid(block: SquadEmpriseBlock, history: SquadMatchHistoryRow[]): MatchGrid {
  const matches = block.matches ?? []
  const byId = historyIndex(history)
  const objects = block.objects ?? []
  const hasKills = matches.some((m) => m.power_weapon_kills != null)
  const present = orderedResources([
    ...objects.map((o) => o.resource),
    ...matches.flatMap((m) => (m.resources ?? []).map((r) => r.resource)),
    ...(hasKills ? [RESOURCE_POWER_WEAPON] : []),
  ])
  const sections = present.map((resource): GridSection => {
    const ofResource = objects.filter((o) => o.resource === resource)
    const measured = ofResource.length > 0 || matches.some((m) => matchResource(m, resource) != null)
    const summary: GridRow | null = measured
      ? { cells: matches.map((m) => filmGate(m, resource) ?? valueCell(matchResource(m, resource)?.taken, [])) }
      : null
    const items = ofResource.map((object): GridRow => ({
      object,
      cells: matches.map((m) => {
        const gate = filmGate(m, resource)
        if (gate) return gate
        const o = matchResource(m, resource)?.objects?.find((x) => x.key === object.key)
        return o ? valueCell(o.taken, whoOf(o), o.pads_emptied) : { kind: 'none' }
      }),
    }))
    const kills: GridRow | null =
      resource === RESOURCE_POWER_WEAPON && hasKills
        ? { cells: matches.map((m) => valueCell(m.power_weapon_kills, [])) }
        : null
    return { resource, summary, items, kills }
  })
  return { columns: matches.map((m) => matchInfo(m, byId)), sections }
}
