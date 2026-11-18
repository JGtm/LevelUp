/**
 * relief.i18n.ts — LES TEXTES DU FOND « RELIEF » ET DE LA COUPE PAR ÉTAGE du panneau de la vue du
 * rejeu : le contrat (`ReliefText`) et ses deux langues. Le fond lui-même, l'interrupteur des traits du
 * relief et le groupe « Étage » (pastilles, curseur, état d'une carte sans maillage).
 *
 * UN SOUS-DICTIONNAIRE À PART, comme `lives.i18n.ts` : `fr.ts` et `en.ts` sont au plafond de taille
 * du dépôt, et ces textes forment un tout. Servis par `REPLAY_TEXT[locale].relief` ; la parité FR/EN est
 * tenue par le typage (`satisfies ReliefText` dans chaque langue).
 *
 * LES ÉTAGES SONT DES RANGS, JAMAIS DES HAUTEURS (décision D-14) : une unité du jeu vaut environ 3 m,
 * un « +2,2 m » serait faux. Le rang 0 est « Bas », les suivants « Étage 1 », « Étage 2 »… Le nombre
 * d'anneaux que porte un pion à cet étage (le langage d'étage des marqueurs) se dit à l'infobulle.
 */

export interface ReliefText {
  /** Le fond « Relief » : le terrain vu de dessus, en gris teinté (option du segmenté « Fond »). */
  background: string
  backgroundHint: string
  /** La part du relief tirée du maillage de navigation (D-16), ajoutée à l'infobulle du fond quand elle est connue. */
  backgroundMeshFmt: (percent: number) => string
  /** L'interrupteur « Traits du relief » : ruptures, courbes de niveau, liseré, retenu par fond. */
  lines: string
  linesHint: string
  /** Le groupe « Étage » : titre, (i), « Tout » et ses infobulles. */
  floorTitle: string
  floorInfo: string
  floorAll: string
  floorAllHint: string
  /** Le libellé d'un étage par son rang : 0 = « Bas », puis « Étage 1 »… */
  floorRankFmt: (rank: number) => string
  /** L'infobulle d'une pastille : le libellé de l'étage et le nombre d'anneaux d'un pion à cet étage. */
  floorChipHintFmt: (label: string, rings: number) => string
  /** Le curseur de hauteur : libellé visible et nom accessible. */
  floorHeight: string
  floorHeightAria: string
  /** Carte sans maillage de navigation : le bloc d'explication à la place de la commande d'étage (D-7). */
  floorNoMesh: string
}

export const FR_RELIEF = {
  background: 'Relief',
  backgroundHint:
    'Le terrain vu de dessus, en gris teinté : plus clair plus haut, ombré du nord-ouest, découpé à la zone jouable.',
  backgroundMeshFmt: (percent) => `Relief tiré du maillage de navigation sur ${percent} % de la carte.`,
  lines: 'Traits du relief',
  linesHint:
    'Ruptures de hauteur, courbes de niveau sur les pentes et liseré de la zone jouable, nets à tout zoom. Allumés d’office sur le fond Relief, retenus pour chaque fond.',
  floorTitle: 'Étage',
  floorInfo:
    'Coupe du terrain par hauteur : l’étage choisi s’affiche en plein, les étages du dessous pâles, ceux du dessus presque effacés. Aucun joueur n’est masqué.',
  floorAll: 'Tout',
  floorAllHint: 'Aucune coupe : tous les étages en plein.',
  floorRankFmt: (rank) => (rank === 0 ? 'Bas' : `Étage ${rank}`),
  floorChipHintFmt: (label, rings) =>
    `${label} : ${rings === 0 ? 'aucun anneau' : `${rings} anneau${rings > 1 ? 'x' : ''}`}`,
  floorHeight: 'Hauteur',
  floorHeightAria: 'Hauteur de la coupe',
  floorNoMesh: 'Carte sans maillage de navigation : le relief est affiché en entier, sans coupe par étage.',
} satisfies ReliefText

export const EN_RELIEF = {
  background: 'Relief',
  backgroundHint:
    'The terrain seen from above, in tinted grey: lighter where higher, shaded from the north-west, cut to the playable area.',
  backgroundMeshFmt: (percent) => `Relief taken from the navigation mesh on ${percent}% of the map.`,
  lines: 'Relief lines',
  linesHint:
    'Height breaks, contour lines on slopes and an outline of the playable area, sharp at any zoom. On by default with the Relief background, remembered for each background.',
  floorTitle: 'Floor',
  floorInfo:
    'Terrain cut by height: the chosen floor is drawn in full, floors below pale, floors above almost erased. No player is hidden.',
  floorAll: 'All',
  floorAllHint: 'No cut: every floor drawn in full.',
  floorRankFmt: (rank) => (rank === 0 ? 'Low' : `Floor ${rank}`),
  floorChipHintFmt: (label, rings) => `${label}: ${rings === 0 ? 'no rings' : `${rings} ring${rings > 1 ? 's' : ''}`}`,
  floorHeight: 'Height',
  floorHeightAria: 'Cut height',
  floorNoMesh: 'Map without a navigation mesh: the relief is shown whole, with no floor cut.',
} satisfies ReliefText
