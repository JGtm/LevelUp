/**
 * objectif.logic.ts — LES MODÈLES PURS des quatre cartes d'objectif de l'onglet Contributions
 * (lot L3 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, maquette C3EW).
 *
 * Trois cartes lisent le bloc `formes_retenues` (une ligne par joueur et par match, les deux
 * camps, sur le périmètre D2) ; la quatrième lit `squad_objective_history`, calculé côté Go
 * (les soirées précédentes ne sont pas dans le périmètre de la page).
 *
 * LES RÔLES (D7) : la partition prendre / défendre / tenir que le serveur publie sur chaque
 * colonne. Les colonnes FACULTATIVES (prises nettes, lues du film) restent hors des rôles —
 * mêmes règles que `aggregateRole` et que le calcul Go de l'historique. Elles restent, en
 * revanche, des lignes du rapport de force et des fiches : c'est une action que l'on montre.
 *
 * UNE PART PAR MATCH, ET CHAQUE MATCH PÈSE PAREIL (D7) : quatre Bases à 200 actions n'écrasent
 * pas trois Drapeau à 60. Un rôle que le lobby n'a pas touché sur un match (0 sur 0) ne pèse
 * pas dans la moyenne : ce n'est pas une part nulle, c'est une absence.
 *
 * Pur : aucun React, aucune couleur, aucune chaîne de langue.
 */
import { asDominance, type DominanceValue, type OutcomeValue } from '@/components/charts/outcomeSequence'
import type {
  SquadFormesBlock,
  SquadFormesMatch,
  SquadFormesObjectiveColumn,
  SquadMatchHistoryRow,
  SquadObjectiveEvening,
  SquadObjectiveHistory,
} from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { outcomeCodeToValue } from '@/lib/outcome'

import {
  OBJECTIVE_ROLES,
  aggregateColumns,
  columnsOfFamily,
  matchesOfFamily,
  objectiveCell,
  objectiveFamilies,
  objectiveMatches,
  type ObjectiveRole,
} from '../formes/model/objectives'

/** Le minimum de matchs à objectif pour tracer une courbe (D6, `min_objective_matches`). */
export const OBJECTIVE_MIN_MATCHES = 3

// ---------------------------------------------------------------------------
// Rapport de force par famille de mode
// ---------------------------------------------------------------------------

/** Une action : ce que notre camp a fait, ce que l'adversaire a fait, notre part (0..1). */
export interface BalanceLine {
  key: string
  duration: boolean
  us: number
  them: number
  share: number
}

export interface BalanceFamily {
  family: string
  matches: number
  roles: { role: ObjectiveRole; lines: BalanceLine[] }[]
}

/**
 * buildObjectiveBalance — une famille par cadre, les actions rangées par rôle dans l'ordre du
 * serveur. L'adversaire = le lobby moins notre camp. Une colonne que personne n'a touchée sur
 * les matchs qui la mesurent n'a pas de ligne.
 */
export function buildObjectiveBalance(block: SquadFormesBlock): BalanceFamily[] {
  const main = block.main_xuid ?? ''
  return objectiveFamilies(block).map((family) => {
    const matches = matchesOfFamily(block, family)
    const roles = OBJECTIVE_ROLES.map((role) => ({
      role,
      lines: columnsOfFamily(block, family)
        .filter((c) => c.role === role)
        .flatMap((c): BalanceLine[] => {
          const agg = aggregateColumns(matches, main, [c])
          if (agg.lobby <= 0) return []
          return [{ key: c.key, duration: !!c.duration, us: agg.team, them: agg.lobby - agg.team, share: agg.team / agg.lobby }]
        }),
    })).filter((r) => r.lines.length > 0)
    return { family, matches: matches.length, roles }
  })
}

// ---------------------------------------------------------------------------
// Rapport de force au fil de la session
// ---------------------------------------------------------------------------

/** Un rôle sur un match : notre camp, le lobby, notre part (null = rôle non joué) et le cumul. */
export interface FilRole {
  us: number
  lobby: number
  share: number | null
  cumulative: number | null
}

export interface FilMatch {
  matchId: string
  startTime: string
  map: string
  family: string
  /** Résultat joint depuis `match_history` (L3.2) — absent quand la ligne manque. */
  outcome: OutcomeValue | null
  score: string | null
  dominance: DominanceValue | undefined
  roles: Record<ObjectiveRole, FilRole>
}

/** Colonnes d'un rôle sur un match, sans les grandeurs facultatives (D7). */
function roleColumns(match: SquadFormesMatch, role: ObjectiveRole): SquadFormesObjectiveColumn[] {
  return (match.objective?.columns ?? []).filter((c) => c.role === role && !c.optional)
}

