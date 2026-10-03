/**
 * exportFormats.ts — LES FORMATS DE SORTIE DE L'EXPORT VIDEO, et la geometrie qui en decoule.
 *
 * # POURQUOI UN CATALOGUE FERME (plan « formats video standard », 2026-09-16, D2)
 *
 * L'export encodait la toile TELLE QU'AFFICHEE : largeur de la fenetre, hauteur offerte par
 * l'ecran, le tout multiplie par la densite de pixels. Deux exports du meme match sortaient donc
 * en dimensions ET en proportions differentes selon la machine — inutilisable au montage comme
 * au partage. Le fichier ne depend plus que d'un choix : 1920x1080 (defaut) ou 1280x720, en 16:9.
 *
 * # LA MISE EN PAGE EST LA MEME POUR LES DEUX FORMATS
 *
 * La carte et les surimpressions se mettent en page dans un cadre LOGIQUE de 960x540 px CSS
 * (`EXPORT_LAYOUT`), puis ce cadre est rendu a la densite qui donne le format demande :
 * x2 pour 1080p, x4/3 pour 720p. Consequence voulue : un clip 720p montre EXACTEMENT la meme
 * composition qu'un clip 1080p — memes cadrage, memes tailles relatives de texte et de pions.
 *
 * 540 de haut n'est pas arbitraire : c'est le milieu de la plage de hauteurs de la toile a
 * l'ecran (360 a 720, cf. `useReplayView`), donc des etiquettes et des pions dessines en px
 * CSS gardent a l'export le poids qu'ils ont a l'ecran. Et a x2, c'est le rendu mesure net du
 * 2026-08-28 (sous-echantillonnage chroma de H.264 : la toile doublee ne perd rien) — la mesure
 * avait ete faite a 480 x2, la cible est desormais 540 x2.
 *
 * # POURQUOI LE 720p EST RENDU DIRECTEMENT, SANS RENDU 1080p REDUIT
 *
 * Decision ecrite par raisonnement, NON MESUREE dans un navigateur (a verifier au gate visuel).
 *
 * La perte de nettete mesuree le 2026-08-28 vient du sous-echantillonnage chroma 4:2:0 : l'encodeur
 * garde un echantillon de couleur pour 2x2 pixels de l'image QU'IL ENCODE. Cette perte est donc
 * une propriete de la grille de SORTIE (1280x720), pas du chemin par lequel on l'a remplie.
 * Rendre en 1920x1080 puis reduire ne rajoute aucun pixel de couleur au fichier : la reduction
 * moyenne les details fins AVANT l'encodeur (un flou de luminance en plus), et l'encodeur
 * sous-echantillonne ensuite la meme grille. Le rendu direct a x4/3, lui, rasterise traits et
 * texte a leur taille finale avec l'antialiasing analytique du canvas — l'equivalent d'une
 * reduction par moyenne de surface, sans le flou du reechantillonnage et avec le lissage des
 * polices a la bonne taille.
 *
 * Il evite aussi deux couts sans benefice : une copie pleine image par image dans une toile hors
 * ecran, et le recours a une mise a l'echelle IMPLICITE de `VideoEncoder` (une image plus grande
 * que la configuration), que la specification WebCodecs ne garantit pas d'un navigateur a
 * l'autre. Ce qui reste perdu en 720p — le demi-pixel de couleur sur un trait d'un px CSS rendu
 * a 1,33 px — est le prix du format choisi, et c'est pour cela que 1080p est le defaut.
 */

/** Les identifiants PERSISTES (cf. `replayPreferences.ts`) : ne jamais les renommer. */
export type ExportFormatId = '1080p' | '720p'

export interface ExportFormat {
  id: ExportFormatId
  /** Dimensions du FICHIER, en pixels. Paires, 16:9 exact. */
  width: number
  height: number
}

/** Le catalogue, dans l'ordre ou le dialogue le propose. */
export const EXPORT_FORMATS: readonly ExportFormat[] = [
  { id: '1080p', width: 1920, height: 1080 },
  { id: '720p', width: 1280, height: 720 },
]

export const EXPORT_FORMAT_IDS: readonly ExportFormatId[] = EXPORT_FORMATS.map((f) => f.id)

/** Le format d'un export quand rien n'est choisi, ou quand le choix lu est invalide (D4). */
export const DEFAULT_EXPORT_FORMAT_ID: ExportFormatId = '1080p'

/** Le cadre LOGIQUE de mise en page de l'export, en px CSS (cf. l'en-tete). */
export const EXPORT_LAYOUT = { width: 960, height: 540 } as const

/**
 * LE CADRAGE DU CLIP (decision D6, 2026-09-16), quand la carte est zoomee a l'ecran :
 * - `whole` (defaut) : la carte entiere, rendue a 1x — le zoom de l'utilisateur n'est PAS
 *   modifie, il est simplement ignore le temps de l'export ;
 * - `current` : le palier et le centre de l'ecran, elargis au cadre 16:9.
 * Contextuel, donc JAMAIS memorise : un zoom est un geste du moment, pas une habitude.
 */
export type ExportFraming = 'whole' | 'current'

/** Le cadrage d'un export quand rien n'est choisi. */
export const DEFAULT_EXPORT_FRAMING: ExportFraming = 'whole'

/**
 * La geometrie d'un export : le cadre logique ou la carte se met en page, la densite a
 * laquelle il est rendu (`width * pixelRatio` et `height * pixelRatio` donnent le format), et
 * le cadrage de la carte dans ce cadre.
 */
export interface ExportLayout {
  format: ExportFormat
  width: number
  height: number
  pixelRatio: number
  framing: ExportFraming
}

/** exportFormatOf — le format d'un identifiant ; inconnu, absent ou invalide -> le defaut. */
export function exportFormatOf(id: string | null | undefined): ExportFormat {
  return (
    EXPORT_FORMATS.find((f) => f.id === id) ??
    (EXPORT_FORMATS.find((f) => f.id === DEFAULT_EXPORT_FORMAT_ID) as ExportFormat)
  )
}

/**
 * exportLayoutFor — le cadre logique et la densite d'un format.
 *
 * INDEPENDANTE DU `devicePixelRatio` PAR CONSTRUCTION : elle ne lit rien de l'ecran. C'est ce
 * qui fait que deux machines differentes sortent le meme fichier.
 */
export function exportLayoutFor(
  format: ExportFormat,
  framing: ExportFraming = DEFAULT_EXPORT_FRAMING,
): ExportLayout {
  return {
    format,
    framing,
    width: EXPORT_LAYOUT.width,
    height: EXPORT_LAYOUT.height,
    pixelRatio: format.height / EXPORT_LAYOUT.height,
  }
}
