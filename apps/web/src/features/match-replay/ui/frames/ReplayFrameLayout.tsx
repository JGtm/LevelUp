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
 * `colonne` peut être CACHÉE (`colonneCachee`) : elle reste dans l'arbre, avec le fil qu'elle porte, mais n'est
 * pas affichée et le plan prend sa largeur (Cinéma et Plein cadre quand la rangée de fiches est posée et qu'aucune
 * fiche de joueur n'est ouverte).
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
 * l'application ; le Standard est de plus une page bornée et centrée (`standardPage`), son en-tête et son
 * corps dans la même boîte.
 */
import type { CSSProperties, ReactNode, Ref } from 'react'

import { REPLAY_FULLSCREEN_FRAME } from '../../hooks/useReplayFullscreen'
import { COTES_COLONNE_STANDARD } from '../../model/cotesStandard'
import type { ReplayFrame } from '../../model/replayFrame'
import { STANDARD_PAGE_CLASS, STANDARD_PAGE_STYLE } from './standardPage'

export interface ReplayFrameLayoutProps {
  frame: ReplayFrame
  tete: ReactNode
  avis: ReactNode
  rangee: ReactNode
  /** Absente tant que le rejeu n'est pas prêt : ni la grille ni la colonne ne se montent alors. */
  scene: ReactNode
  colonne: ReactNode
  /** La colonne reste dans l'arbre, cachée : le plan prend toute la largeur. */
  colonneCachee?: boolean
  /** Le corps du cadre, quand le cadre le mesure (la rangée des équipes dépend de sa largeur). */
  bodyRef?: Ref<HTMLDivElement>
}

/** Les classes de chaque zone par cadre : la table est fermée, un cadre ajouté la complète à la compilation. */
const FRAME_CLASSES: Record<ReplayFrame, { root: string; body: string; bodyStyle?: CSSProperties }> = {
  // Le Standard est une page bornée et centrée : la boîte de `standardPage` porte la largeur et les marges.
  standard: { root: 'flex flex-col', body: `space-y-4 pb-3 ${STANDARD_PAGE_CLASS}`, bodyStyle: STANDARD_PAGE_STYLE },
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
 *
 * LA LARGEUR DE LA COLONNE SE DIT PAR CADRE. Le Standard a une colonne fixe (`COTES_COLONNE_STANDARD`),
 * passée par une variable CSS : une classe de Tailwind ne se compose pas depuis un nombre, et la variable
 * garde le nombre dans la cote. Le Cinéma et le Plein cadre gardent la colonne souple de 30 rem.
 */
const GRID_CLASSES = 'grid gap-3 xl:items-stretch'
const STANDARD_COLUMN_VAR = '--replay-colonne-standard'

const GRID_BY_FRAME: Record<ReplayFrame, { className: string; style?: CSSProperties }> = {
  standard: {
    className: 'xl:grid-cols-[minmax(0,1fr)_var(--replay-colonne-standard)]',
    style: { [STANDARD_COLUMN_VAR]: `${COTES_COLONNE_STANDARD.largeurPx}px` } as CSSProperties,
  },
  cinema: { className: 'xl:grid-cols-[minmax(0,1fr)_minmax(0,30rem)]' },
  'plein-cadre': { className: 'xl:grid-cols-[minmax(0,1fr)_minmax(0,30rem)]' },
}

/** Le gabarit à une seule colonne : le plan prend la largeur que la colonne cachée ne réclame plus. */
const GRID_SANS_COLONNE = 'xl:grid-cols-[minmax(0,1fr)]'

export function ReplayFrameLayout({ frame, tete, avis, rangee, scene, colonne, colonneCachee = false, bodyRef }: ReplayFrameLayoutProps) {
  const classes = FRAME_CLASSES[frame]
  const grid = GRID_BY_FRAME[frame]
  return (
    <div className={classes.root} data-replay-frame={frame}>
      {tete}
      <div ref={bodyRef} className={classes.body} style={classes.bodyStyle} data-replay-body={frame}>
        {avis}
        {rangee}
        {scene != null && (
          <div
            className={`${GRID_CLASSES} ${colonneCachee ? GRID_SANS_COLONNE : grid.className}`}
            style={colonneCachee ? undefined : grid.style}
            data-replay-grid={frame}
          >
            {/* `min-w-0` : sans lui, un contenu large ferait déborder la colonne au lieu de la
                contraindre — c'est elle que mesure le ResizeObserver de la toile. `relative` : le
                repère de l'écran de victoire, qui coiffe toute la colonne du plan. */}
            <section className="relative min-w-0">{scene}</section>
            <aside className={colonneCachee ? 'relative hidden' : 'relative'} data-replay-column={colonneCachee ? 'cachee' : 'affichee'}>
              {colonne}
            </aside>
          </div>
        )}
      </div>
    </div>
  )
}
