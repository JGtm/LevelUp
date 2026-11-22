/**
 * ReplayShowAllButton — « Tout afficher », en tête de la colonne des équipes et de la rangée des équipes :
 * il réaffiche d'un geste tous les joueurs masqués. Les deux lieux ne le montrent que si au moins un joueur est
 * masqué (`showAllVisible`). Même contrat que les boutons du rejeu : la souris ne prend pas le focus, Espace reste
 * « Lecture » (`PRESS_GUARDS`).
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { PRESS_GUARDS } from './replayPressGuards'

export function ReplayShowAllButton({ locale, onShowAll, className = '' }: {
  locale: ReplayLocale
  onShowAll: () => void
  className?: string
}) {
  return (
    <button
      type="button"
      data-replay-show-all=""
      className={`cursor-pointer rounded-md border border-border bg-background px-2 text-3xs font-medium leading-4 text-foreground hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${className}`}
      {...PRESS_GUARDS}
      onClick={onShowAll}
    >
      {REPLAY_TEXT[locale].playerPanel.tile.showAll}
    </button>
  )
}
