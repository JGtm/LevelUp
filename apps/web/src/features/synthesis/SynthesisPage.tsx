/**
 * SynthesisPage --- Vue synthese / bilan periodique (Slice 7).
 * Types ref: SynthesisPageResponse, SynthesisKPIs, ComparisonMetricItem, HeatmapCell, TopWeekItem
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { useSynthesisPage } from './queries'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { OutcomeBar } from '@/components/ui/outcome-bar'
import { ProportionalBar } from '@/components/ui/proportional-bar'
import { SynthesisRelationsPreview } from './SynthesisRelationsPreview'
import { SynthesisKillTypesDonut } from './SynthesisKillTypesDonut'
import { SynthesisWeaponKillsChart } from './SynthesisWeaponKillsChart'
import { SynthesisOutcomesByGroupChart } from './SynthesisOutcomesByGroupChart'
import { SynthesisTopWeeksChart } from './SynthesisTopWeeksChart'
import { SynthesisHeatmapChart } from './SynthesisHeatmapChart'
import { SynthesisBipolaireChart } from './SynthesisBipolaireChart'
import { PeriodePill, SaisonPill, DEFAULT_PERIOD } from '@/components/shell/FilterOmnibar'
import { useActiveSeason, seasonToPeriod } from '@/features/squad/useActiveSeason'
import type {
  PeriodInput,
  SynthesisDetailedStats,
  SynthesisOverview,
  SynthesisQueryRequest,
  SynthesisWeaponKillEntry,
} from '@/lib/api/types'

// ─── Sous-composants ──────────────────────────────────────────────────────────

// ─── Bloc 1 — Vue d'ensemble (D4) ─────────────────────────────────────────────

function AccentCard({ label, value, accent }: { label: string; value: string; accent: SemanticToken }) {
  return (
    <div className="rounded-lg overflow-hidden border">
      <div className="h-[3px]" style={{ backgroundColor: tokenCssVar(accent) }} />
      <div className="p-3">
        <span className="text-xs text-muted-foreground block">{label}</span>
        <span className="text-xl font-bold">{value}</span>
      </div>
    </div>
  )
}

interface SynthesisOverviewSectionProps {
  overview: SynthesisOverview
  detailedStats?: SynthesisDetailedStats
  topWeaponKills?: SynthesisWeaponKillEntry[]
}
function SynthesisOverviewSection({ overview, detailedStats, topWeaponKills }: SynthesisOverviewSectionProps) {
  const { data: fieldMappings } = useFieldMappings()
  const labelOf = (key: string): string =>
    fieldMappings?.fields[key]?.label ?? key
  // P4.4 (revue 2026-04-29 B3) : K/D agrégé canonique sum/sum depuis l'API.
  const kd = overview.total_kdr != null
    ? overview.total_kdr.toFixed(2)
    : overview.total_deaths > 0
    ? (overview.total_kills / overview.total_deaths).toFixed(2)
    : String(overview.total_kills)

  const hasIncidents = detailedStats != null &&
    (detailedStats.total_betrayals > 0 || detailedStats.total_suicides > 0)

  return (
    <Card>
      <CardHeader><CardTitle>Vue d'ensemble</CardTitle></CardHeader>
      <CardContent>

        {/* Statistiques détaillées intégrées */}
        {detailedStats && (
          <div className="space-y-5">

            <div>
              {/* Donut + 3 cartes côte à côte */}
              <div className="flex gap-4 items-stretch">
                <div className="flex-1 min-w-0">
                  <SynthesisKillTypesDonut stats={detailedStats} height={260} />
                </div>
                <div className="flex flex-col gap-3 w-[22rem] shrink-0">

                  {/* Taux de victoire */}
                  <div className="flex-1 flex flex-col items-center justify-center rounded-lg border border-border bg-card px-3 py-2 text-center">
                    <p className="text-xs text-muted-foreground">{labelOf('win_rate')}</p>
                    <p className="text-xl font-bold text-primary">{`${(overview.win_rate * 100).toFixed(0)}%`}</p>
                    <div className="mt-1.5 w-full">
                      <OutcomeBar wins={overview.total_wins} draws={overview.total_ties} dnfs={overview.total_dnf} losses={overview.total_losses} />
                    </div>
                    <div className="mt-1 flex justify-center gap-2 text-xs font-semibold tabular-nums">
                      <span style={{ color: tokenCssVar('outcome-win') }}>{overview.total_wins}</span>
                      {overview.total_ties > 0 && <span style={{ color: tokenCssVar('outcome-draw') }}>{overview.total_ties}</span>}
                      {overview.total_dnf > 0 && <span style={{ color: tokenCssVar('outcome-dnf') }}>{overview.total_dnf}</span>}
                      <span style={{ color: tokenCssVar('outcome-loss') }}>{overview.total_losses}</span>
                    </div>
                  </div>

                  {/* FDA */}
                  <div className="flex-1 flex flex-col items-center justify-center rounded-lg border border-border bg-card px-3 py-2 text-center">
                    <p className="text-xs text-muted-foreground">{labelOf('kd_ratio')}</p>
                    <p className="text-xl font-bold text-primary">{kd}</p>
                    <div className="mt-1.5 w-full">
                      <ProportionalBar segments={[
                        { value: overview.total_kills,   color: 'outcome-win' },
                        { value: overview.total_assists,  color: 'outcome-draw' },
                        { value: overview.total_deaths,   color: 'outcome-loss' },
                      ]} />
                    </div>
                    <div className="mt-1 flex justify-center gap-2 text-xs font-semibold tabular-nums">
                      <span style={{ color: tokenCssVar('outcome-win') }}>{overview.total_kills}</span>
                      <span style={{ color: tokenCssVar('outcome-draw') }}>{overview.total_assists}</span>
                      <span style={{ color: tokenCssVar('outcome-loss') }}>{overview.total_deaths}</span>
                    </div>
                  </div>

                  {/* Incidents */}
                  {hasIncidents && (
                    <div className="flex-1 flex flex-col items-center justify-center rounded-lg border border-border bg-card px-3 py-2 text-center">
                      <p className="text-xs text-muted-foreground">Incidents</p>
                      <div className="mt-1.5 w-full">
                        <ProportionalBar segments={[
                          { value: detailedStats.total_betrayals, color: 'warning' },
                          { value: detailedStats.total_suicides,  color: 'outcome-dnf' },
                        ]} />
                      </div>
                      <div className="mt-1 flex justify-center gap-2 text-xs font-semibold tabular-nums">
                        {detailedStats.total_betrayals > 0 && (
                          <span style={{ color: tokenCssVar('warning') }}>{detailedStats.total_betrayals} trahisons</span>
                        )}
                        {detailedStats.total_suicides > 0 && (
                          <span style={{ color: tokenCssVar('outcome-dnf') }}>{detailedStats.total_suicides} suicides</span>
                        )}
                      </div>
                    </div>
                  )}

                </div>
              </div>

              {/* Tir / Dégâts / Fun à gauche, Frags par arme à droite */}
              <div className="flex gap-4 mt-4 items-stretch">
                <div className="flex flex-col gap-4 w-[21rem] shrink-0">
                  {(overview.longest_win_streak ?? 0) > 1 && (
                    <AccentCard label="Victoires consécutives (max)" value={String(overview.longest_win_streak)} accent="outcome-win" />
                  )}
                  <div className="grid grid-cols-2 gap-2">
                    {overview.best_kills_match != null && (
                      <AccentCard label={`Meilleur match · ${labelOf('kills').toLowerCase()}`} value={String(overview.best_kills_match)} accent="outcome-win" />
                    )}
                    <AccentCard label="Folie meurtrière (max)" value={detailedStats.max_killing_spree.toLocaleString('fr-FR')} accent="outcome-win" />
                  </div>

                  <div className="grid grid-cols-2 gap-2">
                    <AccentCard label="Frags parfaits" value={detailedStats.total_perfect_kills.toLocaleString('fr-FR')} accent="perf-tier-3" />
                    <AccentCard label="Tirs à la tête" value={detailedStats.total_headshot_kills.toLocaleString('fr-FR')} accent="perf-tier-2" />
                  </div>

                  <div>
                    <div className="grid grid-cols-2 gap-2">
                      <AccentCard label="Tirs effectués" value={detailedStats.total_shots_fired.toLocaleString('fr-FR')} accent="info" />
                      <AccentCard label="Tirs au but"    value={detailedStats.total_shots_hit.toLocaleString('fr-FR')}   accent="info" />
                      {detailedStats.total_shots_fired > 0 && (
                        <AccentCard
                          label="Précision brute"
                          value={`${((detailedStats.total_shots_hit / detailedStats.total_shots_fired) * 100).toFixed(1)}%`}
                          accent="info"
                        />
                      )}
                    </div>
                  </div>

                  <div>
                    <div className="grid grid-cols-2 gap-2">
                      <AccentCard label="Dégâts infligés" value={Math.round(detailedStats.total_damage_dealt).toLocaleString('fr-FR')} accent="outcome-win" />
                      <AccentCard label="Dégâts reçus"    value={Math.round(detailedStats.total_damage_taken).toLocaleString('fr-FR')} accent="outcome-loss" />
                    </div>
                  </div>

                  {(detailedStats.total_vehicles_destroyed > 0 || detailedStats.total_hijacks > 0) && (
                    <div>
                      <div className="grid grid-cols-2 gap-2">
                        {detailedStats.total_vehicles_destroyed > 0 && (
                          <AccentCard label="Véhicules détruits" value={detailedStats.total_vehicles_destroyed.toLocaleString('fr-FR')} accent="warning" />
                        )}
                        {detailedStats.total_hijacks > 0 && (
                          <AccentCard label="Hijacks" value={detailedStats.total_hijacks.toLocaleString('fr-FR')} accent="chart-series-4" />
                        )}
                      </div>
                    </div>
                  )}
                </div>

                {topWeaponKills && topWeaponKills.length > 0 && (
                  <div className="flex-1 min-w-0 flex flex-col">
                    <SynthesisWeaponKillsChart weapons={topWeaponKills} fillHeight />
                  </div>
                )}
              </div>
            </div>

          </div>
        )}

      </CardContent>
    </Card>
  )
}



