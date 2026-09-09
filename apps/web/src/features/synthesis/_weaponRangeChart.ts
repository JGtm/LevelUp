/**
 * _weaponRangeChart — LA PORTÉE D'USAGE PAR ARME, mes frags ET mes morts sur la MÊME ligne.
 *
 * Grammaire reprise du graphe de portée par match validé le 2026-09-02
 * (`features/match-view/_killDistanceChart.ts`) : un bâton par arme, un losange sur la
 * valeur centrale. Transposition à l'agrégat multi-matchs (décision D6 du plan
 * `.ai/PLAN_DUELS_PORTEE_2026-09-06.md`) : le bâton court du 10e au 90e centile et le
 * losange marque la MÉDIANE — sur des centaines de frags, min et max décrivent deux
 * accidents, pas une portée. L'utilisateur a demandé le 2026-09-06 la FUSION des deux
 * graphes jumeaux : une ligne par arme, deux bâtons (frags en haut, morts en bas), même axe.
 *
 * POURQUOI UNE SÉRIE `custom` ET PAS bar+scatter comme le graphe de match : sur un axe de
 * catégories, un `scatter` se pose au CENTRE de la bande et ne sait pas se décaler d'un
 * demi-bâton — le losange de la médiane des morts tomberait entre les deux bâtons. Le
 * `renderItem` place les quatre formes au pixel, à partir des coordonnées que l'API ECharts
 * rend elle-même : la position est exacte, pas approchée.
 *
 * CE QUI EST UNE PORTÉE D'USAGE, JAMAIS UNE PORTÉE D'ARME (D8) : ces nombres décrivent LE
 * JOUEUR — à quelle distance il frague et à quelle distance il meurt — pas la balistique du
 * BR75. Aucun libellé de ce module ne dit « portée de l'arme ».
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
import type { WeaponRangeRow, WeaponRangeSide } from '@/lib/api/types'
import type { ManifestLocale } from '@/lib/i18n/format'

import { resolveWeaponLabel } from './weaponRange_logic'

/** Hauteur d'une bande : deux bâtons + leur écart y tiennent (26 px suffisaient à un seul). */
export const WEAPON_RANGE_ROW_PX = 34
/** Épaisseur d'un bâton, et écart entre les deux bâtons d'une même ligne. */
const BAR_HEIGHT = 7
const BAR_GAP = 4
/** Demi-diagonale du losange de médiane. */
const DIAMOND_RADIUS = 5

/**
 * WEAPON_KEYS_WITHOUT_RANGE — les pseudo-armes pour lesquelles une DISTANCE N'A PAS DE SENS.
 *
 * « Chute et environnement » (`hinf_environment`) agrège les morts par chute, écrasement ou
 * élément de décor : la « distance tueur → victime » y mesure l'écart entre la victime et
 * un point de géométrie, pas une portée d'engagement. La ligne polluait l'axe et tirait la
 * médiane (retrait demandé le 2026-09-09).
 *
 * FILTRE CÔTÉ FRONT, ET C'EST UN PIS-ALLER ASSUMÉ : la place durable de cette exclusion est
 * le classifieur de sources côté Go (`port.KillSourceClassifier`), qui sait déjà rattacher
 * un `source_tag` à sa nature — le contrat de portée ne porte pas la classe de l'arme, donc
 * le front n'a que la clé pour trancher. Un `Set` et non un test d'égalité : un autre titre
 * ajoute sa clé sans réécrire la condition.
 */
const WEAPON_KEYS_WITHOUT_RANGE = new Set(['hinf_environment'])

/** Une arme projetée pour les DEUX graphes de la section — côté absent = `null`. */
export interface WeaponRangeLine {
  weaponKey: string
  label: string
  kills: WeaponRangeSide | null
  deaths: WeaponRangeSide | null
}

/**
 * weaponRangeLines — la projection des lignes du contrat.
 *
 * L'ORDRE DU BACKEND EST CONSERVÉ (médiane des frags croissante, cf. `mergeWeaponSides`
 * côté Go) : le graphe se lit comme un continuum du contact à la longue portée. Rien n'est
 * retrié ici — deux tris du même fait divergeraient au premier changement de doctrine.
 *
 * Le contrat OMET un côté non mesuré (jamais un zéro, qui se lirait « mesuré, à zéro
 * mètre ») ; la projection le normalise en `null` pour que le rendu ait un seul cas à
 * traiter.
 */
export function weaponRangeLines(
  weapons: readonly WeaponRangeRow[] | null | undefined,
  locale: ManifestLocale,
): WeaponRangeLine[] {
  return (weapons ?? [])
    .filter((w) => !WEAPON_KEYS_WITHOUT_RANGE.has(w.weapon_key))
    .map((w) => ({
      weaponKey: w.weapon_key,
      label: resolveWeaponLabel(w, locale),
      kills: w.kills ?? null,
      deaths: w.deaths ?? null,
    }))
}

