// cross-feature-allow: helpers de tri (NUMERIC_SORT, localeTextSortingFn) importés de
// features/explorer, comme le voisin SquadSynergyHistoryTable (baseline du ratchet) —
// 3e consommateur hors explorer au 2026-08-25 (avec MatchEncountersTable/MatchScoreboard) :
// la règle des 2 copies rend leur déménagement vers lib/ dû ; à faire dans un lot dédié
// qui migre les 4 importeurs d'un coup (pas ici, fix hors périmètre du chantier notion5).
/**
 * SquadAssistPairsTable — « qui assiste qui » dans l'escouade (page Synergies).
 *
 * Colonnes : Assistant | Bénéficiaire | Assistances | Part | Éliminations volées.
 *
 * LES DEUX COLONNES DE VOLUME SONT AFFICHÉES ENSEMBLE — brut ET part — sans bascule
 * (décision produit) : le brut dit combien, la part dit ce que ça pèse dans l'escouade,
 * et l'un sans l'autre se lit mal (3 assistances sur 6 n'est pas 3 sur 300).
 *
 * LE BANDEAU DE COUVERTURE N'EST PAS DÉCORATIF. L'assistance se lit dans le film du
 * match, et les films Theater EXPIRENT côté serveur : le manque est définitif, pas un
 * retard. Un pourcentage calculé sur une fraction de la sélection SANS dire laquelle
 * serait un chiffre non reproductible — le bandeau vit donc AU-DESSUS du tableau, pas
 * en note de bas de page.
 *
 * Le bloc entier est absent du contrat quand rien n'a été mesuré : ce composant n'est
 * alors pas monté (cf. SquadSynergiesPage). Il rend son état vide seulement pour le cas
 * « mesuré, aucune assistance INTERNE à l'escouade » — un fait, pas un vide.
 *
 * TanStack Table v8, tri client sur toutes les colonnes (helpers partagés
 * `explorerMatchesClientSort.ts`), pas de pagination : une escouade a au plus quelques
 * dizaines de paires.
 */
