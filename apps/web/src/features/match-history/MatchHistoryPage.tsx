/**
 * MatchHistoryPage — page de l'historique des parties.
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { PageHeader } from '@/components/shell/PageHeader'
import { Spinner } from '@/components/ui/spinner'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import { MatchHistoryTable } from './MatchHistoryTable'
import { useMatchHistory, useMatchHistoryExport } from './queries'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'

export function MatchHistoryPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const filterContextHash = useGlobalFilterStore((s) => s.filterContextHash)

  const [page, setPage] = useState(1)
  const [sortField, setSortField] = useState('start_time')
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('desc')

  const request = {
    filters: filterContext,
    pagination: { page, page_size: 50 },
    include_export_hint: true,
  }

  const { data, isLoading, isError, refetch } = useMatchHistory(
    playerSlug,
    request,
    filterContextHash,
    page,
  )

  const exportMutation = useMatchHistoryExport(playerSlug)

  function handleSort(field: string) {
    if (field === sortField) {
      setSortDirection((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortField(field)
      setSortDirection('desc')
    }
    setPage(1)
  }

  function handleExport() {
    exportMutation.mutate(
      { filters: filterContext },
      {
        onSuccess: (token) => {
          window.open(token.download_path, '_blank')
        },
      },
    )
  }

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Historique des parties"
        subtitle={data ? `${data.summary.total_matches_scoped.toLocaleString('fr-FR')} parties dans la période` : undefined}
      />

      <div className="p-6">
        {/* Sprint 54-B : avertissement privacy */}
        <PrivacyBanner warning={data?.privacy_warning} className="mb-4" />

        {isLoading ? (
          <div className="flex justify-center py-16">
            <Spinner size="lg" label="Chargement de l'historique…" />
          </div>
        ) : isError ? (
          <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-6 text-center">
            <p className="font-medium text-destructive">Impossible de charger l'historique.</p>
            <button onClick={() => refetch()} className="mt-2 text-sm text-primary underline">
              Réessayer
            </button>
          </div>
        ) : data ? (
          <MatchHistoryTable
            rows={data.table.items}
            pagination={data.table.pagination}
            sortField={sortField}
            sortDirection={sortDirection}
            onSort={handleSort}
            onPage={(p) => setPage(p)}
            onExport={handleExport}
            exporting={exportMutation.isPending}
            playerSlug={playerSlug}
            filterHash={filterContextHash}
            page={page}
          />
        ) : null}
      </div>
    </div>
  )
}
