/**
 * mapBackgroundVariant — LES FONDS D'UNE CARTE (rendu, plan, relief) et le choix entre eux.
 *
 * Un fond publié porte son image « rendu » et, quand ils ont été cuits, une image « plan » (vue
 * plan à hauteurs : silhouette du sol teintée par altitude, murs, étages recouverts en pointillé)
 * et un « relief » (hauteurs et drapeaux par pixel, plus ses traits et ses sols : le terrain vu de
 * dessus en gris teinté). Les trois ont les mêmes dimensions et le même calage : changer de fond
 * ne change ni la projection, ni l'orientation, ni le zoom, ni le survol.
 *
 * Le choix de l'utilisateur est une préférence globale (`replay-background-style`) ; ce module ne
 * fait que la lire comme une variante à demander. La préférence ne vaut que pour un fond qui a
 * l'image demandée : sans relief, ou quand le relief ne se lit pas (`ReliefError`), c'est le plan
 * qui est posé ; sans plan non plus, le rendu. Rien n'est dit à l'utilisateur dans ces replis.
 *
 * LE RELIEF N'EST PAS UNE IMAGE À DÉCODER comme les deux autres : ses pixels sont des hauteurs, que
 * `lib/replay/relief` teinte et coupe. Seuls le rendu et le plan se chargent en `HTMLImageElement`
 * (`MapBackgroundImageVariant`) ; le relief a ses propres fichiers (`reliefDataPath`).
 */

/** Un fond posé : `rendu`, `plan` (vue plan à hauteurs) ou `relief` (hauteurs teintées). */
export type MapBackgroundVariant = 'rendu' | 'plan' | 'relief'

/** Les fonds qui se chargent comme une image à décoder : le relief en est exclu. */
export type MapBackgroundImageVariant = Exclude<MapBackgroundVariant, 'relief'>

/** Les variantes, dans l'ordre où toutes les commandes les proposent (Plan, Rendu, Relief). */
export const MAP_BACKGROUND_VARIANTS: readonly MapBackgroundVariant[] = ['plan', 'rendu', 'relief']

/** Le plan est le défaut (décision du 2026-11-16, maquettes des réapparitions) ; une carte sans image plan sert le rendu. */
export const DEFAULT_MAP_BACKGROUND_VARIANT: MapBackgroundVariant = 'plan'

/** La valeur de `variante=` que l'API accepte pour chacune des deux images non rendues. */
const VARIANT_QUERY: Record<Exclude<MapBackgroundVariant, 'rendu'>, string> = {
  plan: 'variante=plan',
  relief: 'variante=relief',
}

/** Le suffixe de l'adresse du fichier de données du relief, sœur de l'adresse de l'image. */
const RELIEF_DATA_SUFFIX = '/background.relief.json'

/**
 * normalizeBackgroundStyle lit une valeur persistée de façon tolérante : tout ce qui n'est pas une
 * variante connue (absent, écrit par une autre version) vaut le défaut (le plan).
 */
export function normalizeBackgroundStyle(raw: unknown): MapBackgroundVariant {
  return raw === 'rendu' || raw === 'plan' || raw === 'relief' ? raw : DEFAULT_MAP_BACKGROUND_VARIANT
}

/**
 * Ce que le fond offre en fait de relief : le sidecar en porte-t-il un (`reliefImage` et `reliefData`),
 * et sa lecture a-t-elle échoué (`ReliefError`) ? Un relief qui a échoué n'est plus offert : le plan le remplace.
 */
export interface ReliefOffer {
  available: boolean
  failed: boolean
}

/** Un relief est utilisable quand le sidecar le porte et qu'il ne s'est pas révélé illisible. */
export function reliefUsable(relief: ReliefOffer): boolean {
  return relief.available && !relief.failed
}

/** L'image plan d'un fond : le nom de fichier du sidecar (`planImage`), ou un booléen quand l'appelant sait déjà si elle existe. */
export type PlanImageOffer = string | boolean | null | undefined

