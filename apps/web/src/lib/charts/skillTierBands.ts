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
 * sous-rang).
 */
import type { ManifestLocale } from '@/lib/i18n/format'
import type { SkillTierGrid, SkillTier } from '@/lib/skillTiers'

const STEP = 100
const MIN_BANDS = 3
const OPEN_TOP = 9000 // au-delà = palier ouvert (Onyx), pas de sous-palier au-dessus

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

/**
 * Bandes de sous-palier en markArea ECharts, clippées à [yMin, yMax], shading
 * neutre alterné (splitAreaA/B) + label. Parité alternée sur les bandes
 * visibles pour un damier lisible quel que soit le zoom.
 */
export function buildSkillTierMarkArea(
  locale: ManifestLocale,
  yMin: number,
  yMax: number,
  grid: SkillTierGrid,
  tc: MarkAreaTheme,
) {
  const data: Array<[object, object]> = []
  let parity = 0
  for (const tier of grid.tiers) {
    const n = Math.max(1, tier.subTiers)
    const width = (tier.max - tier.min) / n
    for (let k = 0; k < n; k++) {
      const lo = Math.max(tier.min + k * width, yMin)
      const hi = Math.min(tier.min + (k + 1) * width, yMax)
      if (hi <= lo) continue
      data.push([
        {
          yAxis: lo,
          name: subTierLabel(grid, tier, k, locale),
          itemStyle: { color: parity % 2 === 0 ? tc.splitAreaA : tc.splitAreaB },
          label: { show: true, position: 'insideTopLeft' as const, fontSize: 9, color: tc.axisLabel, opacity: 0.7 },
        },
        { yAxis: hi },
      ])
      parity++
    }
  }
  return { silent: true, data }
}
