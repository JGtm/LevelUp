/**
 * useReplayViewport — CE QUE L'ÉCRAN LAISSE AU TERRAIN : sa largeur, et la hauteur libre.
 *
 * # LE PROBLÈME (demande utilisateur du 2026-09-02)
 *
 * « Le canevas doit toujours être visible dans son entièreté niveau hauteur — ça englobe la
 * map et le rejeu mais aussi les contrôles de lecture. » La hauteur du terrain était une
 * CONSTANTE (480 px). Additionnée au reste de la pile — NavL1, fil d'Ariane, bandeau de score,
 * frise, barre de lecture — il fallait ~965 px de viewport pour tout voir. Or :
 *
 * - 1080p à 100 % → ~960 px : juste en dessous ;
 * - 1080p à 125 % (le réglage par défaut de la plupart des portables) → ~768 px ;
 * - 1080p à 150 % → ~640 px ; 1366x768 → ~660 px.
 *
 * Autrement dit : ça ne tenait pas sur la configuration la plus répandue.
 *
 * # CE HOOK OFFRE, IL NE DÉCIDE PAS
 *
 * Il rend une hauteur LIBRE, pas la hauteur du terrain. La différence est réelle : passé un
 * certain point, la largeur de la colonne devient le facteur limitant du cadrage et un pixel de
 * hauteur de plus n'agrandit plus la carte — il ajoute une bande vide. Ce point dépend du ratio
 * de la carte, que seul `useReplayView` connaît : c'est donc lui qui retient
 * `min(offre, usefulHeight)`. Ici, on ne mesure que la place.
 *
 * # POURQUOI LA QUANTIFICATION ET LE DÉLAI SONT OBLIGATOIRES
 *
 * Avant, le cadrage (`canvasView`) ne changeait que sur la largeur, qui est stable. Une hauteur
 * dérivée du viewport rend le redimensionnement de fenêtre CONTINU — et chaque changement de
 * cadrage recuit les quatre calques statiques, dont le sol et ses ~45 000 cellules. Sans garde,
 * tirer le bord d'une fenêtre déclencherait des dizaines de cuissons par seconde.
 *
 * Ce n'est pas un risque de mémoire : chaque cuisson REMPLACE la précédente (le hook des
 * calques réaffecte une référence, l'ancienne toile est collectée) et leur nombre est fixé à
 * quatre. C'est un risque de CPU/GC : des mégaoctets alloués et jetés en boucle, donc un onglet
 * qui se fige. D'où deux garde-fous, l'un et l'autre nécessaires :
 *
 * - la QUANTIFICATION (`VIEWPORT_HEIGHT_STEP`) : une variation de moins de 8 px ne produit même
 *   pas une nouvelle valeur, donc aucun rendu et aucune cuisson ;
 * - le DÉLAI (`VIEWPORT_SETTLE_MS`) : un glissement de bord de fenêtre ne cuit qu'une fois, à la
 *   fin du geste, au lieu d'une fois par image.
 *
 * Le plafond dur (`CANVAS_HEIGHT_CEILING`) est le troisième garde-fou, celui de la mémoire : il
 * borne la surface des quatre calques quelle que soit la taille de l'écran.
 *
 * # POURQUOI DEUX SOURCES D'ÉVÉNEMENTS
 *
 * Le `ResizeObserver` seul ne suffit pas. Rétrécir une fenêtre EN HAUTEUR ne change pas la
 * largeur du conteneur : l'observateur ne se déclencherait pas, et le terrain garderait une
 * hauteur devenue trop grande. D'où l'écoute de `resize` sur la fenêtre, en plus.
 */
import { useEffect, useState, type RefObject } from 'react'

import {
  CANVAS_HEIGHT_CEILING,
  CANVAS_HEIGHT_DEFAULT,
  CANVAS_HEIGHT_MIN,
} from './useReplayView'

/** Le blanc laissé sous la carte : sans lui, elle touche le bas de la fenêtre. */
export const VIEWPORT_BOTTOM_MARGIN = 16
/** Le pas de la hauteur — cf. l'en-tête, garde-fou n°1 contre la tempête de cuissons. */
export const VIEWPORT_HEIGHT_STEP = 8
/** Le temps d'immobilité exigé avant de remesurer — garde-fou n°2. */
export const VIEWPORT_SETTLE_MS = 120

export interface ReplayViewport {
  /** Largeur du conteneur en px CSS ; 0 tant qu'il n'est pas mesuré. */
  width: number
  /** Hauteur que l'écran laisse au terrain, entre le plancher et le plafond dur. */
  freeHeight: number
}

