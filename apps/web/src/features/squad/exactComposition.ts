/**
 * exactComposition — défaut de l'option « composition stricte » (page Escouade).
 *
 * L'option restreint la population aux matchs joués avec EXACTEMENT la
 * composition sélectionnée (joueur principal + coéquipiers cochés). Elle est
 * cochée par défaut : la lecture attendue d'une page Escouade est « nous, cette
 * équipe-là », pas « ces matchs commencés ensemble avec, parfois, un cinquième
 * joueur connu dans le lobby ».
 *
 * Le choix de l'utilisateur est persisté par joueur en localStorage
 * (`squad-exact-composition-{playerSlug}`), sous forme de `String(boolean)`.
 * Cette fonction est le seul endroit qui traduit la valeur stockée en état
 * initial : un décochage explicite reste respecté, tout le reste retombe sur le
 * défaut coché (pas de clé versionnée — une bascule de défaut ne réécrit pas un
 * choix délibéré).
 */

/**
 * État initial de l'option « composition stricte » à partir de la valeur
 * localStorage brute (`null` quand la clé n'existe pas).
 */
export function exactCompositionDefault(stored: string | null): boolean {
  // Seul un « false » explicitement stocké (décochage délibéré) désactive
  // l'option ; l'absence de clé (premier passage, autre navigateur) comme une
  // valeur illisible retombent sur le défaut coché.
  return stored !== 'false'
}
