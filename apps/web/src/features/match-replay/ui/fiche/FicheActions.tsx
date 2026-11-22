/**
 * FicheActions — L'ACTION « SEUL » de la fiche d'un joueur : un bouton à bascule (`aria-pressed`) pleine
 * largeur, qui masque tous les autres joueurs ; actif, il se lit « Seul · tout réafficher » et réaffiche tout. Suivre
 * et Masquer ne sont plus dans la fiche : ils sont sur les tuiles et les lignes (`ReplayTileActionButtons`).
 * « Actif » se lit sur l'ensemble des joueurs masqués (`isSolo`) : réafficher un joueur à la main sort de « Seul ».
 * Le bouton est accessible au clavier : le focus se voit par un anneau.
 *
 * La souris ne prend pas le focus et Espace reste « Lecture » (`PRESS_GUARDS`), comme pour tout bouton du rejeu.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../../i18n/i18n'
import { PRESS_GUARDS } from '../replayPressGuards'

/** Le contour : une colonne pleine largeur, 32 px, texte de 12 px, anneau de focus. */
const BUTTON_CLASS =
  'h-8 w-full min-w-0 cursor-pointer truncate rounded-lg border border-border px-2 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring'

/** L'action active est pleine ; au repos, un fond léger au survol. */
const ACTIVE_CLASS = 'bg-primary text-primary-foreground hover:bg-primary/90'
const IDLE_CLASS = 'bg-background text-foreground hover:bg-accent hover:text-accent-foreground'

interface FicheActionsProps {
  /** Le nom accessible du groupe : le nom du joueur. */
  name: string
  /** Vrai quand ce joueur est « seul » : tous les autres joueurs sont masqués. */
  solo: boolean
  /** « Seul » sur ce joueur, ou tout réafficher s'il l'est déjà. */
  onSolo: () => void
  locale: ReplayLocale
}

export function FicheActions({ name, solo, onSolo, locale }: FicheActionsProps) {
  const text = REPLAY_TEXT[locale].playerPanel.solo
  return (
    <div role="group" aria-label={name} data-replay-fiche-actions="">
      <button
        type="button"
        className={`${BUTTON_CLASS} ${solo ? ACTIVE_CLASS : IDLE_CLASS}`}
        aria-pressed={solo}
        title={solo ? text.hintActive : text.hint}
        {...PRESS_GUARDS}
        onClick={onSolo}
      >
        {solo ? text.labelActive : text.label}
      </button>
    </div>
  )
}
