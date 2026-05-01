/**
 * MatchHistoryPage — page de l'historique des parties.
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import { SessionMultiSelect } from '@/components/ui/SessionMultiSelect'
import {
  OutcomeSequenceTape,
  type OutcomePoint,
  type OutcomeValue,
} from '@/components/charts/OutcomeSequenceTape'
import { MatchHistoryTable } from './MatchHistoryTable'
import { useMatchHistory, useMatchHistoryExport } from './queries'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { useAppShellStore } from '@/stores/appShellStore'
import { SessionBriefing } from '@/features/_shared/SessionBriefing'

const OUTCOME_FROM_CODE: Record<number, OutcomeValue> = {
  1: 'tie',
  2: 'win',
  3: 'loss',
  4: 'dnf',
}


export function MatchHistoryPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const filterContextHash = useGlobalFilterStore((s) => s.filterContextHash)
  const locale = useAppShellStore((s) => s.locale)

  const [page, setPage] = useState(1)
  const [sortField, setSortField] = useState('start_time')
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('desc')
  // §5 plan Squad/Sessions : filtre multi-sessions solo persisté localStorage.
  const sessionStorageKey = `stats-sessions-${playerSlug}`
  const [pickedSoloSessionLabels, setPickedSoloSessionLabelsRaw] = useState<string[]>(() => {
    try {
      const stored = localStorage.getItem(sessionStorageKey)
      return stored ? (JSON.parse(stored) as string[]) : []
    } catch {
      return []
    }
  })
  const applySoloSessionLabels = (labels: string[]) => {
    setPickedSoloSessionLabelsRaw(labels)
    try {
      localStorage.setItem(sessionStorageKey, JSON.stringify(labels))
    } catch {
      // localStorage indisponible — ignorer.
    }
    setPage(1) // reset pagination quand le filtre change
  }

  const request = {
    filters: filterContext,
    pagination: { page, page_size: 50 },
    include_export_hint: true,
    picked_solo_session_labels: pickedSoloSessionLabels.length > 0 ? pickedSoloSessionLabels : undefined,
  }

  const { data, isLoading, isError, refetch } = useMatchHistory(
    playerSlug,
    request,
    filterContextHash,
    page,
    pickedSoloSessionLabels,
  )

  const exportMutation = useMatchHistoryExport(playerSlug)
  const { data: fieldMappings } = useFieldMappings()

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
      <div className="p-6">
        {/* Sprint 54-B : avertissement privacy */}
        <PrivacyBanner warning={data?.privacy_warning} className="mb-4" />

        {/* SessionBriefing — KPI bar en haut de page (mode solo) */}
        {data?.briefing_kpis && (
          <div className="mb-4">
            <SessionBriefing kpis={data.briefing_kpis} />
          </div>
        )}

        {/* §5 plan Squad/Sessions : filtre multi-sessions solo */}
        {(data?.session_labels?.solo?.length ?? 0) > 0 && (
          <div className="mb-4">
            <SessionMultiSelect
              sessions={data!.session_labels.solo}
              selected={pickedSoloSessionLabels}
              onChange={applySoloSessionLabels}
              locale={locale}
            />
          </div>
        )}

        {isLoading ? (
          null
        ) : isError ? (
          <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-6 text-center">
            <p className="font-medium text-destructive">Impossible de charger l'historique.</p>
            <button onClick={() => refetch()} className="mt-2 text-sm text-primary underline">
              Réessayer
            </button>
          </div>
        ) : data ? (
          <>
            {data.table.items.length > 0 && (
              <div className="mb-4">
                <OutcomeSequenceTape
                  matches={[...data.table.items]
                    .reverse()
                    .filter((r) => r.outcome_code !== null)
                    .map<OutcomePoint>((r) => ({
                      matchId: r.match_id,
                      outcome: OUTCOME_FROM_CODE[r.outcome_code!] ?? 'dnf',
                      map: r.map_ui,
                      mode: r.mode_ui,
                    }))}
                  labels={{
                    win: fieldMappings?.outcomes?.['win']?.label ?? 'win',
                    loss: fieldMappings?.outcomes?.['loss']?.label ?? 'loss',
                    tie: fieldMappings?.outcomes?.['tie']?.label ?? 'tie',
                    dnf: fieldMappings?.outcomes?.['dnf']?.label ?? 'dnf',
                  }}
                />
              </div>
            )}
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
          </>
        ) : null}
      </div>
    </div>
  )
}
