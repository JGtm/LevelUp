/**
 * useAnchoredPanel — LA POSITION D'UN PANNEAU FLOTTANT, ancrée sur le bouton qui l'ouvre.
 *
 * # POURQUOI (demande utilisateur du 2026-09-02)
 *
 * « Est-ce que ce tiroir sort du cadre ou pas ? Je pense que si ça reste centré par son milieu
 * au-dessus du bouton des réglages, et si la taille de l'écran le permet, c'est bien qu'il
 * puisse s'afficher au-dessus du killfeed ou des fiches. »
 *
 * La réponse à la question était NON : le tiroir vivait en `absolute inset-y-0 right-0` DANS
 * la carte du rejeu, dont le `overflow-hidden` le découpait au bord. Il ne pouvait donc ni
 * dépasser sur la colonne de droite, ni être plus large que ce que la carte voulait bien lui
 * laisser — et c'est cette largeur (264 px utiles) qui interdisait les calques sur deux
 * colonnes depuis le 2026-08-29.
 *
 * Sortir du cadre demande de quitter le flux : le panneau se rend en PORTAIL sur `body`, en
 * `position: fixed`, et se replace lui-même à partir du rectangle de son déclencheur.
 *
 * # LES TROIS CONTRAINTES, DANS L'ORDRE OÙ ELLES SE RÉSOLVENT
 *
 * 1. CENTRÉ SUR LE BOUTON, puis ramené dans la fenêtre. Le bouton vit au bout de la barre de
 *    lecture, tout à droite : un panneau large centré sur lui déborderait à droite. Le
 *    `clamp` le ramène, ce qui le décentre légèrement — un panneau tronqué serait pire.
 * 2. IL OUVRE VERS LE HAUT. Le bouton est en bas de la carte ; vers le bas, il n'y a rien.
 *    D'où un ancrage par le BAS (`bottom`), qui fait grandir le panneau vers le haut sans
 *    qu'on ait à connaître sa hauteur.
 * 3. IL NE DÉPASSE JAMAIS L'ÉCRAN. `maxHeight` vaut ce qui reste au-dessus du bouton ; au-delà,
 *    c'est le panneau qui défile, pas la page. Sans cette borne, un tiroir plus haut que la
 *    fenêtre sortirait par le haut, là où rien ne le rattrape.
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

/**
 * Le repli : centré dans la fenêtre. Il sert au premier rendu et au cas où le déclencheur n'est
 * pas mesurable (panneau monté seul, bouton démonté sous un panneau ouvert). On ne rend jamais
 * nul : un panneau de réglages qui s'évapore laisserait un geste sans effet.
 */
function centeredFallback(): AnchoredPosition {
  const w = typeof window === 'undefined' ? 0 : window.innerWidth
  const h = typeof window === 'undefined' ? 0 : window.innerHeight
  return {
    left: Math.max((w - PANEL_FALLBACK_WIDTH) / 2, EDGE),
    bottom: EDGE,
    maxHeight: Math.max(h - 2 * EDGE, MIN_HEIGHT),
  }
}

export interface AnchoredPosition {
  left: number
  bottom: number
  maxHeight: number
}

export function useAnchoredPanel(
  triggerRef: RefObject<HTMLElement | null> | undefined,
  width: number,
): AnchoredPosition {
  // L'ÉTAT DÉMARRE CENTRÉ, JAMAIS NUL. Un premier rendu nul ferait naître le panneau APRÈS le
  // montage, et tout ce qui l'attend au montage — le focus qui doit y entrer, notamment —
  // s'exécuterait sur un panneau qui n'existe pas encore. Le `useLayoutEffect` ci-dessous
  // corrige la position avant la peinture : le centrage n'est donc jamais vu.
  const [pos, setPos] = useState<AnchoredPosition>(centeredFallback)

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
        setPos({ ...centeredFallback(), left: Math.max((window.innerWidth - width) / 2, EDGE) })
        return
      }
      const r = el.getBoundingClientRect()
      const centered = r.left + r.width / 2 - width / 2
      const maxLeft = Math.max(window.innerWidth - width - EDGE, EDGE)
      setPos({
        left: Math.min(Math.max(centered, EDGE), maxLeft),
        bottom: Math.max(window.innerHeight - r.top + GAP, EDGE),
        maxHeight: Math.max(r.top - GAP - EDGE, MIN_HEIGHT),
      })
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
