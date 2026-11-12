/**
 * useAnchoredPanel — LA POSITION D'UN PANNEAU FLOTTANT, ancrée sur le bouton qui l'ouvre.
 *
 * # POURQUOI UN PORTAIL EN POSITION FIXE
 *
 * Le panneau doit pouvoir dépasser la carte du rejeu (sur la colonne de droite, au-dessus du
 * fil ou des fiches) quand l'écran le permet. Or la carte porte `overflow-hidden` : aucun
 * `z-index` ne fait sortir un enfant d'un ancêtre qui découpe. Le panneau se rend donc en
 * PORTAIL sur `body`, en `position: fixed`, et se replace lui-même à partir du rectangle de son
 * déclencheur.
 *
 * # LES TROIS CONTRAINTES, DANS L'ORDRE OÙ ELLES SE RÉSOLVENT
 *
 * 1. BORD DROIT ALIGNÉ sur celui du bouton, puis ramené dans la fenêtre. Le bouton vit dans le
 *    coin haut droit de la carte : un panneau large aligné sur son bord gauche déborderait à
 *    droite. Le `clamp` le ramène — un panneau tronqué serait pire.
 * 2. IL OUVRE VERS LE BAS, sous le bouton, ancré par le HAUT (`top`).
 * 3. IL NE DÉPASSE JAMAIS L'ÉCRAN. `maxHeight` vaut ce qui reste sous le bouton ; au-delà, c'est
 *    le panneau qui défile, pas la page. Sans cette borne, un panneau plus haut que la fenêtre
 *    sortirait par le bas, là où rien ne le rattrape.
 *
 * # POURQUOI `scroll` EN CAPTURE
 *
 * Le déclencheur vit dans un conteneur qui défile (`<main>` du shell), pas dans la fenêtre.
 * Un écouteur `scroll` posé sur `window` sans capture ne verrait jamais ce défilement-là :
 * l'événement ne remonte pas. En capture, il passe par `window` à la descente — c'est le seul
 * moyen d'entendre TOUS les conteneurs sans les connaître.
 */
import { useLayoutEffect, useState, type RefObject } from 'react'

/** L'écart entre le panneau et son déclencheur. */
const GAP = 8
/** La marge minimale gardée contre chaque bord de la fenêtre. */
const EDGE = 8
/** En dessous, un panneau n'a plus rien à montrer : il défile déjà à l'intérieur. */
const MIN_HEIGHT = 160
/** Largeur supposée par le repli centré, avant que l'appelant n'impose la sienne. */
const PANEL_FALLBACK_WIDTH = 416

export interface AnchoredPosition {
  left: number
  top: number
  maxHeight: number
}

/**
 * Le repli : centré dans la fenêtre. Il sert au premier rendu et au cas où le déclencheur n'est
 * pas mesurable (panneau monté seul, bouton démonté sous un panneau ouvert). On ne rend jamais
 * nul : un panneau de réglages qui s'évapore laisserait un geste sans effet.
 */
function centeredFallback(width: number): AnchoredPosition {
  const w = typeof window === 'undefined' ? 0 : window.innerWidth
  const h = typeof window === 'undefined' ? 0 : window.innerHeight
  return {
    left: Math.max((w - width) / 2, EDGE),
    top: EDGE,
    maxHeight: Math.max(h - 2 * EDGE, MIN_HEIGHT),
  }
}

/**
 * La position d'un panneau de largeur `width` pour un déclencheur de rectangle `r`, dans une
 * fenêtre `viewport`. Pure : c'est ce qu'on mesure en test.
 */
export function anchoredPosition(
  r: Pick<DOMRect, 'right' | 'bottom'>,
  viewport: { width: number; height: number },
  width: number,
): AnchoredPosition {
  const maxLeft = Math.max(viewport.width - width - EDGE, EDGE)
  return {
    left: Math.min(Math.max(r.right - width, EDGE), maxLeft),
    top: Math.max(r.bottom + GAP, EDGE),
    maxHeight: Math.max(viewport.height - r.bottom - GAP - EDGE, MIN_HEIGHT),
  }
}

export function useAnchoredPanel(
  triggerRef: RefObject<HTMLElement | null> | undefined,
  width: number,
): AnchoredPosition {
  // L'ÉTAT DÉMARRE CENTRÉ, JAMAIS NUL. Un premier rendu nul ferait naître le panneau APRÈS le
  // montage, et tout ce qui l'attend au montage — le focus qui doit y entrer, notamment —
  // s'exécuterait sur un panneau qui n'existe pas encore. Le `useLayoutEffect` ci-dessous
  // corrige la position avant la peinture : le centrage n'est donc jamais vu.
  const [pos, setPos] = useState<AnchoredPosition>(() => centeredFallback(PANEL_FALLBACK_WIDTH))

  // `useLayoutEffect` et non `useEffect` : la mesure doit être posée AVANT la peinture, sinon
  // le panneau apparaît une image au coin haut-gauche puis saute à sa place.
  useLayoutEffect(() => {
    function place() {
      const el = triggerRef?.current
      // SANS DÉCLENCHEUR, ON CENTRE — on ne disparaît PAS. Le cas se produit hors de la page
      // (un test qui monte le panneau seul) mais aussi en production, si le bouton se démonte
      // sous le panneau ouvert. Un panneau de réglages qui s'évapore laisserait l'utilisateur
      // devant un geste sans effet ; centré, il reste utilisable et refermable.
      if (!el) {
        setPos(centeredFallback(width))
        return
      }
      setPos(
        anchoredPosition(
          el.getBoundingClientRect(),
          { width: window.innerWidth, height: window.innerHeight },
          width,
        ),
      )
    }

    place()
    window.addEventListener('resize', place)
    window.addEventListener('scroll', place, true)
    return () => {
      window.removeEventListener('resize', place)
      window.removeEventListener('scroll', place, true)
    }
  }, [triggerRef, width])

  return pos
}