/**
 * fitCanvasHeight — de l'espace libre à la hauteur offerte : quantifier, puis borner.
 *
 * PURE ET EXPORTÉE, parce que c'est ICI qu'est la décision et que c'est elle qu'on teste. Sous
 * le plancher on ne rétrécit plus : une carte de 200 px ne se lit pas, mieux vaut laisser la
 * page défiler — le défilement montre alors quelque chose, ce qui est son rôle.
 */
export function fitCanvasHeight(free: number): number {
  const stepped = Math.floor(free / VIEWPORT_HEIGHT_STEP) * VIEWPORT_HEIGHT_STEP
  return Math.min(Math.max(stepped, CANVAS_HEIGHT_MIN), CANVAS_HEIGHT_CEILING)
}

/**
 * L'ancêtre qui défile, cherché par son style calculé plutôt que par sa balise : le hook n'a
 * pas à savoir que le shell de l'application met son `<main>` en `overflow-y: auto`. Absent
 * (tests unitaires, rendu hors shell), on retombe sur la fenêtre.
 */
function scrollParentOf(el: HTMLElement): HTMLElement | null {
  for (let p = el.parentElement; p; p = p.parentElement) {
    const overflowY = getComputedStyle(p).overflowY
    if (overflowY === 'auto' || overflowY === 'scroll') return p
  }
  return null
}

/**
 * L'espace vertical réellement disponible sous le haut du conteneur.
 *
 * MESURÉ DANS LE CADRE QUI DÉFILE, PAS DANS LA FENÊTRE. Pris depuis la fenêtre, le calcul
 * dépendrait de la position de défilement : descendre dans la page ferait remonter le haut du
 * conteneur, donc grandir le terrain, donc grandir la page — une boucle où défiler agrandit ce
 * qu'on regarde. Rapporté au conteneur de défilement et à son `scrollTop`, le résultat est le
 * même à toutes les positions : c'est bien « est-ce que ça tient quand on est en haut ? ».
 *
 * `chrome` = tout ce que le conteneur porte EN PLUS du terrain (ses marges internes, la frise,
 * la barre de lecture). C'est la seule part incompressible ; le terrain prend ce qui reste.
 */
function freeSpaceFor(el: HTMLElement, chrome: number): number {
  const rect = el.getBoundingClientRect()
  const scroller = scrollParentOf(el)
  if (!scroller) return window.innerHeight - rect.top - chrome - VIEWPORT_BOTTOM_MARGIN
  const box = scroller.getBoundingClientRect()
  const topInScroller = rect.top - box.top + scroller.scrollTop
  return scroller.clientHeight - topInScroller - chrome - VIEWPORT_BOTTOM_MARGIN
}

/**
 * @param ref  le conteneur mesuré : il porte le terrain ET tout ce qui l'accompagne.
 * @param canvasRef  la toile elle-même. Elle sert à ISOLER le chrome par soustraction, et c'est
 *   pour cela qu'on la lit dans le DOM plutôt que de suivre la hauteur qu'on vient de servir :
 *   le DOM dit ce qui est réellement peint, une valeur recopiée ne dirait que ce qu'on croit
 *   avoir demandé — et à la première divergence, la mesure suivante croirait que le chrome a
 *   changé de taille et partirait en oscillation.
 */
export function useReplayViewport(
  ref: RefObject<HTMLElement | null>,
  canvasRef: RefObject<HTMLCanvasElement | null>,
): ReplayViewport {
  const [viewport, setViewport] = useState<ReplayViewport>({
    width: 0,
    freeHeight: CANVAS_HEIGHT_DEFAULT,
  })

  useEffect(() => {
    const el = ref.current
    if (!el) return
    let timer = 0

    function measure() {
      if (!el) return
      const painted = canvasRef.current?.getBoundingClientRect().height ?? 0
      const chrome = Math.max(el.offsetHeight - painted, 0)
      const next = {
        width: Math.max(Math.floor(el.getBoundingClientRect().width), 0),
        freeHeight: fitCanvasHeight(freeSpaceFor(el, chrome)),
      }
      // ÉGALITÉ D'ABORD : la mesure qui suit un ajustement de hauteur retombe sur la même
      // valeur (le chrome, lui, n'a pas bougé). Rendre l'objet précédent arrête la chaîne
      // net — sans quoi chaque cuisson en déclencherait une autre.
      setViewport((prev) =>
        prev.width === next.width && prev.freeHeight === next.freeHeight ? prev : next,
      )
    }

    function settle() {
      window.clearTimeout(timer)
      timer = window.setTimeout(measure, VIEWPORT_SETTLE_MS)
    }

    measure()
    const ro = new ResizeObserver(settle)
    ro.observe(el)
    window.addEventListener('resize', settle)
    return () => {
      window.clearTimeout(timer)
      ro.disconnect()
      window.removeEventListener('resize', settle)
    }
  }, [ref, canvasRef])

  return viewport
}
