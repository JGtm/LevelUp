/**
 * endPoint.ts — LE POINT FINAL GROSSI d'une courbe ECharts (source unique, CLAUDE.md n° 6).
 *
 * Une courbe qui porte une valeur au bout (cumul, écart cumulé, « ce soir ») grossit le point
 * qui porte cette valeur et le lisère à la couleur de la carte pour le détacher de la courbe ;
 * les autres points gardent leur taille. Le motif était écrit à la main à quatre endroits
 * (écart au FDA attendu, fil de l'objectif, fil des ressources, soirée après soirée des
 * ressources ; plus le soir de l'objectif) — revue L6.1 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, constat R7. Garde-rail :
 * `endPoint.guard.test.ts` interdit la ré-écriture locale.
 *
 * QUEL POINT : par défaut le DERNIER point non nul (la fin d'un cumul) ; `at` en désigne un autre
 * (« ce soir », qui n'est pas forcément le dernier point porteur d'une valeur). Un point absent
 * (`null`) à cet index : rien n'est grossi — jamais un point voisin à sa place.
 */

/** Diamètre du point final (rayon ~4,5 px des maquettes). */
export const END_POINT_SIZE = 9

/** Un point de série : un nombre brut, un objet de donnée ECharts, ou un trou. */
export type SeriesDatum = number | { value: unknown; itemStyle?: object; [key: string]: unknown } | null

export interface EndPointOptions {
  /** L'index du point à grossir ; défaut : le dernier point non nul. */
  at?: number
  /** Diamètre (défaut `END_POINT_SIZE`). */
  size?: number
  /** Liseré couleur de carte (défaut 2 px). */
  borderWidth?: number
  /** Couleur du point, quand la série ne la donne pas déjà. */
  color?: string
  /** Champs ajoutés au point grossi (infobulle…), calculés pour son index. */
  extra?: (index: number) => Record<string, unknown>
}

/** L'index du dernier point non nul, -1 sans point. */
export function lastPointIndex(data: readonly SeriesDatum[]): number {
  for (let i = data.length - 1; i >= 0; i--) if (data[i] != null) return i
  return -1
}

/**
 * withEndPoint rend une copie de `data` dont le point final (voir `EndPointOptions.at`) est
 * grossi et liseré à la couleur de la carte `card` ; les autres points sont rendus tels quels.
 */
export function withEndPoint(data: readonly SeriesDatum[], card: string, opts: EndPointOptions = {}): SeriesDatum[] {
  const at = opts.at ?? lastPointIndex(data)
  const point = at >= 0 && at < data.length ? data[at] : null
  if (point == null) return [...data]
  const base = typeof point === 'number' ? { value: point } : point
  const grown = {
    ...base,
    symbol: 'circle',
    symbolSize: opts.size ?? END_POINT_SIZE,
    itemStyle: {
      ...(base.itemStyle ?? {}),
      ...(opts.color ? { color: opts.color } : {}),
      borderColor: card,
      borderWidth: opts.borderWidth ?? 2,
    },
    ...(opts.extra ? opts.extra(at) : {}),
  }
  return data.map((d, i) => (i === at ? grown : d))
}