import { useMemo, useState } from 'react'
import {
  type ColumnDef,
  type SortingState,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'

import type { SquadAssistPair, SquadAssistPairs } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { HeaderLabelTooltip } from '@/lib/table/columnMeta'
import {
  NUMERIC_SORT,
  localeTextSortingFn,
} from '@/features/explorer/explorerMatchesClientSort'
import { getSquadText } from './i18n'

// Même classe d'en-tête que SquadSynergyHistoryTable — les deux tableaux vivent dans la
// même page et doivent s'aligner au pixel.
const ASSISTS_TH_CLASS =
  'px-2 py-2 text-left whitespace-nowrap text-xs font-medium text-muted-foreground border-r border-border last:border-r-0'

function sortAriaValue(dir: false | 'asc' | 'desc'): 'ascending' | 'descending' | 'none' {
  if (dir === 'asc') return 'ascending'
  if (dir === 'desc') return 'descending'
  return 'none'
}

interface SquadAssistPairsTableProps {
  block: SquadAssistPairs
}

export function SquadAssistPairsTable({ block }: SquadAssistPairsTableProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const labels = t.assists
  const intlLocale = t.intlLocale

  // `pairs` est un tableau NULLABLE au contrat (toute tranche Go sort ainsi) : comblé
  // ici, à la frontière, une seule fois.
  const pairs = useMemo(() => block.pairs ?? [], [block.pairs])
  // Dénominateur de la part : le TOTAL SERVEUR des assistances internes, pas la somme
  // des lignes visibles. Les deux coïncident aujourd'hui, mais dériver le dénominateur
  // de l'affichage le ferait mentir le jour où une ligne serait filtrée.
  const totalAssists = block.total_assists
  // Pourcentage LOCALISÉ : le français met une espace insécable avant le « % », pas
  // l'anglais. Intl s'en charge — un `${x} %` en dur serait faux dans une des deux
  // langues, et l'autre l'écrirait avec une espace ordinaire (coupure de ligne).
  const pctFmt = useMemo(
    () =>
      new Intl.NumberFormat(intlLocale, {
        style: 'percent',
        minimumFractionDigits: 1,
        maximumFractionDigits: 1,
      }),
    [intlLocale],
  )

  const columns = useMemo<ColumnDef<SquadAssistPair>[]>(
    () => [
      {
        id: 'assistant',
        header: labels.colAssistant,
        accessorFn: (r) => r.assist_gamertag,
        sortingFn: localeTextSortingFn,
        sortDescFirst: false,
        cell: (ctx) => <span className="font-medium">{ctx.row.original.assist_gamertag}</span>,
      },
      {
        id: 'killer',
        header: labels.colKiller,
        accessorFn: (r) => r.killer_gamertag,
        sortingFn: localeTextSortingFn,
        sortDescFirst: false,
        cell: (ctx) => ctx.row.original.killer_gamertag,
      },
      {
        id: 'count',
        header: labels.colCount,
        accessorFn: (r) => r.assist_count,
        ...NUMERIC_SORT,
        meta: { headerTooltip: labels.colCountTooltip },
        cell: (ctx) => (
          <span className="font-mono tabular-nums">{ctx.row.original.assist_count}</span>
        ),
      },
      {
        id: 'share',
        header: labels.colShare,
        accessorFn: (r) => (totalAssists > 0 ? r.assist_count / totalAssists : 0),
        ...NUMERIC_SORT,
        meta: { headerTooltip: labels.colShareTooltip },
        cell: (ctx) => (
          <span className="font-mono tabular-nums">
            {totalAssists > 0 ? pctFmt.format(ctx.row.original.assist_count / totalAssists) : '-'}
          </span>
        ),
      },
      {
        id: 'stolen',
        header: labels.colStolen,
        accessorFn: (r) => r.stolen_count,
        ...NUMERIC_SORT,
        meta: { headerTooltip: labels.colStolenTooltip },
        cell: (ctx) => (
          <span className="font-mono tabular-nums">{ctx.row.original.stolen_count}</span>
        ),
      },
    ],
    [labels, totalAssists, pctFmt],
  )

  // Tri initial : aucun. L'ordre serveur (assistances décroissantes) reste affiché tant
  // qu'aucun en-tête n'est cliqué — même règle que le tableau d'historique voisin.
  const [sorting, setSorting] = useState<SortingState>([])
  const table = useReactTable<SquadAssistPair>({
    data: pairs,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  })

  const coverage = (
    <p
      className="text-xs text-muted-foreground"
      data-testid="squad-assist-pairs-coverage"
      title={labels.coverageHint}
    >
      {labels.coverage(block.matches_measured, block.matches_total)}
    </p>
  )

  if (pairs.length === 0) {
    // Mesuré, mais aucune assistance entre membres de l'escouade. La couverture reste
    // affichée : sans elle, « aucune » ressemblerait à « rien mesuré ».
    return (
      <div className="space-y-2" data-testid="squad-assist-pairs-table">
        {coverage}
        <div className="flex min-h-[100px] items-center justify-center rounded-md border border-border text-sm text-muted-foreground">
          {labels.noPairs}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-2" data-testid="squad-assist-pairs-table">
      {coverage}
      <div className="overflow-x-auto rounded-md border border-border">
        <table className="w-full text-sm">
          <thead className="bg-muted border-b">
            {table.getHeaderGroups().map((hg) => (
              <tr key={hg.id}>
                {hg.headers.map((h) => {
                  const content = h.isPlaceholder
                    ? null
                    : flexRender(h.column.columnDef.header, h.getContext())
                  const tip = h.column.columnDef.meta?.headerTooltip
                  const sortDir = h.column.getIsSorted()
                  return (
                    <th key={h.id} className={ASSISTS_TH_CLASS} aria-sort={sortAriaValue(sortDir)}>
                      <HeaderLabelTooltip text={tip}>
                        <button
                          type="button"
                          onClick={h.column.getToggleSortingHandler()}
                          aria-label={labels.sortByAriaLabel(String(content ?? ''))}
                          className={`group inline-flex items-center gap-1 whitespace-nowrap transition-colors hover:text-foreground${sortDir ? ' text-foreground' : ''}`}
                        >
                          {content}
                          {sortDir && (
                            <span aria-hidden="true" className="text-2xs leading-none">
                              {sortDir === 'asc' ? '▲' : '▼'}
                            </span>
                          )}
                        </button>
                      </HeaderLabelTooltip>
                    </th>
                  )
                })}
              </tr>
            ))}
          </thead>
          <tbody className="divide-y divide-border">
            {table.getRowModel().rows.map((row) => (
              <tr key={row.id} className="transition-colors hover:bg-primary/10">
                {row.getVisibleCells().map((cell) => (
                  <td
                    key={cell.id}
                    className="px-2 py-2 whitespace-nowrap border-r border-border last:border-r-0"
                  >
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
