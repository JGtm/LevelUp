/**
 * _elevation — LE NUAGE DU DÉNIVELÉ : distance en abscisse, hauteur signée en ordonnée.
 *
 * Forme M1 de la maquette MAQUETTE_DENIVELE_V2_2026-09-22.html (décision D24) : « la
 * dimension de l'arme on s'en fiche un peu sur le dénivelé, l'important c'est de voir où on
 * meurt et où on frag ». La clé du nuage est donc LE CÔTÉ — mes frags, mes morts — et jamais
 * l'arme, qui ne reparaît que dans l'infobulle d'un point.
 *
 * L'AXE DU DÉNIVELÉ EST SYMÉTRIQUE, ET C'EST LA SEULE DÉCISION DE GÉOMÉTRIE DU FICHIER. Un
 * axe calé sur les extrêmes observés placerait la ligne de zéro n'importe où : sur une soirée
 * à un seul frag d'en bas, « à niveau » se lirait en haut du cadre. Le zéro est le milieu, la
 * bande à niveau l'entoure, et un point au-dessus de la ligne veut dire ce qu'il montre.
 *
 * LA BANDE À NIVEAU VAUT ±1 m, la MÊME demi-largeur que l'agrégat Go
 * (`analysis.WeaponRangeLevelBandM`, sonde du 2026-09-06 : |dz| médian de 0,4 à 0,9 m sur
 * quatre cartes). Elle n'est pas une donnée : c'est le repère qui sépare une marche d'un
 * décalage de capsule entre deux joueurs debout sur le même sol.
 *
 * LE LOBBY EST UN FOND, PAS UNE SÉRIE. Il se pose SOUS les points du joueur (`z` plus bas,
 * opacité basse) et n'entre ni dans l'infobulle ni dans le clic : 90 points gris qui
 * captureraient le survol rendraient les 25 points colorés inatteignables.
 */
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getGridBase,
  getTooltipBase,
  type EChartsThemeColors,
} from '@/components/charts/_utils'
import type { MatchElevationKill } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

/**
 * Demi-largeur de la bande « à niveau », en mètres. MIROIR de la constante Go
 * `analysis.WeaponRangeLevelBandM` — la carte et l'agrégat ne peuvent pas découper le monde
 * à deux endroits différents.
 */
export const ELEVATION_LEVEL_BAND_M = 1

/** Plancher de l'axe du dénivelé, en mètres : sous lui, la bande occuperait tout le cadre. */
export const ELEVATION_MIN_HALF_SPAN_M = 3

/** Un point du nuage, tel que le graphe le porte (identité comprise, pour le clic). */
export interface ElevationPoint {
  /** [distance (m), dénivelé signé (m)] — la géométrie, et rien d'autre. */
  value: [number, number]
  /** Instant sur l'horloge du MATCH : ce que le lien vers le rejeu transporte. */
  timeMs: number
  /** `kill` ou `death`, du point de vue du joueur consulté. */
  side: string
  weapon: string
  opponent: string
}

/** Les bornes du cadre : l'ordonnée est symétrique, l'abscisse part de zéro. */
export interface ElevationBounds {
  yMax: number
  xMax: number
}

/**
 * elevationBounds rend le cadre du nuage depuis TOUS les points affichés (lobby compris quand
 * il l'est) : un fond qui déborderait du cadre serait rogné, et le rognage se lit comme une
 * donnée absente.
 *
 * L'ordonnée est arrondie au mètre supérieur et ne descend jamais sous
 * `ELEVATION_MIN_HALF_SPAN_M` : sur un match entièrement joué à plat, un axe à ±0,4 m ferait
 * d'un décalage de capsule une montagne.
 */
export function elevationBounds(...groups: readonly ElevationPoint[][]): ElevationBounds {
  let yMax = ELEVATION_MIN_HALF_SPAN_M
  let xMax = 0
  for (const g of groups) {
    for (const p of g) {
      yMax = Math.max(yMax, Math.abs(p.value[1]))
      xMax = Math.max(xMax, p.value[0])
    }
  }
  return { yMax: Math.ceil(yMax), xMax: Math.max(Math.ceil(xMax), 1) }
}

/**
 * elevationPoints projette les lignes du contrat en points du nuage. Une ligne sans géométrie
 * lisible (valeur absente ou non finie) est ÉCARTÉE : un point à zéro serait un engagement au
 * contact et à niveau qui n'a jamais eu lieu.
 */
export function elevationPoints(
  kills: readonly MatchElevationKill[] | null | undefined,
  locale: Locale,
): ElevationPoint[] {
  const out: ElevationPoint[] = []
  for (const k of kills ?? []) {
    if (!Number.isFinite(k.distance_m) || !Number.isFinite(k.delta_z_m)) continue
    out.push({
      value: [k.distance_m, k.delta_z_m],
      timeMs: k.time_ms,
      side: k.side,
      weapon: (locale === 'fr' ? k.weapon : k.weapon_en) ?? '',
      opponent: k.opponent ?? '',
    })
  }
  return out
}

