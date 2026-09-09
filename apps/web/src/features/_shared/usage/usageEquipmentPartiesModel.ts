/**
 * usageEquipmentPartiesModel.ts — LES DEUX DONUTS du bloc « servi ou gâché » (décisions
 * P10/P11, PLAN_EQUIPEMENT_GACHIS_2026-09-09, étapes E5.9/E6.3).
 *
 * UN SEUL ANNEAU, DES PARTS EXCLUSIVES faisant 100 % du lobby (P10). L'emboîtement se
 * lit par CONTIGUÏTÉ : les arcs sont rangés moi → mes amis → reste de mon équipe → eux,
 * et les deux sous-totaux « mon escouade » (moi + mes amis) et « mon équipe » (+ le
 * reste de mon équipe) sont écrits SOUS la légende — jamais un second anneau.
 *
 * LES VALEURS SONT SUR LES ARCS, PAS DANS LA LÉGENDE (P11) : ce module ne pose que le
 * NOM et la couleur dans `legendRows` ; les comptes bruts vivent dans `series[].datapoints[].valueLabel`
 * (rendu par `DonutChart` via `arcLabelKind="value"`), les sous-totaux sont les SEULS
 * pourcentages du bloc.
 *
 * P14 (amendement consigné au plan, §3.3) : la part « Eux » emprunte `team-enemy` — une
 * amende délibérée à la règle locale du bloc usage qui voulait l'adversaire hachuré et
 * anonyme. Dans un donut, une part doit être une part ; l'adversaire reste agrégé et non
 * nommé, seule sa traduction visuelle change.
 *
 * COULEURS DE JOUEUR : `features/squad/colors.ts` est la source unique (même précédent
 * que `usageGrids.ts` → `usagePlayerInk`, déjà en place dans ce dossier) — un ami suivi
 * garde la même couleur sur la Synthèse et l'Escouade.
 *
 * Pur : aucun React, aucune lecture de store.
 */
import { SQUAD_MAIN_PLAYER_TOKEN, SQUAD_TEAMMATE_COLOR_TOKENS } from '@/features/squad/colors'
import type { SemanticToken } from '@/lib/accessibility'
import type { EquipmentUsageParties, SessionUsageSquadPlayer } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPointDonut } from '@/components/charts/DonutChart'

import { formatUsageCount, formatUsagePct } from './usageFormat'
import type { UsageText } from './usageI18n'

export interface UsageDonutLegendRow {
  token: SemanticToken
  label: string
}

export interface UsageDonutSubtotal {
  label: string
  /** Un pourcentage du lobby, déjà formaté (locale) — le seul type de valeur qu'un
   *  sous-total porte (les parts elles-mêmes sont des comptes, sur les arcs). */
  value: string
}

export interface UsageDonutModel {
  series: ChartSeries<ChartPointDonut>[]
  sliceColors: Record<string, SemanticToken>
  centerValue: string
  centerLabel: string
  legendRows: UsageDonutLegendRow[]
  subtotals: UsageDonutSubtotal[]
}

interface Slice {
  name: string
  token: SemanticToken
  value: number
}

/**
 * buildPartiesDonutModel — construit le donut à partir des CINQ comptes exclusifs
 * publiés par le Go (P10/P11 : aucune part n'y est calculée côté serveur).
 *
 * `null` quand :
 *   - `parties` est absent (scope sans aucun match à camp connu, cf. domain Go) ;
 *   - `lobby_total <= 0` (rien à répartir).
 * Dans les deux cas l'appelant masque le donut — jamais un anneau à zéro (P13/règle du
 * bloc : une absence de mesure ne se maquille pas en donnée).
 *
 * Les amis suivis dont le compte est nul (absent de `by_friend`, ou 0) n'ont pas de
 * part : un arc à zéro n'ajoute rien et complique la légende pour rien.
 */
export function buildPartiesDonutModel(
  parties: EquipmentUsageParties | null | undefined,
  trackedPlayers: SessionUsageSquadPlayer[],
  centerLabel: string,
  t: UsageText,
  locale: Locale,
): UsageDonutModel | null {
  if (parties == null || parties.lobby_total <= 0) return null

  const count = (v: number) => formatUsageCount(v, locale)
  const byFriendValue = new Map((parties.by_friend ?? []).map((f) => [f.xuid, f.value]))

  const slices: Slice[] = []
  if (parties.player > 0) {
    slices.push({ name: t.donutMe, token: SQUAD_MAIN_PLAYER_TOKEN, value: parties.player })
  }
  const friendSlices: Slice[] = []
  trackedPlayers.forEach((p, i) => {
    const value = byFriendValue.get(p.xuid) ?? 0
    if (value <= 0) return
    friendSlices.push({
      name: p.gamertag,
      token: SQUAD_TEAMMATE_COLOR_TOKENS[i % SQUAD_TEAMMATE_COLOR_TOKENS.length],
      value,
    })
  })
  slices.push(...friendSlices)
  if (parties.rest_of_team > 0) {
    slices.push({ name: t.segTeamRest, token: 'team-ally', value: parties.rest_of_team })
  }
  if (parties.opponents > 0) {
    slices.push({ name: t.segEnemy, token: 'team-enemy', value: parties.opponents })
  }
  if (slices.length === 0) return null

  const sliceColors: Record<string, SemanticToken> = {}
  const datapoints: ChartPointDonut[] = slices.map((s) => {
    sliceColors[s.name] = s.token
    return { name: s.name, value: s.value, valueLabel: count(s.value) }
  })
  const legendRows: UsageDonutLegendRow[] = slices.map((s) => ({ token: s.token, label: s.name }))

  const hasFriends = friendSlices.length > 0
  const squadTotal = parties.player + friendSlices.reduce((a, s) => a + s.value, 0)
  const teamTotal = squadTotal + parties.rest_of_team
  const subtotals: UsageDonutSubtotal[] = hasFriends
    ? [
        { label: t.donutSquadSubtotal, value: formatUsagePct((squadTotal / parties.lobby_total) * 100, locale) },
        { label: t.rowMyTeam, value: formatUsagePct((teamTotal / parties.lobby_total) * 100, locale) },
      ]
    : [{ label: t.rowMyTeam, value: formatUsagePct((teamTotal / parties.lobby_total) * 100, locale) }]

  return {
    series: [{ key: 'parties', datapoints }],
    sliceColors,
    centerValue: count(parties.lobby_total),
    centerLabel,
    legendRows,
    subtotals,
  }
}
