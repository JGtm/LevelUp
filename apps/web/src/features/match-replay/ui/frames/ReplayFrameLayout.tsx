/**
 * ReplayFrameLayout — LA STRUCTURE UNIQUE DES TROIS CADRES de la page du rejeu (Standard, Cinéma,
 * Plein cadre), à zones nommées.
 *
 * # CINQ ZONES, MÊME ARBRE DANS CHAQUE CADRE
 *
 *  - `tete`    : l'en-tête du cadre ;
 *  - `avis`    : ce que la page dit hors du rejeu (bandeau de génération, chargement, absence, erreur) ;
 *  - `rangee`  : la rangée au-dessus du plan, que seuls le Cinéma et le Plein cadre peuvent poser ;
 *  - `scene`   : la toile et ce qui la coiffe ;
 *  - `colonne` : la colonne à côté du plan.
 * `scene` est rendue au MÊME endroit de l'arbre quel que soit le cadre : React en garde l'instance, et
 * avec elle la position de lecture, le son, le zoom, un export en cours et les calques cuits. Basculer de
 * cadre ne change que les classes de la table ci-dessous et le contenu de `tete`, `rangee` et `colonne`.
 *
 * # CE QUE CHAQUE CADRE CHANGE
 *
 * Le Plein cadre est le cadre d'écran de `REPLAY_FULLSCREEN_FRAME` : il recouvre la fenêtre et défile
 * lui-même, `useReplayViewport` mesure la hauteur offerte au terrain par rapport à lui. Son corps se
 * centre dans la hauteur restante sous l'en-tête (marge automatique : elle se réduit à zéro quand le
 * contenu déborde, et c'est le cadre qui défile). Standard et Cinéma restent dans le flux de
 * l'application.
 */
import type { ReactNode } from 'react'

import { REPLAY_FULLSCREEN_FRAME } from '../../hooks/useReplayFullscreen'
import type { ReplayFrame } from '../../model/replayFrame'

export interface ReplayFrameLayoutProps {
  frame: ReplayFrame
  tete: ReactNode
  avis: ReactNode
  rangee: ReactNode
  /** Absente tant que le rejeu n'est pas prêt : ni la grille ni la colonne ne se montent alors. */
  scene: ReactNode
  colonne: ReactNode
}

/** Les classes de chaque zone par cadre : la table est fermée, un cadre ajouté la complète à la compilation. */
const FRAME_CLASSES: Record<ReplayFrame, { root: string; body: string }> = {
  standard: { root: 'flex flex-col', body: 'space-y-4 px-6 pb-3' },
  cinema: { root: 'flex flex-col', body: 'space-y-2 px-6 pb-2' },
  'plein-cadre': {
    root: `${REPLAY_FULLSCREEN_FRAME} flex flex-col`,
    body: 'my-auto shrink-0 space-y-2',
  },
}

/**
 * LA GRILLE DU PLAN ET DE SA COLONNE. C'est le plan qui impose la hauteur de la rangée : la colonne est
 * `relative` et son contenu `absolute inset-0` à partir de `xl`, donc un fil de 80 éliminations ou un BTB
 * à 24 fiches ne peuvent pas l'étirer. Sous 1280 px (`xl`) le plan reprend toute la largeur et la colonne
 * passe dessous.
 */
const GRID_CLASSES = 'grid gap-3 xl:grid-cols-[minmax(0,1fr)_minmax(0,30rem)] xl:items-stretch'

export function ReplayFrameLayout({ frame, tete, avis, rangee, scene, colonne }: ReplayFrameLayoutProps) {
  const classes = FRAME_CLASSES[frame]
  return (
    <div className={classes.root} data-replay-frame={frame}>
      {tete}
      <div className={classes.body}>
        {avis}
        {rangee}
        {scene != null && (
          <div className={GRID_CLASSES}>
            {/* `min-w-0` : sans lui, un contenu large ferait déborder la colonne au lieu de la
                contraindre — c'est elle que mesure le ResizeObserver de la toile. `relative` : le
                repère de l'écran de victoire, qui coiffe toute la colonne du plan. */}
            <section className="relative min-w-0">{scene}</section>
            <aside className="relative">{colonne}</aside>
          </div>
        )}
      </div>
    </div>
  )
}
