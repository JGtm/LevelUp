/**
 * ImpactSessionRow — une session du tableau « Impact des coéquipiers » : sa ligne (cellules des colonnes)
 * et, quand elle est ouverte, la ligne dépliée qui porte sa matrice de rôles. Sorti de
 * `SquadImpactSessionsTable` pour tenir le plafond de longueur de fonction.
 */
import { Fragment } from 'react'
import { flexRender, type Row } from '@tanstack/react-table'

import type { Locale } from '@/lib/i18n/locale'

import type { SquadText } from '../i18n'
import { SessionMatrix } from './SessionMatrix'
import { ALL_SESSIONS_KEY, type ImpactRowView, type ImpactSessionsView } from './impactSessions.logic'
import type { ImpactSessionsText } from './impactSessionsText'

/** Le fond d'une ligne : teinte de la session sélectionnée, ou survol ; un peu plus soutenu quand elle est ouverte. */
function rowTone(selected: boolean, open: boolean): string {
  if (selected) return open ? 'bg-info/15' : 'bg-info/10'
  return open ? 'bg-muted/60' : 'hover:bg-muted/60'
}

interface Props {
  row: Row<ImpactRowView>
  /** Rang de la ligne : toute ligne après la première porte un filet au-dessus. */
  index: number
  /** La session fait partie de la sélection du filtre de la page (ligne teintée). */
  selected: boolean
  /** Nombre de colonnes du tableau (étendue de la ligne dépliée). */
  columnCount: number
  view: ImpactSessionsView
  impact: SquadText['impact']
  text: ImpactSessionsText
  locale: Locale
  inkOf: (gamertag: string) => string
}

export function ImpactSessionRow({ row, index, selected, columnCount, view, impact, text, locale, inkOf }: Props) {
  return (
    <Fragment>
      <tr
        className={`cursor-pointer ${rowTone(selected, row.getIsExpanded())} ${index > 0 ? 'border-t border-border/60' : ''}`}
        data-testid={row.original.kind === 'all' ? 'squad-impact-row-all' : 'squad-impact-row'}
        onClick={() => row.toggleExpanded()}
      >
        {row.getVisibleCells().map((cell) => (
          <td key={cell.id} className="whitespace-nowrap px-3.5 py-2.5">
            {flexRender(cell.column.columnDef.cell, cell.getContext())}
          </td>
        ))}
      </tr>
      {row.getIsExpanded() && (
        <tr>
          <td colSpan={columnCount} className="bg-muted/40 p-0">
            <SessionMatrix
              row={row.original}
              view={view}
              impact={impact}
              text={text}
              locale={locale}
              inkOf={inkOf}
              ariaLabel={row.id === ALL_SESSIONS_KEY ? text.totalsLabel : text.matrixLabel(row.original.label)}
            />
          </td>
        </tr>
      )}
    </Fragment>
  )
}
