/**
 * axisFootProblems — vérifie, sur une OPTION ECharts déjà construite, que les étiquettes de l'axe X
 * et la bande tiennent dans le pied réservé (`grid[0].bottom`). Lit l'option telle qu'ECharts la
 * recevra (formatters appelés, marges et hauteurs de ligne lues), donc indépendant de
 * `layoutAxisFoot` : un nombre réécrit à la main qui ne tient plus est attrapé ici. Pour les tests.
 */
interface AxisLike {
  show?: boolean
  data?: unknown[]
  axisLabel?: {
    margin?: number
    lineHeight?: number
    formatter?: (value: string, index: number) => string
  }
}

interface GridLike {
  bottom?: number
  height?: number
}

interface OptionLike {
  grid?: GridLike | GridLike[]
  xAxis?: AxisLike | AxisLike[]
}

const asList = <T>(v: T | T[] | undefined): T[] => (v === undefined ? [] : Array.isArray(v) ? v : [v])

/** Hauteur du bloc d'étiquettes d'un axe, ou un problème si elle ne se lit pas. */
function blockHeight(axis: AxisLike | undefined, name: string, problems: string[]): number {
  if (!axis || axis.show === false || !axis.axisLabel?.formatter) return 0
  const { formatter, margin, lineHeight } = axis.axisLabel
  const count = Math.max(axis.data?.length ?? 0, 1)
  let lines = 0
  for (let i = 0; i < count; i++) {
    const text = formatter(String(i), i)
    if (text !== '') lines = Math.max(lines, text.split('\n').length)
  }
  if (lines === 0) return 0
  if (typeof margin !== 'number' || typeof lineHeight !== 'number') {
    problems.push(`${name} : margin et lineHeight doivent être écrits (le pied se calcule dessus)`)
    return 0
  }
  return margin + lines * lineHeight
}

/** Les débordements du pied ; vide quand la bande et les étiquettes tiennent sous la grille. */
export function axisFootProblems(option: OptionLike): string[] {
  const problems: string[] = []
  const grids = asList(option.grid)
  const axes = asList(option.xAxis)
  const foot = grids[0]?.bottom
  if (typeof foot !== 'number') return ['grid[0].bottom doit être un nombre']
  const above = blockHeight(axes[0], 'xAxis[0]', problems)
  const band = grids[1]
  if (!band) {
    if (above > foot) problems.push(`étiquettes (${above} px) > bas de grille (${foot} px)`)
    return problems
  }
  const bandBottomEdge = foot - (band.bottom ?? 0)
  const bandTop = bandBottomEdge - (band.height ?? 0)
  const below = blockHeight(axes[1], 'xAxis[1]', problems)
  if (bandTop < above) problems.push(`la bande (haut à ${bandTop} px) recouvre les étiquettes de l'axe (${above} px)`)
  if (bandBottomEdge + below > foot) {
    problems.push(`bande + étiquettes (${bandBottomEdge + below} px) > bas de grille (${foot} px)`)
  }
  return problems
}
