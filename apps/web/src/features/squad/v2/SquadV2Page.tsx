/**
 * SquadV2Page — page Squad V2 finale (chunk S12).
 *
 * Compose les wrappers ECharts (S10) et les composants tableau/galerie (S10b)
 * sur le payload backend complet (S11). Resout l'i18n via le squadManifest
 * (squad.toml) et la locale active du store appShell.
 *
 * Sticky legend (FloatingLegend) intercale entre les sections pour rappeler
 * l'ordre stable des couleurs du squad.
 */
import { BarStackedChart } from '@/components/charts/BarStackedChart'
import { BarGroupedChart } from '@/components/charts/BarGroupedChart'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import { Heatmap2DChart } from '@/components/charts/Heatmap2DChart'
import { RadarChart } from '@/components/charts/RadarChart'
import { formatMessage } from '@/lib/i18n/format'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { useAppShellStore } from '@/stores/appShellStore'

import { OutcomeSequenceTape, type OutcomePoint } from '@/components/charts/OutcomeSequenceTape'
import { SessionBriefing } from '@/features/_shared/SessionBriefing'
import { HistoryTable } from './components/HistoryTable'
import { SquadCombatProfileRow } from './components/SquadCombatProfileRow'
import { WeaponsTable } from './components/WeaponsTable'
import { MedalsGallery } from './components/MedalsGallery'
import { useSquadV2 } from './queries'
import type { SquadPeriod } from './types'

export interface SquadV2PageProps {
  playerSlug: string
  teammates?: string[]
  period?: SquadPeriod
  experienceTypes?: string[]
  playlists?: string[]
  maps?: string[]
  modes?: string[]
}

type SquadLocale = 'fr' | 'en'

function isSquadLocale(loc: string | undefined): loc is SquadLocale {
  return loc === 'fr' || loc === 'en'
}

function useSquadV2Translator() {
  const rawLocale = useAppShellStore((s) => s.locale)
  const locale: SquadLocale = isSquadLocale(rawLocale) ? rawLocale : 'fr'
  return (key: SquadManifestKey, vars?: Record<string, unknown>) =>
    formatMessage(squadManifest, key, locale, vars)
}

// SquadEngagementSection lazy-loaded pour eviter la cyclic deps avec @/features/squad
import { SquadEngagementSection } from '@/features/engagement/SquadEngagementSection'
import { FeatureGate } from '@/lib/capabilities/FeatureGate'

