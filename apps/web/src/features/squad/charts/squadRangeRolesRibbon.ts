/**
 * squadRangeRolesRibbon — l'option ECharts du ruban « Rôles de portée » de l'Escouade : une ligne
 * par joueur (son nom à gauche, à sa couleur), une colonne par match du plus ancien au plus
 * récent (mêmes étiquettes « #N · carte » que les autres graphes par match de la page), une case
 * par match joué, à la couleur du RÔLE lu sur la fenêtre glissante (Ligne de front / Polyvalent /
 * Tireur d'élite : rampe ordinale chaude -> neutre -> froide, jetons `range-role-*`).
 *
 * Une case sans rôle reste VIDE (fond de la carte, sans motif) : fenêtre incomplète ou match sous
 * `PLANCHER_MESURE` frags mesurés. Aucun marquage visuel du petit échantillon : il se lit dans
 * l'infobulle de la case (« 3 frags mesurés, match peu sûr »). Une case vide reste survolable, une
 * série transparente la porte.
 *
 * Le nuage de rôles du Résumé solo et des Sessions garde `squadRangeRolesChart.ts` : ce ruban est
 * propre à l'Escouade, où la lecture d'une soirée à plusieurs tient en une grille joueur x match.
 */
import type { ChartOption } from '@/components/charts/chartOption'

import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
  legendEntries,
} from '@/components/charts/_utils'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'

import {
  ROLES_ORDONNES,
  type CelluleRuban,
  type LigneRuban,
  type RoleDePortee,
} from '../squadRangeRoles.logic'

/** Jeton de chaque rôle : rampe ordinale chaude (front) -> neutre (polyvalent) -> froide (sniper). */
export const JETON_ROLE: Record<RoleDePortee, SemanticToken> = {
  front: 'range-role-front',
  polyvalent: 'range-role-mid',
  sniper: 'range-role-far',
}

export interface RangeRolesRibbonOpts {
  /** Étiquettes des matchs (« #N · carte »), indexées par ordre du match. */
  categories: string[]
  /** Encre par gamertag — la même source que le reste de la page. */
  couleurs: Record<string, string>
  libelles: {
    xAxis: string
    roles: Record<RoleDePortee, string>
    tooltipRole: (role: string) => string
    tooltipNoRole: string
    tooltipDelta: (delta: string) => string
    tooltipMeasured: (n: number) => string
    tooltipUnsure: (n: number) => string
  }
  /** Écart en mètres, SIGNÉ (« +2,0 », « −1,5 »), séparateurs de la langue. */
  fmtDeltaM: (v: number) => string
}

/** Pas de hauteur d'une ligne de joueur (px) : la hauteur de la carte en découle. */
export const HAUTEUR_LIGNE = 36
/** Marges du graphe hors lignes : titre de l'axe, étiquettes des matchs, légende. */
export const HAUTEUR_MARGES = 130

/** Hauteur de la carte pour `n` joueurs. */
export function hauteurRuban(n: number): number {
  return Math.max(200, HAUTEUR_MARGES + n * HAUTEUR_LIGNE)
}

interface CaseDatum {
  value: [number, number, number]
  tip: string
}

/** L'infobulle d'une case : joueur, match, rôle, écart signé, frags mesurés (petit échantillon dit ici). */
function infobulle(ligne: LigneRuban, c: CelluleRuban, opts: RangeRolesRibbonOpts): string {
  const l = opts.libelles
  const lignes = [
    `<b>${escapeHtml(ligne.gamertag)}</b>`,
    escapeHtml(opts.categories[c.ordre] ?? `#${c.ordre + 1}`),
  ]
  if (c.role) lignes.push(escapeHtml(l.tooltipRole(l.roles[c.role])))
  else if (c.plein) lignes.push(escapeHtml(l.tooltipNoRole))
  lignes.push(escapeHtml(l.tooltipDelta(opts.fmtDeltaM(c.ecartM))))
  lignes.push(escapeHtml(c.plein ? l.tooltipMeasured(c.mesures) : l.tooltipUnsure(c.mesures)))
  return lignes.join('<br/>')
}

