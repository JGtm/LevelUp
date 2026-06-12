/**
 * sparkline.ts — géométrie pure d'une sparkline (mini-tendance sans axes),
 * testable sans DOM. Rendu SVG inline (flat hard-edge) plutôt qu'ECharts :
 * léger, pas de lazy-load par cellule, pas de mock canvas dans les tests.
 */

/** Arrondi à 0,1 px pour des `points` SVG compacts et stables. */
function r1(n: number): number {
  return Math.round(n * 10) / 10
}

/**
 * sparklinePoints — convertit une série en attribut `points` pour
 * `<polyline>`. x réparti uniformément sur [0, width] ; y normalisé sur
 * [pad, height-pad] selon min/max (origine SVG en haut → série inversée pour
 * que « plus grand = plus haut »). Série constante → ligne médiane. Série vide
 * → "". `pad` réserve une marge verticale pour ne pas rogner le trait.
 */
export function sparklinePoints(
  values: number[],
  width: number,
  height: number,
  pad = 1.5,
): string {
  if (values.length === 0) return ''
  if (values.length === 1) {
    const y = r1(height / 2)
    return `0,${y} ${r1(width)},${y}`
  }
  const min = Math.min(...values)
  const max = Math.max(...values)
  const range = max - min
  const innerH = height - pad * 2
  const stepX = width / (values.length - 1)
  return values
    .map((v, i) => {
      const x = i * stepX
      const norm = range === 0 ? 0.5 : (v - min) / range
      const y = pad + (1 - norm) * innerH
      return `${r1(x)},${r1(y)}`
    })
    .join(' ')
}

/**
 * lastPoint — coordonnées du dernier point (pour matérialiser la valeur
 * courante d'un dot). null si série vide.
 */
export function lastPoint(
  values: number[],
  width: number,
  height: number,
  pad = 1.5,
): { x: number; y: number } | null {
  const pts = sparklinePoints(values, width, height, pad)
  if (!pts) return null
  const parts = pts.split(' ')
  const [x, y] = parts[parts.length - 1].split(',').map(Number)
  return { x, y }
}
