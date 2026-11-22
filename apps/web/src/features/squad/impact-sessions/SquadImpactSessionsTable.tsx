/**
 * SquadImpactSessionsTable — « Impact des coéquipiers » (Escouade › Contributions) : un tableau, une
 * ligne par session de la plus récente à la plus ancienne, précédée de « Toutes les sessions ».
 *
 * Colonnes : « Session » (libellé, nombre de matchs, chevron), « Bilan » (victoires, barre, défaites),
 * puis un joueur par colonne (mini-barre centrée sur zéro et net par match). Chaque ligne se déplie sur
 * la matrice de sa session (`SessionMatrix`) ; plusieurs lignes peuvent être ouvertes à la fois. Les
 * sessions sélectionnées dans le filtre de la page s'ouvrent d'office et sont teintées ; sans sélection,
 * la plus récente s'ouvre. Pleine largeur, colonnes fixes (rien ne bouge au dépliage), défilement
 * horizontal dans le cadre sous 900 px.
 *
 * Données : `squad_impact_sessions`, mises en forme par `impactSessions.logic.ts`. Patron des lignes
 * dépliables : `match-view/MatchScoreboard.tsx` (TanStack Table, `getRowId`, état `expanded`). Le clic
 * sur la ligne et le bouton de sa première cellule (Entrée, Espace) la déplient.
 */
