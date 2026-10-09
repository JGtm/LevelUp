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
 * `readStoredExactComposition` / `writeStoredExactComposition` sont les seuls
 * endroits qui touchent à cette clé, et `exactCompositionDefault` le seul qui
 * traduit la valeur stockée en état initial : un décochage explicite reste respecté, tout le
 * reste retombe sur le défaut coché (pas de clé versionnée — une bascule de défaut ne réécrit
 * pas un choix délibéré).
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

/** Clé de stockage de l'option pour un joueur. */
function storageKey(playerSlug: string): string {
  return `squad-exact-composition-${playerSlug}`
}

/** Option stockée pour ce joueur ; le défaut coché si rien n'est lisible. */
export function readStoredExactComposition(playerSlug: string): boolean {
  try {
    return exactCompositionDefault(localStorage.getItem(storageKey(playerSlug)))
  } catch {
    return exactCompositionDefault(null)
  }
}

/** Écrit l'option de ce joueur ; un stockage indisponible n'interrompt rien. */
export function writeStoredExactComposition(playerSlug: string, value: boolean): void {
  try {
    localStorage.setItem(storageKey(playerSlug), String(value))
  } catch {
    /* stockage indisponible : l'option vit alors en mémoire */
  }
}