/** Ce que la section fournit au constructeur d'option — aucune couleur n'est décidée ici. */
export interface ElevationOptionInput {
  kills: ElevationPoint[]
  deaths: ElevationPoint[]
  /** Fond du lobby : vide quand le bouton « comparer au lobby » est relâché. */
  lobby: ElevationPoint[]
  /** Médiane du dénivelé du lobby, tracée SEULEMENT si le fond est affiché. */
  lobbyMedian: number | null
  tc: EChartsThemeColors
  colors: { kills: string; deaths: string; lobby: string; band: string; zero: string }
  labels: {
    kills: string
    deaths: string
    lobby: string
    xAxis: string
    yAxis: string
    lobbyMedian: (m: number) => string
  }
  /** Infobulle d'UN point, déjà formatée et déjà échappée par l'appelant si besoin. */
  tooltip: (p: ElevationPoint) => string
}

/** Clés de série stables : le clic les relit pour savoir s'il doit naviguer. */
export const ELEVATION_SERIES_KILLS = 'elevation-kills'
export const ELEVATION_SERIES_DEATHS = 'elevation-deaths'
export const ELEVATION_SERIES_LOBBY = 'elevation-lobby'

/**
 * buildElevationOption compose l'option ECharts. PURE et exportée pour être testée sans monter
 * le composant : la bande, la ligne de zéro et les bornes se vérifient sur l'objet.
 */
export function buildElevationOption(input: ElevationOptionInput): EChartsCoreOption {
  const { kills, deaths, lobby, tc, colors, labels } = input
  const bounds = elevationBounds(kills, deaths, lobby)
  const axis = getAxisBase(tc)

  const scatter = (id: string, name: string, data: ElevationPoint[], color: string, faded: boolean) => ({
    id,
    name,
    type: 'scatter' as const,
    data,
    symbolSize: faded ? 5 : 8,
    z: faded ? 2 : 5,
    silent: faded,
    itemStyle: { color, opacity: faded ? 0.3 : 0.85 },
  })

  const series: Record<string, unknown>[] = []
  if (lobby.length > 0) {
    series.push(scatter(ELEVATION_SERIES_LOBBY, labels.lobby, lobby, colors.lobby, true))
  }
  // LA BANDE ET LA LIGNE DE ZÉRO SONT PORTÉES PAR LA PREMIÈRE SÉRIE VISIBLE, jamais par une
  // série fantôme : une série vide de plus apparaîtrait dans la légende d'ECharts et dans le
  // décompte des séries des tests.
  series.push({
    ...scatter(ELEVATION_SERIES_KILLS, labels.kills, kills, colors.kills, false),
    markArea: {
      silent: true,
      itemStyle: { color: colors.band, opacity: 0.35 },
      data: [[{ yAxis: -ELEVATION_LEVEL_BAND_M }, { yAxis: ELEVATION_LEVEL_BAND_M }]],
    },
    markLine: {
      silent: true,
      symbol: 'none',
      label: { show: false },
      lineStyle: { color: colors.zero, type: 'solid', width: 1 },
      data: [{ yAxis: 0 }],
    },
  })
  series.push({
    ...scatter(ELEVATION_SERIES_DEATHS, labels.deaths, deaths, colors.deaths, false),
    ...(lobby.length > 0 && input.lobbyMedian != null
      ? {
          markLine: {
            silent: true,
            symbol: 'none',
            lineStyle: { color: colors.lobby, type: 'dashed', width: 1 },
            label: {
              formatter: labels.lobbyMedian(input.lobbyMedian),
              color: tc.axisLabel,
              fontSize: 10,
              position: 'insideEndTop',
            },
            data: [{ yAxis: input.lobbyMedian }],
          },
        }
      : {}),
  })

  return {
    backgroundColor: CHART_BG,
    grid: getGridBase({ bottom: 48, left: 52 }),
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: (params: unknown) => {
        const p = params as { data?: ElevationPoint }
        return p.data ? input.tooltip(p.data) : ''
      },
    },
    xAxis: {
      ...axis,
      type: 'value',
      min: 0,
      max: bounds.xMax,
      name: labels.xAxis,
      nameLocation: 'middle',
      nameGap: 26,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    yAxis: {
      ...axis,
      type: 'value',
      min: -bounds.yMax,
      max: bounds.yMax,
      name: labels.yAxis,
      nameLocation: 'middle',
      nameGap: 36,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    series,
  }
}

/**
 * elevationTooltipLines rend les lignes de l'infobulle d'un point, DÉJÀ ÉCHAPPÉES (le
 * gamertag et le libellé d'arme sont des données non constantes : l'infobulle d'ECharts est
 * rendue en innerHTML).
 *
 * Trois lignes au plus, dans l'ordre de ce qu'on cherche : quoi et où, avec quoi et quand,
 * puis l'invitation au clic.
 */
export function elevationTooltipLines(parts: readonly string[]): string {
  return parts.filter((s) => s !== '').map((s) => escapeHtml(s)).join('<br>')
}
