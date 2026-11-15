/**
 * TimeseriesPage — onglet "Progression".
 *
 * Découpé depuis TimeseriesPage.tsx (audit #6 god-file split).
 * Contenu : premier frag / première mort, cadences par minute, profil d'intensité,
 * performance, spree/headshots, rank score, skill rank perf, efficiency,
 * engagement, puis le tableau historique en pied d'onglet.
 *
 * L'INTENSITÉ EST REMONTÉE juste après les cadences par minute : les deux disent COMMENT
 * le rythme se répartit dans un match, elles se lisent l'une après l'autre. Les usages
 * d'équipement et les formes retenues ont quitté cet onglet pour « Usages » (tout ce qui
 * vient du film décodé y est réuni, cf. TimeseriesPage.usages.tsx).
 */
import { useMemo } from 'react'
import { PAIR_ROW_CLASS } from '@/components/ui/pair-row'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { EfficiencyTooltipText } from '@/components/charts/EfficiencyTooltipText'
import { intensityTooltipText } from '@/components/charts/intensityTooltipText'
import { FirstBloodLanes } from '@/components/charts/FirstBloodLanes'
import { firstBloodMaxSec, toFirstBloodSeries } from '@/features/_shared/firstBlood'
import {
  TimeseriesPerformanceTrend,
  TimeseriesSpreeHeadshots,
  TimeseriesRankScore,
  TimeseriesSkillRankPerformance,
  TimeseriesCareerXP,
} from './TimeseriesFormCharts'
import { TimeseriesPerMinuteTrend, type PerMinuteTrendLabels } from './TimeseriesPerMinuteTrend'
import { paceHasDamage } from './timeseriesPerMinute.logic'
import {
  TimeseriesEfficiency,
  TimeseriesIntensityProfile,
} from './TimeseriesSquadAdapted'
import { EngagementTimeseriesSection } from '@/features/engagement/EngagementTimeseriesSection'
import { TimeseriesEngagementGapTrend } from './TimeseriesEngagementGapTrend'
import { TimeseriesCoordinationSection } from './TimeseriesCoordinationSection'
import { FeatureGate } from '@/lib/capabilities/FeatureGate'
import { useCapability } from '@/lib/capabilities/capabilities'
import { useDataCapability } from '@/lib/capabilities/dataCapabilities'
import { useProvidesDamageTaken } from '@/lib/damage/effectiveHp'
import { ExplorerMatchesTable } from '@/features/explorer/ExplorerMatchesTable'
import { TimeseriesSkillProgression } from './TimeseriesSkillProgression'
import type { FilterContextInput, TimeseriesPageResponse, ExplorerMatchRow } from '@/lib/api/types'
import type { FieldMappingsResponse } from '@/lib/i18n/fieldMappings'
import type { TimeseriesManifestKey } from '@/lib/i18n/generated/timeseries'
import type { Locale } from '@/lib/i18n/locale'
import { CUMULATIVE_GAP_CHARTS_SHOWN } from '@/lib/charts/cumulativeGapChartsVisibility'

export interface TimeseriesProgressionTabProps {
  data: TimeseriesPageResponse
  playerSlug: string
  locale: Locale
  t: (key: TimeseriesManifestKey) => string
  fieldMappings: FieldMappingsResponse | undefined
  soloFilterContext: FilterContextInput
  filterContextHash: string
  explorerMatchRows: ExplorerMatchRow[] | undefined
}

/**
 * Hauteur du graphe « Stats par minute » — la voisine de rangée de « Premier frag /
 * première mort ». Les deux cartes de la première rangée la partagent (2026-09-19).
 */
const PER_MINUTE_CHART_HEIGHT = 360

