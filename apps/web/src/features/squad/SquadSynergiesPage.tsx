/**
 * SquadSynergiesPage — onglet Synergies de l'Escouade.
 *
 * Distingue 2 états vides diagnosticables :
 *  - no_selection : aucun coéquipier confirmé.
 *  - invalid_selection : confirmedGts > 0 mais selectedRows vide.
 */
import { useMemo } from 'react'

import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { useAppShellStore } from '@/stores/appShellStore'
import { useCapability } from '@/lib/capabilities/capabilities'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { OutcomeSequenceTape, type OutcomePoint } from '@/components/charts/OutcomeSequenceTape'
import { asDominance } from '@/components/charts/outcomeSequence'
import { ReviewBadge } from '@/components/charts/ReviewBadge'
import { dominanceLabels as buildDominanceLabels } from '@/lib/narrative/dominance'
import { outcomeCodeToTapeValue } from '@/lib/outcome'
import { useSquadContext } from './SquadContext'
import { getSquadText } from './i18n'
import { WinRateVsHistoryBulletChart } from './WinRateVsHistoryBulletChart'
import { MapPerfVsHistoryChart } from './MapPerfVsHistoryChart'
import { SquadMapHeatmapChart } from './SquadMapHeatmapChart'
import { SquadSessionTimelineChart } from './SquadSessionTimelineChart'
import { SquadAssistPairsTable } from './SquadAssistPairsTable'
import { SquadSynergyHistoryTable } from './SquadSynergyHistoryTable'
import { SquadImpactScoreboard } from './SquadImpactScoreboard'
import { MedalDigest } from './MedalDigest'
import { SquadFragSection } from './SquadFragSection'
import { SquadFdaGapCumulativeCard } from './SquadFdaGapCumulativeCard'
import { getSquadPlayerColors } from './colors'

export function SquadSynergiesPage() {
  const { selectedRows, confirmedGamertags, pageData, playerSlug } = useSquadContext()
  const { data: mappings } = useFieldMappings()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  // FDA attendu natif (Infinite déclare `expected_stats`, Halo 5 non) → gate
  // PARENT du card « Écart cumulé au FDA attendu » : pas de colonne vide dans la
  // rangée 1 de SquadFragSection (le card conserve son self-gate en profondeur).
  const hasExpectedStats = useCapability('expected_stats')
  // Libellés des drapeaux de dominance (bande de résultats) — table canonique
  // partagée avec la colonne Dominance de l'Explorateur. Mémoïsé : la bande
  // recalcule son option ECharts quand cette référence change. Déclaré AVANT les
  // retours anticipés (règle des hooks).
  const tapeDominanceLabels = useMemo(() => buildDominanceLabels(locale), [locale])

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

  // Section « frags » (relocalisée depuis Contributions) : mêmes couleurs/ordre
  // que SquadContributionsPage — main_player (casse serveur) puis coéquipiers,
  // restreint aux joueurs ayant des frag_classes ou une performance_series.
  const mainPlayerKey = pageData?.main_player ?? playerSlug
  const playerColors = getSquadPlayerColors(mainPlayerKey, confirmedGamertags)
  const playerOrder = [mainPlayerKey, ...confirmedGamertags].filter(
    (p) => pageData?.frag_classes?.[p] || pageData?.performance_series?.[p],
  )

  const mapAssets = mappings?.assets?.['map']
  const mapLabelOf = (mapUI: string) => mapAssets?.[mapUI]?.label ?? mapUI
  const mapBreakdown = pageData?.map_breakdown ?? []
  const matchHistory = pageData?.match_history ?? []
  const sessionTimeline = pageData?.session_timeline ?? []
  const mapHeatmap = pageData?.map_heatmap
  // assist_pairs n'est PAS comblé par un défaut : son absence est un ÉTAT (aucun match
  // de la sélection n'a d'assistance mesurée — dont le cas d'un titre sans décodeur de
  // film). Le bloc n'est alors pas monté du tout, plutôt que d'afficher un cadre vide
  // qui laisserait croire à une escouade sans entraide.
  const assistPairs = pageData?.assist_pairs

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
          title={
            <span className="flex items-center gap-1.5">
              {t.charts.winRateVsHistoryBulletTitle}
              <InfoTooltip content={t.charts.winRateVsHistoryBulletMapCountTooltip} />
            </span>
          }
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
        <p className="mb-1 flex items-center text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {t.charts.outcomeSequenceTitle}
          <ReviewBadge reviewKey="squad.outcome_tape" />
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
              // Absent (0/undefined, ex. Halo 5 sans timeline de score) → aucun
              // marqueur dessiné, aucun suffixe de tooltip.
              dominance: asDominance(m.dominance_flag),
            }))}
            labels={outcomeLabels}
            dominanceLabels={tapeDominanceLabels}
          />
        ) : (
          <p className="text-sm text-muted-foreground">{t.empty.noBlockData}</p>
        )}
      </div>
      <SquadSynergyHistoryTable rows={matchHistory} playerSlug={playerSlug} />
      {assistPairs && (
        <section>
          <p className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">
            {t.assists.title}
          </p>
          <SquadAssistPairsTable block={assistPairs} />
        </section>
      )}
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
      {/* Section « frags » (relocalisée depuis Contributions), juste avant « Impact
          des coéquipiers ». Rangée 1 : sur Infinite « Écart cumulé au FDA attendu »
          (gate `expected_stats`) à GAUCHE de « Répartition des frags » ; sur Halo 5
          « Répartition » | « Précision par rôle ». Puis « Outils de destruction ».
          Le card FDA porte ses propres pastilles KPI « écart moyen / match » et
          garde son self-gate capability (défense en profondeur). */}
      <SquadFragSection
        fragClassesByPlayer={pageData?.frag_classes ?? {}}
        weaponKills={pageData?.weapon_kills}
        weaponAccuracy={pageData?.weapon_accuracy}
        playerColors={playerColors}
        playerOrder={playerOrder}
        locale={locale}
        t={t}
        leftOfBreakdown={
          hasExpectedStats ? (
            <SquadFdaGapCumulativeCard
              rowsByPlayer={pageData?.performance_series ?? {}}
              playerOrder={playerOrder}
              colorByPlayer={playerColors}
              t={t}
              emptyMessage={t.empty.noBlockData}
            />
          ) : undefined
        }
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
