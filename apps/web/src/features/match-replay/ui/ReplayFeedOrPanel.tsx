/**
 * ReplayFeedOrPanel — LA CELLULE DU FIL : le fil des événements, ou le panneau de joueur qui prend
 * sa place (lot 4 du plan « maquettes du rejeu »).
 *
 * LE PANNEAU REMPLACE LE FIL À L'ÉCRAN, IL NE LE DÉTRUIT PAS. Le fil reste monté sous le panneau,
 * `invisible` : cette classe retire le fil de l'affichage, du focus clavier et de la lecture d'écran
 * sans lui retirer sa boîte, donc son défilement. « Retour au fil » retrouve ainsi le filtre du menu
 * et la position de défilement tels qu'ils étaient — un `display: none` ou un démontage les
 * perdraient. Le fil reste aussi abonné à l'horloge : sa liste est à jour au retour.
 *
 * La carte n'est jamais couverte : le panneau (`absolute inset-0`) remplit seulement cette cellule,
 * dans la colonne de droite. La hauteur de la cellule est celle du fil ; `min-h-[16rem]` ne vaut que
 * panneau ouvert, pour qu'il ne s'effondre pas quand le fil est court sous `xl`.
 */
import type { ReactNode } from 'react'

interface Props {
  /** Le fil des événements : toujours monté. */
  feed: ReactNode
  /** Le panneau de joueur, ou `null` quand le fil est affiché. */
  panel: ReactNode | null
}

export function ReplayFeedOrPanel({ feed, panel }: Props) {
  const open = panel !== null
  return (
    <div className={`relative flex min-h-0 flex-1 flex-col xl:min-h-[12rem]${open ? ' min-h-[16rem]' : ''}`}>
      <div
        data-testid="replay-feed-slot"
        className={`flex min-h-0 flex-1 flex-col${open ? ' invisible' : ''}`}
        aria-hidden={open ? true : undefined}
      >
        {feed}
      </div>
      {open && <div className="absolute inset-0">{panel}</div>}
    </div>
  )
}
