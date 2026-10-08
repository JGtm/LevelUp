/**
 * impactHistory.logic.ts — la mise en forme PURE du graphe « Points d'impact par soirée et par
 * rôle » (Escouade › Contributions) : soirées et libellés d'axe, piles de segments par barre,
 * nets, extrêmes écrits, infobulles, légende et résumé accessible.
 *
 * Aucun point n'est calculé ici : comptes, points par rôle et nets arrivent du serveur
 * (`squad_impact_history`, barème côté Go). Ce module ne fait qu'ordonner, regrouper les
 * segments voisins de même nuance et formater.
 */
import type { SemanticToken } from '@/lib/accessibility'
import type { SquadImpactEvening, SquadImpactHistory } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { eveningDate } from '../objectif/objectif.logic'
import type { ImpactHistoryText } from './impactHistoryStrings'

/** La nuance de chaque rôle : rampe des gains (1 = contre l'axe) puis des pertes. */
const ROLE_TOKEN: Readonly<Record<string, SemanticToken>> = {
  clutch_finisher: 'impact-gain-1',
  first_blood: 'impact-gain-2',
  silent_hero: 'impact-gain-3',
  top_killer: 'impact-gain-4',
  last_casualty: 'impact-loss-1',
  false_brother: 'impact-loss-2',
  first_group_death: 'impact-loss-3',
  last_group_kill: 'impact-loss-3',
  kamikaze: 'impact-loss-3',
  thief: 'impact-loss-3',
}

/** La nuance d'un rôle ; un rôle inconnu prend le bout de la rampe de son signe. */
export function roleToken(role: string, points: number): SemanticToken {
  return ROLE_TOKEN[role] ?? (points >= 0 ? 'impact-gain-4' : 'impact-loss-3')
}

/** Un segment d'une pile : de `from` à `to` points depuis zéro, d'une nuance. */
export interface ImpactSegment {
  token: SemanticToken
  from: number
  to: number
}

/** Une ligne de l'infobulle : un rôle, son nombre, ses points. */
export interface ImpactTipRole {
  token: SemanticToken
  label: string
  count: number
  points: string
}

/** Une barre : un joueur sur une soirée. */
export interface ImpactBar {
  evening: number
  player: number
  gains: ImpactSegment[]
  losses: ImpactSegment[]
  gainTotal: number
  lossTotal: number
  net: number
  tip: { player: string; evening: string; matches: string; roles: ImpactTipRole[]; net: string }
}

/** Une entrée de légende des rôles. */
export interface ImpactLegendRole {
  token: SemanticToken
  label: string
}

export interface ImpactHistoryView {
  players: string[]
  /** Libellé d'axe par soirée (date, l'heure en seconde ligne si deux soirées partagent la date). */
  axisLabels: string[]
  /** Deux soirées partagent une date : l'axe a besoin de deux lignes. */
  twoLineAxis: boolean
  bars: ImpactBar[]
  /** nets[joueur][soirée]. */
  nets: number[][]
  /** Les nets écrits sur le graphe (le plus haut et le plus bas), clés `joueur:soirée`. */
  extremes: Set<string>
  legend: { gains: ImpactLegendRole[]; losses: ImpactLegendRole[] }
  info: string
  aria: string
}

/** Clé d'un point de courbe (joueur, soirée). */
export const pointKey = (player: number, evening: number) => `${player}:${evening}`

/** Les segments d'une pile depuis zéro ; des rôles voisins de même nuance forment un bloc. */
function stack(roles: { role: string; points: number }[]): ImpactSegment[] {
  const out: ImpactSegment[] = []
  let acc = 0
  for (const r of roles) {
    const token = roleToken(r.role, r.points)
    const prev = out[out.length - 1]
    if (prev && prev.token === token) prev.to += r.points
    else out.push({ token, from: acc, to: acc + r.points })
    acc += r.points
  }
  return out
}

/** Le nom d'une soirée : « Soirée du 07/09 », l'heure ajoutée si la date est partagée. */
function eveningNames(evenings: SquadImpactEvening[], locale: Locale, t: ImpactHistoryText) {
  const dates = evenings.map((e) => eveningDate(e.start_time, locale))
  const shared = dates.map((d) => dates.filter((x) => x === d).length > 1)
  const hours = evenings.map((e) => t.hourOf(new Date(e.start_time)))
  return {
    dates,
    twoLineAxis: shared.some(Boolean),
    axisLabels: dates.map((d, i) => (shared[i] ? `${d}\n${hours[i]}` : d)),
    names: dates.map((d, i) => (shared[i] ? t.eveningAt(d, hours[i]) : t.eveningOf(d))),
  }
}