export function TimeseriesProgressionTab({
  data,
  playerSlug,
  locale,
  t,
  fieldMappings,
  soloFilterContext,
  filterContextHash,
  explorerMatchRows,
}: TimeseriesProgressionTabProps) {
  const emptyMsg = t('timeseries.empty.no_data_description')
  // « Premier frag / première mort » : série solo servie par le payload de page —
  // même scope (filtres L2) que tous les autres blocs Progression.
  const firstBlood = useMemo(() => toFirstBloodSeries(data.first_blood), [data.first_blood])
  // « Stats par minute » : les dégâts par minute s'ajoutent en courbes (axe de droite) quand les matchs
  // en portent ; le titre les nomme alors, avec une aide sur le calcul. Les dégâts subis sont retirés
  // d'un titre qui ne les fournit pas (Halo 5).
  const withDamageTaken = useProvidesDamageTaken()
  const matchRows = data.match_rows
  const perMinuteHasDamage = useMemo(() => paceHasDamage(matchRows ?? [], withDamageTaken), [matchRows, withDamageTaken])
  const killsLabel = fieldMappings?.fields['kills_per_minute']?.label ?? 'Frags / min'
  const deathsLabel = fieldMappings?.fields['deaths_per_minute']?.label ?? 'Morts / min'
  const assistsLabel = fieldMappings?.fields['assists_per_minute']?.label ?? 'Assistances / min'
  const damageDealtLabel = t('timeseries.progression.damage_dealt_per_minute')
  const damageTakenLabel = t('timeseries.progression.damage_taken_per_minute')
  const damageAxis = t('timeseries.progression.per_minute_damage_axis')
  const perMinuteSuffix = t('timeseries.progression.per_minute_suffix')
  const perMinuteLabels = useMemo<PerMinuteTrendLabels>(
    () => ({
      kills: killsLabel,
      deaths: deathsLabel,
      assists: assistsLabel,
      damageDealt: damageDealtLabel,
      damageTaken: damageTakenLabel,
      damageAxis,
      suffix: perMinuteSuffix,
    }),
    [killsLabel, deathsLabel, assistsLabel, damageDealtLabel, damageTakenLabel, damageAxis, perMinuteSuffix],
  )
  // Charts dépendant du rating de skill (CSR/LUSR) : masqués pour un titre sans
  // système de rang. NO-OP halo_infinite (déclare 'ranked' + 'lusr'). RankScore
  // (score + placement de match) reste générique. Les deux hooks sont appelés
  // inconditionnellement (règles des hooks) avant la combinaison.
  const hasRanked = useCapability('ranked')
  const hasLusr = useCapability('lusr')
  const hasSkillRating = hasRanked || hasLusr
  // XP de carrière estimée : la capability du titre (analytics.career_xp_estimate, Infinite) garde la carte
  // même sans match estimé sur le périmètre (Firefight seul : titre et état vide, pas un trou) ; la présence
  // d'un match estimé la monte aussi, le temps que la capability se charge. Titre sans capability : carte absente.
  const careerXpSupported = useDataCapability('analytics.career_xp_estimate')
  const hasCareerXP = careerXpSupported || (data.match_rows ?? []).some((r) => r.career_xp_estimated != null)
  return (
    <div className="space-y-8">
      {/* Premier frag / première mort (gauche, bande solo) | timeseries.14 — Par
          minute (droite). Titre et état vide portés par le manifest partagé
          first_blood — même vocabulaire que l'Escouade et les Sessions.

          MÊME HAUTEUR QUE SA VOISINE (2026-09-19) : la bande était dérivée du nombre de
          pistes — le tiers de la hauteur de « Stats par minute » —, et la rangée se lisait
          bancale. `items-stretch` + la hauteur de la carte voisine : les pistes se
          répartissent dans la hauteur donnée, et la légende se pose au ras du bas. */}
      <div className="grid grid-cols-1 items-stretch gap-4 lg:grid-cols-2">
        <FirstBloodLanes
          data={firstBlood}
          maxSec={firstBloodMaxSec(firstBlood)}
          emptyMessage={emptyMsg}
          height={PER_MINUTE_CHART_HEIGHT}
        />

        <TimeseriesPerMinuteTrend
          title={
            perMinuteHasDamage ? (
              <span className="flex items-center gap-1.5">
                {t('timeseries.progression.per_minute_damage_title')}
                <InfoTooltip content={t('timeseries.progression.per_minute_damage_info')} />
              </span>
            ) : (
              t('timeseries.progression.per_minute_title')
            )
          }
          emptyMessage={emptyMsg}
          rows={matchRows ?? []}
          labels={perMinuteLabels}
          withTaken={withDamageTaken}
        />
      </div>

      {/* Intensité — profil médian des parts de frags par phase + enveloppe
          P25–P75 (panneau solo pleine largeur). */}
      <TimeseriesIntensityProfile
        title={
          <span className="flex items-center gap-1.5">
            {t('timeseries.progression.intensity_title')}
            <InfoTooltip content={intensityTooltipText(locale)} />
          </span>
        }
        emptyMessage={emptyMsg}
        rows={data.intensity_rows ?? []}
        teamRows={data.intensity_rows_team ?? undefined}
        lobbyRows={data.intensity_rows_lobby ?? undefined}
        medianLabel={t('timeseries.progression.intensity_median')}
        envelopeLabel={t('timeseries.progression.intensity_envelope')}
        refLabel={t('timeseries.progression.intensity_ref')}
        playerLabel={t('timeseries.progression.intensity_player')}
        teamLabel={t('timeseries.progression.intensity_team')}
        lobbyLabel={t('timeseries.progression.intensity_lobby')}
      />

      {/* timeseries.12 (gauche) | timeseries.16 (droite) */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <TimeseriesPerformanceTrend
          title={t('timeseries.summary.perf_label')}
          emptyMessage={emptyMsg}
          rows={data.match_rows ?? []}
          smoothingLabel={t('timeseries.summary.trend')}
        />

        <TimeseriesSpreeHeadshots
          title={t('timeseries.progression.spree_headshots_title')}
          emptyMessage={emptyMsg}
          rows={data.match_rows ?? []}
          spreeLabel={t('timeseries.progression.spree_label')}
          headshotsLabel={fieldMappings?.fields['headshot_kills']?.label ?? t('timeseries.progression.headshot_kills_fallback')}
          perfectLabel={
            fieldMappings?.fields['perfect_kills']?.label ??
            t('timeseries.progression.perfect_kills')
          }
        />
      </div>

      {/* Progression CSR (classé) ou LUSR (non classé) | XP de carrière (estimée)
          côte à côte (demande utilisateur 2026-07-24) ; chacun repasse pleine
          largeur si l'autre est masqué (titre sans rang / sans capability XP).
          La « Répartition des frags » vit désormais sur l'onglet Résumé (sunburst
          v2 unifié, D7) : plus de donut kill-type ici. */}
      <div className={hasSkillRating && hasCareerXP ? 'grid grid-cols-1 gap-4 lg:grid-cols-2' : ''}>
        {hasSkillRating && (
          <TimeseriesSkillProgression rows={data.match_rows ?? []} locale={locale} emptyMessage={emptyMsg} />
        )}
        {hasCareerXP && (
          <TimeseriesCareerXP
            title={
              <span className="flex items-center gap-1.5">
                {t('timeseries.progression.career_xp_title')}
                <InfoTooltip content={t('timeseries.progression.career_xp_tooltip')} />
              </span>
            }
            emptyMessage={emptyMsg}
            rows={data.match_rows ?? []}
            cumulativeLabel={t('timeseries.progression.career_xp_cumulative')}
            perMatchLabel={t('timeseries.progression.career_xp_per_match')}
          />
        )}
      </div>

      {/* timeseries.19 — Score & rang de match (générique : personal_score +
          placement). | Skill rank + Performance (CSR/LUSR, masqué sans rang).
          Sans rang, Score & rang passe pleine largeur (pas de colonne vide). */}
      <div className={hasSkillRating ? 'grid grid-cols-1 gap-4 lg:grid-cols-2' : ''}>
        <TimeseriesRankScore
          title={t('timeseries.progression.rank_score_title')}
          emptyMessage={emptyMsg}
          rows={data.match_rows ?? []}
          scoreLabel={fieldMappings?.fields['personal_score']?.label ?? t('timeseries.distributions.personal_score_fallback')}
          rankLabel={
            fieldMappings?.fields['rank']?.label ??
            t('timeseries.progression.rank')
          }
        />

        {hasSkillRating && (
          <TimeseriesSkillRankPerformance
            title={t('timeseries.progression.rank_perf_title')}
            emptyMessage={emptyMsg}
            rows={data.match_rows ?? []}
            ratingLabel={t('timeseries.progression.rank')}
            perfLabel={
              fieldMappings?.fields['performance_score']?.label ??
              t('timeseries.summary.perf_label')
            }
          />
        )}
      </div>

      {/* Rendement & Résistance (gauche) | « Appui reçu » (droite), sur UNE rangée à
          hauteur égale (retour utilisateur du 2026-10-07). « Appui reçu » : deux séries de
          bâtons par soirée, aucune requête neuve — le bloc arrive avec cette réponse de page.
          La carte se retire quand le titre ne sert pas de bloc de coordination (Rendement
          reprend alors toute la rangée) ; elle est CONSERVÉE, vide et nommée, quand le bloc
          est servi mais indisponible sur le périmètre (D8). */}
      <div className={PAIR_ROW_CLASS}>
        <TimeseriesEfficiency
          title={
            <span className="flex items-center gap-1.5">
              {t('timeseries.progression.efficiency_title')}
              <InfoTooltip content={<EfficiencyTooltipText locale={locale} />} />
            </span>
          }
          emptyMessage={emptyMsg}
          rows={data.match_rows ?? []}
          rendementLabel={t('timeseries.progression.rate_offensive')}
          resistanceLabel={t('timeseries.progression.rate_defensive')}
          refLabel={t('timeseries.progression.ref_one_life')}
          perFragLabel={t('timeseries.progression.per_effective_frag')}
          perDeathLabel={t('timeseries.progression.per_death')}
        />
        <TimeseriesCoordinationSection block={data.coordination} locale={locale} />
      </div>

      {/* « Balance des dégâts cumulée » N'EST PLUS SUR CET ONGLET (demande utilisateur du
          2026-09-22) : elle se monte désormais sur le Résumé, sous « Assistances »
          (cf. TimeseriesPage.summary.tsx). */}

      {/* Engagement. EngagementTimeseriesSection rend déjà sa propre ChartCard avec titre
          interne, donc pas de wrapper supplémentaire (sinon double titre). Gaté sur
          `engagement`. */}
      <FeatureGate capability="engagement">
        {/* Engagement (gauche) | Écart d'engagement cumulé (droite), sur UNE rangée
            (2026-09-13) : les deux lisent la même série et se répondent — empilés, il
            fallait faire défiler pour comparer une pointe d'engagement à son effet cumulé.
            L'Écart réutilise la même query (dédup cache TanStack Query). */}
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <EngagementTimeseriesSection
            playerSlug={playerSlug}
            filters={soloFilterContext}
            filterHash={filterContextHash}
            limit={30}
          />
          {CUMULATIVE_GAP_CHARTS_SHOWN && (
            <TimeseriesEngagementGapTrend
              playerSlug={playerSlug}
              filters={soloFilterContext}
              filterHash={filterContextHash}
              limit={30}
            />
          )}
        </div>
      </FeatureGate>

      {/* Historique des matchs — tableau Explorer standalone (sans bloc ni titre)
          en bas de Progression. Reflète le scope solo du filtre global (mêmes
          matchs que les charts ci-dessus). */}
      {explorerMatchRows && explorerMatchRows.length > 0 && (
        <ExplorerMatchesTable
          rows={explorerMatchRows}
          playerSlug={playerSlug}
          alwaysShowPagination
          // I16 : même endpoint/tri backend (date DESC) que le mode Matchs →
          // le défaut interne de ExplorerMatchesTable reproduit déjà l'ordre
          // serveur, aucun `defaultSort` à surcharger (cf. ExplorerPage.playerMode).
          sortable
        />
      )}

    </div>
  )
}
