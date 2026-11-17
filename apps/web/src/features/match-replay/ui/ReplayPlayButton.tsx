/**
 * ReplayPlayButton — LE BOUTON DE LECTURE / PAUSE, le seul bouton plein d'une barre de lecture. Le nom
 * accessible dit ce que le CLIC va faire, l'icône dit où l'on en est. Inerte pendant un export. Partagé par la
 * barre complète et la barre d'une ligne : l'appelant donne la taille (`sizeClass` ou `style`).
 */
import type { CSSProperties } from 'react'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { PauseIcon, PlayIcon } from './ReplayTransportIcons'

interface ReplayPlayButtonProps {
  playing: boolean
  onTogglePlay: () => void
  disabled: boolean
  locale: ReplayLocale
  /** Les classes de taille (et de disposition) : posées entre `inline-flex` et le centrage. */
  sizeClass?: string
  style?: CSSProperties
}

export function ReplayPlayButton({ playing, onTogglePlay, disabled, locale, sizeClass = '', style }: ReplayPlayButtonProps) {
  const t = REPLAY_TEXT[locale]
  return (
    <button
      type="button"
      onClick={onTogglePlay}
      disabled={disabled}
      aria-label={playing ? t.pause : t.play}
      title={`${playing ? t.pause : t.play} (${t.keySpace})`}
      style={style}
      className={`inline-flex ${sizeClass} items-center justify-center rounded-full bg-primary text-primary-foreground transition-colors ${
        disabled ? 'cursor-not-allowed opacity-40' : 'cursor-pointer hover:bg-primary/90 active:bg-primary/80'
      }`}
    >
      {playing ? <PauseIcon /> : <PlayIcon />}
    </button>
  )
}
