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
import { OutcomeSequenceTape, type OutcomePoint } from '@/components/charts/OutcomeSequenceTape'
import { outcomeCodeToTapeValue } from '@/lib/outcome'
import { useSquadContext } from './SquadContext'
import { getSquadText } from './i18n'
import { WinRateVsHistoryBulletChart } from './WinRateVsHistoryBulletChart'
import { MapPerfVsHistoryChart } from './MapPerfVsHistoryChart'
import { SquadMapHeatmapChart } from './SquadMapHeatmapChart'
import { SquadSessionTimelineChart } from './SquadSessionTimelineChart'
import { SquadSynergyHistoryTable } from './SquadSynergyHistoryTable'
import { SquadImpactScoreboard } from './SquadImpactScoreboard'
import { MedalDigest } from './MedalDigest'

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
      {/* Graphes toujours montés : ChartCard affiche son état vide (titre +
          message) au lieu de faire disparaître le bloc quand mapBreakdown
          est vide ou sans champs de performance. */}
      <div className="grid grid-cols-2 gap-4">
        <WinRateVsHistoryBulletChart
          title={t.charts.winRateVsHistoryBulletTitle}
          emptyMessage={t.empty.noBlockData}
          rows={mapBreakdown}
          mapLabelOf={mapLabelOf}
          sessionLabel={t.charts.winRateVsHistorySession}
          historyLabel={t.charts.winRateVsHistoryHistory}
          parityLabel={t.charts.winRateVsHistoryBulletParity}
          zeroWinrateLabel={t.charts.winRateVsHistoryBulletZero}
          countsLabel={t.charts.winRateVsHistoryBulletCounts}
        />
        <MapPerfVsHistoryChart
          title={t.charts.mapPerfVsHistoryTitle}
          emptyMessage={t.empty.noBlockData}
          rows={mapBreakdown}
          mapLabelOf={mapLabelOf}
          sessionLabel={t.charts.mapPerfVsHistorySession}
          historyLabel={t.charts.mapPerfVsHistoryHistory}
        />
      </div>
      {/* Séquence des résultats : on garde le libellé + un message court quand
          il n'y a pas d'historique, au lieu de masquer le bloc. */}
      <div>
        <p className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {t.charts.outcomeSequenceTitle}
        </p>
        {matchHistory.length > 0 ? (
          <OutcomeSequenceTape
            // matchHistory arrive DESC (récent→ancien) ; on inverse pour afficher
            // du plus vieux au plus récent (gauche→droite).
            matches={[...matchHistory].reverse().map<OutcomePoint>((m) => ({
              outcome: outcomeCodeToTapeValue(m.outcome),
              matchId: m.match_id,
              map: m.map_ui || undefined,
              mode: m.mode_ui || m.pair_name || undefined,
            }))}
            labels={outcomeLabels}
          />
        ) : (
          <p className="text-sm text-muted-foreground">{t.empty.noBlockData}</p>
        )}
      </div>
      <SquadSynergyHistoryTable rows={matchHistory} playerSlug={playerSlug} />
      <SquadMapHeatmapChart
        title={t.heatmap.title}
        emptyMessage={t.empty.noBlockData}
        data={mapHeatmap && (mapHeatmap.players?.length ?? 0) > 0 && (mapHeatmap.maps_topn?.length ?? 0) > 0 ? mapHeatmap : undefined}
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
      <SquadSessionTimelineChart
        title={t.timeline.title}
        emptyMessage={t.empty.noBlockData}
        rows={sessionTimeline}
        perfLabel={t.timeline.perf}
        winRateLabel={t.timeline.winRate}
        mmrLabel={t.timeline.teamMmr}
        perfAxisLabel={t.timeline.perfAxis}
        mmrAxisLabel={t.timeline.mmrAxis}
      />
      {/* Sections non-graphes toujours montées : titre + état vide géré par le
          composant (cadre bordé / carte), au lieu de disparaître. */}
      <section className="space-y-3">
        <h3 className="text-base font-semibold text-foreground">{t.impact.title}</h3>
        <SquadImpactScoreboard
          matrix={pageData?.impact_matrix ?? { matches: [], players: [], cells: [], badge_ord: [] }}
        />
      </section>
      <section className="space-y-3">
        <h3 className="text-base font-semibold text-foreground">{t.medals.title}</h3>
        <MedalDigest
          entries={pageData?.medal_digest ?? []}
          mainPlayer={pageData?.main_player ?? playerSlug}
          t={t.medals}
        />
      </section>
    </div>
  )
}