// ─── Page principale ──────────────────────────────────────────────────────────

export function SynthesisPage() {
  const { playerSlug } = useParams({ from: '/players/$playerSlug/synthesis' })
  const { filterContext } = useGlobalFilterStore()

  // ── Filtres période / saison (pending → committed) ──────────────────────────
  const [pendingPeriod, setPendingPeriod] = useState<PeriodInput>(DEFAULT_PERIOD)
  const [committedPeriod, setCommittedPeriod] = useState<PeriodInput>(DEFAULT_PERIOD)
  const [activePopover, setActivePopover] = useState<'periode' | 'saison' | null>(null)

  const togglePopover = (which: 'periode' | 'saison') =>
    setActivePopover((cur) => (cur === which ? null : which))
  const closeAll = () => setActivePopover(null)

  const { seasons, activeSeason } = useActiveSeason(pendingPeriod)

  const isDirty =
    pendingPeriod.start_date !== committedPeriod.start_date ||
    pendingPeriod.end_date !== committedPeriod.end_date

  function handleAnalyser() {
    setCommittedPeriod(pendingPeriod)
    closeAll()
  }

  const hasPeriod = !!(committedPeriod.start_date || committedPeriod.end_date)
  const request: SynthesisQueryRequest = {
    filters: filterContext,
    period: 'all',
    start_date: hasPeriod ? committedPeriod.start_date : undefined,
    end_date: hasPeriod ? committedPeriod.end_date : undefined,
  }
  const { data, isLoading, isError, error } = useSynthesisPage(playerSlug, committedPeriod, request)

  const comparisonMetrics = data?.comparison_metrics ?? []
  const topWeeks = data?.top_weeks ?? []
  const { data: fieldMappings } = useFieldMappings()
  const labelOf = (key: string): string => fieldMappings?.fields[key]?.label ?? key

  if (isLoading) return null
  if (isError) return <div className="p-8 text-center text-destructive">Erreur : {String(error)}</div>
  if (!data) {
    return (
      <div className="px-6">
        <EmptyStateCard
          title="Synthèse indisponible"
          description="Aucune charge utile n'a été renvoyée pour cette page. Vérifie les agrégats solo/escouade et le contexte de filtres."
        />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      {/* ─── Barre période / saison ─────────────────────────────────────────── */}
      <div className="sticky top-0 z-20 border-b border-border" style={{ background: 'var(--background)' }}>
        <div className="flex min-h-10 items-center gap-1.5 px-4 py-1.5">
          {seasons.length > 0 && (
            <SaisonPill
              open={activePopover === 'saison'}
              onToggle={() => togglePopover('saison')}
              onClose={closeAll}
              seasons={seasons}
              activeSeason={activeSeason}
              onSelectSeason={(s) => setPendingPeriod(seasonToPeriod(s))}
              onClear={() => setPendingPeriod(DEFAULT_PERIOD)}
            />
          )}
          <PeriodePill
            open={activePopover === 'periode'}
            onToggle={() => togglePopover('periode')}
            onClose={closeAll}
            period={pendingPeriod}
            onSetPeriod={setPendingPeriod}
          />
          <div className="flex-1" />
          <button
            type="button"
            onClick={handleAnalyser}
            className={[
              'shrink-0 rounded-md px-2.5 py-1 text-xs font-medium transition-colors',
              isDirty
                ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                : 'border border-input bg-background text-muted-foreground hover:bg-muted',
            ].join(' ')}
          >
            Analyser
          </button>
          {hasPeriod && (
            <button
              type="button"
              onClick={() => { setPendingPeriod(DEFAULT_PERIOD); setCommittedPeriod(DEFAULT_PERIOD) }}
              className="shrink-0 text-xs text-muted-foreground transition-colors hover:text-destructive"
              title="Réinitialiser la période"
            >
              ↺
            </button>
          )}
        </div>
      </div>

      {/* Bloc 1 — Vue d'ensemble D4 */}
      {data.overview && (
        <SynthesisOverviewSection
          overview={data.overview}
          detailedStats={data.detailed_stats}
          topWeaponKills={data.top_weapon_kills}
        />
      )}

      {/* synthesis.05 — Bipolaire Solo vs Escouade */}
      <SynthesisBipolaireChart
        title="Comparaison Solo / Escouade"
        metrics={comparisonMetrics}
        fieldLabels={comparisonMetrics.map((m) => labelOf(m.label))}
      />

      {/* synthesis.03 — Heatmap activité jour × heure */}
      <SynthesisHeatmapChart title="Activité par jour et heure" cells={data.heatmap_data ?? []} />

      {/* synthesis.04 — Top semaines */}
      {topWeeks.length > 0 && (
        <SynthesisTopWeeksChart title="Matchs Top vs Total par semaine" weeks={topWeeks} />
      )}

      {/* Relations / Rivalités D6 */}
      {data.rivalries_preview &&
        (data.rivalries_preview.top_teammates.length > 0 || data.rivalries_preview.top_enemies.length > 0) && (
          <SynthesisRelationsPreview playerSlug={playerSlug} preview={data.rivalries_preview} />
        )}

      {/* synthesis.01 + synthesis.02 — Répartition carte / mode D7 */}
      {data.breakdowns && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {data.breakdowns.top_maps.length > 0 && (
            <SynthesisOutcomesByGroupChart
              title="Par carte"
              entries={data.breakdowns.top_maps.map((m) => ({
                name: m.map_name,
                wins: m.wins,
                losses: m.losses,
                ties: m.ties,
                unfinished: m.unfinished,
              }))}
              height={360}
            />
          )}
          {data.breakdowns.top_modes.length > 0 && (
            <SynthesisOutcomesByGroupChart
              title="Par mode"
              entries={data.breakdowns.top_modes.map((m) => ({
                name: m.mode_name,
                wins: m.wins,
                losses: m.losses,
                ties: m.ties,
                unfinished: m.unfinished,
              }))}
              height={360}
            />
          )}
        </div>
      )}
    </div>
  )
}