/**
 * weaponRangeCategoryLabel — le nom de l'arme, nu.
 *
 * L'étiquette portait jusqu'au 2026-09-09 le suffixe « ×281/402 » (frags puis morts
 * mesurés). Retiré à la demande de l'utilisateur : deux nombres collés à chaque nom d'arme
 * allongeaient la gouttière de gauche sans être lus, alors que les DEUX effectifs sont déjà
 * écrits dans l'infobulle de la ligne (« Mes frags — 281 : … ») et dans le tableau
 * dépliable, où ils ont une colonne à eux.
 *
 * La fonction reste la SOURCE UNIQUE de l'étiquette : les deux graphes de la section
 * (portée et dénivelé) partagent le même axe de catégories, et deux fabrications
 * séparées divergeraient au premier changement.
 */
export function weaponRangeCategoryLabel(line: WeaponRangeLine): string {
  return line.label
}

/** Hauteur du graphe : une bande par arme, plus la place des axes. */
export function weaponRangeChartHeight(lineCount: number): number {
  return 48 + WEAPON_RANGE_ROW_PX * lineCount
}

/**
 * weaponRangeAxisMax — la borne haute de l'axe des distances, COMMUNE aux deux côtés.
 *
 * Un axe par côté rendrait les deux bâtons d'une ligne incomparables — c'est-à-dire
 * exactement ce que la fusion des deux graphes cherchait à éviter. Arrondi aux 5 m
 * supérieurs pour que la dernière graduation soit lisible, plancher à 5 m (un corpus tout
 * au contact ne doit pas produire un axe dégénéré).
 */
export function weaponRangeAxisMax(lines: readonly WeaponRangeLine[]): number {
  let max = 0
  for (const line of lines) {
    for (const side of [line.kills, line.deaths]) {
      if (side && side.p90 > max) max = side.p90
    }
  }
  return Math.max(5, Math.ceil(max / 5) * 5)
}

/** Coordonnées rendues par l'API `renderItem` d'ECharts (sous-ensemble utilisé ici). */
interface RenderItemApi {
  coord(point: [number, number]): number[]
}
interface RenderItemParams {
  dataIndex: number
}
/** Un nœud graphique du groupe rendu — forme minimale consommée par ECharts. */
interface RenderItemChild {
  type: 'rect' | 'polygon'
  shape: Record<string, unknown>
  style: Record<string, unknown>
}

export interface WeaponRangeOptionInput {
  /** Lignes DANS L'ORDRE DU BACKEND — l'inversion de l'axe Y se fait ici, pas chez l'appelant. */
  lines: readonly WeaponRangeLine[]
  tc: EChartsThemeColors
  /** Encres résolues par l'appelant (tokens sémantiques) : bâtons, losange, contour. */
  killsColor: string
  deathsColor: string
  medianColor: string
  /** Contour du losange = fond de carte : il se détache du bâton sans couleur nouvelle. */
  cardColor: string
  /** Formate une distance (« 12,4 m ») — la locale vit chez l'appelant. */
  fmtDistance: (m: number) => string
  /** Libellés d'infobulle, déjà localisés. */
  labels: { kills: string; deaths: string; percentiles: string; noMeasure: string }
}

/** Les quatre encres du `renderItem`, plus les lignes déjà retournées pour l'axe. */
interface RangeRenderInput {
  ordered: readonly WeaponRangeLine[]
  killsColor: string
  deathsColor: string
  medianColor: string
  cardColor: string
}

/**
 * makeRangeRenderItem — le `renderItem` de la série `custom`, fabriqué à part.
 *
 * Extrait de `buildWeaponRangeOption` le 2026-09-06 (revue adversariale du lot 5, seuil de
 * 80 lignes par fonction) : la GÉOMÉTRIE et l'ASSEMBLAGE de l'option sont deux sujets, et le
 * test de géométrie pince déjà cette fonction à travers une API `renderItem` factice.
 * Aucun changement de rendu — les mêmes formes, aux mêmes pixels.
 */