import { Fragment, useMemo, useState } from 'react'
import {
  type ColumnDef,
  type ExpandedState,
  type Updater,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'

import { WinLossSplitBar } from '@/features/_shared/EncounterSplitBars'
import { tokenCssVar } from '@/lib/accessibility'
import type { SquadImpactSessions } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { SquadText } from '../i18n'
import { PlayerName, SessionMatrix } from './SessionMatrix'
import {
  ALL_SESSIONS_KEY,
  buildImpactSessionsView,
  formatSignedNet,
  initialOpenState,
  netBarRatio,
  netSign,
  type ImpactPlayerView,
  type ImpactRowView,
  type ImpactSessionsView,
} from './impactSessions.logic'
import { IMPACT_SESSIONS_TEXT, type ImpactSessionsText } from './impactSessionsText'

const SESSION_COL_PX = 200
const BILAN_COL_PX = 130

interface Props {
  data: SquadImpactSessions | null | undefined
  /** Libellés des sessions sélectionnées dans le filtre de la page (suffixe de compte volatil toléré). */
  pickedSessionLabels: readonly string[]
  /** gamertag → encre CSS du joueur (palette des joueurs de l'Escouade). */
  inkOf: (gamertag: string) => string
  locale: Locale
  impact: SquadText['impact']
  /** Message du cadre vide (réponse sans session à montrer). */
  emptyMessage: string
}

function toExpanded(keys: readonly string[]): ExpandedState {
  return Object.fromEntries(keys.map((k) => [k, true]))
}

/** Le net par match d'un joueur : mini-barre centrée sur zéro et valeur signée. */
function NetCell({ player, locale, text }: { player: ImpactPlayerView; locale: Locale; text: ImpactSessionsText }) {
  if (player.netPerMatch === null) {
    return (
      <span className="text-muted-foreground" title={text.notPlayed}>
        —
      </span>
    )
  }
  const net = player.netPerMatch
  const sign = netSign(net)
  const token = sign === 'loss' ? 'outcome-loss' : 'outcome-win'
  const ratio = netBarRatio(net)
  const value = formatSignedNet(net, locale)
  return (
    <span
      className="flex items-center gap-2.5"
      title={text.netTooltip(value, formatSignedNet(player.points, locale), player.matchesPlayed)}
    >
      <span className="relative h-1.5 max-w-[90px] flex-1 rounded-sm bg-border/60" aria-hidden>
        <i
          className="absolute inset-y-0 rounded-sm"
          style={{
            left: `${ratio >= 0 ? 50 : 50 + ratio * 50}%`,
            width: `${Math.abs(ratio) * 50}%`,
            backgroundColor: tokenCssVar(token),
          }}
        />
        <i className="absolute -inset-y-[3px] left-1/2 w-px bg-muted-foreground/60" />
      </span>
      <span
        className="min-w-[44px] text-right font-semibold tabular-nums"
        style={sign === 'zero' ? undefined : { color: tokenCssVar(token) }}
      >
        {value}
      </span>
    </span>
  )
}

function Chevron({ open }: { open: boolean }) {
  return (
    <svg
      className={`size-3.5 shrink-0 text-muted-foreground transition-transform motion-reduce:transition-none ${open ? 'rotate-90' : ''}`}
      viewBox="0 0 16 16"
      aria-hidden
    >
      <path d="M6 3.5 10.5 8 6 12.5" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function buildColumns(
  view: ImpactSessionsView,
  text: ImpactSessionsText,
  locale: Locale,
  inkOf: (gamertag: string) => string,
): ColumnDef<ImpactRowView>[] {
  return [
    {
      id: 'session',
      header: text.colSession,
      cell: ({ row }) => {
        const r = row.original
        const isAll = r.kind === 'all'
        return (
          <button
            type="button"
            className="flex w-full items-center gap-2 text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
            aria-expanded={row.getIsExpanded()}
          >
            <Chevron open={row.getIsExpanded()} />
            <span className="min-w-0">
              <span className="block truncate font-semibold">{isAll ? text.allSessions : r.label}</span>
              <span className="block text-[12.5px] text-muted-foreground">
                {isAll ? text.allSessionsSub(r.matchCount) : text.sessionSub(r.matchCount)}
              </span>
            </span>
          </button>
        )
      },
    },
    {
      id: 'bilan',
      header: text.colBilan,
      cell: ({ row }) => <WinLossSplitBar wins={row.original.wins} losses={row.original.losses} locale={locale} />,
    },
    ...view.players.map<ColumnDef<ImpactRowView>>((player, i) => ({
      id: `player-${i}`,
      header: () => (
        <span className="text-[13px] font-medium normal-case tracking-normal text-foreground">
          <PlayerName player={player} ink={inkOf(player)} />
        </span>
      ),
      cell: ({ row }) => <NetCell player={row.original.players[i]} locale={locale} text={text} />,
    })),
  ]
}

const HEAD_CELL =
  'truncate border-b border-border px-3.5 pb-2 pt-3 text-left text-[11.5px] font-medium uppercase tracking-wide text-muted-foreground'

/** Le fond d'une ligne : teinte de la session sélectionnée, ou survol ; un peu plus soutenu quand elle est ouverte. */
function rowTone(selected: boolean, open: boolean): string {
  if (selected) return open ? 'bg-info/15' : 'bg-info/10'
  return open ? 'bg-muted/60' : 'hover:bg-muted/60'
}

/**
 * L'état ouvert des lignes : il repart de la sélection quand elle (ou les sessions affichées) change ;
 * entre-temps, les clics de l'utilisateur s'y ajoutent. Ajustement d'état pendant le rendu, sans effet.
 */
function useOpenRows(initialOpen: string[]) {
  const signature = initialOpen.join('|')
  const [state, setState] = useState<{ signature: string; expanded: ExpandedState }>(() => ({
    signature,
    expanded: toExpanded(initialOpen),
  }))
  if (state.signature !== signature) {
    setState({ signature, expanded: toExpanded(initialOpen) })
  }
  const onExpandedChange = (updater: Updater<ExpandedState>) =>
    setState((s) => ({ ...s, expanded: typeof updater === 'function' ? updater(s.expanded) : updater }))
  return { expanded: state.expanded, onExpandedChange }
}

export function SquadImpactSessionsTable({ data, pickedSessionLabels, inkOf, locale, impact, emptyMessage }: Props) {
  const text = IMPACT_SESSIONS_TEXT[locale]
  const view = useMemo(() => buildImpactSessionsView(data), [data])
  const initial = useMemo(
    () => (view ? initialOpenState(view.rows, pickedSessionLabels) : { open: [], selected: [] }),
    [view, pickedSessionLabels],
  )
  const { expanded, onExpandedChange } = useOpenRows(initial.open)
  const columns = useMemo(() => (view ? buildColumns(view, text, locale, inkOf) : []), [view, text, locale, inkOf])
  const table = useReactTable<ImpactRowView>({
    data: view?.rows ?? [],
    columns,
    state: { expanded },
    onExpandedChange,
    getRowId: (r) => r.key,
    getRowCanExpand: () => true,
    getCoreRowModel: getCoreRowModel(),
  })

  if (!view) {
    return (
      <div
        className="flex min-h-[120px] items-center justify-center rounded-md border border-border text-sm text-muted-foreground"
        data-testid="squad-impact-sessions"
      >
        {emptyMessage}
      </div>
    )
  }

  const selected = new Set(initial.selected)
  return (
    <div className="overflow-hidden rounded-md border border-border bg-card" data-testid="squad-impact-sessions">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[900px] table-fixed border-collapse text-sm" aria-label={text.tableLabel}>
          <colgroup>
            <col style={{ width: SESSION_COL_PX }} />
            <col style={{ width: BILAN_COL_PX }} />
            {view.players.map((p) => (
              <col key={p} />
            ))}
          </colgroup>
          <thead>
            <tr>
              {table.getFlatHeaders().map((h) => (
                <th key={h.id} scope="col" className={HEAD_CELL}>
                  {flexRender(h.column.columnDef.header, h.getContext())}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {table.getRowModel().rows.map((row, index) => (
              <Fragment key={row.id}>
                <tr
                  className={`cursor-pointer ${rowTone(selected.has(row.id), row.getIsExpanded())} ${index > 0 ? 'border-t border-border/60' : ''}`}
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
                    <td colSpan={columns.length} className="bg-muted/40 p-0">
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
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
