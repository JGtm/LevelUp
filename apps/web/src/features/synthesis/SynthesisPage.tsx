/* eslint-disable max-lines -- 2026-09-06 (lot v2 D.11, decision utilisateur 4) : hors perimetre du lot D (modele web du rejeu) : l'exemption DATE la dette, elle ne l'absout pas — le decoupage revient au lot qui touchera ce fichier. */
/**
 * SynthesisPage --- Vue synthese / bilan periodique (Slice 7).
 * Types ref: SynthesisPageResponse, SynthesisKPIs, ComparisonMetricItem, HeatmapCell, TopWeekItem
 */
import { useMemo, useState, type ReactNode } from 'react'
import { useParams } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { useSynthesisPage } from './queries'
import { useFiltersPreview } from '@/features/filters/queries'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { OutcomeBar } from '@/components/ui/outcome-bar'
import { ProportionalBar } from '@/components/ui/proportional-bar'
import { useCapability } from '@/lib/capabilities/capabilities'
import { FragSunburst } from '@/components/charts/FragSunburst'
import { FragWeaponBreakdown } from '@/components/charts/FragWeaponBreakdown'
import { SynthesisWeaponAccuracyChart } from './SynthesisWeaponAccuracyChart'
import { useSynthesisFragCharts } from './useSynthesisFragCharts'
import { SynthesisOutcomesByGroupChart } from './SynthesisOutcomesByGroupChart'
import { SynthesisTopWeeksChart } from './SynthesisTopWeeksChart'
import { SynthesisHeatmapChart } from './SynthesisHeatmapChart'
import { SynthesisBipolaireChart } from './SynthesisBipolaireChart'
import { CombatYieldDisplay } from '@/components/ui/combat-yield-display'
import { Badge } from '@/components/ui/badge'
import { offensiveLabel, defensiveLabel, activityLabel } from '@/features/_shared/combatProfileLabels'
import { PeriodePill, SaisonPill, DEFAULT_PERIOD } from '@/components/shell/FilterOmnibar'
import { useActiveSeason, seasonToPeriod } from '@/features/squad/useActiveSeason'
import { MultiSelectFilter, type MultiSelectOption } from '@/features/explorer/MultiSelectFilter'
import { ExperienceDropdown, type Experience } from '@/features/_shared/ExperienceDropdown'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'
import { hijacksLabelKey } from './hijacksLabel'
import type { ManifestLocale } from '@/lib/i18n/format'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import type {
  BestMatchRef,
  CascadeInput,
  CombatProfileBlock,
  FilterContextInput,
  FragDistribution,
  PeriodInput,
  SynthesisDetailedStats,
  SynthesisOverview,
  SynthesisQueryRequest,
  SynthesisWeaponKillEntry,
  SynthesisWeaponAccuracyEntry,
  ObjectiveAggregate,
} from '@/lib/api/types'
import { formatDurationMMSS } from '@/lib/formatters/duration'
// EXPERIENCE_TO_CASCADE + setsEqual : source unique partagée avec useLocalFilterBar (H3).
import { EXPERIENCE_TO_CASCADE, setsEqual } from '@/features/_shared/experienceCascade'

// ─── Sous-composants ──────────────────────────────────────────────────────────

function formatTimePlayed(seconds: number): string {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const parts: string[] = []
  if (d > 0) parts.push(`${d}j`)
  if (h > 0 || d > 0) parts.push(`${h}h`)
  parts.push(`${m}m`)
  return parts.join(' ')
}

// Sous-titre de section (type 6 du catalogue) : petit uppercase semibold + filet 1px.
function SectionSubtitle({ children }: { children: ReactNode }) {
  return (
    <div className="space-y-2">
      <p className="text-3xs font-semibold uppercase tracking-label-md text-foreground/90">{children}</p>
      <div className="h-px w-full rounded-full bg-border" />
    </div>
  )
}

// ─── Bloc 1 — Vue d'ensemble (D4) ─────────────────────────────────────────────

// Libellés des descripteurs : source unique @/features/_shared/combatProfileLabels.

