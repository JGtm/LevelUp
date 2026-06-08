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
import { OutcomeSequenceTape, type OutcomePoint, type OutcomeValue } from '@/components/charts/OutcomeSequenceTape'
import { useSquadContext } from './SquadContext'
import { getSquadText } from './i18n'
import { WinRateVsHistoryBulletChart } from './WinRateVsHistoryBulletChart'
import { MapPerfVsHistoryChart } from './MapPerfVsHistoryChart'
import { SquadMapHeatmapChart } from './SquadMapHeatmapChart'
import { SquadSessionTimelineChart } from './SquadSessionTimelineChart'
import { SquadSynergyHistoryTable } from './SquadSynergyHistoryTable'
import { SquadImpactScoreboard } from './SquadImpactScoreboard'
import { MedalDigest } from './MedalDigest'

function outcomeNumToValue(n: number): OutcomeValue {
  if (n === 2) return 'win'
  if (n === 3) return 'loss'
  if (n === 1) return 'tie'
  return 'dnf'
}

export function SquadSynergiesPage() {
  const { selectedRows, confirmedGamertags, pageData, playerSlug } = useSquadContext()
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
  const matchHistory = pageData?.match_history ?? []
  const sessionTimeline = pageData?.session_timeline ?? []
  const mapHeatmap = pageData?.map_heatmap

  const outcomeLabels = {
    win: mappings?.outcomes?.['win']?.label ?? t.history.outcomeLabel.win,
    loss: mappings?.outcomes?.['loss']?.label ?? t.history.outcomeLabel.loss,
    tie: mappings?.outcomes?.['tie']?.label ?? t.history.outcomeLabel.draw,
    dnf: mappings?.outcomes?.['dnf']?.label ?? t.history.outcomeLabel.dnf,
  }

  return (
    <div className="space-y-4">
      {mapBreakdown.length > 0 && (
        <div className="grid grid-cols-2 gap-4">
          <WinRateVsHistoryBulletChart
            title={t.charts.winRateVsHistoryBulletTitle}
            rows={mapBreakdown}
            mapLabelOf={mapLabelOf}
            sessionLabel={t.charts.winRateVsHistorySession}
            historyLabel={t.charts.winRateVsHistoryHistory}
            parityLabel={t.charts.winRateVsHistoryBulletParity}
            zeroWinrateLabel={t.charts.winRateVsHistoryBulletZero}
          />
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
        </div>
      )}
      {matchHistory.length > 0 && (
        <div>
          <p className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">
            {t.charts.outcomeSequenceTitle}
          </p>
          <OutcomeSequenceTape
            matches={matchHistory.map<OutcomePoint>((m) => ({
              outcome: outcomeNumToValue(m.outcome),
              matchId: m.match_id,
              map: m.map_ui || undefined,
              mode: m.mode_ui || m.pair_name || undefined,
            }))}
            labels={outcomeLabels}
          />
        </div>
      )}
      {matchHistory.length > 0 && (
        <SquadSynergyHistoryTable rows={matchHistory} playerSlug={playerSlug} />
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
      {pageData?.impact_matrix && (
        <section className="space-y-3">
          <h3 className="text-base font-semibold text-foreground">{t.impact.title}</h3>
          <SquadImpactScoreboard matrix={pageData.impact_matrix} />
        </section>
      )}
      {pageData?.medal_digest && pageData.medal_digest.length > 0 && (
        <section className="space-y-3">
          <h3 className="text-base font-semibold text-foreground">{t.medals.title}</h3>
          <MedalDigest
            entries={pageData.medal_digest}
            mainPlayer={pageData.main_player ?? playerSlug}
            t={t.medals}
          />
        </section>
      )}
    </div>
  )
}
