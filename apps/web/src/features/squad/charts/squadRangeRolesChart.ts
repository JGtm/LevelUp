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
   * Écrit le gamertag AU BOUT de sa courbe de tendance, en plus de la légende (E1 : à
   * quatre joueurs et vingt matchs, faire l'aller-retour vers la légende coûte la lecture).
   * Réserve la marge droite qui va avec. Défaut : non.
   */
  etiquetteBout?: boolean
  /**
   * Opacité des points du nuage. Défaut 1 (lot R). Une grandeur dont la LECTURE tient dans
   * la tendance (E1) baisse les points pour que les courbes passent devant.
   */
  opacitePoints?: number
}

/** Marge droite du `grid` : l'étiquette de bout de courbe a besoin de place. */
const MARGE_DROITE = 16
const MARGE_DROITE_ETIQUETTE = 64

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

function bornesY(series: SeriePortee[]): { min: number; max: number } {
  const ecarts = series.flatMap((s) => s.points).map((p) => p.ecartM)
  const amplitude = Math.max(MIN_AMPLITUDE_M, ...ecarts.map((e) => Math.abs(e)))
  const borne = Math.ceil(amplitude) + 1
  return { min: -borne, max: borne }
}

/** Les trois bandes de rôle, posées sur l'échelle relative. */
function bandesRoles(
  seuils: SeuilsRoles | null,
  bornes: { min: number; max: number },
  libelles: RangeRolesChartOpts['libelles'],
  couleurBande: string,
): Record<string, unknown> | undefined {
  if (!seuils) return undefined
  const bande = (bas: number, haut: number, nom: string) => [
    {
      yAxis: bas,
      name: nom,
      itemStyle: { color: couleurBande, opacity: 0.07 },
      label: { position: 'insideEndTop' },
    },
    { yAxis: haut },
  ]
  return {
    silent: true,
    label: { show: true, fontSize: 10, position: 'insideEndTop' },
    data: [
      bande(bornes.min, seuils.bas, libelles.bandes.front),
      bande(seuils.bas, seuils.haut, libelles.bandes.polyvalent),
      bande(seuils.haut, bornes.max, libelles.bandes.sniper),
    ],
  }
}

/** La série ligne d'un joueur : la moyenne glissante de ses points PLEINS. */
function serieTendance(
  serie: SeriePortee,
  couleur: string,
  nom: string,
  etiquetteBout = false,
): Record<string, unknown> {
  const moyennes = moyenneGlissante(serie.points)
  const data = serie.points.map((p, i) => [p.ordre, moyennes[i]])
  const s: Record<string, unknown> = {
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
  if (etiquetteBout) {
    s.endLabel = {
      show: true,
      formatter: nom,
      color: couleur,
      fontSize: 10,
      fontWeight: 500,
      distance: 6,
    }
  }
  return s
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
  const bornes = bornesY(series)
  const n = opts.categories.length
  const couleurDefaut = resolveToken('info')
  const opacitePoints = opts.opacitePoints ?? 1

  const nuage = series.map((serie, idx) => {
    const couleur = opts.couleurs[serie.gamertag] ?? couleurDefaut
    const data: EchartPointDatum[] = serie.points.map((p) => ({
      value: [p.ordre, p.ecartM],
      symbolSize: taillePoint(p.mesures, opts.mesuresMin, opts.mesuresMax),
      // POINT CREUX sous le plancher : contour pointillé, aucun remplissage. Il reste
      // visible (un essai de style n'est pas une erreur de mesure) mais ne se confond
      // jamais avec une médiane tenue.
      itemStyle: p.plein
        ? { color: couleur, borderColor: tc.card, borderWidth: 2, opacity: opacitePoints }
        : {
            color: 'transparent',
            borderColor: couleur,
            borderWidth: 2,
            borderType: 'dashed',
            opacity: opacitePoints,
          },
      raw: p,
    }))
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
      const bandes = bandesRoles(opts.seuils, bornes, opts.libelles, tc.axisLine)
      if (bandes) s.markArea = bandes
    }
    return s
  })

  const tendances = series.map((serie) =>
    serieTendance(
      serie,
      opts.couleurs[serie.gamertag] ?? couleurDefaut,
      serie.gamertag,
      opts.etiquetteBout ?? false,
    ),
  )

  return {
    backgroundColor: CHART_BG,
    grid: {
      top: 24,
      bottom: 78,
      left: 56,
      right: opts.etiquetteBout ? MARGE_DROITE_ETIQUETTE : MARGE_DROITE,
    },
    tooltip: { ...getTooltipBase(tc), trigger: 'item', formatter: formatTooltip(opts) },
    legend: {
      ...getLegendBase(tc),
      data: legendEntries(
        series.map((s) => ({
          name: s.gamertag,
          color: opts.couleurs[s.gamertag] ?? couleurDefaut,
        })),
      ),
    },
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