function CombatProfileInlineRow({ combatProfile, locale }: { combatProfile: CombatProfileBlock; locale: ManifestLocale }) {
  const t = (key: keyof typeof synthesisManifest) => synthesisManifest[key][locale]
  const hasStyles =
    combatProfile.style_offensive != null ||
    combatProfile.style_defensive != null ||
    combatProfile.style_activity != null
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
      <span className="text-xs text-muted-foreground shrink-0">
        {combatProfile.match_count} {t('synthesis.combat_profile.matches_analyzed')}
      </span>
      {hasStyles && (
        <>
          {combatProfile.style_offensive && (
            <Badge variant="outline" className="text-xs">
              {offensiveLabel(combatProfile.style_offensive, locale)}
            </Badge>
          )}
          {combatProfile.style_defensive && (
            <Badge variant="outline" className="text-xs">
              {defensiveLabel(combatProfile.style_defensive, locale)}
            </Badge>
          )}
          {combatProfile.style_activity && (
            <Badge variant="secondary" className="text-xs">
              {activityLabel(combatProfile.style_activity, locale)}
            </Badge>
          )}
        </>
      )}
      <CombatYieldDisplay
        className="ml-auto w-full sm:max-w-[560px]"
        label={t('synthesis.combat_profile.yield_resistance')}
        offensiveConversion={combatProfile.avg_oc}
        defensiveResistance={combatProfile.avg_dr}
        dmgPerKill={combatProfile.dmg_per_kill}
        dmgPerDeath={combatProfile.dmg_per_death}
      />
    </div>
  )
}

interface AccentCardProps {
  label: string
  value: string
  accent: SemanticToken
  onOpenMatch?: () => void
  openMatchLabel?: string
}
function AccentCard({ label, value, accent, onOpenMatch, openMatchLabel }: AccentCardProps) {
  return (
    <div className="rounded-lg overflow-hidden border border-border bg-card">
      <div className="h-[3px]" style={{ backgroundColor: tokenCssVar(accent) }} />
      <div className="p-3">
        <div className="flex items-center gap-1.5">
          <span className="text-xs text-muted-foreground block">{label}</span>
          {onOpenMatch && (
            <button
              type="button"
              onClick={onOpenMatch}
              aria-label={openMatchLabel}
              className="group flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
            >
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="currentColor" className="h-3 w-3 opacity-50 group-hover:opacity-100 transition-opacity" aria-hidden="true">
                <path d="M6.22 8.72a.75.75 0 0 0 1.06 1.06l5.22-5.22v1.69a.75.75 0 0 0 1.5 0v-3.5a.75.75 0 0 0-.75-.75h-3.5a.75.75 0 0 0 0 1.5h1.69L6.22 8.72Z" />
                <path d="M3.5 6.75c0-.69.56-1.25 1.25-1.25H7A.75.75 0 0 0 7 4H4.75A2.75 2.75 0 0 0 2 6.75v4.5A2.75 2.75 0 0 0 4.75 14h4.5A2.75 2.75 0 0 0 12 11.25V9a.75.75 0 0 0-1.5 0v2.25c0 .69-.56 1.25-1.25 1.25h-4.5c-.69 0-1.25-.56-1.25-1.25v-4.5Z" />
              </svg>
            </button>
          )}
        </div>
        <span className="text-xl font-bold">{value}</span>
      </div>
    </div>
  )
}

