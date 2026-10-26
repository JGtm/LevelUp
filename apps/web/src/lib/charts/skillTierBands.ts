/**
 * skillTierBands — helpers partagés pour cadrer un axe Y de classement (LUSR/CSR)
 * sur la magnitude de la session et dessiner les bandes de SOUS-PALIER.
 *
 * Consommé par le graphe « Classement » par-match (features/timeseries) et le
 * graphe « Évolution LUSR / CSR » de la carrière (features/career).
 *
 * Cadrage : `frameToData` cadre [min, max] sur les sous-paliers contenant les
 * données (+ 1 sous-palier de marge de chaque côté, plancher MIN_BANDS) au lieu
 * du tier entier — pour révéler le mouvement par-match sur les sessions courtes
 * sans amplifier le bruit (le plancher empêche d'étaler 1 pas de quantification
 * sur tout le graphe). L'échelle dépend de la grille passée : LUSR = legacy
 * 1000-2000, CSR = Halo brut. Le palier Onyx (ouvert vers le haut) est arrondi
 * au STEP au lieu d'utiliser la borne sentinelle 9999.
 *
 * Bandes : `buildSkillTierMarkArea` dessine une bande par sous-palier en shading
 * neutre alterné (theme splitAreaA/B) + label « Or III » / « Diamond 3 ». Pas de
 * couleur par rang (zéro token de plus, identique LUSR/CSR ; zoomé dans un tier
 * la couleur serait un aplat constant — c'est le label qui porte l'identité du
 * sous-rang). Le label est centré VERTICALEMENT dans sa bande : posé en haut, il
 * collait à la borne du sous-palier suivant et une valeur au milieu d'une bande
 * se lisait « entre » deux rangs. Quand un sous-palier visible ferait moins de
 * MIN_LABEL_PX à l'écran, les bandes passent au palier entier (« Diamant »), et
 * une bande trop fine pour son label le masque : jamais de libellés empilés.
 */
import type { ManifestLocale } from '@/lib/i18n/format'
import type { SkillTierGrid, SkillTier } from '@/lib/skillTiers'

const STEP = 100
const MIN_BANDS = 3
const OPEN_TOP = 9000 // au-delà = palier ouvert (Onyx), pas de sous-palier au-dessus
const MIN_LABEL_PX = 12 // hauteur sous laquelle un label de bande (fontSize 9) chevauche son voisin

const ROMAN = ['I', 'II', 'III', 'IV', 'V', 'VI'] as const

/** Bornes de sous-paliers de la grille (triées, dédupliquées). Le palier ouvert
 *  (Onyx) ne contribue que sa borne d'entrée. */
function subTierEdges(grid: SkillTierGrid): number[] {
  const edges: number[] = []
  for (const tier of grid.tiers) {
    if (tier.max >= OPEN_TOP) {
      edges.push(tier.min)
      continue
    }
    const n = Math.max(1, tier.subTiers)
    const width = (tier.max - tier.min) / n
    for (let k = 0; k <= n; k++) edges.push(tier.min + k * width)
  }
  return Array.from(new Set(edges)).sort((a, b) => a - b)
}

/**
 * Cadre [min, max] de l'axe Y sur les sous-paliers contenant les données, avec
 * 1 sous-palier de marge de chaque côté et un plancher de MIN_BANDS sous-paliers
 * visibles. Snappé sur les bornes de sous-palier pour que les bandes remplissent
 * le cadre. Ne descend jamais sous 0 ; n'utilise jamais la borne ouverte d'Onyx
 * (9999) comme plafond.
 */
export function frameToData(dataMin: number, dataMax: number, grid: SkillTierGrid): { min: number; max: number } {
  const edges = subTierEdges(grid)
  const topFinite = edges[edges.length - 1] // borne d'entrée du palier ouvert (Onyx)

  // Palier ouvert : pas de sous-palier au-dessus → arrondir au STEP.
  if (dataMax >= topFinite) {
    const ref = Math.min(dataMin, topFinite)
    const loEdge = [...edges].reverse().find(e => e <= ref) ?? ref
    let max = Math.ceil(dataMax / STEP) * STEP
    if (max <= topFinite) max = topFinite + STEP
    return { min: Math.max(0, loEdge), max }
  }

  // Edge juste sous dataMin et juste au-dessus de dataMax.
  const firstAbove = edges.findIndex(e => e > dataMin)
  let loIdx = firstAbove <= 0 ? 0 : firstAbove - 1
  let hiIdx = edges.findIndex(e => e >= dataMax)
  if (hiIdx < 0) hiIdx = edges.length - 1

  // 1 sous-palier de marge de chaque côté.
  if (loIdx > 0) loIdx--
  if (hiIdx < edges.length - 1) hiIdx++

  // Plancher : au moins MIN_BANDS sous-paliers visibles.
  while (hiIdx - loIdx < MIN_BANDS) {
    if (loIdx > 0) loIdx--
    else if (hiIdx < edges.length - 1) hiIdx++
    else break
  }
  return { min: Math.max(0, edges[loIdx]), max: edges[hiIdx] }
}

