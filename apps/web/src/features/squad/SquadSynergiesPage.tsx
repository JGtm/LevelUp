/**
 * SquadSynergiesPage — onglet Synergies de l'Escouade.
 *
 * Distingue 2 états vides diagnosticables :
 *  - no_selection : aucun coéquipier confirmé.
 *  - invalid_selection : confirmedGts > 0 mais selectedRows vide.
 */
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { useSquadContext } from './SquadContext'
import { getSquadText } from './i18n'
import { WinRateVsHistoryChart } from './WinRateVsHistoryChart'
import { WinRateVsHistoryBulletChart } from './WinRateVsHistoryBulletChart'
import { MapPerfVsHistoryChart } from './MapPerfVsHistoryChart'
import { SquadMapHeatmapChart } from './SquadMapHeatmapChart'
import { SquadSessionTimelineChart } from './SquadSessionTimelineChart'

export function SquadSynergiesPage() {
  const { selectedRows, confirmedGamertags, pageData } = useSquadContext()
  const { data: mappings } = useFieldMappings()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)

  const hasSelection = confirmedGamertags.length > 0
  const hasRows = selectedRows.length > 0

  if (!hasSelection) {
    return (
      <Card>
        <CardContent className="pt-4">
          <EmptyStateNotice
            title={t.empty.noSelectionTitle}
            description={t.empty.noSelectionDescription}
          />
        </CardContent>
      </Card>
    )
  }

  if (!hasRows) {
    return (
      <Card>
        <CardContent className="pt-4">
          <EmptyStateNotice
            title={t.empty.invalidSelectionTitle}
            description={t.empty.invalidSelectionDescription}
          />
        </CardContent>
      </Card>
    )
  }

  const mapAssets = mappings?.assets?.['map']
  const mapLabelOf = (mapUI: string) => mapAssets?.[mapUI]?.label ?? mapUI
  const mapBreakdown = pageData?.map_breakdown ?? []
  const sessionTimeline = pageData?.session_timeline ?? []
  const mapHeatmap = pageData?.map_heatmap

  return (
    <div className="space-y-4">
      {mapBreakdown.length > 0 && (
        <WinRateVsHistoryChart
          title={t.charts.winRateVsHistoryTitle}
          rows={mapBreakdown}
          mapLabelOf={mapLabelOf}
          sessionLabel={t.charts.winRateVsHistorySession}
          historyLabel={t.charts.winRateVsHistoryHistory}
        />
      )}
      {mapBreakdown.length > 0 && (
        <WinRateVsHistoryBulletChart
          title={t.charts.winRateVsHistoryBulletTitle}
          rows={mapBreakdown}
          mapLabelOf={mapLabelOf}
          sessionLabel={t.charts.winRateVsHistorySession}
          historyLabel={t.charts.winRateVsHistoryHistory}
          parityLabel={t.charts.winRateVsHistoryBulletParity}
          zeroWinrateLabel={t.charts.winRateVsHistoryBulletZero}
        />
      )}
      {mapBreakdown.some(
        (r) => r.performance_avg !== undefined && r.historical_performance_avg !== undefined,
      ) && (
        <MapPerfVsHistoryChart
          title={t.charts.mapPerfVsHistoryTitle}
          rows={mapBreakdown}
          mapLabelOf={mapLabelOf}
          sessionLabel={t.charts.mapPerfVsHistorySession}
          historyLabel={t.charts.mapPerfVsHistoryHistory}
        />
      )}
      {mapHeatmap && mapHeatmap.players.length > 0 && mapHeatmap.maps_topn.length > 0 && (
        <SquadMapHeatmapChart
          title={t.heatmap.title}
          data={mapHeatmap}
          mapLabelOf={mapLabelOf}
          pieceLabels={{
            tier1: t.heatmap.pieceTier1,
            tier2: t.heatmap.pieceTier2,
            tier3: t.heatmap.pieceTier3,
            tier4: t.heatmap.pieceTier4,
            tier5: t.heatmap.pieceTier5,
          }}
          noScoreLabel={t.heatmap.noScore}
        />
      )}
      {sessionTimeline.length > 0 && (
        <SquadSessionTimelineChart
          title={t.timeline.title}
          rows={sessionTimeline}
          perfLabel={t.timeline.perf}
          winRateLabel={t.timeline.winRate}
          mmrLabel={t.timeline.teamMmr}
          perfAxisLabel={t.timeline.perfAxis}
          mmrAxisLabel={t.timeline.mmrAxis}
        />
      )}
    </div>
  )
}