/** Notre camp et le lobby d'un match sur un rôle. Camp inconnu → null. */
function matchRole(match: SquadFormesMatch, role: ObjectiveRole): { us: number; lobby: number } | null {
  if (match.player_team == null) return null
  const cols = roleColumns(match, role)
  let us = 0
  let lobby = 0
  for (const p of match.objective?.players ?? []) {
    const v = cols.reduce((a, c) => a + (p.values?.[c.key] ?? 0), 0)
    lobby += v
    if (p.team_id === match.player_team) us += v
  }
  return { us, lobby }
}

/**
 * buildSessionFil — les matchs à objectif du périmètre, dans l'ordre chronologique, avec la part
 * de chaque rôle et son cumul (moyenne des parts depuis le premier match, D7). Le résultat, le
 * score et la dominance viennent de l'historique de la page, joint par `match_id` : un match
 * sans ligne d'historique garde sa case, sans résultat.
 */
export function buildSessionFil(block: SquadFormesBlock, history: SquadMatchHistoryRow[]): FilMatch[] {
  const byId = new Map(history.map((h) => [h.match_id, h]))
  const matches = [...objectiveMatches(block)].sort((a, b) =>
    (a.start_time ?? '').localeCompare(b.start_time ?? ''),
  )
  const sums: Record<ObjectiveRole, { sum: number; n: number }> = {
    take: { sum: 0, n: 0 },
    defend: { sum: 0, n: 0 },
    hold: { sum: 0, n: 0 },
  }
  return matches.map((m) => {
    const h = byId.get(m.match_id)
    const roles = {} as Record<ObjectiveRole, FilRole>
    for (const role of OBJECTIVE_ROLES) {
      const r = matchRole(m, role)
      const share = r != null && r.lobby > 0 ? r.us / r.lobby : null
      if (share != null) {
        sums[role].sum += share
        sums[role].n += 1
      }
      roles[role] = {
        us: r?.us ?? 0,
        lobby: r?.lobby ?? 0,
        share,
        cumulative: sums[role].n > 0 ? sums[role].sum / sums[role].n : null,
      }
    }
    return {
      matchId: m.match_id,
      startTime: m.start_time ?? '',
      map: h?.map_ui || m.map_label || '',
      family: m.objective?.family ?? '',
      outcome: h ? outcomeCodeToValue(h.outcome) : null,
      score: h?.score_label || null,
      dominance: asDominance(h?.dominance_flag),
      roles,
    }
  })
}

// ---------------------------------------------------------------------------
// Répartition de l'objectif dans l'escouade
// ---------------------------------------------------------------------------

/** Une fiche : un joueur de l'escouade, ou le reste du camp (`xuid` null). */
export interface SheetOwner {
  xuid: string | null
}

export interface SheetLine {
  key: string
  duration: boolean
  /** Une valeur par fiche, dans l'ordre des fiches. */
  values: number[]
  /** Le maximum de la ligne sur toutes les fiches : l'échelle de ses barres. */
  max: number
}

export interface ObjectiveSheets {
  owners: SheetOwner[]
  families: { family: string; lines: SheetLine[] }[]
  /** Par fiche : ses totaux de rôle (colonnes facultatives exclues), dans l'ordre OBJECTIVE_ROLES. */
  roleTotals: number[][]
  /** Par fiche : le rôle où elle pèse le plus dans notre camp (null si elle n'a rien fait). */
  dominant: (ObjectiveRole | null)[]
}

/** La fiche d'un joueur de notre camp sur un match : la sienne, ou « reste du camp ». */
function sheetIndex(xuid: string, squadIndex: Map<string, number>, restIndex: number): number {
  return squadIndex.get(xuid) ?? restIndex
}

/**
 * buildObjectiveSheets — une fiche par joueur de l'escouade, plus le reste du camp. Mêmes
 * lignes (les colonnes de chaque famille) dans le même ordre sur toutes les fiches.
 */
