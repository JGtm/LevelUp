/**
 * hoverLayers.ts — LE GESTE DE SURVOL, DISTRIBUÉ AUX CALQUES QUI L'ATTENDENT.
 *
 * EXTRACTION IMPOSÉE PAR LE CLIQUET DE TAILLE DU CANVAS (`placementFamily.guard.test.ts`) : le
 * fichier était PILE à son plafond et le lot de l'export hors temps réel devait y brancher une
 * commande de plus. La règle du dépôt est d'extraire plutôt que de relever le plafond.
 *
 * LA DÉCOUPE ÉTAIT DÉJÀ ÉCRITE DANS LE CANVAS : « TROIS CALQUES SURVOLABLES SUR UNE SEULE
 * BALISE (poses, emplacements d'arme, drapeaux) : chacun rejoue le survol sur SA donnée, le
 * canvas passe le geste ». Ce fichier EST cette phrase. Le canvas n'avait rien à décider là —
 * il recopiait le même geste trois fois, deux fois de suite.
 *
 * PAS DE REACT ICI, ET C'EST DÉLIBÉRÉ. Un hook aurait invité à mémoïser, donc à déclarer des
 * dépendances sur un tableau reconstruit à chaque rendu — une échappatoire de lint pour un gain
 * nul : ces deux fonctions ne franchissent aucune frontière de mémoïsation, elles se posent sur
 * une balise `<canvas>` du même rendu. Une fonction pure se teste aussi sans monter un composant.
 *
 * IL N'Y A AUCUNE LOGIQUE DE SURVOL ICI : elle vit dans le hook de chaque calque
 * (`usePlacementHover`, `useReplayWeaponPads`, `useReplayFlagCarries`). Ce qui se centralise est
 * la DISTRIBUTION — et le fait qu'un quatrième calque survolable s'ajoutera en UNE ligne, à un
 * seul endroit, plutôt qu'en deux endroits du canvas.
 */
import type { PointerEvent } from 'react'

/** Ce qu'un calque doit savoir faire pour recevoir le geste. */
export interface HoverLayer {
  onPointerMove: (e: PointerEvent<HTMLCanvasElement>) => void
  onPointerLeave: () => void
}

/** Les deux gestionnaires à poser tels quels sur la balise `<canvas>`. */
export interface HoverHandlers {
  onPointerMove: (e: PointerEvent<HTMLCanvasElement>) => void
  onPointerLeave: () => void
}

/**
 * hoverHandlers rend les gestionnaires qui servent TOUS les calques donnés.
 *
 * AUCUN CALQUE N'INTERROMPT LES SUIVANTS : chacun décide seul s'il est concerné. Un calque qui
 * « consommerait » le geste rendrait le survol dépendant de l'ordre de déclaration — ce que le
 * canvas ne faisait pas, et qu'il ne faut surtout pas introduire en factorisant.
 */
export function hoverHandlers(...layers: readonly HoverLayer[]): HoverHandlers {
  return {
    onPointerMove: (e) => {
      for (const l of layers) l.onPointerMove(e)
    },
    onPointerLeave: () => {
      for (const l of layers) l.onPointerLeave()
    },
  }
}
