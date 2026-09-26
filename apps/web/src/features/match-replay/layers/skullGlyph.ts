/**
 * skullGlyph.ts — LE GLYPHE DU CRÂNE d'Oddball, tracé UNE seule fois pour ses DEUX calques.
 *
 * POURQUOI CENTRALISÉ (2026-08-28). Le crâne se dessinait en DEUX exemplaires — un disque nu
 * recopié dans `objectiveObjectsLayer` (le crâne LIBRE, au sol) et dans `skullCarrierLayer` (le
 * crâne SUR SON PORTEUR). La doctrine « le MÊME disque, pour qu'on reconnaisse le même objet » ne
 * tenait alors que par vigilance ; une seule fonction la rend vraie par construction, et c'est
 * elle qu'il faut pour que les deux crânes changent d'habillage ENSEMBLE.
 *
 * SON HABILLAGE EST CELUI DU DRAPEAU (retour utilisateur du 2026-08-28 : « prendre son icône et
 * le mettre comme on a mis le drapeau — taille et contour »). Le drapeau n'est PAS une vignette du
 * jeu (cf. `flagCarriesLayer`, en-tête) : c'est un glyphe canvas posé sur un LISERÉ à l'encre du
 * FOND, qui lui rend un bord franc sur un fond de carte photographique, dans les deux thèmes. Le
 * crâne reprend ce liseré, à la MÊME cote (`FLAG_OUTLINE_PAD`), et grandit pour peser autant que
 * l'aile du drapeau — le disque à r=5 se perdait sur un fond chargé.
 *
 * LA FORME RESTE UNE BOULE, distincte de la hampe + fanion du drapeau : deux objets de mode du
 * même glyphe seraient indiscernables sur une carte où les deux paraissent, et le rejeu se lit
 * aussi en niveaux de gris à l'impression. Elle porte DEUX ORBITES creusées à l'encre du fond pour
 * se lire « crâne » et non « pastille » — le minimum qui évoque la vignette d'Oddball sans exiger
 * un dessin fin illisible à cette taille.
 *
 * PAS D'ICÔNE DU JEU ICI, ET C'EST UNE LIMITE DE PÉRIMÈTRE, PAS UN CHOIX GRAPHIQUE. Le crâne a
 * bien une vignette extraite (`jeu/contour-25.png`, `nom_jeu: skull`, tag weap `0017592c`), mais
 * son URL n'est PAS cuite dans le document du rejeu — contrairement aux vignettes d'armes
 * (`weaponLabels[id].img`) — et le client ne DEVINE jamais un index d'atlas (le piège
 * `killfeed-NN`, documenté au garde-rail du son : un index qui bouge d'une saison casse la
 * jointure EN SILENCE). La brancher proprement demande un lot Go qui publie l'URL dans le
 * document ; d'ici là, ce glyphe canvas tient le rôle, exactement comme pour le drapeau.
 */
import type { XY } from '../../../lib/replay/replayLogic'

/**
 * SKULL_GLYPH_RADIUS — rayon du disque (le crâne), en pixels canvas.
 *
 * AGRANDI le 2026-08-28 (r=5 → r=7) pour peser autant que le drapeau (aile ~9,4 px de large) :
 * un disque nu de rayon 5 se confondait avec les repères voisins sur un fond de carte chargé.
 * PLUS PETIT QUE LE MARQUEUR D'UN JOUEUR quand même — le crâne est un objet, pas un acteur.
 */
export const SKULL_GLYPH_RADIUS = 7

/** Débord du liseré au-delà du disque, en pixels — MÊME COTE que le drapeau (`FLAG_OUTLINE_PAD`). */
const SKULL_LISERE_PAD = 1.6
/**
 * Largeur du trait de liseré. Le trait de canvas est CENTRÉ sur son chemin : sa moitié
 * intérieure (`SKULL_LISERE_PAD` vers le centre) est recouverte par le remplissage du disque,
 * il ne reste que le débord extérieur — un bord franc, la technique des vignettes de socle et
 * du drapeau.
 */
const SKULL_LISERE_WIDTH = 2 * SKULL_LISERE_PAD

/** Une orbite : son rayon et son décalage depuis le centre. Deux points qui font lire « crâne ». */
const SKULL_EYE_RADIUS = 1.4
const SKULL_EYE_DX = 2.7
const SKULL_EYE_DY = 0.6

/** Ce que le tracé d'un crâne a besoin de savoir (règle des 5 paramètres). */
export interface SkullGlyphPaint {
  /** Encre du disque : le neutre du thème, résolu par l'appelant (jamais une couleur en dur). */
  ink: string
  /**
   * Encre du FOND : le liseré posé SOUS le disque, ET les orbites creusées dedans — c'est celle
   * du drapeau (`markInk.outline` = `--background`), pas une arête de mise en page.
   */
  outline: string
  alpha: number
}

/**
 * drawSkullGlyph pose le crâne à `center` : un liseré à l'encre du fond, un disque à l'encre
 * neutre par-dessus, deux orbites creusées à l'encre du fond.
 *
 * LE POINT SERVI EST LE CENTRE, jamais un point décalé : c'est l'appelant qui applique son propre
 * décalage — le crâne libre se pose sur sa position publiée, le crâne porté au-DESSUS du marqueur
 * de son porteur. Le glyphe, lui, est le même.
 */
export function drawSkullGlyph(ctx: CanvasRenderingContext2D, center: XY, paint: SkullGlyphPaint): void {
  ctx.globalAlpha = paint.alpha
  // LE LISERÉ D'ABORD, à l'encre du fond : le remplissage recouvre ensuite sa moitié intérieure,
  // il ne reste que le débord — le bord franc que le drapeau a gagné le 2026-08-27.
  ctx.beginPath()
  ctx.arc(center.x, center.y, SKULL_GLYPH_RADIUS, 0, Math.PI * 2)
  ctx.lineWidth = SKULL_LISERE_WIDTH
  ctx.strokeStyle = paint.outline
  ctx.stroke()
  // LE DISQUE, à l'encre neutre : il cache la moitié intérieure du liseré.
  ctx.beginPath()
  ctx.arc(center.x, center.y, SKULL_GLYPH_RADIUS, 0, Math.PI * 2)
  ctx.fillStyle = paint.ink
  ctx.fill()
  // LES DEUX ORBITES, creusées à l'encre du fond : c'est ce qui fait lire « crâne » et non
  // « pastille ». À l'encre du fond, elles contrastent avec le neutre du disque dans les deux
  // thèmes, comme le liseré.
  ctx.fillStyle = paint.outline
  ctx.beginPath()
  ctx.arc(center.x - SKULL_EYE_DX, center.y - SKULL_EYE_DY, SKULL_EYE_RADIUS, 0, Math.PI * 2)
  ctx.fill()
  ctx.beginPath()
  ctx.arc(center.x + SKULL_EYE_DX, center.y - SKULL_EYE_DY, SKULL_EYE_RADIUS, 0, Math.PI * 2)
  ctx.fill()
}