export function buildObjectiveSheets(block: SquadFormesBlock): ObjectiveSheets {
  const squad = block.squad ?? []
  const owners: SheetOwner[] = [...squad.map((p) => ({ xuid: p.xuid })), { xuid: null }]
  const squadIndex = new Map(squad.map((p, i) => [p.xuid, i]))
  const rest = squad.length
  const families = objectiveFamilies(block).map((family) => {
    const matches = matchesOfFamily(block, family)
    const lines = columnsOfFamily(block, family).map((col) => {
      const values = owners.map(() => 0)
      for (const m of matches) {
        for (const p of m.objective?.players ?? []) {
          if (m.player_team == null || p.team_id !== m.player_team) continue
          const v = objectiveCell(m, p.xuid, col)
          if (v != null) values[sheetIndex(p.xuid, squadIndex, rest)] += v
        }
      }
      return { key: col.key, duration: !!col.duration, values, max: Math.max(0, ...values) }
    })
    return { family, lines }
  })
  const roleTotals = owners.map(() => OBJECTIVE_ROLES.map(() => 0))
  for (const m of objectiveMatches(block)) {
    OBJECTIVE_ROLES.forEach((role, ri) => {
      const cols = roleColumns(m, role)
      for (const p of m.objective?.players ?? []) {
        if (m.player_team == null || p.team_id !== m.player_team) continue
        const v = cols.reduce((a, c) => a + (p.values?.[c.key] ?? 0), 0)
        roleTotals[sheetIndex(p.xuid, squadIndex, rest)][ri] += v
      }
    })
  }
  const campTotals = OBJECTIVE_ROLES.map((_, ri) => roleTotals.reduce((a, t) => a + t[ri], 0))
  const dominant = roleTotals.map((totals) => dominantRole(totals, campTotals))
  return { owners, families, roleTotals, dominant }
}

/** Le rôle où la fiche pèse le plus dans notre camp (part du camp), null si aucune part. */
function dominantRole(totals: number[], campTotals: number[]): ObjectiveRole | null {
  let best: ObjectiveRole | null = null
  let bestShare = 0
  OBJECTIVE_ROLES.forEach((role, ri) => {
    const share = campTotals[ri] > 0 ? totals[ri] / campTotals[ri] : 0
    if (share > bestShare) {
      bestShare = share
      best = role
    }
  })
  return best
}

// ---------------------------------------------------------------------------
// Rapport de force, soirée après soirée
// ---------------------------------------------------------------------------

export interface EveningPoint {
  current: boolean
  startTime: string
  matches: number
  wins: number
  families: { family: string; matches: number }[]
  /** Parts en POURCENT (0..100) par rôle ; null = rôle non joué ce soir-là. */
  shares: Record<ObjectiveRole, number | null>
}

export type EveningsView =
  | { kind: 'belowMinimum'; matches: number; below: number; withObjective: number }
  | { kind: 'noHistory'; current: EveningPoint }
  | { kind: 'chart'; points: EveningPoint[]; medians: Record<ObjectiveRole, number | null> }

function toPoint(e: SquadObjectiveEvening, current: boolean): EveningPoint {
  const pct = (v: number | undefined) => (v == null ? null : v * 100)
  return {
    current,
    startTime: e.start_time,
    matches: e.objective_matches,
    wins: e.wins,
    families: e.families ?? [],
    shares: { take: pct(e.take), defend: pct(e.defend), hold: pct(e.hold) },
  }
}

/** La médiane d'une liste de nombres (null si vide). */
export function median(values: number[]): number | null {
  if (values.length === 0) return null
  const s = [...values].sort((a, b) => a - b)
  const mid = (s.length - 1) / 2
  return (s[Math.floor(mid)] + s[Math.ceil(mid)]) / 2
}

/**
 * buildEveningsView — ce que la carte montre : la note « sous le minimum » (soirée affichée
 * sous trois matchs à objectif, D6), la note « première soirée » (aucune soirée précédente),
 * ou le graphe : les soirées précédentes puis ce soir, et la médiane des précédentes par rôle.
 */
export function buildEveningsView(h: SquadObjectiveHistory): EveningsView {
  const min = h.min_objective_matches || OBJECTIVE_MIN_MATCHES
  if (h.current.objective_matches < min) {
    return {
      kind: 'belowMinimum',
      matches: h.current.objective_matches,
      below: h.evenings_below_minimum,
      withObjective: h.evenings_with_objective,
    }
  }
  const current = toPoint(h.current, true)
  const previous = (h.previous ?? []).map((e) => toPoint(e, false))
  if (previous.length === 0) return { kind: 'noHistory', current }
  const medians = {} as Record<ObjectiveRole, number | null>
  for (const role of OBJECTIVE_ROLES) {
    medians[role] = median(previous.flatMap((p) => (p.shares[role] == null ? [] : [p.shares[role] as number])))
  }
  return { kind: 'chart', points: [...previous, current], medians }
}

/** « 4 B · 3 D » — le mélange de modes d'une soirée, par abréviation de famille. */
export function familyMix(families: { family: string; matches: number }[], abbr: Record<string, string>): string {
  return families.map((f) => `${f.matches} ${abbr[f.family] ?? f.family}`).join(' · ')
}

/** « 06/04 » — le jour et le mois d'une soirée, dans l'heure locale. */
export function eveningDate(iso: string, locale: Locale): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString(locale === 'fr' ? 'fr-FR' : 'en-GB', { day: '2-digit', month: '2-digit' })
}
