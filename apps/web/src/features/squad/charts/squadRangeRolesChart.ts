/**
 * squadRangeRolesChart — l'option ECharts du nuage « Rôles de portée ».
 *
 * PAS LE WRAPPER `<ScatterChart>` GÉNÉRIQUE, pour deux raisons soudées au contrat : il
 * n'expose qu'un `symbolSize` UNIFORME par série (ici la taille dit les frags mesurés, et
 * un point sous le plancher se dessine CREUX) et il ne porte ni `markArea` ni `markLine`
 * (les trois bandes de rôle et la ligne du lobby). Même pattern que
 * `SquadIsolementNuageCard` : composer `<ChartCard>` avec un `buildOption` custom.
 *
 * L'axe X est CATÉGORIEL — un match par position, du plus ancien au plus récent — et ses
 * étiquettes « #N · carte » ont le gabarit des autres graphes par match de la page
 * (`squadEfficiencyChart` : même séparateur, même troncature, même fontSize, même
 * intervalle).
 */
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
  legendEntries,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'

import {
  moyenneGlissante,
  taillePoint,
  type PointPortee,
  type RoleDePortee,
  type SeriePortee,
  type SeuilsRoles,
} from '../squadRangeRoles.logic'

export interface RangeRolesChartOpts {
  categories: string[]
  seuils: SeuilsRoles | null
  /** Encre par gamertag — la même source que le reste de la page. */
  couleurs: Record<string, string>
  /** Extrêmes RÉELS des frags mesurés : l'échelle de taille des points. */
  mesuresMin: number
  mesuresMax: number
  libelles: {
    xAxis: string
    yAxis: string
    lobbyLine: string
    bandes: Record<RoleDePortee, string>
    tooltipMedian: (m: string) => string
    tooltipDelta: (delta: string) => string
    tooltipMeasured: (n: number) => string
  }
  /** Formateur des mètres (locale de l'app). */
  fmtM: (v: number) => string
  /**
   * Légende des SÉRIES (une entrée par joueur). `false` sur un nuage à une seule série —
   * Timeseries, où le joueur consulté est seul : nommer une série unique n'apprend rien, et
   * la légende des encodages (point creux, tendance) reste rendue par la carte.
   */
  legende?: boolean
  /** Encre des séries sans entrée dans `couleurs` (défaut : jeton `info`). */
  couleurDefaut?: string

  // ── Ouverture au scope « un seul joueur » (Sessions, lot W / D23-4) ─────────────
  // Le nuage de période des Sessions est CE nuage à une série : même axe, mêmes bandes,
  // même fenêtre glissante, mêmes points creux. Les quatre options ci-dessous portent la
  // seule chose qui lui est propre — la session mise en surbrillance dans sa période.

  /**
   * Bornes Y PLANCHER imposées (échelle partagée A/B du drawer de comparaison). Le nuage
   * les ÉLARGIT s'il déborde : une borne imposée ne doit jamais rogner un point mesuré.
   */
  yDomain?: [number, number]
  /**
   * Encre d'un point quand elle ne dit PAS l'identité du joueur mais son appartenance
   * (Sessions : période en gris, matchs de la session en couleur pleine). Absente :
   * l'encre du joueur, comme sur l'Escouade.
   */
  encrePoint?: (p: PointPortee) => string
  /**
   * Fenêtre de matchs mise en surbrillance sur l'axe X (indices INCLUSIFS dans
   * `categories`) : le fond qui dit « cette session » dans la période.
   */
  surbrillance?: { debut: number; fin: number; label: string; couleur: string }
  /** Masque la légende du graphe — à une seule série, son nom n'apprend rien. */
  masquerLegende?: boolean
}

/** La donnée d'un point du nuage : sa position, et le point BRUT pour l'infobulle. */
interface EchartPointDatum {
  value: [number, number]
  symbolSize: number
  itemStyle: Record<string, unknown>
  raw: PointPortee
}

