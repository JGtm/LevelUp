/**
 * ReplayCanvasBlock — LE BLOC DE LA TOILE (`data-replay-plan-block`, que la fiche en fenêtre lit pour s'ancrer sous son
 * haut) : le cadre arrondi qui porte le plan (`children`, posé dans une zone relative où s'ancrent la légende et les
 * superpositions), et la barre de lecture (`transport`) placée selon `placement`. Il ne sait rien de ce qu'il porte :
 * la toile lui remet des nœuds.
 *
 * L'ARBRE NE CHANGE PAS D'UNE PLACE À L'AUTRE : la colonne mesurée (`containerRef`) contient toujours le bloc, puis la
 * barre, et seules leurs classes changent. C'est ce qui garde la toile et la barre montées quand le cadre change (la
 * lecture continue). La colonne contient le plan ET la barre : `useReplayViewport` déduit le chrome par soustraction.
 *
 * `placement` (cf. `barPlacement`) :
 *  - `in-block` (Standard) : pas de cadre propre, le cadre pose UN bloc autour de la carte et de sa colonne
 *    (`ReplayFrameLayout`) ; la barre complète va bord à bord sous le plan, dans le flux ;
 *  - `overlay` (Plein cadre, barre fine) : le bloc est arrondi, et la barre est posée sur le bas du plan, dans le
 *    cadre, hors du flux : le terrain reprend toute sa hauteur ;
 *  - `below` (Cinéma, et Plein cadre sans barre fine) : le bloc arrondi se ferme sur le plan, et la barre est SOUS lui,
 *    hors du bloc, sur le fond de la page, à `blocBarreEcartPx` du bloc.
 */
import type { CSSProperties, ReactNode, Ref } from 'react'

import { COTES_BARRE_COMPACTE as C } from '../model/cotesBarreCompacte'
import type { BarPlacement } from '../model/transportBarLogic'

/**
 * Le cadre du bloc : `overflow-hidden` retient les coins carrés de la carte dans son rayon arrondi ; le panneau de la
 * vue en sort donc par un portail (`useAnchoredPanel`).
 */
const FRAME_CLASS = 'overflow-hidden rounded-lg border border-border bg-card'

/** Le bloc, selon la place de la barre : sans cadre, ou cadré (et fermé par sa marge basse quand la barre est dessous). */
const BLOCK_CLASS: Record<BarPlacement, string> = {
  'in-block': 'relative',
  overlay: `relative ${FRAME_CLASS}`,
  below: `relative pb-3 ${FRAME_CLASS}`,
}

/**
 * La barre, selon sa place : dans le flux (Standard, et sous le bloc), ou posée sur le bas du plan DANS le cadre (le
 * liseré d'un pixel et les coins arrondis du cadre lui sont rendus).
 */
const BAR_CLASS: Record<BarPlacement, string | undefined> = {
  'in-block': undefined,
  overlay: 'absolute inset-x-px bottom-px z-20 overflow-hidden rounded-b-[7px]',
  below: undefined,
}

export function ReplayCanvasBlock({
  containerRef, placement, transport, children,
}: {
  containerRef: Ref<HTMLDivElement>
  placement: BarPlacement
  transport: ReactNode
  children: ReactNode
}) {
  const barStyle: CSSProperties | undefined = placement === 'below' ? { marginTop: C.blocBarreEcartPx } : undefined
  return (
    // Colonne canvas : c'est ELLE que le ResizeObserver mesure. Elle occupe toute la largeur : le panneau la recouvre
    // au lieu de lui prendre sa place, donc le canvas ne se retaille pas à l'ouverture.
    <div ref={containerRef} className="relative min-w-0">
      <div className={BLOCK_CLASS[placement]} data-replay-plan-block="">
        <div className="relative p-3 pb-0">{children}</div>
      </div>
      <div className={BAR_CLASS[placement]} style={barStyle} data-replay-bar-slot={placement}>
        {transport}
      </div>
    </div>
  )
}
