/**
 * ReplayFeedOrPanel — LA CELLULE DU FIL : le fil des événements, seul (Cinéma, Plein cadre), ou dans la colonne du
 * Standard, où la fiche du joueur prend la place de toute la colonne.
 *
 * LA FICHE REMPLACE LE FIL À L'ÉCRAN, ELLE NE LE DÉTRUIT PAS. Le fil reste monté sous la fiche, `invisible` : cette
 * classe retire le fil de l'affichage, du focus clavier et de la lecture d'écran sans lui retirer sa boîte, donc son
 * défilement. « Fil complet » et « Équipes » retrouvent ainsi le filtre du menu et la position de défilement tels
 * qu'ils étaient — un `display: none` ou un démontage les perdraient. Le fil reste aussi abonné à l'horloge : sa
 * liste est à jour au retour.
 *
 * LA FICHE REMPLIT TOUTE LA COLONNE (`above`). Ce qui se tient au-dessus du fil dans la colonne — les équipes du
 * Standard — se donne par `above` : il est monté avant la cellule, et la fiche ouverte le recouvre avec le fil.
 * Comme le fil, il reste MONTÉ sous la fiche, `invisible` (boîte et défilement gardés, hors focus et hors lecture
 * d'écran), et la fiche, `absolute inset-0`, se place contre la colonne elle-même, que son conteneur pose en
 * `relative`. `min-h-[16rem]` ne vaut que fiche ouverte, pour que la cellule ne s'effondre pas quand le fil est
 * court sous `xl`. Sans `above`, la cellule ne porte que le fil : le Cinéma et le Plein cadre ouvrent la fiche en
 * fenêtre (`ReplayPlayerPopover`).
 */
import { Fragment, type ReactNode } from 'react'

interface FeedProps {
  /** Le fil des événements : toujours monté. */
  feed: ReactNode
  /**
   * La hauteur minimale de la cellule, px, quand le fil est affiché : elle remplace le plancher par défaut. La
   * fiche ouverte garde le sien (`min-h-[16rem]`).
   */
  minHeightPx?: number
}

/** Le fil seul, ou le fil sous ce qui se tient au-dessus de lui et sous la fiche (`null` : le fil est affiché). */
type Props = FeedProps & ({ above?: undefined; panel?: null } | { above: ReactNode; panel: ReactNode | null })

export function ReplayFeedOrPanel({ feed, panel, minHeightPx, above }: Props) {
  const open = panel !== null && panel !== undefined
  const wholeColumn = above !== undefined
  return (
    <Fragment>
      {wholeColumn && (
        // `contents` : la boîte du bloc est celle de son contenu, comme sans l'enveloppe ; la visibilité, elle, se transmet.
        <div
          data-testid="replay-above-slot"
          className={`contents${open ? ' invisible' : ''}`}
          aria-hidden={open ? true : undefined}
        >
          {above}
        </div>
      )}
      <div
        className={`flex min-h-0 flex-1 flex-col xl:min-h-[12rem]${open ? ' min-h-[16rem]' : ''}`}
        style={!open && minHeightPx !== undefined ? { minHeight: minHeightPx } : undefined}
      >
        <div
          data-testid="replay-feed-slot"
          className={`flex min-h-0 flex-1 flex-col${open ? ' invisible' : ''}`}
          aria-hidden={open ? true : undefined}
        >
          {feed}
        </div>
        {open && <div className="absolute inset-0">{panel}</div>}
      </div>
    </Fragment>
  )
}