/** Le barème en une phrase : rôles de même valeur réunis, gains puis pertes. */
export function scaleSentence(scale: { role: string; points: number }[], t: ImpactHistoryText): string {
  const groups: { points: number; labels: string[] }[] = []
  for (const s of scale) {
    const label = t.roles[s.role] ?? s.role
    const last = groups[groups.length - 1]
    if (last && last.points === s.points) last.labels.push(label)
    else groups.push({ points: s.points, labels: [label] })
  }
  const part = (gs: typeof groups) => gs.map((g) => `${t.scaleJoin(g.labels)} ${t.signed(g.points)}`).join(', ')
  return [part(groups.filter((g) => g.points > 0)), part(groups.filter((g) => g.points < 0))]
    .filter(Boolean)
    .join(' ; ')
}

/** La légende des rôles : une entrée par rôle, sauf les pertes qui partagent une nuance. */
function legendOf(scale: { role: string; points: number }[], t: ImpactHistoryText) {
  const entry = (s: { role: string; points: number }) => ({
    token: roleToken(s.role, s.points),
    label: `${t.roles[s.role] ?? s.role} ${t.signed(s.points)}`,
  })
  const gains = scale.filter((s) => s.points > 0).map(entry)
  const lossScale = scale.filter((s) => s.points < 0)
  const byToken = new Map<SemanticToken, { role: string; points: number }[]>()
  for (const s of lossScale) {
    const token = roleToken(s.role, s.points)
    byToken.set(token, [...(byToken.get(token) ?? []), s])
  }
  const losses = [...byToken.entries()].map(([token, group]) =>
    group.length === 1
      ? entry(group[0])
      : { token, label: t.legendGrouped(t.signed(group[0].points), group.map((g) => t.roles[g.role] ?? g.role)) },
  )
  return { gains, losses }
}

/** Les deux nets écrits : le plus haut et le plus bas de tout le graphe. */
function extremesOf(nets: number[][]): Set<string> {
  let hi: [number, number, number] | null = null
  let lo: [number, number, number] | null = null
  nets.forEach((row, p) =>
    row.forEach((v, e) => {
      if (!hi || v > hi[0]) hi = [v, p, e]
      if (!lo || v < lo[0]) lo = [v, p, e]
    }),
  )
  const out = new Set<string>()
  for (const x of [hi, lo] as ([number, number, number] | null)[]) if (x) out.add(pointKey(x[1], x[2]))
  return out
}

/** La vue du graphe ; null quand il n'y a ni soirée ni joueur. */
export function buildImpactHistoryView(
  h: SquadImpactHistory,
  locale: Locale,
  t: ImpactHistoryText,
): ImpactHistoryView | null {
  const evenings = h.evenings ?? []
  const players = h.players ?? []
  if (evenings.length === 0 || players.length === 0) return null
  const scale = h.scale ?? []
  const { dates, twoLineAxis, axisLabels, names } = eveningNames(evenings, locale, t)
  const nets = players.map(() => evenings.map(() => 0))
  const bars: ImpactBar[] = []
  evenings.forEach((ev, e) => {
    players.forEach((gt, p) => {
      const entry = (ev.players ?? []).find((x) => x.player === gt)
      const roles = entry?.roles ?? []
      const net = entry?.points ?? 0
      nets[p][e] = net
      const gains = roles.filter((r) => r.points > 0)
      const losses = roles.filter((r) => r.points < 0)
      bars.push({
        evening: e,
        player: p,
        gains: stack(gains),
        losses: stack(losses),
        gainTotal: gains.reduce((a, r) => a + r.points, 0),
        lossTotal: losses.reduce((a, r) => a + r.points, 0),
        net,
        tip: {
          player: gt,
          evening: names[e],
          matches: t.matchesWins(ev.matches, ev.wins),
          roles: roles.map((r) => ({
            token: roleToken(r.role, r.points),
            label: t.roles[r.role] ?? r.role,
            count: r.count,
            points: t.signed(r.points),
          })),
          net: t.signed(net),
        },
      })
    })
  })
  const perPlayer = players.map((gt, p) => t.ariaPlayer(gt, nets[p].map(t.signed).join(', '))).join(' ; ')
  return {
    players,
    axisLabels,
    twoLineAxis,
    bars,
    nets,
    extremes: extremesOf(nets),
    legend: legendOf(scale, t),
    info: t.info(t.scaleJoin(players), scaleSentence(scale, t)),
    aria: t.ariaSummary(dates[0], dates[dates.length - 1], perPlayer),
  }
}

/** Sur téléphone, une date sur deux, comptée depuis la dernière soirée (toujours écrite). */
export function showAxisLabel(index: number, count: number, narrow: boolean): boolean {
  return !narrow || (count - 1 - index) % 2 === 0
}
