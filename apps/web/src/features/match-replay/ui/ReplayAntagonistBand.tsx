/**
 * ReplayAntagonistBand — LA BANDE DES TITRES de la frise : une rangée de quatorze pixels AU-DESSUS de
 * la rangée du joueur regardé, où chaque PRISE de titre (némésis, souffre-douleur) est une icône posée
 * à l'instant de l'élimination qui a fait passer le titre (D-13 du plan des antagonistes).
 *
 * EXTRAIT DE `ReplayTimelineTracks.tsx`, qui est au-delà du seuil de taille : l'hôte ne connaît que
 * l'emplacement (`antagonistBand`, un nœud posé en tête de sa grille) ; ce composant porte tout le reste.
 *
 * LA RANGÉE EST DANS LA GRILLE DES PISTES : une cellule vide sous la colonne des libellés, puis la bande
 * sur la colonne des pistes (`TIMELINE_GRID_COLUMNS`), donc la même largeur et la même abscisse que la
 * rangée du dessous. L'ABSCISSE EST CELLE DE `trackLeft` (jamais recopiée : garde-rail
 * `timelineGeometry.guard.test.ts`).
 *
 * UNE BANDE VIDE RESTE UNE BANDE. Sans point de vue ou sans prise de titre, la rangée garde ses quatorze
 * pixels et ne porte aucun texte : passer d'un joueur à l'autre ne décale pas la frise.
 *
 * CE SONT DES BOUTONS, la seule exception de la frise avec les portes de présence et les vignettes de
 * média : le survol (`title`, lu par la couche des infobulles du rejeu) et le clic (poser le curseur à
 * l'image de la prise, sans mettre en pause, comme une porte de présence) exigent que la bande reçoive
 * le pointeur. Le nom accessible est « titre. phrase » ; l'infobulle, « titre », un saut de ligne,
 * « phrase » (comme les boutons des tuiles). La souris ne prend pas le focus et Espace reste « Lecture »
 * (`PRESS_GUARDS`).
 *
 * L'ENCRE de l'icône est celle du camp de son tenant, vu du point de vue (jetons) : un adversaire est
 * à l'encre adverse.
 */
import { useMemo } from 'react'

import type { FilmAllegiance } from '@/lib/replay/filmAllegiance'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { ANTAGONISTS_TEXT } from '../i18n/antagonists.i18n'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { buildAntagonistBand } from '../model/antagonistBandLogic'
import { antagonistNameOf } from '../model/antagonistNames'
import { campInk } from '../model/campInk'
import { cardName } from '../model/playerCardReadings'
import { trackedPlayerOf } from '../model/playerPanelLogic'
import type { ReplayAntagonists } from '../model/replayAntagonists'
import { trackLeft, trackScale } from '../model/replayTimelineTracksLogic'
import type { ReplayWindowBounds } from '../model/replayWindow'
import { AntagonistIcon } from './AntagonistIcon'
import { PRESS_GUARDS } from './replayPressGuards'

/** Le côté de l'icône dans son bouton de quatorze pixels. */
const ICON_PX = 12

export interface ReplayAntagonistBandProps {
  /** La fabrique des antagonistes du match (`useReplayAntagonists`) : la bande lit l'index du point de vue. */
  antagonists: ReplayAntagonists
  /** Le point de vue de la page : un xuid de feuille ou une clé du film ; `null` = personne, bande vide. */
  viewpoint: string | null
  players: readonly ReplayPlayer[]
  /** L'allégeance du film vue du point de vue : l'encre du camp de chaque tenant. */
  allegiance: Pick<FilmAllegiance, 'ofXuid'>
  /** La fenêtre de jeu et la durée du film : l'échelle des pistes (`trackScale`), la même que celle des marques. */
  playWindow: ReplayWindowBounds | null
  frameCount: number
  frameIntervalMs: number
  locale: ReplayLocale
  /** Poser le curseur à une image, sans pause (celui des portes de présence). */
  onSeekFrame: (frame: number) => void
}

export function ReplayAntagonistBand({
  antagonists, viewpoint, players, allegiance, playWindow, frameCount, frameIntervalMs, locale, onSeekFrame,
}: ReplayAntagonistBandProps) {
  const items = useMemo(() => {
    const t = REPLAY_TEXT[locale]
    const subject = trackedPlayerOf(viewpoint, players)
    if (!subject) return []
    return buildAntagonistBand(antagonists.forViewpoint(viewpoint).changes, {
      nameOf: antagonistNameOf(players, t),
      viewpointName: cardName(subject, t),
      frameIntervalMs,
      scale: trackScale(playWindow, frameCount),
      text: ANTAGONISTS_TEXT[locale],
    })
  }, [antagonists, viewpoint, players, playWindow, frameCount, frameIntervalMs, locale])

  return (
    <>
      <span aria-hidden="true" />
      <div className="relative h-[14px]" data-replay-antagonist-band="">
        {items.map((item) => (
          <button
            key={item.key}
            type="button"
            onClick={() => onSeekFrame(item.frame)}
            className="absolute top-0 flex h-[14px] w-[14px] -translate-x-1/2 items-center justify-center rounded-[2px] transition-colors hover:bg-accent focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            style={{ left: trackLeft(item.ratio), color: campInk(allegiance.ofXuid(item.holder)) }}
            aria-label={`${item.title}. ${item.phrase}`}
            title={`${item.title}\n${item.phrase}`}
            {...PRESS_GUARDS}
          >
            <AntagonistIcon role={item.role} size={ICON_PX} />
          </button>
        ))}
      </div>
    </>
  )
}
