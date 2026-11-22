/**
 * FicheActions — LES TROIS ACTIONS SUR LA CARTE de la fiche d'un joueur, Suivre, Seul et Masquer, en UN SEGMENT :
 * trois colonnes égales, collées sous un même contour arrondi, séparées par un filet ; l'action active est PLEINE
 * (fond et encre du jeton principal), les autres sont au repos. Chaque colonne reste un bouton à bascule
 * (`aria-pressed`), accessible au clavier : le focus se voit par un anneau rentrant, que le contour arrondi ne rogne
 * pas. Les exclusions entre actions et leur remise à zéro sont dans `playerPanelLogic` ; ce composant ne décide rien.
 *
 * La souris ne prend pas le focus et Espace reste « Lecture » (`PRESS_GUARDS`), comme pour tout bouton du rejeu.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../../i18n/i18n'
import type { PlayerPanelAction } from '../../model/playerPanelLogic'
import { PRESS_GUARDS } from '../replayPressGuards'

/** L'ordre des trois actions : celui de la maquette. */
const ACTIONS: readonly PlayerPanelAction[] = ['follow', 'solo', 'hide']

/** Le contour du segment : trois colonnes égales, coins arrondis, tout ce qui dépasse est rogné. */
const SEGMENT_CLASS = 'grid grid-cols-3 overflow-hidden rounded-lg border border-border'

/** Une colonne : 32 px, texte de 12 px, filet à gauche sauf la première, anneau de focus rentrant. */
const COLUMN_CLASS =
  'h-8 min-w-0 cursor-pointer truncate px-2 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring [&:not(:first-child)]:border-l [&:not(:first-child)]:border-border'

/** L'action active est pleine ; les autres sont au repos, un fond léger au survol. */
const ACTIVE_CLASS = 'bg-primary text-primary-foreground hover:bg-primary/90'
const IDLE_CLASS = 'bg-background text-foreground hover:bg-accent hover:text-accent-foreground'

interface FicheActionsProps {
  /** Le nom accessible du groupe : le nom du joueur. */
  name: string
  /** Quelle action est active. */
  actions: Readonly<Record<PlayerPanelAction, boolean>>
  onToggle: (action: PlayerPanelAction) => void
  locale: ReplayLocale
}

export function FicheActions({ name, actions, onToggle, locale }: FicheActionsProps) {
  const text = REPLAY_TEXT[locale].playerPanel
  return (
    <div className={SEGMENT_CLASS} role="group" aria-label={name} data-replay-fiche-actions="">
      {ACTIONS.map((action) => (
        <button
          key={action}
          type="button"
          className={`${COLUMN_CLASS} ${actions[action] ? ACTIVE_CLASS : IDLE_CLASS}`}
          aria-pressed={actions[action]}
          title={text.actionHint[action]}
          {...PRESS_GUARDS}
          onClick={() => onToggle(action)}
        >
          {text.action[action]}
        </button>
      ))}
    </div>
  )
}