function donnees(lignes: LigneRuban[], opts: RangeRolesRibbonOpts, role: RoleDePortee | null): CaseDatum[] {
  return lignes.flatMap((ligne, y) =>
    ligne.cellules
      .filter((c) => c.role === role)
      .map((c) => ({ value: [c.ordre, y, 1] as [number, number, number], tip: infobulle(ligne, c, opts) })),
  )
}

interface RenderApi {
  coord: (v: number[]) => number[]
  size: (v: number[]) => number[]
  value: (i: number) => number
  /** Le style de l'élément, tiré de l'`itemStyle` de la série (couleur du rôle). */
  style: () => Record<string, unknown>
}

/** Écart (px) entre deux cases : la grille se lit case par case. */
const ECART_CASE = 2
const RAYON_CASE = 2

/**
 * Une série de cases (rectangles sur deux axes catégoriels). Série custom et non heatmap : le
 * type heatmap d'ECharts exige un visualMap, que ni la couleur d'un rôle (une teinte par
 * série, nommée dans la légende) ni la case vide n'ont de raison de porter.
 */
function serieCases(name: string, data: CaseDatum[], fill: string, z: number): unknown {
  return {
    type: 'custom',
    name,
    z,
    data,
    itemStyle: { color: fill },
    encode: { x: 0, y: 1 },
    renderItem: (_params: unknown, api: unknown) => {
      const a = api as RenderApi
      const [cx, cy] = a.coord([a.value(0), a.value(1)])
      const [w, h] = a.size([1, 1])
      return {
        type: 'rect',
        shape: { x: cx - w / 2 + ECART_CASE / 2, y: cy - h / 2 + ECART_CASE / 2, width: w - ECART_CASE, height: h - ECART_CASE, r: RAYON_CASE },
        style: a.style(),
      }
    },
  }
}

const tipOf = (p: { data?: { tip?: string } }) => p.data?.tip ?? ''

export function buildRangeRolesRibbonOption(lignes: LigneRuban[], opts: RangeRolesRibbonOpts): ChartOption {
  if (lignes.length === 0) return { backgroundColor: CHART_BG }
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const n = opts.categories.length
  const couleurRole = (role: RoleDePortee) => resolveToken(JETON_ROLE[role])
  const gamertags = lignes.map((l) => l.gamertag)

  const seriesRoles = ROLES_ORDONNES.map((role) =>
    serieCases(opts.libelles.roles[role], donnees(lignes, opts, role), couleurRole(role), 2),
  )
  // Cases sans rôle : transparentes (le fond de la carte), survolables pour dire pourquoi.
  const serieVides = serieCases('', donnees(lignes, opts, null), 'transparent', 1)

  return {
    backgroundColor: CHART_BG,
    animation: false,
    grid: { top: 12, bottom: 78, left: 112, right: 16 },
    legend: {
      ...getLegendBase(tc),
      bottom: 4,
      left: 'center',
      data: legendEntries(ROLES_ORDONNES.map((r) => ({ name: opts.libelles.roles[r], color: couleurRole(r) }))),
    },
    tooltip: { ...getTooltipBase(tc), trigger: 'item', formatter: tipOf },
    xAxis: {
      ...axis,
      type: 'category',
      data: opts.categories,
      name: opts.libelles.xAxis,
      nameLocation: 'middle',
      nameGap: 32,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      splitArea: { show: false },
      splitLine: { show: false },
      // `hideOverlap` : en colonne étroite, les étiquettes qui se chevaucheraient sont masquées ;
      // l'infobulle d'une case nomme toujours son match.
      axisLabel: { ...axis.axisLabel, fontSize: 9, interval: n > 12 ? Math.floor(n / 8) : 0, hideOverlap: true },
    },
    yAxis: {
      ...axis,
      type: 'category',
      data: gamertags,
      inverse: true,
      splitArea: { show: false },
      splitLine: { show: false },
      axisTick: { show: false },
      axisLine: { show: false },
      // Le nom de chaque joueur à sa couleur : l'identité se lit sans légende.
      axisLabel: {
        ...axis.axisLabel,
        fontWeight: 600,
        fontSize: 12,
        color: (value: string) => opts.couleurs[value] ?? tc.text,
      },
    },
    series: [...seriesRoles, serieVides],
  }
}
