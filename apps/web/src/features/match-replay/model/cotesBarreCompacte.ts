/**
 * cotesBarreCompacte.ts — LES COTES DE LA BARRE DE LECTURE D'UNE LIGNE (`ui/ReplayTransportCompact.tsx`)
 * et de sa frise (`ui/ReplayTimelineCompact.tsx`), en pixels. Les composants lisent ce tableau et
 * n'écrivent aucun nombre.
 *
 * Elles viennent de la maquette de la rangée (classes `.tp`, `.play`, `.ibtn`, `.time`, `.scrub`, `.ticks`,
 * `.rail`, `.dom`, `.knob`, `.vsep`) : la frise compte 30 px de haut, et ses quatre couches
 * (repères, rail, bande de dominance, curseur) y ont chacune leur ligne — le centre du curseur
 * tombe sur celui du rail.
 *
 * Le bouton « Plan tourné » (`planTourne*`, classe `.sel.on`) : capsule de 30 px, marge
 * intérieure de 10 px, rayon de 8 px, icône de 14 px, bordure d'1 px à moitié de la couleur de suivi.
 *
 * Toutes s'appliquent en `style` ou par une variable CSS (`friseVars`) : une classe Tailwind
 * arbitraire ne naît que si elle est écrite en clair, une interpolation ne produit aucune règle.
 */
import { THUMB_PX } from './replayTimelineTracksLogic'

export interface CotesBarreCompacte {
  /** Hauteur de la barre, écart entre deux commandes, marge latérale du socle. */
  barreHauteurPx: number
  ecartPx: number
  margeXPx: number
  /** Barre sous le bloc du plan, sur le fond de la page : sa marge latérale, et l'écart qui la sépare du bloc. */
  margeXHorsBlocPx: number
  blocBarreEcartPx: number
  /** Le bouton de lecture, rond et plein. */
  lecturePx: number
  /** Une commande ordinaire : carré de `boutonPx`, rayon `boutonRayonPx`, sans bordure ni fond au repos (le fond ne vient qu'au survol). */
  boutonPx: number
  boutonRayonPx: number
  /** Le séparateur vertical entre deux groupes : hauteur et marge de chaque côté. */
  separateurHauteurPx: number
  separateurMargePx: number
  /** Le temps « instant / durée » : largeur minimale (la frise ne bouge pas quand un chiffre change) et marge. */
  tempsLargeurMinPx: number
  tempsMargePx: number
  /** La frise : hauteur, marge de chaque côté, largeur minimale (sous elle, la barre passe sur deux lignes). */
  friseHauteurPx: number
  friseMargePx: number
  friseLargeurMinPx: number
  /** Sur deux lignes : hauteur de la ligne de la frise, seule en haut, et marge verticale de chaque commande de la seconde ligne. */
  deuxLignesFriseHauteurPx: number
  deuxLignesCommandeMargeYPx: number
  /** La forme fine (Plein cadre, en lecture) : hauteur de la barre, et échelle des commandes (la barre pleine, réduite). */
  fineHauteurPx: number
  fineEchelle: number
  /** Combien de temps la barre garde sa forme pleine après le dernier geste (survol quitté, touche), en millisecondes. */
  fineRetenuMs: number
  /** Les repères d'éliminations, au-dessus du rail : bande, largeur et hauteur d'un repère. */
  reperesBandePx: number
  repereLargeurPx: number
  repereHauteurPx: number
  /** Le rail : distance au haut de la frise et épaisseur. */
  railHautPx: number
  railEpaisseurPx: number
  /** La bande de dominance, dessous : distance au haut de la frise et épaisseur. */
  dominanceHautPx: number
  dominanceEpaisseurPx: number
  /** Le curseur : diamètre visible, et halo autour. */
  curseurPx: number
  curseurHaloPx: number
  /** Le bouton « Plan tourné » : hauteur, marge intérieure de chaque côté, rayon, écart icône-libellé, côté de l'icône, bordure (px) et part de couleur de suivi de la bordure (%). */
  planTourneHauteurPx: number
  planTourneMargeXPx: number
  planTourneRayonPx: number
  planTourneEcartPx: number
  planTourneIconePx: number
  planTourneBordurePx: number
  planTourneBordurePct: number
  /** Un repère de média sur la bande des repères : largeur d'une photo et largeur minimale d'un clip. */
  mediaPhotoPx: number
  mediaClipMinPx: number
}

export const COTES_BARRE_COMPACTE: CotesBarreCompacte = Object.freeze({
  barreHauteurPx: 56,
  ecartPx: 6,
  margeXPx: 12,
  margeXHorsBlocPx: 2,
  blocBarreEcartPx: 6,
  lecturePx: 36,
  boutonPx: 30,
  boutonRayonPx: 8,
  separateurHauteurPx: 20,
  separateurMargePx: 4,
  tempsLargeurMinPx: 84,
  tempsMargePx: 4,
  friseHauteurPx: 30,
  friseMargePx: 10,
  friseLargeurMinPx: 240,
  deuxLignesFriseHauteurPx: 38,
  deuxLignesCommandeMargeYPx: 1,
  fineHauteurPx: 32,
  fineEchelle: 0.8,
  fineRetenuMs: 1500,
  reperesBandePx: 9,
  repereLargeurPx: 2,
  repereHauteurPx: 8,
  railHautPx: 13,
  railEpaisseurPx: 4,
  dominanceHautPx: 23,
  dominanceEpaisseurPx: 3,
  curseurPx: 12,
  curseurHaloPx: 4,
  planTourneHauteurPx: 30,
  planTourneMargeXPx: 10,
  planTourneRayonPx: 8,
  planTourneEcartPx: 6,
  planTourneIconePx: 14,
  planTourneBordurePx: 1,
  planTourneBordurePct: 50,
  mediaPhotoPx: 10,
  mediaClipMinPx: 6,
})

/**
 * Les variables CSS de la frise : les classes du champ (`[&::-webkit-slider-thumb]:h-[var(--bc-boite)]`)
 * les lisent, ce qui évite d'y écrire un nombre. La boîte du curseur est la largeur que le navigateur
 * lui réserve (`THUMB_PX`), celle que suppose la géométrie des repères ; le curseur visible
 * (`curseurPx`) est plus petit, et l'écart est rendu en bordure transparente (`--bc-bord`).
 */
export function friseVars(c: CotesBarreCompacte = COTES_BARRE_COMPACTE): Record<string, string> {
  return {
    '--bc-frise': `${c.friseHauteurPx}px`,
    '--bc-boite': `${THUMB_PX}px`,
    '--bc-bord': `${Math.max(0, (THUMB_PX - c.curseurPx) / 2)}px`,
    '--bc-halo': `${c.curseurHaloPx}px`,
    // Le curseur WebKit se pose en haut de la piste du champ : il se centre par une marge haute.
    '--bc-haut': `${(c.friseHauteurPx - THUMB_PX) / 2}px`,
  }
}