interface SynthesisOverviewSectionProps {
  overview: SynthesisOverview
  detailedStats?: SynthesisDetailedStats
  topWeaponKills?: SynthesisWeaponKillEntry[]
  fragDistribution?: FragDistribution | null
  weaponAccuracy?: SynthesisWeaponAccuracyEntry[]
  combatProfile?: CombatProfileBlock | null
  objectiveStats?: ObjectiveAggregate | null
  playerSlug: string
}
function SynthesisOverviewSection({ overview, detailedStats, topWeaponKills, fragDistribution, weaponAccuracy, combatProfile, objectiveStats, playerSlug }: SynthesisOverviewSectionProps) {
  const { data: fieldMappings } = useFieldMappings()
  const labelOf = (key: string): string =>
    fieldMappings?.fields[key]?.label ?? key
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const t = (key: keyof typeof synthesisManifest) => synthesisManifest[key][locale]
  // Libellé « vol à la tire » PAR TITRE : Halo 5 = « Vol à la tire », Infinite (défaut)
  // = « Dépositaire ». Sélection par currentTitleSlug (précédent NavL2 : slug== toléré
  // côté front pour un simple choix de libellé, jamais côté Go). Logique extraite en
  // helper pur (hijacksLabelKey) pour testabilité hors rendu.
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const hijacksLabel = t(hijacksLabelKey(currentTitleSlug))
  // Format nombre locale-aware (séparateurs "12 345" FR / "12,345" EN) — I2.
  const numLoc = intlLocale(locale)
  const navigateToMatch = useNavigateToMatch(playerSlug)
  const openMatchLabel = t('synthesis.kpi.open_match')
  // Helper : crée le handler onOpenMatch d'une carte si le ref backend est présent.
  // Les match_ids des cartes "Top X" servent à un usage one-shot (pas de matchIds[] groupé).
  const handlerFor = (ref: BestMatchRef | null | undefined) =>
    ref ? () => navigateToMatch(ref.match_id) : undefined
  // P4.4 (revue 2026-04-29 B3) : K/D agrégé canonique sum/sum depuis l'API.
  const kd = overview.total_kdr != null
    ? overview.total_kdr.toFixed(2)
    : overview.total_deaths > 0
    ? (overview.total_kills / overview.total_deaths).toFixed(2)
    : String(overview.total_kills)

  const hasIncidents = detailedStats != null &&
    (detailedStats.total_betrayals > 0 || detailedStats.total_suicides > 0)

  // Mécaniques natives Halo 5 (assassinats + compétences spartiate) : section gatée
  // par la capability du titre (masquée pour Infinite qui ne les fournit pas).
  const hasKillMechanics = useCapability('native_kill_mechanics')

  // Dégâts reçus : indispo pour Halo 5 (total_damage_taken non-nullable = 0 →
  // afficherait un « 0 » trompeur). Capability-gated → retrait silencieux de la
  // carte quand le titre ne fournit pas la donnée.
  const hasDamageTaken = useCapability('damage_taken')

  // Précision par arme : Halo 5 natif (table weapon_accuracy). Capability-gated →
  // le graphe est masqué pour les titres qui ne fournissent pas la donnée (Infinite).
  const hasWeaponAccuracy = useCapability('weapon_accuracy')
  // KPI objectifs (CTF/Zones/Oddball) : gated capability + data-driven (KPI > 0 seulement).
  const hasObjectiveStats = useCapability('objective_stats')

  // Graphes frags : état survol LIÉ + « Détails des frags » + coach, remontés ici car le
  // sunburst (rangée 1) et le breakdown (rangée 2) sont sur DEUX rangées distinctes.
  const frags = useSynthesisFragCharts(fragDistribution, topWeaponKills)

  return (
    <section className="space-y-3">
      <header><h3 className="text-base font-semibold text-foreground">{t('synthesis.section.overview')}</h3></header>

        {/* Profil de combat — sous-titre (type 6) + card pleine largeur. */}
        {combatProfile != null && (
          <div className="space-y-2">
            <SectionSubtitle>{t('synthesis.combat_profile.title')}</SectionSubtitle>
            <div className="rounded-lg border border-border bg-card p-4">
              <CombatProfileInlineRow combatProfile={combatProfile} locale={locale} />
            </div>
          </div>
        )}

        {/* Meilleures stats — sous-titre (type 6) + top records cliquables. La colonne
            accuracy est déjà en 0..100 (cf. scoreboard / table de session) → pas de ×100. */}
        {(overview.best_kills_match != null || detailedStats != null || overview.best_kda_ref || overview.best_perf_ref || overview.best_accuracy_ref || overview.best_damage_ref || overview.best_headshots_ref || overview.best_personal_score_ref) && (
          <div className="space-y-2">
            <SectionSubtitle>{t('synthesis.section.top_stats')}</SectionSubtitle>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4 xl:grid-cols-8">
              {overview.best_kills_match != null && (
                <AccentCard
                  label={`${labelOf('kills')} (max)`}
                  value={String(overview.best_kills_match)}
                  accent="outcome-win"
                  onOpenMatch={handlerFor(overview.best_kills_ref)}
                  openMatchLabel={openMatchLabel}
                />
              )}
              {detailedStats?.max_killing_spree != null && (
                <AccentCard
                  label={t('synthesis.kpi.killing_spree_max')}
                  value={detailedStats.max_killing_spree.toLocaleString(numLoc)}
                  accent="outcome-win"
                  onOpenMatch={handlerFor(overview.best_killing_spree_ref)}
                  openMatchLabel={openMatchLabel}
                />
              )}
              {overview.best_kda_ref && (
                <AccentCard
                  label={t('synthesis.kpi.top_kda')}
                  value={overview.best_kda_ref.value.toFixed(2)}
                  accent="perf-tier-3"
                  onOpenMatch={handlerFor(overview.best_kda_ref)}
                  openMatchLabel={openMatchLabel}
                />
              )}
              {overview.best_perf_ref && (
                <AccentCard
                  label={t('synthesis.kpi.top_perf')}
                  value={Math.round(overview.best_perf_ref.value).toLocaleString(numLoc)}
                  accent="perf-tier-3"
                  onOpenMatch={handlerFor(overview.best_perf_ref)}
                  openMatchLabel={openMatchLabel}
                />
              )}
              {overview.best_accuracy_ref && (
                <AccentCard
                  label={t('synthesis.kpi.top_accuracy')}
                  value={`${overview.best_accuracy_ref.value.toFixed(1)}%`}
                  accent="info"
                  onOpenMatch={handlerFor(overview.best_accuracy_ref)}
                  openMatchLabel={openMatchLabel}
                />
              )}
              {overview.best_damage_ref && (
                <AccentCard
                  label={t('synthesis.kpi.top_damage')}
                  value={Math.round(overview.best_damage_ref.value).toLocaleString(numLoc)}
                  accent="outcome-win"
                  onOpenMatch={handlerFor(overview.best_damage_ref)}
                  openMatchLabel={openMatchLabel}
                />
              )}
              {overview.best_headshots_ref && (
                <AccentCard
                  label={t('synthesis.kpi.top_headshots')}
                  value={Math.round(overview.best_headshots_ref.value).toLocaleString(numLoc)}
                  accent="perf-tier-2"
                  onOpenMatch={handlerFor(overview.best_headshots_ref)}
                  openMatchLabel={openMatchLabel}
                />
              )}
              {overview.best_personal_score_ref && (
                <AccentCard
                  label={t('synthesis.kpi.top_personal_score')}
                  value={Math.round(overview.best_personal_score_ref.value).toLocaleString(numLoc)}
                  accent="chart-series-4"
                  onOpenMatch={handlerFor(overview.best_personal_score_ref)}
                  openMatchLabel={openMatchLabel}
                />
              )}
            </div>
          </div>
        )}

        {/* Statistiques détaillées intégrées */}
        {detailedStats && (
          <div className="space-y-5">

            <div>
              {/* Carte frags v2 + 3 cartes côte à côte */}
              {/* Rangée 1 : « Répartition des frags » (sunburst LARGE) + colonne KPI Taux de
                  victoire / FDA / Incidents. Survol LIÉ au breakdown de la rangée 2 (état `frags`). */}
              <div className="flex gap-4 items-stretch">
                {fragDistribution && (
                  <div className="flex min-w-0 flex-1 flex-col gap-2">
                    <FragSunburst
                      distribution={fragDistribution}
                      maxWidthPx={520}
                      hideCenterLabel
                      legendSide="left"
                      externalHoveredClass={frags.hovered}
                      onClassHover={frags.setHovered}
                    />
                    {frags.insightText && (
                      <p className="w-full text-xs leading-snug text-muted-foreground">
                        <span className="font-semibold text-foreground">
                          {t('synthesis.coach.label')}
                          {' : '}
                        </span>
                        {frags.insightText}
                      </p>
                    )}
                  </div>
                )}
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
                      <p className="text-xs text-muted-foreground">{t('synthesis.combat_profile.incidents')}</p>
                      <div className="mt-1.5 w-full">
                        <ProportionalBar segments={[
                          { value: detailedStats.total_betrayals, color: 'warning' },
                          { value: detailedStats.total_suicides,  color: 'outcome-dnf' },
                        ]} />
                      </div>
                      <div className="mt-1 flex justify-center gap-2 text-xs font-semibold tabular-nums">
                        {detailedStats.total_betrayals > 0 && (
                          <span style={{ color: tokenCssVar('warning') }}>{detailedStats.total_betrayals} {t('synthesis.combat_profile.betrayals')}</span>
                        )}
                        {detailedStats.total_suicides > 0 && (
                          <span style={{ color: tokenCssVar('outcome-dnf') }}>{detailedStats.total_suicides} {t('synthesis.combat_profile.suicides')}</span>
                        )}
                      </div>
                    </div>
                  )}

                </div>
              </div>

              {/* Rangée 2 : colonne KPI Tir / Dégâts + « Détails des frags » (breakdown, survol
                  lié au sunburst de la rangée 1) + « Précision par arme » (H5, gaté weapon_accuracy). */}
              <div className="flex gap-4 mt-4 items-stretch">
                <div className="flex flex-col justify-between gap-4 w-[21rem] shrink-0">
                  {(overview.longest_win_streak ?? 0) > 1 && (
                    <AccentCard label={t('synthesis.kpi.win_streak_max')} value={String(overview.longest_win_streak)} accent="outcome-win" />
                  )}

                  {(overview.longest_loss_streak ?? 0) > 1 && (
                    <AccentCard label={t('synthesis.kpi.loss_streak_max')} value={String(overview.longest_loss_streak)} accent="outcome-loss" />
                  )}

                  <div className="grid grid-cols-2 gap-2">
                    <AccentCard label={t('synthesis.combat_profile.perfect_kills')} value={detailedStats.total_perfect_kills.toLocaleString(numLoc)} accent="perf-tier-3" />
                    <AccentCard label={fieldMappings?.fields['headshot_kills']?.label ?? 'Tirs à la tête'} value={detailedStats.total_headshot_kills.toLocaleString(numLoc)} accent="perf-tier-2" />
                  </div>

                  <div>
                    <div className="grid grid-cols-2 gap-2">
                      <AccentCard label={fieldMappings?.fields['shots_fired']?.label ?? 'Tirs effectués'} value={detailedStats.total_shots_fired.toLocaleString(numLoc)} accent="info" />
                      <AccentCard label={fieldMappings?.fields['shots_hit']?.label ?? 'Tirs au but'}      value={detailedStats.total_shots_hit.toLocaleString(numLoc)}   accent="info" />
                      {detailedStats.total_shots_fired > 0 && (
                        <AccentCard
                          label={t('synthesis.kpi.raw_accuracy')}
                          value={`${((detailedStats.total_shots_hit / detailedStats.total_shots_fired) * 100).toFixed(1)}%`}
                          accent="info"
                        />
                      )}
                      {(detailedStats.total_time_played_seconds ?? 0) > 0 && (
                        <AccentCard
                          label={labelOf('time_played_seconds')}
                          value={formatTimePlayed(detailedStats.total_time_played_seconds!)}
                          accent="info"
                        />
                      )}
                    </div>
                  </div>

                  <div>
                    <div className="grid grid-cols-2 gap-2">
                      <AccentCard label={fieldMappings?.fields['damage_dealt']?.label ?? 'Dégâts infligés'} value={Math.round(detailedStats.total_damage_dealt).toLocaleString(numLoc)} accent="outcome-win" />
                      {hasDamageTaken && (
                        <AccentCard label={fieldMappings?.fields['damage_taken']?.label ?? 'Dégâts reçus'} value={Math.round(detailedStats.total_damage_taken).toLocaleString(numLoc)} accent="outcome-loss" />
                      )}
                    </div>
                  </div>

                  {/* Compétences spartiate & assassinats (Halo 5 — capability-gated, cumul sur le scope). */}
                  {hasKillMechanics &&
                    (detailedStats.total_assassinations + detailedStats.total_ground_pound_kills + detailedStats.total_shoulder_bash_kills) > 0 && (
                    <div>
                      <div className="grid grid-cols-3 gap-2">
                        <AccentCard label={t('synthesis.charts.kill_type_assassination')} value={detailedStats.total_assassinations.toLocaleString(numLoc)} accent="chart-series-2" />
                        <AccentCard label={t('synthesis.charts.kill_type_ground_pound')} value={detailedStats.total_ground_pound_kills.toLocaleString(numLoc)} accent="chart-series-3" />
                        <AccentCard label={t('synthesis.charts.kill_type_shoulder_bash')} value={detailedStats.total_shoulder_bash_kills.toLocaleString(numLoc)} accent="chart-series-4" />
                      </div>
                    </div>
                  )}

                  {(detailedStats.total_vehicles_destroyed > 0 || detailedStats.total_hijacks > 0) && (
                    <div>
                      <div className="grid grid-cols-2 gap-2">
                        {detailedStats.total_vehicles_destroyed > 0 && (
                          <AccentCard label={t('synthesis.kpi.vehicles_destroyed')} value={detailedStats.total_vehicles_destroyed.toLocaleString(numLoc)} accent="warning" />
                        )}
                        {detailedStats.total_hijacks > 0 && (
                          <AccentCard label={hijacksLabel} value={detailedStats.total_hijacks.toLocaleString(numLoc)} accent="chart-series-4" />
                        )}
                      </div>
                    </div>
                  )}

                  {/* Objectifs (CTF/Zones/Oddball) — capability-gated + data-driven (KPI > 0). */}
                  {hasObjectiveStats && objectiveStats && (
                    <div>
                      <p className="mb-1 text-xs font-semibold text-muted-foreground">{t('synthesis.kpi.objectives_title')}</p>
                      <div className="grid grid-cols-3 gap-2">
                        {(objectiveStats.flag_captures ?? 0) > 0 && (
                          <AccentCard label={t('synthesis.kpi.flag_captures')} value={(objectiveStats.flag_captures ?? 0).toLocaleString(numLoc)} accent="chart-series-1" />
                        )}
                        {(objectiveStats.flag_returns ?? 0) > 0 && (
                          <AccentCard label={t('synthesis.kpi.flag_returns')} value={(objectiveStats.flag_returns ?? 0).toLocaleString(numLoc)} accent="chart-series-2" />
                        )}
                        {(objectiveStats.flag_steals ?? 0) > 0 && (
                          <AccentCard label={t('synthesis.kpi.flag_steals')} value={(objectiveStats.flag_steals ?? 0).toLocaleString(numLoc)} accent="chart-series-3" />
                        )}
                        {(objectiveStats.flag_carrier_seconds ?? 0) > 0 && (
                          <AccentCard label={t('synthesis.kpi.flag_carrier_time')} value={formatDurationMMSS(objectiveStats.flag_carrier_seconds)} accent="chart-series-4" />
                        )}
                        {(objectiveStats.zone_captures ?? 0) > 0 && (
                          <AccentCard label={t('synthesis.kpi.zone_captures')} value={(objectiveStats.zone_captures ?? 0).toLocaleString(numLoc)} accent="chart-series-1" />
                        )}
                        {(objectiveStats.zone_secures ?? 0) > 0 && (
                          <AccentCard label={t('synthesis.kpi.zone_secures')} value={(objectiveStats.zone_secures ?? 0).toLocaleString(numLoc)} accent="chart-series-2" />
                        )}
                        {(objectiveStats.zone_seconds ?? 0) > 0 && (
                          <AccentCard label={t('synthesis.kpi.zone_time')} value={formatDurationMMSS(objectiveStats.zone_seconds)} accent="chart-series-3" />
                        )}
                        {(objectiveStats.skull_grabs ?? 0) > 0 && (
                          <AccentCard label={t('synthesis.kpi.skull_grabs')} value={(objectiveStats.skull_grabs ?? 0).toLocaleString(numLoc)} accent="chart-series-1" />
                        )}
                        {(objectiveStats.skull_carrier_seconds ?? 0) > 0 && (
                          <AccentCard label={t('synthesis.kpi.skull_carrier_time')} value={formatDurationMMSS(objectiveStats.skull_carrier_seconds)} accent="chart-series-4" />
                        )}
                      </div>
                    </div>
                  )}
                </div>
                {fragDistribution && (
                  <div className="flex min-w-0 flex-1 flex-col">
                    <FragWeaponBreakdown
                      weapons={frags.breakdown}
                      title={frags.detailTitle}
                      hoveredClass={frags.hovered}
                      onClassHover={frags.setHovered}
                      fillHeight
                      // fluid : le tracé prend flex-1, la légende de pied reste collée au
                      // bas de la card (sinon flex-1 laisserait un vide sous le footer).
                      fluid
                    />
                  </div>
                )}
                {hasWeaponAccuracy && (
                  <div className="flex min-w-0 flex-1 flex-col">
                    <SynthesisWeaponAccuracyChart
                      weapons={weaponAccuracy ?? []}
                      weaponKills={topWeaponKills}
                      hoveredClass={frags.hovered}
                      onClassHover={frags.setHovered}
                      fillHeight
                    />
                  </div>
                )}

              </div>

            </div>

          </div>
        )}


    </section>
  )
}



