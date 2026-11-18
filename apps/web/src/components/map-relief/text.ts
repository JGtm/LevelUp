/**
 * Textes des commandes du fond Relief des vues 2D sans panneau « Vue » (onglet Tactique, « Occupation du
 * terrain ») — dictionnaire partagé par ces deux vues : chaque langue de `config/locales.toml` figure ici
 * (parité par typage). Le rejeu a le sien (`features/match-replay/i18n/relief.i18n.ts`), qui y ajoute le langage des
 * anneaux des marqueurs.
 *
 * LES ÉTAGES SONT DES RANGS, JAMAIS DES HAUTEURS (décision D-14) : une unité du jeu vaut environ 3 m. Le rang 0 est
 * « Bas », les suivants « Étage 1 », « Étage 2 »…
 */
import type { Locale } from '@/lib/i18n/locale'

export interface ReliefViewText {
  /** La pilule « Fond » et ses trois valeurs. */
  backgroundLabel: string
  plan: string
  render: string
  relief: string
  /** Infobulle de la valeur « Relief ». */
  reliefHint: string
  /** La part du relief tirée du maillage de navigation (D-16), ajoutée à l'infobulle de « Relief » quand elle est connue. */
  reliefMeshFmt: (percent: number) => string
  /** Le bouton « Traits du relief » et son infobulle. */
  lines: string
  linesHint: string
  /** La pilule « Étage » : étiquette, infobulle de l'étiquette, « Tout » et son infobulle. */
  floorLabel: string
  floorInfo: string
  floorAll: string
  floorAllHint: string
  /** Le libellé d'un étage par son rang : 0 = « Bas », puis « Étage 1 »… */
  floorRankFmt: (rank: number) => string
  /** Nom accessible du curseur de hauteur. */
  floorHeightAria: string
}

export const RELIEF_VIEW_TEXT: Record<Locale, ReliefViewText> = {
  fr: {
    backgroundLabel: 'Fond',
    plan: 'Plan',
    render: 'Rendu',
    relief: 'Relief',
    reliefHint:
      'Le terrain vu de dessus, en gris teinté : plus clair plus haut, ombré du nord-ouest, découpé à la zone jouable.',
    reliefMeshFmt: (percent) => `Relief tiré du maillage de navigation sur ${percent} % de la carte.`,
    lines: 'Traits du relief',
    linesHint:
      'Ruptures de hauteur, courbes de niveau sur les pentes et liseré de la zone jouable, nets à tout zoom. Allumés d’office sur le fond Relief, retenus pour chaque fond.',
    floorLabel: 'Étage',
    floorInfo:
      'Coupe du terrain par hauteur : l’étage choisi s’affiche en plein, les étages du dessous pâles, ceux du dessus presque effacés.',
    floorAll: 'Tout',
    floorAllHint: 'Aucune coupe : tous les étages en plein.',
    floorRankFmt: (rank) => (rank === 0 ? 'Bas' : `Étage ${rank}`),
    floorHeightAria: 'Hauteur de la coupe',
  },
  en: {
    backgroundLabel: 'Background',
    plan: 'Plan',
    render: 'Render',
    relief: 'Relief',
    reliefHint:
      'The terrain seen from above, in tinted grey: lighter where higher, shaded from the north-west, cut to the playable area.',
    reliefMeshFmt: (percent) => `Relief taken from the navigation mesh on ${percent}% of the map.`,
    lines: 'Relief lines',
    linesHint:
      'Height breaks, contour lines on slopes and an outline of the playable area, sharp at any zoom. On by default with the Relief background, remembered for each background.',
    floorLabel: 'Floor',
    floorInfo:
      'Terrain cut by height: the chosen floor is drawn in full, floors below pale, floors above almost erased.',
    floorAll: 'All',
    floorAllHint: 'No cut: every floor drawn in full.',
    floorRankFmt: (rank) => (rank === 0 ? 'Low' : `Floor ${rank}`),
    floorHeightAria: 'Cut height',
  },
}
