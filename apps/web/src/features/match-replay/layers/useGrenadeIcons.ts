/**
 * useGrenadeIcons — LES VIGNETTES DE TYPE DE GRENADE, chargées et TEINTES une fois.
 *
 * POURQUOI TEINDRE, ET POURQUOI PAS À L'IMAGE. Ce que le dépôt sert sont des MASQUES du HUD
 * (blanc/gris + alpha) : posés tels quels, ils seraient blancs dans les deux thèmes. Le canvas,
 * lui, ne sait pas teindre un masque au moment du dessin — il faut un canvas hors écran par
 * vignette. Le faire soixante fois par seconde coûterait pour rien : l'encre ne change qu'avec
 * le thème, le document qu'au chargement.
 *
 * EXTRAIT DE `ReplayCanvas.tsx` LE 2026-08-18 (fusion des lots R2-P et R2-V) : le canvas porte
 * un seuil de taille (`max-lines` eslint, R5) que les deux lots ont rempli, et la
 * règle du seuil est d'extraire avant d'ajouter. Ce bloc part le plus proprement — il ne
 * connaît ni les réglages, ni le cadrage, ni l'image courante.
 *
 * IL PREND `redraw` ET NON `draw` : le chargement d'une image est asynchrone, et la scène doit
 * se repeindre quand elle arrive. `redraw` est la référence STABLE du canvas (elle lit la
 * version courante de `draw`), donc ce hook peut être appelé AVANT que `draw` n'existe — ce
 * qu'il faut, puisque la boucle de dessin lit la table qu'il rend.
 */
import { useEffect, useRef, type RefObject } from 'react'

import { tintedIconCanvas } from './replayDraw'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

/**
 * useGrenadeIcons rend la table `rang -> vignette teinte`, remplie au fil des chargements.
 *
 * Une table PAR RÉFÉRENCE, pas un état : la remplir ne doit pas re-rendre la page (le canvas la
 * lit pendant qu'il dessine). Un rang sans visuel reste absent — l'appelant garde alors son
 * tracé de repli, jamais la vignette d'une autre grenade.
 */
export function useGrenadeIcons(
  grenadeLabels: ReplayDocumentReady['grenadeLabels'],
  ink: string,
  redraw: () => void,
): RefObject<Map<number, HTMLCanvasElement>> {
  const iconsRef = useRef<Map<number, HTMLCanvasElement>>(new Map())
  useEffect(() => {
    // Une table NEUVE à chaque cuisson : garder l'ancienne servirait des vignettes teintes
    // au thème précédent le temps que les images se rechargent.
    const map = new Map<number, HTMLCanvasElement>()
    iconsRef.current = map
    grenadeLabels.forEach((lbl, rank) => {
      if (!lbl.img) return
      const im = new Image()
      im.onload = () => {
        map.set(rank, tintedIconCanvas(im, ink))
        redraw()
      }
      im.src = lbl.img
    })
  }, [grenadeLabels, ink, redraw])
  return iconsRef
}