/** Libellé d'un sous-palier : « Or III » (LUSR/roman) ou « Diamond 3 » (CSR/arabe).
 *  Palier sans sous-palier (Onyx) → nom du tier seul. */
function subTierLabel(grid: SkillTierGrid, tier: SkillTier, subIndex: number, locale: ManifestLocale): string {
  const name = locale === 'fr' ? tier.fr : tier.en
  if (tier.subTiers <= 1) return name
  if (grid.subTierStyle === 'roman') return `${name} ${ROMAN[subIndex] ?? subIndex + 1}`
  return `${name} ${subIndex + 1}`
}

interface MarkAreaTheme {
  splitAreaA: string
  splitAreaB: string
  axisLabel: string
}

export interface SkillTierMarkAreaInput {
  locale: ManifestLocale
  yMin: number
  yMax: number
  grid: SkillTierGrid
  theme: MarkAreaTheme
  /** Hauteur de la zone de tracé en pixels (hauteur du graphe moins les marges
   *  haut/bas). Une estimation basse est sûre : elle replie plus tôt au palier. */
  plotHeightPx: number
}

interface Band {
  lo: number
  hi: number
  label: string
}

/** Bandes visibles (clippées à [yMin, yMax]) : une par sous-palier, ou une par
 *  palier entier quand `byTier`. */
function visibleBands(input: SkillTierMarkAreaInput, byTier: boolean): Band[] {
  const { grid, locale, yMin, yMax } = input
  const bands: Band[] = []
  for (const tier of grid.tiers) {
    const n = byTier ? 1 : Math.max(1, tier.subTiers)
    const width = (tier.max - tier.min) / n
    for (let k = 0; k < n; k++) {
      const lo = Math.max(tier.min + k * width, yMin)
      const hi = Math.min(tier.min + (k + 1) * width, yMax)
      if (hi <= lo) continue
      const label = byTier ? (locale === 'fr' ? tier.fr : tier.en) : subTierLabel(grid, tier, k, locale)
      bands.push({ lo, hi, label })
    }
  }
  return bands
}

/** Vrai quand un sous-palier visible ferait moins de MIN_LABEL_PX : on passe alors
 *  au palier entier. Mesuré sur la largeur NOMINALE du sous-palier (pas la portion
 *  clippée, qui est masquée bande par bande). */
function subTiersTooThin(input: SkillTierMarkAreaInput, pxPerPoint: number): boolean {
  const { grid, yMin, yMax } = input
  return grid.tiers.some(tier => {
    if (tier.max <= yMin || tier.min >= yMax || tier.subTiers <= 1) return false
    return ((tier.max - tier.min) / tier.subTiers) * pxPerPoint < MIN_LABEL_PX
  })
}

/**
 * Bandes de palier en markArea ECharts, clippées à [yMin, yMax], shading neutre
 * alterné (splitAreaA/B) + label centré verticalement. Parité alternée sur les
 * bandes visibles pour un damier lisible quel que soit le zoom.
 */
export function buildSkillTierMarkArea(input: SkillTierMarkAreaInput) {
  const { yMin, yMax, theme: tc, plotHeightPx } = input
  const pxPerPoint = yMax > yMin ? plotHeightPx / (yMax - yMin) : 0
  const bands = visibleBands(input, subTiersTooThin(input, pxPerPoint))
  const data: Array<[object, object]> = bands.map((band, i) => [
    {
      yAxis: band.lo,
      name: band.label,
      itemStyle: { color: i % 2 === 0 ? tc.splitAreaA : tc.splitAreaB },
      label: {
        show: (band.hi - band.lo) * pxPerPoint >= MIN_LABEL_PX,
        position: 'insideLeft' as const,
        fontSize: 9,
        color: tc.axisLabel,
        opacity: 0.7,
      },
    },
    { yAxis: band.hi },
  ])
  return { silent: true, data }
}