/**
 * resolveBackgroundVariant dit quel fond poser. Le relief quand l'utilisateur le veut ET qu'il est
 * offert (`relief`, absent = jamais offert) ; le plan quand il est voulu, ou quand le relief voulu ne
 * peut pas l'être, ET que le fond porte une image plan (`planImage` du sidecar) ; le rendu sinon.
 * Sans le troisième argument la réponse est toujours une image (plan ou rendu) : c'est la forme
 * des vues qui n'ont pas encore de relief.
 */
export function resolveBackgroundVariant(
  style: MapBackgroundVariant,
  planImage: PlanImageOffer,
): MapBackgroundImageVariant
export function resolveBackgroundVariant(
  style: MapBackgroundVariant,
  planImage: PlanImageOffer,
  relief: ReliefOffer,
): MapBackgroundVariant
export function resolveBackgroundVariant(
  style: MapBackgroundVariant,
  planImage: PlanImageOffer,
  relief: ReliefOffer = { available: false, failed: false },
): MapBackgroundVariant {
  if (style === 'relief' && reliefUsable(relief)) return 'relief'
  return (style === 'plan' || style === 'relief') && !!planImage ? 'plan' : 'rendu'
}

/**
 * baseImageVariant dit quelle IMAGE charger pour un fond voulu : le rendu ou le plan, tels quels ; pour
 * le relief, l'image plan quand le fond en porte une, sinon le rendu. C'est ce qui est posé tant que
 * le relief se charge, et ce qui le remplace s'il échoue ; l'image plan sert aussi de masque aux zones.
 */
export function baseImageVariant(
  wanted: MapBackgroundVariant,
  planImage: PlanImageOffer,
): MapBackgroundImageVariant {
  if (wanted === 'relief') return planImage ? 'plan' : 'rendu'
  return wanted
}

/**
 * isWallPlanDrawn dit si le calque « Plan des murs » se trace : l'interrupteur est allumé ET l'image
 * posée n'est pas le plan, qui porte déjà les murs (les tracer en double les épaissirait). Le relief ne
 * porte pas les murs : le calque s'y trace, indépendamment des traits du relief. Le calque reste un
 * réglage de l'utilisateur : posé de nouveau sur le rendu ou le relief, il reparaît tel qu'il l'a laissé.
 */
export function isWallPlanDrawn(showWallPlan: boolean, variante: MapBackgroundVariant): boolean {
  return showWallPlan && variante !== 'plan'
}

/**
 * backgroundImagePath ajoute la variante à l'adresse de l'image (le rendu garde l'adresse nue). Pour
 * `relief`, l'adresse est celle de l'image des hauteurs, pas celle d'un fond à poser tel quel.
 */
export function backgroundImagePath(path: string, variante: MapBackgroundVariant): string {
  return variante === 'rendu' ? path : `${path}?${VARIANT_QUERY[variante]}`
}

/** reliefDataPath donne l'adresse des données du relief (JSON en clair) à partir de celle de l'image du fond. */
export function reliefDataPath(imagePath: string): string {
  return imagePath.replace(/\/background\.png$/, RELIEF_DATA_SUFFIX)
}

/**
 * loadedBackground choisit ce qui est posé parmi les images chargées et dit quelle variante c'est :
 * le plan quand il est demandé ET chargé, sinon le rendu (repli ou rendu demandé), sinon rien — la
 * variante vaut alors celle qu'on a demandée. Les murs se décident sur cette variante RÉELLE.
 */
export function loadedBackground<T>(
  demandee: MapBackgroundImageVariant,
  plan: T | undefined,
  rendu: T | undefined,
): { image: T | null; variante: MapBackgroundImageVariant } {
  if (demandee === 'plan' && plan !== undefined) return { image: plan, variante: 'plan' }
  if (rendu !== undefined) return { image: rendu, variante: 'rendu' }
  return { image: null, variante: demandee }
}
