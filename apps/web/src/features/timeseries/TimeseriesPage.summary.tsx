/**
 * TimeseriesPage — onglet "Summary".
 *
 * Découpé depuis TimeseriesPage.tsx (audit #6 god-file split).
 * Contenu : outcome sequence + KDA trend + KDA density + avg life + assists +
 * top weapons + KDA trend value + perf session/week/month + map win-rate/perf.
 */
import { useMemo, useState } from 'react'

import { OutcomeSequenceTape, type OutcomePoint } from '@/components/charts/OutcomeSequenceTape'
import { asDominance } from '@/components/charts/outcomeSequence'
import { ReviewBadge } from '@/components/charts/ReviewBadge'
import { dominanceLabels as buildDominanceLabels } from '@/lib/narrative/dominance'
import { outcomeCodeToTapeValue } from '@/lib/outcome'
import { TimeseriesKdaTrend } from './TimeseriesKdaTrend'
import { TimeseriesKdaDensity } from './TimeseriesKdaDensity'
import { FragSunburst } from '@/components/charts/FragSunburst'
import { FragWeaponBreakdown } from '@/components/charts/FragWeaponBreakdown'
import { buildFragDetailBreakdown } from '@/components/charts/fragDetailBreakdown'
// Précision par arme : réutilise le graphe Synthesis (recoloré par classe + survol lié)
// — même choix que SessionFragCard. Import cross-feature durable déclaré
// (timeseries=>synthesis, cf. tools/lint-cross-feature-imports.mjs), analogue à
// session-detail=>synthesis.
import { SynthesisWeaponAccuracyChart } from '@/features/synthesis/SynthesisWeaponAccuracyChart'
import {
  TimeseriesAssistsTrend,
  TimeseriesAvgLifeTrend,
  TimeseriesKdaValueTrend,
} from './TimeseriesFormCharts'
import { TimeseriesFdaGapTrend } from './TimeseriesFdaGapTrend'
import { TimeseriesSessionPerformance } from './TimeseriesSquadAdapted'
import { WinRateVsHistoryBulletChart } from '@/features/squad/WinRateVsHistoryBulletChart'
import { MapPerfVsHistoryChart } from '@/features/squad/MapPerfVsHistoryChart'
import { useCapability } from '@/lib/capabilities/capabilities'
import { formatMessage } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import { useAppShellStore } from '@/stores/appShellStore'
import type { FieldMappingsResponse } from '@/lib/i18n/fieldMappings'
import type { TimeseriesPageResponse } from '@/lib/api/types'
import type { TimeseriesManifestKey } from '@/lib/i18n/generated/timeseries'

export interface OutcomeLabels {
  win: string
  loss: string
  tie: string
  dnf: string
  unknown: string
}

export interface TimeseriesSummaryTabProps {
  data: TimeseriesPageResponse
  t: (key: TimeseriesManifestKey) => string
  fieldMappings: FieldMappingsResponse | undefined
  outcomeLabels: OutcomeLabels
  mapLabelOf: (mapUI: string) => string
}

