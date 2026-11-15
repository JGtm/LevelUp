/**
 * mapBackgroundVariant — LES DEUX IMAGES D'UN FOND DE CARTE et le choix entre elles.
 *
 * Un fond publié porte son image « rendu » et, quand le plan a été cuit, une image « plan » (vue
 * plan à hauteurs : silhouette du sol teintée par altitude, murs, étages recouverts en pointillé).
 * Les deux ont les mêmes dimensions et le même calage : changer d'image ne change ni la projection,
 * ni l'orientation, ni le zoom, ni le survol.
 *
 * Le choix de l'utilisateur est une préférence globale (`replay-background-style`) ; ce module ne
 * fait que la lire comme une variante à demander : la préférence ne vaut que pour un fond qui a un
 * plan, sinon l'image rendu est servie sans rien dire.
 */

/** La variante d'image d'un fond : `rendu` (le fond) ou `plan` (la vue plan à hauteurs). */
export type MapBackgroundVariant = 'rendu' | 'plan'

/** Les variantes, dans l'ordre où la commande les propose. */
export const MAP_BACKGROUND_VARIANTS: readonly MapBackgroundVariant[] = ['rendu', 'plan']

/** Le rendu est le défaut : c'est le fond d'avant le plan. */
export const DEFAULT_MAP_BACKGROUND_VARIANT: MapBackgroundVariant = 'rendu'

/** La valeur de `variante=` que l'API accepte pour l'image plan. */
const PLAN_QUERY = 'variante=plan'

/**
 * normalizeBackgroundStyle lit une valeur persistée de façon tolérante : tout ce qui n'est pas une
 * variante connue (absent, écrit par une autre version) vaut le rendu.
 */
export function normalizeBackgroundStyle(raw: unknown): MapBackgroundVariant {
  return raw === 'plan' ? 'plan' : DEFAULT_MAP_BACKGROUND_VARIANT
}

/**
 * resolveBackgroundVariant dit quelle image demander : le plan quand l'utilisateur le veut ET que
 * le fond en porte un (`planImage` du sidecar), le rendu sinon.
 */
export function resolveBackgroundVariant(
  style: MapBackgroundVariant,
  planImage: string | null | undefined,
): MapBackgroundVariant {
  return style === 'plan' && !!planImage ? 'plan' : 'rendu'
}

/**
 * isWallPlanDrawn dit si le calque « Plan des murs » se trace : l'interrupteur est allumé ET l'image
 * posée n'est pas le plan, qui porte déjà les murs (les tracer en double les épaissirait). Le calque
 * reste un réglage de l'utilisateur : posé de nouveau sur le rendu, il reparaît tel qu'il l'a laissé.
 */
export function isWallPlanDrawn(showWallPlan: boolean, variante: MapBackgroundVariant): boolean {
  return showWallPlan && variante === 'rendu'
}

/** backgroundImagePath ajoute la variante à l'adresse de l'image (le rendu garde l'adresse nue). */
export function backgroundImagePath(path: string, variante: MapBackgroundVariant): string {
  return variante === 'plan' ? `${path}?${PLAN_QUERY}` : path
}

/**
 * loadedBackground choisit ce qui est posé parmi les images chargées et dit quelle variante c'est :
 * le plan quand il est demandé ET chargé, sinon le rendu (repli ou rendu demandé), sinon rien — la
 * variante vaut alors celle qu'on a demandée. Les murs se décident sur cette variante RÉELLE.
 */
export function loadedBackground<T>(
  demandee: MapBackgroundVariant,
  plan: T | undefined,
  rendu: T | undefined,
): { image: T | null; variante: MapBackgroundVariant } {
  if (demandee === 'plan' && plan !== undefined) return { image: plan, variante: 'plan' }
  if (rendu !== undefined) return { image: rendu, variante: 'rendu' }
  return { image: null, variante: demandee }
}