function makeRangeRenderItem({
  ordered,
  killsColor,
  deathsColor,
  medianColor,
  cardColor,
}: RangeRenderInput) {
  return (params: RenderItemParams, api: RenderItemApi) => {
    const line = ordered[params.dataIndex]
    const children: RenderItemChild[] = []
    if (!line) return { type: 'group', children }
    const yCenter = api.coord([0, params.dataIndex])[1]
    const x = (v: number) => api.coord([v, 0])[0]
    const push = (side: WeaponRangeSide | null, color: string, dy: number) => {
      if (!side) return
      children.push({
        type: 'rect',
        shape: {
          x: x(side.p10),
          y: yCenter + dy - BAR_HEIGHT / 2,
          // Plancher de 2 px : une portée quasi ponctuelle reste visible (un bâton de
          // largeur nulle disparaîtrait, et l'arme avec lui).
          width: Math.max(2, x(side.p90) - x(side.p10)),
          height: BAR_HEIGHT,
          r: 4,
        },
        style: { fill: color },
      })
      const cx = x(side.median)
      const cy = yCenter + dy
      children.push({
        type: 'polygon',
        shape: {
          points: [
            [cx, cy - DIAMOND_RADIUS],
            [cx + DIAMOND_RADIUS, cy],
            [cx, cy + DIAMOND_RADIUS],
            [cx - DIAMOND_RADIUS, cy],
          ],
        },
        style: { fill: medianColor, stroke: cardColor, lineWidth: 1 },
      })
    }
    const offset = BAR_HEIGHT / 2 + BAR_GAP / 2
    push(line.kills, killsColor, -offset)
    push(line.deaths, deathsColor, +offset)
    return { type: 'group', children }
  }
}

/**
 * rangeTooltipSideLine — une ligne d'infobulle pour un côté : « Mes frags — 281 : 7,1 m ·
 * 13,6 m · 24,9 m », ou « Mes frags — aucune mesure ».
 *
 * Le nom de l'arme comme les nombres formatés passent par `escapeHtml` : l'infobulle d'ECharts
 * est du HTML, et un libellé de registre n'est pas une source de confiance.
 */
function rangeTooltipSideLine(
  name: string,
  side: WeaponRangeSide | null,
  fmtDistance: (m: number) => string,
  noMeasure: string,
): string {
  if (!side) return `${escapeHtml(name)} — ${escapeHtml(noMeasure)}`
  const low = escapeHtml(fmtDistance(side.p10))
  const median = escapeHtml(fmtDistance(side.median))
  const high = escapeHtml(fmtDistance(side.p90))
  return `${escapeHtml(name)} — ${side.measured} : ${low} · <b>${median}</b> · ${high}`
}

/**
 * buildWeaponRangeOption — l'option ECharts du graphe de portée.
 *
 * Contrat de rendu, ligne par ligne : deux rectangles arrondis p10 → p90 décalés de part et
 * d'autre du centre de bande (frags AU-DESSUS, morts en dessous — l'ordre de la légende), et
 * un losange par médiane. Un côté absent ne dessine RIEN de son côté ; l'infobulle le dit
 * (« aucune mesure »), elle n'invente pas un zéro.
 */
export function buildWeaponRangeOption({
  lines,
  tc,
  killsColor,
  deathsColor,
  medianColor,
  cardColor,
  fmtDistance,
  labels,
}: WeaponRangeOptionInput): EChartsCoreOption {
  // Premier du backend = plus courte portée = EN HAUT : l'axe Y d'ECharts empile du bas
  // vers le haut, donc la liste se lit à l'envers au montage.
  const ordered = [...lines].reverse()
  const xMax = weaponRangeAxisMax(lines)
  const axis = getAxisBase(tc)
  const renderItem = makeRangeRenderItem({
    ordered,
    killsColor,
    deathsColor,
    medianColor,
    cardColor,
  })
  const sideLine = (name: string, side: WeaponRangeSide | null) =>
    rangeTooltipSideLine(name, side, fmtDistance, labels.noMeasure)

  return {
    backgroundColor: CHART_BG,
    grid: getGridBase({ top: 8, bottom: 24, left: 8 }),
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: (params: unknown) => {
        const p = params as { dataIndex?: number }
        const line = p?.dataIndex != null ? ordered[p.dataIndex] : undefined
        if (!line) return ''
        return [
          `<b>${escapeHtml(line.label)}</b> — ${escapeHtml(labels.percentiles)}`,
          sideLine(labels.kills, line.kills),
          sideLine(labels.deaths, line.deaths),
        ].join('<br/>')
      },
    },
    xAxis: {
      type: 'value',
      max: xMax,
      ...axis,
      axisLabel: { ...axis.axisLabel, formatter: (v: number) => fmtDistance(v) },
    },
    yAxis: {
      type: 'category',
      data: ordered.map(weaponRangeCategoryLabel),
      ...axis,
      splitLine: { show: false },
    },
    series: [
      {
        type: 'custom',
        renderItem,
        // `encode` déclare à ECharts quelles dimensions bornent l'axe : la valeur 0 et la
        // borne haute, pour que l'échelle couvre exactement l'étendue voulue même si le
        // `renderItem` dessine, lui, aux coordonnées de chaque côté.
        encode: { x: [1, 2], y: 0 },
        data: ordered.map((_, i) => [i, 0, xMax]),
      },
    ],
  }
}