/** Bornes de l'axe Y : l'amplitude observée, arrondie au mètre et jamais plus serrée
 *  que ±`MIN_AMPLITUDE_M` — une escouade homogène ne doit pas voir son bruit grossi. */
const MIN_AMPLITUDE_M = 5

function bornesY(
  series: SeriePortee[],
  plancher?: [number, number],
): { min: number; max: number } {
  const ecarts = series.flatMap((s) => s.points).map((p) => p.ecartM)
  const impose = plancher ? Math.max(Math.abs(plancher[0]), Math.abs(plancher[1])) : 0
  const amplitude = Math.max(MIN_AMPLITUDE_M, impose, ...ecarts.map((e) => Math.abs(e)))
  const borne = Math.ceil(amplitude) + 1
  return { min: -borne, max: borne }
}

/**
 * Les zones de fond : les trois bandes de rôle, plus — quand l'appelant en pose une — la
 * fenêtre de matchs en surbrillance. Un `markArea` par série et pas deux : ECharts n'en
 * tient qu'un, les deux familles partagent donc ses données.
 */
function zonesDeFond(
  seuils: SeuilsRoles | null,
  bornes: { min: number; max: number },
  opts: RangeRolesChartOpts,
  couleurBande: string,
): Record<string, unknown> | undefined {
  const libelles = opts.libelles
  const data: unknown[] = []
  if (seuils) {
    const bande = (bas: number, haut: number, nom: string) => [
      {
        yAxis: bas,
        name: nom,
        itemStyle: { color: couleurBande, opacity: 0.07 },
        label: { position: 'insideEndTop' },
      },
      { yAxis: haut },
    ]
    data.push(
      bande(bornes.min, seuils.bas, libelles.bandes.front),
      bande(seuils.bas, seuils.haut, libelles.bandes.polyvalent),
      bande(seuils.haut, bornes.max, libelles.bandes.sniper),
    )
  }
  const sur = opts.surbrillance
  if (sur) {
    // Les demi-pas débordent la première et la dernière colonne de la fenêtre : une
    // session d'UN match doit quand même dessiner une bande, pas un trait de largeur nulle.
    data.push([
      {
        xAxis: sur.debut - 0.5,
        name: sur.label,
        itemStyle: { color: sur.couleur, opacity: 0.14 },
        label: { position: 'insideTop' },
      },
      { xAxis: sur.fin + 0.5 },
    ])
  }
  if (data.length === 0) return undefined
  return { silent: true, label: { show: true, fontSize: 10, position: 'insideEndTop' }, data }
}

/** La série ligne d'un joueur : la moyenne glissante de ses points PLEINS. */
function serieTendance(
  serie: SeriePortee,
  couleur: string,
  nom: string,
): Record<string, unknown> {
  const moyennes = moyenneGlissante(serie.points)
  const data = serie.points.map((p, i) => [p.ordre, moyennes[i]])
  return {
    type: 'line',
    name: nom,
    data,
    showSymbol: false,
    connectNulls: true,
    silent: true,
    lineStyle: { color: couleur, width: 1.5, opacity: 0.7 },
    itemStyle: { color: couleur },
    z: 2,
  }
}

/** L'infobulle d'un point : le joueur, le match, la médiane, l'écart, les frags mesurés. */
function formatTooltip(opts: RangeRolesChartOpts) {
  const l = opts.libelles
  return (params: unknown): string => {
    const data = (params as { data?: EchartPointDatum }).data
    if (!data?.raw) return ''
    const p = data.raw
    return [
      `<b>${escapeHtml(p.gamertag)}</b>`,
      escapeHtml(opts.categories[p.ordre] ?? `#${p.ordre + 1}`),
      escapeHtml(l.tooltipMedian(opts.fmtM(p.medianeM))),
      escapeHtml(l.tooltipDelta(opts.fmtM(p.ecartM))),
      escapeHtml(l.tooltipMeasured(p.mesures)),
    ].join('<br/>')
  }
}