export function TimeseriesSummaryTab({
  data,
  t,
  fieldMappings,
  outcomeLabels,
  mapLabelOf,
}: TimeseriesSummaryTabProps) {
  const emptyMsg = t('timeseries.empty.no_data_description')
  // MMR par match indisponible (titre sans `team_mmr`, ex. Halo 5) → la série MMR
  // de « Performance par session » est retirée (data + axe + légende).
  const hasTeamMmr = useCapability('team_mmr')
  // FDA attendu natif (titre avec `expected_stats`, ex. Halo Infinite) → le chart
  // « Écart au FDA attendu » se place à DROITE du FDA (2 colonnes) au lieu d'un bloc
  // pleine largeur en dessous. Absente (Halo 5) → FDA seul, pleine largeur.
  const hasExpectedStats = useCapability('expected_stats')
  const soloPerf = data.solo_session_perf
  const soloGranularity: 'session' | 'week' | 'month' =
    soloPerf?.granularity === 'week' || soloPerf?.granularity === 'month'
      ? soloPerf.granularity
      : 'session'
  const soloPerfTitle =
    soloGranularity === 'week'
      ? t('timeseries.summary.solo_perf_week')
      : soloGranularity === 'month'
        ? t('timeseries.summary.solo_perf_month')
        : t('timeseries.summary.solo_perf_session')

  // Répartition des frags v2 (MÊME rendu que Match view / Sessions) : survol LIÉ
  // sunburst ↔ 2e graphe + résolveurs i18n partagés (fragsManifest, clé detail_title
  // « Outils de destruction »).
  const appLocale = useAppShellStore((s) => s.locale)
  // Libellés des drapeaux de dominance (bande de résultats) — table canonique
  // partagée avec la colonne Dominance de l'Explorateur. Mémoïsé : la bande
  // recalcule son option ECharts quand cette référence change.
  const dominanceLabels = useMemo(() => buildDominanceLabels(appLocale), [appLocale])
  const [hoveredClass, setHoveredClass] = useState<string | null>(null)
  const classLabel = (c: string) => formatMessage(fragsManifest, `frags.class.${c}` as never, appLocale)
  const roleLabel = (r: string) => formatMessage(fragsManifest, `frags.role.${r}` as never, appLocale)
  const detailTitle = formatMessage(fragsManifest, 'frags.charts.detail_title', appLocale)
  // Armes du registre normalisées (label/kills/classe) : alimentent « Outils de
  // destruction » (buildFragDetailBreakdown) ET la recoloration par classe du graphe
  // précision (l'entrée précision de l'API ne porte pas la classe).
  const topWeaponsMapped = (data.top_weapons ?? []).map((w) => ({ label: w.label, kills: w.kills, class: w.class }))
  const breakdown = buildFragDetailBreakdown(data.frag_distribution ?? null, topWeaponsMapped, { roleLabel, classLabel })
  // Précision par arme native (Halo 5) ; vide sur Infinite → 2e rangée = tendance FDA seule.
  const accuracy = data.weapon_accuracy ?? []
  const fdaTrend = (
    <TimeseriesKdaValueTrend
      title={fieldMappings?.fields['kda']?.label ?? 'FDA'}
      emptyMessage={emptyMsg}
      rows={data.match_rows ?? []}
      fdaLabel={fieldMappings?.fields['kda']?.label ?? 'FDA'}
      smoothingLabel={t('timeseries.summary.trend')}
    />
  )
  // Écart cumulé au FDA attendu (D1 + retouche UX 2026-07-24) — aire cumulée
  // signée réel vs attendu + courbe FDA attendu PAR MATCH (axe secondaire),
  // MÊME forme que Sessions (SessionFdaGapCumulative). Self-gate
  // `expected_stats` (null sur Halo 5). Hauteur alignée sur le FDA (360) pour
  // la disposition côte à côte.
  const fdaGapTrend = (
    <TimeseriesFdaGapTrend
      title={t('timeseries.summary.fda_gap_title')}
      emptyMessage={emptyMsg}
      height={360}
      rows={data.match_rows ?? []}
      locale={appLocale}
      labels={{
        series: t('timeseries.summary.fda_gap_series'),
        real: t('timeseries.summary.fda_gap_real'),
        expected: t('timeseries.summary.fda_gap_expected'),
        gap: t('timeseries.summary.fda_gap_gap'),
        avgCaption: t('timeseries.summary.fda_gap_avg_caption'),
        perMatch: t('timeseries.summary.fda_gap_per_match'),
      }}
    />
  )
  return (
    <div className="space-y-6">
      {/* Séquence des résultats — en haut. Libellé conservé + message court
          quand aucun match, au lieu de masquer le bloc. */}
      <div data-testid="timeseries-outcome-sequence">
        <p className="mb-1 flex items-center text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {t('timeseries.summary.outcome_sequence')}
          <ReviewBadge reviewKey="timeseries.outcome_tape" />
        </p>
        {(data.match_rows ?? []).length > 0 ? (
          <OutcomeSequenceTape
            matches={(data.match_rows ?? []).map<OutcomePoint>((r) => ({
              outcome: outcomeCodeToTapeValue(r.outcome),
              matchId: r.match_id,
              mode: r.playlist_name || undefined,
              // Absent (0/undefined, ex. Halo 5 sans timeline de score) → aucun
              // marqueur dessiné, aucun suffixe de tooltip.
              dominance: asDominance(r.dominance_flag),
            }))}
            labels={{
              win: outcomeLabels.win,
              loss: outcomeLabels.loss,
              tie: fieldMappings?.outcomes?.tie?.label ?? 'Égalité',
              dnf: outcomeLabels.unknown,
            }}
            dominanceLabels={dominanceLabels}
          />
        ) : (
          <p className="text-sm text-muted-foreground">{emptyMsg}</p>
        )}
      </div>

      {/* Évolution Frags / Morts (gauche) | Distribution FDA (droite) */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <TimeseriesKdaTrend
          title={t('timeseries.summary.kd_timeline_title')}
          emptyMessage={emptyMsg}
          rows={data.match_rows ?? []}
          labels={{
            kills: fieldMappings?.fields['kills']?.label ?? 'Frags',
            deaths: fieldMappings?.fields['deaths']?.label ?? 'Morts',
            yAxis: t('timeseries.summary.kd_yaxis'),
          }}
        />

        <TimeseriesKdaDensity
          title={t('timeseries.summary.kda_distribution_title')}
          emptyMessage={emptyMsg}
          buckets={data.distributions_tab.kda_buckets ?? []}
          rows={data.match_rows ?? []}
          labels={{
            density: t('timeseries.summary.density'),
            rug: fieldMappings?.fields['kda']?.label ?? 'FDA',
            xAxis: fieldMappings?.fields['kda']?.label ?? 'FDA',
            mean: t('timeseries.summary.mean'),
            median: t('timeseries.summary.median'),
          }}
        />
      </div>

      {/* Durée de vie moyenne (gauche) | Assistances (droite) */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <TimeseriesAvgLifeTrend
          title={fieldMappings?.fields['avg_life_seconds']?.label ?? 'Durée de vie moyenne'}
          emptyMessage={emptyMsg}
          rows={data.match_rows ?? []}
          lifeLabel={t('timeseries.summary.avg_life_axis')}
        />

        <TimeseriesAssistsTrend
          title={fieldMappings?.fields['assists']?.label ?? 'Assistances'}
          emptyMessage={emptyMsg}
          rows={data.match_rows ?? []}
          assistsLabel={fieldMappings?.fields['assists']?.label ?? 'Assistances'}
          smoothingLabel={t('timeseries.summary.trend')}
        />
      </div>

      {/* Répartition des frags v2 — MÊME rendu que Match view / Sessions : sunburst
          hiérarchique classe→rôle (compteur seul, légende à gauche, maxW 480) +
          « Outils de destruction » (armes du registre + détail mêlée/grenade/capacités),
          survol LIÉ. title-agnostic (Infinite sans capacités spartanes, Halo 5 avec).
          Rend null si aucun frag. */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <FragSunburst
          distribution={data.frag_distribution}
          externalHoveredClass={hoveredClass}
          onClassHover={setHoveredClass}
          hideCenterLabel
          maxWidthPx={480}
          legendSide="left"
          className="lg:col-span-2"
        />
        <FragWeaponBreakdown
          weapons={breakdown}
          title={detailTitle}
          hoveredClass={hoveredClass}
          onClassHover={setHoveredClass}
          className="lg:col-span-1"
          heightScale={1.1}
        />
      </div>

      {/* Précision par arme (Halo 5 natif, survol lié au sunburst) | Tendance FDA. Titre
          sans précision native (Infinite → weapon_accuracy vide) : la tendance FDA occupe
          seule la largeur (pas de colonne vide). */}
      {/* Précision par arme (Halo 5 natif) | FDA : la précision native est Halo-5-only
          et exclut `expected_stats` → le chart Écart (masqué sur H5) ne s'insère pas ici. */}
      {accuracy.length > 0 ? (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <SynthesisWeaponAccuracyChart
            weapons={accuracy}
            weaponKills={topWeaponsMapped}
            hoveredClass={hoveredClass}
            onClassHover={setHoveredClass}
            fillHeight
          />
          {fdaTrend}
        </div>
      ) : hasExpectedStats ? (
        // FDA | Écart au FDA attendu, côte à côte (D1 : l'Écart à droite du FDA).
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          {fdaTrend}
          {fdaGapTrend}
        </div>
      ) : (
        // Ni précision native ni FDA attendu → FDA seul, pleine largeur.
        <div className="grid grid-cols-1 gap-6">{fdaTrend}</div>
      )}

      {/* Performance solo par session/semaine/mois — agrégat backend
          sur tous les matchs solo (cross-session, granularité auto). */}
      <TimeseriesSessionPerformance
        title={soloPerfTitle}
        emptyMessage={emptyMsg}
        points={soloPerf?.points ?? []}
        granularity={soloGranularity}
        perfLabel={
          fieldMappings?.fields['performance_score']?.label ??
          t('timeseries.summary.perf_label')
        }
        winRateLabel={
          fieldMappings?.fields['win_rate']?.label ??
          t('timeseries.summary.win_rate_label')
        }
        mmrLabel={fieldMappings?.fields['team_mmr']?.label ?? 'MMR équipe'}
        showMmr={hasTeamMmr}
      />

      {/* Taux de victoire (gauche) | Performance par carte (droite) —
          réutilisation des wrappers Escouade (teammates.02 + .13). */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <WinRateVsHistoryBulletChart
          title={t('timeseries.summary.winrate_vs_history_title')}
          emptyMessage={emptyMsg}
          rows={data.map_breakdown ?? []}
          mapLabelOf={mapLabelOf}
          sessionLabel={t('timeseries.summary.session_label')}
          historyLabel={t('timeseries.summary.history_label')}
          parityLabel={t('timeseries.summary.parity_label')}
          zeroWinrateLabel={t('timeseries.summary.zero_winrate_label')}
        />
        <MapPerfVsHistoryChart
          title={t('timeseries.summary.mapperf_vs_history_title')}
          emptyMessage={emptyMsg}
          rows={data.map_breakdown ?? []}
          mapLabelOf={mapLabelOf}
          sessionLabel={t('timeseries.summary.session_label')}
          historyLabel={t('timeseries.summary.history_label')}
        />
      </div>
    </div>
  )
}
