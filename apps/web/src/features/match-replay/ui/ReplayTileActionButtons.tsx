/**
 * ReplayTileActionButtons — LES DEUX ACTIONS D'UNE TUILE OU D'UNE LIGNE : la cible (Suivre / Ne plus suivre)
 * et l'œil (Masquer / Afficher), posés en haut à droite de la tuile ou de la ligne qui les porte.
 *
 * VISIBILITÉ. Au repos, un bouton est invisible ; il apparaît au survol de la tuile OU quand le focus est dedans
 * (`group-hover`, `group-focus-within` : la tuile porte `group`), et reste visible tant qu'il est actif — un joueur
 * suivi ou masqué se lit sans survol. Sur un écran sans survol (`hover: none`) les boutons au repos sont visibles
 * à 60 %. Un bouton invisible reste focalisable au clavier.
 *
 * CONTRAT DU BOUTON. `aria-pressed` dit l'état (suivi, masqué) ; le nom accessible dit le geste (« Suivre X »,
 * « Ne plus suivre X », « Masquer X », « Afficher X ») ; l'infobulle (`title`, lue par la couche des infobulles de
 * l'application) écrit le titre puis une phrase. Les boutons sont DANS la tuile, qui est elle-même un bouton : un
 * clic sur l'un n'ouvre pas la fiche (`stopPropagation`), Entrée non plus ; Espace reste « Lecture » et la souris ne
 * prend pas le focus (`PRESS_GUARDS`).
 *
 * Aucune couleur en dur : les encres sont des jetons (`primary`, `muted-foreground`, `foreground`, `background`).
 */
import type { KeyboardEvent, ReactNode } from 'react'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { TileActions } from '../model/playerTileActions'
import { PRESS_GUARDS } from './replayPressGuards'

/** Un bouton : 16 px, rayon, anneau de focus. */
const BUTTON =
  'inline-flex h-4 w-4 cursor-pointer items-center justify-center rounded transition-opacity focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring'
/** Au repos : invisible, sauf au survol ou au focus de la tuile, et à 60 % sans survol. */
const AT_REST =
  'bg-background/80 text-muted-foreground opacity-0 hover:text-foreground group-hover:opacity-100 group-focus-within:opacity-100 [@media(hover:none)]:opacity-60'
/** Actif : plein et toujours visible. */
const ACTIVE = 'bg-primary text-primary-foreground opacity-100'

/** La cible : un cercle et quatre repères. */
function TargetIcon() {
  return (
    <svg viewBox="0 0 16 16" width="12" height="12" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" aria-hidden="true" focusable="false">
      <circle cx="8" cy="8" r="4.5" />
      <path d="M8 1v3M8 12v3M1 8h3M12 8h3" />
    </svg>
  )
}

/** L'œil ouvert (le joueur est affiché : le geste le masquerait) ou barré (le joueur est masqué). */
function EyeIcon({ crossed }: { crossed: boolean }) {
  return (
    <svg viewBox="0 0 16 16" width="12" height="12" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" focusable="false">
      <path d="M1 8s2.5-4.5 7-4.5S15 8 15 8s-2.5 4.5-7 4.5S1 8 1 8z" />
      <circle cx="8" cy="8" r="2" />
      {crossed && <path d="M2.5 13.5l11-11" />}
    </svg>
  )
}

interface ActionButtonProps {
  label: string
  title: string
  pressed: boolean
  onAct: () => void
  children: ReactNode
}

function ActionButton({ label, title, pressed, onAct, children }: ActionButtonProps) {
  return (
    <button
      type="button"
      className={`${BUTTON} ${pressed ? ACTIVE : AT_REST}`}
      aria-pressed={pressed}
      aria-label={label}
      title={title}
      onMouseDown={PRESS_GUARDS.onMouseDown}
      onKeyDown={(e: KeyboardEvent) => {
        PRESS_GUARDS.onKeyDown(e)
        // Entrée active le bouton et ne remonte pas à la tuile, qui ouvrirait la fiche.
        if (e.key === 'Enter') e.stopPropagation()
      }}
      onClick={(e) => {
        e.stopPropagation()
        onAct()
      }}
    >
      {children}
    </button>
  )
}

interface ReplayTileActionButtonsProps {
  /** Le nom du joueur, tel que la tuile l'écrit : il entre dans le nom accessible et l'infobulle. */
  name: string
  actions: TileActions
  locale: ReplayLocale
  /** La place des boutons dans la tuile ou la ligne (position, écart). */
  className: string
}

export function ReplayTileActionButtons({ name, actions, locale, className }: ReplayTileActionButtonsProps) {
  const t = REPLAY_TEXT[locale].playerPanel.tile
  const { followed, hidden } = actions
  return (
    <span className={className} data-replay-tile-actions="">
      <ActionButton
        label={followed ? t.unfollowLabelFmt(name) : t.followLabelFmt(name)}
        title={followed ? `${t.unfollow}\n${t.unfollowHint}` : `${t.follow}\n${t.followHintFmt(name)}`}
        pressed={followed}
        onAct={actions.onToggleFollow}
      >
        <TargetIcon />
      </ActionButton>
      <ActionButton
        label={hidden ? t.showLabelFmt(name) : t.hideLabelFmt(name)}
        title={hidden ? `${t.show}\n${t.showHintFmt(name)}` : `${t.hide}\n${t.hideHintFmt(name)}`}
        pressed={hidden}
        onAct={actions.onToggleHidden}
      >
        <EyeIcon crossed={hidden} />
      </ActionButton>
    </span>
  )
}
