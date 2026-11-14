/**
 * alignedZeroAxes — bornes de deux axes verticaux (gauche, droite) qui partagent la MÊME ligne
 * zéro, pour un graphe dont une partie des barres descend sous l'axe (les morts, les dégâts
 * subis) et dont une seconde mesure, d'une autre échelle, s'ajoute sur l'axe de droite.
 *
 * Sans cet alignement, chaque axe place son zéro à sa propre hauteur : une valeur « zéro » de
 * l'axe de droite flotterait au-dessus ou au-dessous de la ligne zéro de l'axe de gauche.
 */

/** Étendue d'un axe : amplitude sous le zéro (`neg`) et au-dessus (`pos`), toutes deux >= 0. */
export interface AxisExtent {
  neg: number
  pos: number
}

export interface AlignedAxes {
  left: [number, number]
  right: [number, number]
}

/** Paliers « jolis » de l'amplitude (mantisse), du plus serré au plus large. */
const NICE_STEPS = [1, 1.5, 2, 3, 4, 5, 6, 8, 10]

/** Plus petit nombre « joli » (1, 1,5, 2, 3, 4, 5, 6, 8 x 10^k) supérieur ou égal à `x`. */
export function niceCeil(x: number): number {
  if (!Number.isFinite(x) || x <= 0) return 0
  const exp = Math.floor(Math.log10(x))
  const base = 10 ** exp
  const mantissa = x / base
  const step = NICE_STEPS.find((s) => s >= mantissa - 1e-9) ?? 10
  return step * base
}

function negativeShare(e: AxisExtent): number {
  const total = e.neg + e.pos
  return total > 0 ? e.neg / total : 0
}

/**
 * Bornes alignées : la part de hauteur sous le zéro est la plus grande des deux axes, chaque axe
 * est ensuite élargi (jamais rogné) à une amplitude « jolie » qui contient ses valeurs. Deux
 * étendues nulles rendent des bornes de repli (un axe ne peut pas être vide).
 */
export function alignedZeroAxes(left: AxisExtent, right: AxisExtent): AlignedAxes {
  const share = Math.min(Math.max(negativeShare(left), negativeShare(right)), 0.95)
  const bounds = (e: AxisExtent): [number, number] => {
    const span = niceCeil(Math.max(share > 0 ? e.neg / share : 0, e.pos / (1 - share)))
    if (span <= 0) return [-share, 1 - share]
    return [-share * span, (1 - share) * span]
  }
  return { left: bounds(left), right: bounds(right) }
}
