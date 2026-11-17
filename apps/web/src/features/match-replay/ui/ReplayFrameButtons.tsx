/**
 * ReplayFrameButtons — LES ICÔNES DE CADRE ET DU PLEIN ÉCRAN, à droite d'une barre de lecture :
 * Standard, Cinéma, Plein cadre, puis le plein écran du système. Partagé par les barres de lecture,
 * dans les trois cadres ; l'en-tête de page ne porte aucun sélecteur de cadre.
 *
 *  - L'icône du cadre courant est marquée (`aria-pressed`). Chaque icône est nommée par le
 *    dictionnaire (`frames.frameTip`), qui y rappelle la touche : « Cinéma (T) », « Plein cadre (F) ».
 *  - Un clic mène au cadre par `selectFrame`, jamais par une règle de cadre écrite ici. Seule
 *    l'icône du Plein cadre COURANT est autre chose : elle en SORT par `leaveFullFrame`, le chemin
 *    de « Quitter » (le cadre d'où l'on vient, et le plein écran du système refermé), que le modèle
 *    admet même pendant un export.
 *  - PENDANT UN EXPORT VIDÉO les trois icônes sont inertes (`frameControl.disabled`), sauf cette
 *    sortie : le cadre ne change pas sous la toile qui produit le clip.
 *  - Le plein écran du système n'est pas un cadre : il s'ajoute à n'importe lequel, son nom dit ce
 *    que le clic va faire (entrer ou sortir) et il est inerte pendant un export (`fullscreen.disabled`).
 */
import type { ComponentType } from 'react'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { ReplayFrameControl } from '../hooks/useReplayFrame'
import type { ReplayFullscreen } from '../hooks/useReplayFullscreen'
import type { ReplayFrame } from '../model/replayFrame'
import { leaveFullFrame, selectFrame } from './frames/selectFrame'
import { PRESS_GUARDS } from './replayPressGuards'
import { CinemaFrameIcon, FullFrameIcon, FullscreenIcon, StandardFrameIcon } from './ReplayTransportIcons'

const FRAME_ICONS: readonly { frame: ReplayFrame; icon: ComponentType }[] = [
  { frame: 'standard', icon: StandardFrameIcon },
  { frame: 'cinema', icon: CinemaFrameIcon },
  { frame: 'plein-cadre', icon: FullFrameIcon },
]

/** Une icône ronde de la rangée : l'état marqué prend le fond d'accent, l'inerte s'éteint. */
function iconClass(pressed: boolean, disabled: boolean): string {
  const state = pressed
    ? 'bg-accent text-accent-foreground'
    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
  const cursor = disabled ? 'cursor-not-allowed opacity-40' : 'cursor-pointer'
  return `inline-flex h-8 w-8 items-center justify-center rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${cursor} ${state}`
}

interface ReplayFrameButtonsProps {
  frameControl: ReplayFrameControl
  fullscreen: ReplayFullscreen
  locale: ReplayLocale
}

export function ReplayFrameButtons({ frameControl, fullscreen, locale }: ReplayFrameButtonsProps) {
  const t = REPLAY_TEXT[locale]
  const fullscreenLabel = fullscreen.active ? t.fullscreenExit : t.fullscreenEnter
  return (
    <div className="ml-1.5 flex items-center gap-0.5" data-replay-frame-buttons="">
      <div role="group" aria-label={t.frames.groupLabel} className="flex items-center gap-0.5">
        {FRAME_ICONS.map(({ frame, icon: Icon }) => {
          const pressed = frameControl.frame === frame
          const leavesFullFrame = pressed && frame === 'plein-cadre'
          const disabled = frameControl.disabled && !leavesFullFrame
          const label = t.frames.frameTip(frame)
          return (
            <button
              key={frame}
              type="button"
              {...PRESS_GUARDS}
              aria-pressed={pressed}
              aria-label={label}
              title={label}
              disabled={disabled}
              onClick={leavesFullFrame ? () => leaveFullFrame(frameControl, fullscreen) : () => selectFrame(frameControl, frame)}
              className={iconClass(pressed, disabled)}
            >
              <Icon />
            </button>
          )
        })}
      </div>
      <span className="mx-1 h-4 w-px bg-border" aria-hidden="true" />
      <button
        type="button"
        onClick={fullscreen.toggle}
        disabled={fullscreen.disabled}
        aria-label={fullscreenLabel}
        title={fullscreenLabel}
        className={iconClass(false, fullscreen.disabled)}
      >
        <FullscreenIcon exit={fullscreen.active} />
      </button>
    </div>
  )
}
