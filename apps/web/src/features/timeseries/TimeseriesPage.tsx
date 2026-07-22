/**
 * TimeseriesPage — page Séries temporelles (3 onglets).
 *
 * Orchestrateur slim. Chaque onglet est un sous-composant dans son propre fichier
 * (voir audit #6 god-file split) :
 *   - TimeseriesPage.summary.tsx       (onglet "summary")
 *   - TimeseriesPage.distributions.tsx (onglet "distributions")
 *   - TimeseriesPage.progression.tsx   (onglet "progression")
 *
 * Phase 2 P2.E : migration partielle Plotly → ECharts pour les charts qui ont
 * un wrapper ECharts disponible (TimeseriesLineChart, Heatmap2DChart). Les
 * histogrammes, scatters, KDA bars et combat-yield restent sur Plotly en
 * attendant des wrappers ECharts dédiés (différé Phase 3). i18n via
 * `timeseriesManifest` + `formatMessage`.
 */
import { useMemo } from 'react'
import { useParams, useSearch, useNavigate } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { useTimeseriesPage } from './queries'
import { useSoloFilterStore } from '@/stores/soloFilterStore'
import { useExplorerMatches } from '@/features/explorer/queries'
import { normalizeExplorerTableRows } from '@/features/explorer/ExplorerPage.matchesMode'
import { formatMessage } from '@/lib/i18n/format'
import {
  timeseriesManifest,
  type TimeseriesManifestKey,
} from '@/lib/i18n/generated/timeseries'
import { useAppShellStore } from '@/stores/appShellStore'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { SessionBriefing } from '@/features/_shared/SessionBriefing'

import { TimeseriesSummaryTab, type OutcomeLabels } from './TimeseriesPage.summary'
import { TimeseriesDistributionsTabView } from './TimeseriesPage.distributions'
import { TimeseriesProgressionTab } from './TimeseriesPage.progression'

type TabId = 'summary' | 'distributions' | 'progression'

const TAB_KEYS: { id: TabId; key: TimeseriesManifestKey }[] = [
  { id: 'summary', key: 'timeseries.tabs.summary' },
  { id: 'distributions', key: 'timeseries.tabs.distributions' },
  { id: 'progression', key: 'timeseries.tabs.progression' },
]

export function TimeseriesPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const filterContext = useSoloFilterStore((s) => s.filterContext)
  const filterContextHash = useSoloFilterStore((s) => s.filterContextHash)
  const { tab } = useSearch({
    from: '/{-$lang}/t/$titleSlug/players/$playerSlug/stats/timeseries',
  })
  const activeTab: TabId = tab ?? 'summary'
  const navigate = useNavigate({ from: '/{-$lang}/t/$titleSlug/players/$playerSlug/stats/timeseries' })
  const setActiveTab = (next: TabId) => {
    navigate({ search: (prev) => ({ ...prev, tab: next }), replace: true }).catch(() => {})
  }
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: TimeseriesManifestKey) => formatMessage(timeseriesManifest, key, locale)
  const { data: fieldMappings } = useFieldMappings()
  const outcomeLabels: OutcomeLabels = {
    win: fieldMappings?.outcomes?.win?.label ?? t('timeseries.distributions.outcome_win_fallback'),
    loss:
      fieldMappings?.outcomes?.loss?.label ??
      t('timeseries.distributions.outcome_loss_fallback'),
    tie: fieldMappings?.outcomes?.tie?.label ?? 'Égalité',
    dnf: fieldMappings?.outcomes?.dnf?.label ?? 'Abandon',
    unknown:
      fieldMappings?.fields['outcome_unknown']?.label ??
      t('timeseries.distributions.outcome_unknown_fallback'),
  }
  // Resolver de label de carte aligné sur la page Escouade.
  const mapAssets = fieldMappings?.assets?.['map']
  const mapLabelOf = (mapUI: string) => mapAssets?.[mapUI]?.label ?? mapUI

  // Injecter match_context='solo' : cette page n'affiche que les matchs solo.
  const soloFilterContext = useMemo(
    () => ({ ...filterContext, match_context: 'solo' as const }),
    [filterContext],
  )

  const { data, isLoading, isError, refetch } = useTimeseriesPage(
    playerSlug,
    { filters: soloFilterContext },
    filterContextHash,
  )

  // Tableau historique de matchs (reproduit l'Explorer mode "Matchs") affiché
  // en bas de l'onglet Progression. Suit le scope du filtre global solo pour
  // matcher les matchs des charts. Pas de filtres avancés (perf/skill/etc.).
  const explorerMatchesQuery = useExplorerMatches(
    playerSlug,
    {
      filters: soloFilterContext,
      pagination: { page: 1, page_size: 10000 },
      sort_field: 'start_time',
      // ASC pour matcher l'ordre chronologique des charts de progression
      // au-dessus (oldest -> newest). La page Explorer générique reste DESC.
      sort_dir: 'asc',
    },
    filterContextHash,
    activeTab === 'progression',
  )

  if (isLoading) return null

  if (isError) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-destructive">{t('timeseries.errors.load_failed')}</p>
            <button onClick={() => refetch()} className="mt-2 text-sm text-primary underline">
              {t('timeseries.errors.retry')}
            </button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="p-6">
        <EmptyStateCard
          title={t('timeseries.empty.page_title')}
          description={t('timeseries.empty.page_description')}
          actionLabel={t('timeseries.errors.retry')}
          onAction={() => refetch()}
        />
      </div>
    )
  }

  return (
    <div className="flex flex-col">
      {/* Briefing — KPI bar mode solo (rail + grid 7 cards, pas de verdict) */}
      {data.briefing_kpis && (
        <div className="px-6 pt-6 pb-6">
          <SessionBriefing kpis={data.briefing_kpis} />
        </div>
      )}

      {/* Onglets */}
      <div className="flex gap-0 border-b bg-background px-6">
        {TAB_KEYS.map((tab) => (
          <Button
            key={tab.id}
            variant="ghost"
            size="sm"
            onClick={() => setActiveTab(tab.id)}
            className={`rounded-none border-b-2 px-4 py-3 text-sm ${
              activeTab === tab.id
                ? 'border-primary font-semibold text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {t(tab.key)}
          </Button>
        ))}
      </div>

      <div className="p-6 space-y-6">
        {activeTab === 'summary' && (
          <TimeseriesSummaryTab
            data={data}
            t={t}
            fieldMappings={fieldMappings}
            outcomeLabels={outcomeLabels}
            mapLabelOf={mapLabelOf}
          />
        )}

        {activeTab === 'distributions' && (
          <TimeseriesDistributionsTabView
            distributions_tab={data.distributions_tab}
            t={t}
            fieldMappings={fieldMappings}
            outcomeLabels={outcomeLabels}
          />
        )}

        {activeTab === 'progression' && (
          <TimeseriesProgressionTab
            data={data}
            playerSlug={playerSlug}
            locale={locale}
            t={t}
            fieldMappings={fieldMappings}
            soloFilterContext={soloFilterContext}
            filterContextHash={filterContextHash}
            explorerMatchRows={normalizeExplorerTableRows(explorerMatchesQuery.data?.table?.items ?? null)}
          />
        )}
      </div>
    </div>
  )
}
