/**
 * cotesMenuCamera.ts — LES COTES DU MENU DE LA CAMÉRA (`ui/ReplayCameraMenu.tsx`), en pixels : le bouton de
 * la barre compacte et son panneau à deux listes. Le composant lit ce tableau et n'écrit aucun nombre.
 *
 * Elles s'appliquent en `style` : une classe Tailwind arbitraire ne naît que si elle est écrite en clair.
 */
export interface CotesMenuCamera {
  /** Le bouton icône de la barre : même diamètre que le bouton du son, auquel il est accolé. */
  declencheurPx: number
  /** Écart entre le haut de la barre et le bas du panneau, qui s'ouvre vers le haut. */
  panneauEcartPx: number
  /** Largeur de chaque liste (« Suivre », « Point de vue »), et écart entre les deux. */
  listeLargeurPx: number
  listeEcartPx: number
  /** Hauteur maximale du panneau : au-delà, il défile (un camp de dix joueurs et deux listes côte à côte). */
  panneauHauteurMaxPx: number
  /** Hauteur d'une entrée de liste. */
  entreeHauteurPx: number
}

export const COTES_MENU_CAMERA: CotesMenuCamera = Object.freeze({
  declencheurPx: 34,
  panneauEcartPx: 6,
  listeLargeurPx: 176,
  listeEcartPx: 8,
  panneauHauteurMaxPx: 320,
  entreeHauteurPx: 28,
})