// ─── Page principale ──────────────────────────────────────────────────────────

export function SynthesisPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const t = (key: keyof typeof synthesisManifest) => synthesisManifest[key][locale]

  // ── Filtres locaux (pending → committed) ────────────────────────────────────
  // Pattern aligné sur CareerHighlightMatchesSection : state local non synchronisé
  // avec le globalFilterStore (les filtres restent locaux à la page Synthesis).
  // Cascade côté backend : counts cascade-aware via /filters/resolve (useFiltersPreview).
  const [pendingPeriod, setPendingPeriod] = useState<PeriodInput>(DEFAULT_PERIOD)
  const [pendingExperience, setPendingExperience] = useState<Experience>('all')
  const [pendingPlaylists, setPendingPlaylists] = useState<Set<string>>(() => new Set())
  const [pendingModes, setPendingModes] = useState<Set<string>>(() => new Set())

  const [committedPeriod, setCommittedPeriod] = useState<PeriodInput>(DEFAULT_PERIOD)
  const [committedExperience, setCommittedExperience] = useState<Experience>('all')
  const [committedPlaylists, setCommittedPlaylists] = useState<Set<string>>(() => new Set())
  const [committedModes, setCommittedModes] = useState<Set<string>>(() => new Set())

  const [activePopover, setActivePopover] = useState<'periode' | 'saison' | null>(null)

  const togglePopover = (which: 'periode' | 'saison') =>
    setActivePopover((cur) => (cur === which ? null : which))
  const closeAll = () => setActivePopover(null)

  const { seasons, activeSeason } = useActiveSeason(pendingPeriod)

  // ── Cascade locale envoyée au backend (override de la cascade du store global) ──
  const pendingCascade: CascadeInput = useMemo(() => ({
    experience_types: EXPERIENCE_TO_CASCADE[pendingExperience],
    playlists: Array.from(pendingPlaylists),
    modes: Array.from(pendingModes),
    maps: [],
  }), [pendingExperience, pendingPlaylists, pendingModes])

  const committedCascade: CascadeInput = useMemo(() => ({
    experience_types: EXPERIENCE_TO_CASCADE[committedExperience],
    playlists: Array.from(committedPlaylists),
    modes: Array.from(committedModes),
    maps: [],
  }), [committedExperience, committedPlaylists, committedModes])

  // ── Preview cascade-aware via /filters/resolve ──────────────────────────────
  // La page Synthesis maintient ses propres filtres locaux et ignore les filtres
  // globaux (sessions/period auto-snap d'autres pages). Le pendingFilterContext
  // est construit UNIQUEMENT à partir des states pending locaux, sinon on hérite
  // de scopes restrictifs (ex. une session auto-snap avec uniquement des matchs
  // PVE) qui faussent les counts cascade affichés.
  const pendingFilterContext: FilterContextInput = useMemo(() => ({
    filter_mode: 'period',
    period: pendingPeriod,
    sessions: { picked_sessions: [], gap_minutes: 120 },
    cascade: pendingCascade,
  }), [pendingPeriod, pendingCascade])

  const { data: previewData } = useFiltersPreview(playerSlug, pendingFilterContext)
  const available = previewData?.available_options

  const experienceCounts = useMemo(() => {
    const opts = available?.experience_types ?? []
    // Mappe les counts "PVP classé"/"PVP non classé"/"PVE" du backend vers le
    // format {value: 'all'|'ranked'|'unranked', count} du dropdown.
    //
    // 'Toutes' agrège TOUS les counts (ranked + unranked + PVE) — sinon un
    // joueur 100% Firefight verrait 'Toutes' à 0 alors qu'il a bien des matchs.
    // 'Classé' / 'Non classé' restent stricts (PVP uniquement).
    let ranked = 0
    let unranked = 0
    let total = 0
    for (const o of opts) {
      const v = o.value.toLowerCase()
      if (v.includes('non classé') || v.includes('non-classé') || v.includes('unranked')) {
        unranked += o.count
      } else if (v.includes('classé') || v.includes('ranked')) {
        ranked += o.count
      }
      total += o.count
    }
    return [
      { value: 'all' as const, count: total },
      { value: 'ranked' as const, count: ranked },
      { value: 'unranked' as const, count: unranked },
    ]
  }, [available?.experience_types])

  const playlistOptions: MultiSelectOption[] = useMemo(() => {
    return (available?.playlists ?? [])
      .map((p) => ({ value: p.value, label: p.label, count: p.count }))
      .filter((o) => o.count > 0 || pendingPlaylists.has(o.value))
  }, [available?.playlists, pendingPlaylists])

  const modeOptions: MultiSelectOption[] = useMemo(() => {
    return (available?.modes ?? [])
      .map((m) => ({ value: m.value, label: m.label, count: m.count }))
      .filter((o) => o.count > 0 || pendingModes.has(o.value))
  }, [available?.modes, pendingModes])

  const hasActiveFilters =
    !!(committedPeriod.start_date || committedPeriod.end_date) ||
    committedExperience !== 'all' ||
    committedPlaylists.size > 0 ||
    committedModes.size > 0

  const isDirty =
    pendingPeriod.start_date !== committedPeriod.start_date ||
    pendingPeriod.end_date !== committedPeriod.end_date ||
    pendingExperience !== committedExperience ||
    !setsEqual(pendingPlaylists, committedPlaylists) ||
    !setsEqual(pendingModes, committedModes)

  function handleAnalyser() {
    setCommittedPeriod(pendingPeriod)
    setCommittedExperience(pendingExperience)
    setCommittedPlaylists(new Set(pendingPlaylists))
    setCommittedModes(new Set(pendingModes))
    closeAll()
  }

  function handleResetAll() {
    setPendingPeriod(DEFAULT_PERIOD)
    setCommittedPeriod(DEFAULT_PERIOD)
    setPendingExperience('all')
    setCommittedExperience('all')
    setPendingPlaylists(new Set())
    setCommittedPlaylists(new Set())
    setPendingModes(new Set())
    setCommittedModes(new Set())
  }

  const hasPeriod = !!(committedPeriod.start_date || committedPeriod.end_date)
  // Filtres 100% locaux : on n'hérite pas du store global (pattern identique à
  // CareerHighlightMatchesSection). Sinon une session auto-snappée d'une autre
  // page peut pré-filtrer les matchs (ex. session 100% PVE → counts Experience
  // PVP tous à 0).
  const request: SynthesisQueryRequest = {
    filters: {
      filter_mode: 'period',
      period: hasPeriod ? committedPeriod : DEFAULT_PERIOD,
      sessions: { picked_sessions: [], gap_minutes: 120 },
      cascade: committedCascade,
    },
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
          title={t('synthesis.empty.synthesis_unavailable')}
          description="Aucune charge utile n'a été renvoyée pour cette page. Vérifie les agrégats solo/escouade et le contexte de filtres."
        />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      {/* ─── Barre filtres (cascade locale) ─────────────────────────────────── */}
      {/* Ordre : Expérience → Saison → Période → Playlist → Mode. Counts cascade-aware
          calculés via /filters/resolve (useFiltersPreview). Commit atomique au clic Analyser. */}
      <div className="sticky top-0 z-20 px-6" style={{ background: 'var(--background)' }}>
        <div className="flex min-h-10 items-center gap-1.5 border-b border-border py-1.5 flex-wrap">
          <ExperienceDropdown
            value={pendingExperience}
            onChange={setPendingExperience}
            counts={experienceCounts}
            labels={{
              placeholder: t('synthesis.filters.experience'),
              all: t('synthesis.filters.experience_all'),
              ranked: t('synthesis.filters.experience_ranked'),
              unranked: t('synthesis.filters.experience_unranked'),
            }}
          />
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
          <MultiSelectFilter
            options={playlistOptions}
            selected={pendingPlaylists}
            toggle={(v) => {
              setPendingPlaylists((prev) => {
                const next = new Set(prev)
                if (next.has(v)) next.delete(v)
                else next.add(v)
                return next
              })
            }}
            placeholder={t('synthesis.filters.playlists')}
            alwaysShow
            disabled={playlistOptions.length === 0 && pendingPlaylists.size === 0}
          />
          <MultiSelectFilter
            options={modeOptions}
            selected={pendingModes}
            toggle={(v) => {
              setPendingModes((prev) => {
                const next = new Set(prev)
                if (next.has(v)) next.delete(v)
                else next.add(v)
                return next
              })
            }}
            placeholder={t('synthesis.filters.modes')}
            alwaysShow
            disabled={modeOptions.length === 0 && pendingModes.size === 0}
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
          {hasActiveFilters && (
            <button
              type="button"
              onClick={handleResetAll}
              className="shrink-0 text-xs text-muted-foreground transition-colors hover:text-destructive"
              title={t('synthesis.filters.reset')}
            >
              ↺
            </button>
          )}
        </div>
      </div>

      {/* Contenu : padding horizontal standard de l'app (px-6, cf. autres pages
          stats). La barre de filtres sticky garde son px-6 propre au-dessus. */}
      <div className="flex flex-col gap-6 px-6">
      {/* Bloc 1 — Vue d'ensemble D4 (inclut profil de combat inline) */}
      {data.overview && (
        <SynthesisOverviewSection
          overview={data.overview}
          detailedStats={data.detailed_stats}
          topWeaponKills={data.top_weapon_kills}
          fragDistribution={data.frag_distribution}
          weaponAccuracy={data.weapon_accuracy}
          combatProfile={data.combat_profile}
          objectiveStats={data.objective_stats}
          playerSlug={playerSlug}
        />
      )}

      {/* Section — Comparaison Solo / Escouade (synthesis.05). Graphe unique : le
          titre de section porte le nom, le ChartCard est rendu sans barre de titre
          (title omis) pour éviter le doublon. */}
      <section className="space-y-3">
        <header><h3 className="text-base font-semibold text-foreground">{t('synthesis.section.comparison')}</h3></header>
        <SynthesisBipolaireChart
          metrics={comparisonMetrics}
          fieldLabels={comparisonMetrics.map((m) => labelOf(m.label))}
        />
      </section>

      {/* Section — Activité & répartition : heatmap jour×heure (synthesis.03),
          Top vs Total par semaine (synthesis.04), répartition carte/mode (synthesis.01/02).
          Plusieurs graphes → chacun garde sa propre barre de titre. */}
      <section className="space-y-3">
        <header><h3 className="text-base font-semibold text-foreground">{t('synthesis.section.activity_breakdown')}</h3></header>

        <SynthesisHeatmapChart title={t('synthesis.section.activity')} cells={data.heatmap_data ?? []} />

        {topWeeks.length > 0 && (
          <SynthesisTopWeeksChart title={t('synthesis.charts.top_vs_total_per_week')} weeks={topWeeks} />
        )}

        {data.breakdowns && (
          <div className="flex flex-col gap-4">
            {(data.breakdowns.top_maps ?? []).length > 0 && (
              <SynthesisOutcomesByGroupChart
                title={t('synthesis.breakdown.by_map')}
                entries={(data.breakdowns.top_maps ?? []).map((m) => ({
                  name: m.map_name,
                  wins: m.wins,
                  losses: m.losses,
                  ties: m.ties,
                  unfinished: m.unfinished,
                }))}
                height={360}
              />
            )}
            {(data.breakdowns.top_modes ?? []).length > 0 && (
              <SynthesisOutcomesByGroupChart
                title={t('synthesis.breakdown.by_mode')}
                entries={(data.breakdowns.top_modes ?? []).map((m) => ({
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
      </section>
      </div>
    </div>
  )
}