export function buildSquadRangeRolesOption(
  series: SeriePortee[],
  opts: RangeRolesChartOpts,
): EChartsCoreOption {
  if (series.length === 0) return { backgroundColor: CHART_BG }
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const bornes = bornesY(series, opts.yDomain)
  const n = opts.categories.length
  const couleurDefaut = opts.couleurDefaut ?? resolveToken('info')

  const nuage = series.map((serie, idx) => {
    const couleur = opts.couleurs[serie.gamertag] ?? couleurDefaut
    const data: EchartPointDatum[] = serie.points.map((p) => {
      const encre = opts.encrePoint ? opts.encrePoint(p) : couleur
      return {
        value: [p.ordre, p.ecartM],
        symbolSize: taillePoint(p.mesures, opts.mesuresMin, opts.mesuresMax),
        // POINT CREUX sous le plancher : contour pointillé, aucun remplissage. Il reste
        // visible (un essai de style n'est pas une erreur de mesure) mais ne se confond
        // jamais avec une médiane tenue.
        itemStyle: p.plein
          ? { color: encre, borderColor: tc.card, borderWidth: 2 }
          : { color: 'transparent', borderColor: encre, borderWidth: 2, borderType: 'dashed' },
        raw: p,
      }
    })
    const s: Record<string, unknown> = { type: 'scatter', name: serie.gamertag, data, z: 3 }
    // Les overlays du graphe entier (ligne du lobby, bandes de rôle) sont posés UNE SEULE
    // FOIS, sur la première série : ce ne sont pas des marques d'une série.
    if (idx === 0) {
      s.markLine = {
        silent: true,
        symbol: 'none',
        lineStyle: { type: 'dashed', color: tc.axisLabel },
        label: {
          show: true,
          formatter: opts.libelles.lobbyLine,
          color: tc.axisLabel,
          fontSize: 10,
          position: 'insideStartTop',
        },
        data: [{ yAxis: 0 }],
      }
      const zones = zonesDeFond(opts.seuils, bornes, opts, tc.axisLine)
      if (zones) s.markArea = zones
    }
    return s
  })

  const tendances = series.map((serie) =>
    serieTendance(serie, opts.couleurs[serie.gamertag] ?? couleurDefaut, serie.gamertag),
  )

  const sansLegende = opts.masquerLegende || opts.legende === false
  const legende = sansLegende
    ? undefined
    : {
        ...getLegendBase(tc),
        data: legendEntries(
          series.map((s) => ({
            name: s.gamertag,
            color: opts.couleurs[s.gamertag] ?? couleurDefaut,
          })),
        ),
      }

  return {
    backgroundColor: CHART_BG,
    // Sans légende de séries (lots Z1 et W), le bas n'a plus à la loger : la grille descend d'autant.
    grid: { top: 24, bottom: sansLegende ? 52 : 78, left: 56, right: 16 },
    tooltip: { ...getTooltipBase(tc), trigger: 'item', formatter: formatTooltip(opts) },
    // Deux contrats cohabitent (Z1 : `legende: false` -> `{ show: false }` ; W : `masquerLegende` -> absente).
    ...(legende ? { legend: legende } : opts.legende === false ? { legend: { show: false } } : {}),
    xAxis: {
      ...axis,
      type: 'category',
      data: opts.categories,
      name: opts.libelles.xAxis,
      nameLocation: 'middle',
      nameGap: 32,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: { ...axis.axisLabel, fontSize: 9, interval: n > 12 ? Math.floor(n / 8) : 0 },
    },
    yAxis: {
      ...axis,
      type: 'value',
      min: bornes.min,
      max: bornes.max,
      name: opts.libelles.yAxis,
      nameLocation: 'middle',
      nameGap: 40,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: { ...axis.axisLabel, formatter: '{value} m' },
    },
    series: [...nuage, ...tendances],
  }
}

