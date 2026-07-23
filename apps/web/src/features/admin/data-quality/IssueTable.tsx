/**
 * IssueTable — table générique des listes d'inconnus : colonnes configurables,
 * bouton d'action par ligne ouvrant un formulaire inline (une seule ligne
 * ouverte à la fois — état porté par le parent via openID).
 */
import type { ReactNode } from 'react'
import {
  type ColumnDef,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'

import { Button } from '@/components/ui/button'
import type { AdminDataQualityIssue } from '@/lib/api/types'
import { playerScopedHref } from '@/lib/title-routing'
import { useAppShellStore } from '@/stores/appShellStore'
import { adminAbsoluteTime, adminRelativeTime } from '../format'
import { useAdminT, useAdminLocale, type TAdmin } from '../useAdminText'

export interface IssueColumn {
  header: string
  /** Rendu de cellule. */
  cell: (issue: AdminDataQualityIssue) => ReactNode
  align?: 'right'
}

/** Pagination serveur optionnelle (kinds à liste longue, ex. xuids orphelins). */
export interface IssuePagination {
  pageIndex: number
  pageSize: number
  /** Total avant fenêtrage (renvoyé par l'API) — pilote le nombre de pages. */
  total: number
  onPageChange: (pageIndex: number) => void
}

interface IssueTableProps {
  issues: AdminDataQualityIssue[]
  columns: IssueColumn[]
  /** Libellé du bouton d'action par ligne (absent → table lecture seule). */
  actionLabel?: string
  /** ID de la ligne dont le formulaire est ouvert. */
  openID?: string | null
  onAction?: (issue: AdminDataQualityIssue) => void
  /** Formulaire inline rendu sous la ligne ouverte. */
  renderForm?: (issue: AdminDataQualityIssue) => ReactNode
  /** Pagination serveur (absente → pas de pied de page, comportement inchangé). */
  pagination?: IssuePagination
  /** Titre pour construire les liens de match d'exemple (présent → colonne
   *  « Exemples » avec jusqu'à 3 match_id cliquables, ouverture nouvel onglet). */
  matchLinkTitleSlug?: string
}

export function IssueTable({
  issues,
  columns,
  actionLabel,
  openID,
  onAction,
  renderForm,
  pagination,
  matchLinkTitleSlug,
}: IssueTableProps) {
  const tA = useAdminT()
  const locale = useAdminLocale()
  // Joueur cible des liens de match : la vue de match est player-scopée
  // (/t/{slug}/players/{player}/matches/{id}). En admin il n'y a pas de scope
  // joueur — on retient le joueur de session (sinon le premier profil disponible),
  // qui est le propriétaire dont l'historique peuple la base partagée.
  const currentPlayerSlug = useAppShellStore((s) => s.currentPlayer?.player_slug)
  const fallbackPlayerSlug = useAppShellStore((s) => s.availablePlayers[0]?.player_slug)
  const examplePlayerSlug = currentPlayerSlug ?? fallbackPlayerSlug
  const showExamples = Boolean(matchLinkTitleSlug)

  return (
    <div className="flex flex-col gap-2">
    <div className="overflow-x-auto rounded-md border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
            {columns.map((c) => (
              <th key={c.header} className={`px-3 py-2 font-medium ${c.align === 'right' ? 'text-right' : ''}`}>
                {c.header}
              </th>
            ))}
            <th className="px-3 py-2 font-medium text-right">{tA('admin.dq.col_occurrences')}</th>
            <th className="px-3 py-2 font-medium">{tA('admin.dq.col_last_seen')}</th>
            {showExamples && <th className="px-3 py-2 font-medium">{tA('admin.dq.col_examples')}</th>}
            {actionLabel && <th className="px-3 py-2" />}
          </tr>
        </thead>
        <tbody>
          {issues.map((issue) => {
            const rowKey = `${issue.kind}:${issue.asset_kind ?? ''}:${issue.id}`
            return (
              <RowWithForm
                key={rowKey}
                open={openID === rowKey}
                colSpan={columns.length + 2 + (showExamples ? 1 : 0) + (actionLabel ? 1 : 0)}
                form={renderForm?.(issue)}
              >
                {columns.map((c) => (
                  <td
                    key={c.header}
                    className={`px-3 py-2 ${c.align === 'right' ? 'text-right' : ''}`}
                  >
                    {c.cell(issue)}
                  </td>
                ))}
                <td className="px-3 py-2 text-right font-mono text-xs tabular-nums text-foreground">
                  {issue.occurrences}
                </td>
                <td
                  className="px-3 py-2 text-xs text-muted-foreground"
                  title={adminAbsoluteTime(issue.last_seen, locale)}
                >
                  {adminRelativeTime(issue.last_seen, locale)}
                </td>
                {showExamples && (
                  <td className="px-3 py-2">
                    <ExampleMatchLinks
                      ids={issue.example_match_ids}
                      titleSlug={matchLinkTitleSlug as string}
                      playerSlug={examplePlayerSlug}
                      tA={tA}
                    />
                  </td>
                )}
                {actionLabel && (
                  <td className="px-3 py-2 text-right">
                    <Button size="sm" variant="outline" onClick={() => onAction?.(issue)}>
                      {actionLabel}
                    </Button>
                  </td>
                )}
              </RowWithForm>
            )
          })}
        </tbody>
      </table>
    </div>
      {pagination && <IssueTablePagination pagination={pagination} tA={tA} />}
    </div>
  )
}

/**
 * Pied de pagination serveur piloté par TanStack Table (règle 13) : l'état est
 * contrôlé par le parent (page courante), le nombre de pages est dérivé du total
 * via `manualPagination` + `rowCount`. Aucune ligne n'est rendue ici (le corps
 * du tableau reste IssueTable) — l'instance ne sert qu'à la logique de pages.
 */
const NO_ROWS: AdminDataQualityIssue[] = []
const NO_COLS: ColumnDef<AdminDataQualityIssue>[] = []

function IssueTablePagination({ pagination, tA }: { pagination: IssuePagination; tA: TAdmin }) {
  const { pageIndex, pageSize, total, onPageChange } = pagination
  const table = useReactTable({
    data: NO_ROWS,
    columns: NO_COLS,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    rowCount: total,
    state: { pagination: { pageIndex, pageSize } },
  })
  const pageCount = table.getPageCount()
  if (pageCount <= 1) return null
  return (
    <div className="flex items-center justify-end gap-2 px-1 text-xs text-muted-foreground">
      <span className="tabular-nums">{`${tA('admin.dq.pagination_total')} : ${total}`}</span>
      <button
        type="button"
        onClick={() => onPageChange(pageIndex - 1)}
        disabled={!table.getCanPreviousPage()}
        aria-label={tA('admin.dq.pagination_prev')}
        className="rounded border border-border px-2 py-1 leading-none transition-colors hover:text-foreground disabled:opacity-40"
      >
        &lsaquo;
      </button>
      <span className="tabular-nums">{`${pageIndex + 1} / ${pageCount}`}</span>
      <button
        type="button"
        onClick={() => onPageChange(pageIndex + 1)}
        disabled={!table.getCanNextPage()}
        aria-label={tA('admin.dq.pagination_next')}
        className="rounded border border-border px-2 py-1 leading-none transition-colors hover:text-foreground disabled:opacity-40"
      >
        &rsaquo;
      </button>
    </div>
  )
}

/**
 * Liens de match d'exemple (≤3) — ouverture NOUVEL onglet vers la vue de match
 * (player-scopée). match_id tronqué (UUID), complet en title/aria. Sans joueur
 * cible résolu, on affiche l'ID en clair (non cliquable) plutôt que rien.
 */
function ExampleMatchLinks({
  ids,
  titleSlug,
  playerSlug,
  tA,
}: {
  ids: string[] | null | undefined
  titleSlug: string
  playerSlug: string | undefined
  tA: TAdmin
}) {
  const items = (ids ?? []).slice(0, 3)
  if (items.length === 0) {
    return <span className="text-xs text-muted-foreground">—</span>
  }
  return (
    <div className="flex flex-wrap gap-1.5">
      {items.map((id) => {
        const short = id.length > 8 ? `${id.slice(0, 8)}…` : id
        if (!playerSlug) {
          return (
            <span key={id} title={id} className="font-mono text-xs text-muted-foreground">
              {short}
            </span>
          )
        }
        return (
          <a
            key={id}
            href={playerScopedHref(titleSlug, playerSlug, `/matches/${id}`)}
            target="_blank"
            rel="noopener noreferrer"
            title={`${tA('admin.dq.example_open')} · ${id}`}
            aria-label={tA('admin.dq.example_open')}
            className="font-mono text-xs text-muted-foreground underline decoration-dotted underline-offset-2 transition-colors hover:text-foreground hover:decoration-solid"
          >
            {short}
          </a>
        )
      })}
    </div>
  )
}

/** Ligne + rangée formulaire optionnelle (colSpan complet) sous la ligne. */
function RowWithForm({
  open,
  colSpan,
  form,
  children,
}: {
  open: boolean
  colSpan: number
  form?: ReactNode
  children: ReactNode
}) {
  return (
    <>
      <tr className="border-b last:border-b-0 hover:bg-muted/30">{children}</tr>
      {open && form && (
        <tr className="border-b last:border-b-0">
          <td colSpan={colSpan} className="px-3 py-2">
            {form}
          </td>
        </tr>
      )}
    </>
  )
}