export function SquadV2Page({ playerSlug, teammates, period, experienceTypes, playlists, maps, modes }: SquadV2PageProps) {
  const t = useSquadV2Translator()
  const locale = useAppShellStore((s) => s.locale) ?? 'fr'
  const intlLocale = locale === 'en' ? 'en-US' : 'fr-FR'
  const { data: fieldMappings } = useFieldMappings()

  const { data, isLoading, error } = useSquadV2({ playerSlug, teammates, period, experienceTypes, playlists, maps, modes })

  if (isLoading) {
    return (
      <div className="p-6 text-center text-muted-foreground">
        {t('squad.v2.loading')}
      </div>
    )
  }
  if (error) {
    return (
      <div className="p-6 text-center text-destructive">
        {t('squad.v2.error_prefix')} {error.message}
      </div>
    )
  }
  if (!data || data.shared_matches_count === 0) {
    return (
      <div className="p-6 text-center text-muted-foreground">
        {t('squad.v2.empty')}
      </div>
    )
  }

  const charts = data.charts
  const tables = data.tables
  const squadOrder = [data.main_player, ...data.teammates]
  const header = data.header

  // Briefing : alimente <SessionBriefing> en haut de page (rail + verdict + grid).
  // Mode squad si squad_score + player_cards + team_avg_kpis presents.
  const mainXuid = header?.player_cards?.find((c) => c.gamertag === data.main_player)?.xuid ?? ''
  const briefingSquad =
    header?.squad_score && header?.player_cards && header?.team_avg_kpis && header?.kpis_by_xuid && mainXuid
      ? {
          score: header.squad_score,
          players: header.player_cards,
          kpisByXuid: header.kpis_by_xuid,
          teamAvgKpis: header.team_avg_kpis,
          activeXuid: mainXuid,
        }
      : undefined

  return (
    <div className="flex flex-col gap-6 p-6" data-testid="squad-v2-page">
      {/* Briefing — KPIs + verdict squad + drill-down click */}
      {header?.solo_kpis && (
        <SessionBriefing kpis={header.solo_kpis} squad={briefingSquad} />
      )}

      {/* PLAN_COMBAT_PROFILE_WIRING Phase 2 — Profil de combat par joueur */}
      {header?.player_cards && header?.kpis_by_xuid && (
        <SquadCombatProfileRow
          playerCards={header.player_cards}
          kpisByXuid={header.kpis_by_xuid}
        />
      )}

      {/* Séquence des outcomes (S13) */}
      {data.shared_matches.length > 0 && (
        <section data-testid="squad-v2-outcome-sequence">
          <p className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">
            {t('squad.v2.section_outcome_sequence')}
          </p>
          <OutcomeSequenceTape
            matches={[...data.shared_matches].reverse().map<OutcomePoint>((m) => ({
              outcome: m.outcome,
              matchId: m.match_id,
              map: m.map?.default_label,
              mode: m.mode?.default_label,
            }))}
            labels={{
              win: fieldMappings?.outcomes?.['win']?.label ?? 'win',
              loss: fieldMappings?.outcomes?.['loss']?.label ?? 'loss',
              tie: fieldMappings?.outcomes?.['tie']?.label ?? 'tie',
              dnf: fieldMappings?.outcomes?.['dnf']?.label ?? 'dnf',
            }}
          />
        </section>
      )}

      {/* Synergies */}
      {charts && (
        <section data-testid="squad-v2-synergies">
          <h2 className="mb-3 text-lg font-semibold">
            {t('squad.v2.section_synergies')}
          </h2>
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            {charts.map_breakdown_lollipop && (
              <BarStackedChart
                title={t('squad.synergies.lollipop_title')}
                series={[charts.map_breakdown_lollipop]}
                orientation="horizontal"
                componentColors={{
                  win: 'outcome-win',
                  loss: 'outcome-loss',
                  tie: 'outcome-draw',
                  dnf: 'outcome-dnf',
                }}
                componentOrder={['win', 'loss', 'tie', 'dnf']}
              />
            )}
            {charts.heatmap_player_map && (
              <Heatmap2DChart
                title={t('squad.synergies.heatmap_title')}
                series={[charts.heatmap_player_map]}
                paletteMode="divergent"
              />
            )}
            {charts.timeline_multi_player && charts.timeline_multi_player.length > 0 && (
              <TimeseriesLineChart
                title={t('squad.synergies.timeline_title')}
                series={charts.timeline_multi_player}
              />
            )}
            {charts.cadence && (
              <BarStackedChart
                title={t('squad.synergies.cadence_title')}
                series={[charts.cadence]}
              />
            )}
            {charts.intensity_heatmap && (
              <Heatmap2DChart
                title={t('squad.synergies.intensity_title')}
                series={[charts.intensity_heatmap]}
              />
            )}
          </div>
        </section>
      )}

      {/* Contributions */}
      {charts && (
        <section data-testid="squad-v2-contributions">
          <h2 className="mb-3 text-lg font-semibold">
            {t('squad.v2.section_contributions')}
          </h2>
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            {charts.per_minute_stats && (
              <BarGroupedChart
                title={t('squad.contrib.per_minute_title')}
                series={[charts.per_minute_stats]}
              />
            )}
            {charts.frags_deaths_combined && (
              <BarGroupedChart
                title={t('squad.contrib.frags_deaths_title')}
                series={[charts.frags_deaths_combined]}
                componentColors={{
                  kills: 'outcome-win',
                  deaths: 'outcome-loss',
                }}
                componentOrder={['kills', 'deaths']}
              />
            )}
          </div>
        </section>
      )}

      {/* Engagement equipe (Mock 15 v2) — gaté sur `engagement` */}
      <FeatureGate capability="engagement">
        <section data-testid="squad-v2-engagement">
          <SquadEngagementSection
            playerSlug={playerSlug}
            teammates={teammates?.map((x) => ({ xuid: x, gamertag: x }))}
          />
        </section>
      </FeatureGate>

      {/* Radar (S8) */}
      {charts?.radar && charts.radar.length > 0 && (
        <section data-testid="squad-v2-radar">
          <h2 className="mb-3 text-lg font-semibold">
            {t('squad.v2.section_radar')}
          </h2>
          <RadarChart series={charts.radar} />
        </section>
      )}

      {/* Tableaux (S9) */}
      {tables?.history && tables.history.length > 0 && (
        <section data-testid="squad-v2-history">
          <h2 className="mb-3 text-lg font-semibold">
            {t('squad.v2.section_history')}
          </h2>
          <HistoryTable
            rows={tables.history}
            squadOrder={squadOrder}
            locale={intlLocale}
            playerSlug={playerSlug}
            labels={{
              date: t('squad.history.col_date'),
              mode: t('squad.history.col_mode'),
              map: t('squad.history.col_map'),
              outcome: t('squad.history.col_outcome'),
              duration: t('squad.history.col_duration'),
              kdaSuffix: t('squad.history.col_kda'),
            }}
          />
        </section>
      )}

      {tables?.weapons && tables.weapons.length > 0 && (
        <section data-testid="squad-v2-weapons">
          <h2 className="mb-3 text-lg font-semibold">
            {t('squad.v2.section_weapons')}
          </h2>
          <WeaponsTable
            rows={tables.weapons}
            squadOrder={squadOrder}
            labels={{
              weapon: t('squad.weapons.col_weapon'),
              total: t('squad.weapons.col_total'),
              minKills: (n: number) => t('squad.v2.weapons.min_kills', { n }),
              grenadeMelee: t('squad.weapons.grenade_melee_marker'),
            }}
          />
        </section>
      )}

      {tables?.medals && tables.medals.length > 0 && (
        <section data-testid="squad-v2-medals">
          <h2 className="mb-3 text-lg font-semibold">
            {t('squad.v2.section_medals')}
          </h2>
          <MedalsGallery
            entries={tables.medals}
            squadOrder={squadOrder}
            labels={{ emptyMatch: t('squad.medals.empty_match') }}
          />
        </section>
      )}
    </div>
  )
}
