/**
 * SessionChartStack — pile de graphes analytiques d'UNE session.
 *
 * Rendu identique en vue principale (session active) et dans le drawer (session
 * comparée) → comparaison "côte à côte" : on lit la session A à gauche et la session B
 * à droite, mêmes graphes alignés. Pas de graphe combiné A/B.
 *
 * - `dense` (drawer) : tout en 1 colonne (le panneau est déjà étroit).
 *   Vue principale : quelques paires en 2 colonnes.
 * - `participationSide` / `participationColor` : l'axe du profil de participation est à
 *   DROITE + couleur A en vue single, à GAUCHE + couleur B dans le drawer → effet miroir.
 */
import type { SemanticToken } from '@/lib/accessibility'
import type { SessionCompareEntry, SessionDetailMatchRow } from '@/lib/api/types'

import { InfoTooltip } from '@/components/ui/info-tooltip'
import { EfficiencyTooltipText } from '@/components/charts/EfficiencyTooltipText'
import { useAppShellStore } from '@/stores/appShellStore'
import type { CompareScale } from './_compareScale'
import { useSessionT } from './_shared'
import { SessionOutcomeDonut } from './SessionOutcomeDonut'
import { SessionKillsDonut } from './SessionKillsDonut'
import { SessionModeBreakdown } from './SessionModeBreakdown'
import { SessionPlacementBreakdown } from './SessionPlacementBreakdown'
import { SessionFdaRadar } from './SessionFdaRadar'
import { SessionFragsRadar } from './SessionFragsRadar'
import { SessionFdaBars } from './SessionFdaBars'
import { SessionParticipationBars } from './SessionParticipationBars'
import { SessionNetScoreArea } from './SessionNetScoreArea'
import { SessionFdaGapCumulative } from './SessionFdaGapCumulative'
import { SessionNetLivesCumulative } from './SessionNetLivesCumulative'
import { SessionMmrDumbbell } from './SessionMmrDumbbell'
import { SessionPerfTrend } from './SessionPerfTrend'
import { SessionEngagementChart } from './SessionEngagementChart'
import { SessionEngagementCumulative } from './SessionEngagementCumulative'
import { FeatureGate } from '@/lib/capabilities/FeatureGate'
import { useCapability } from '@/lib/capabilities/capabilities'
import { SessionDamageComposite } from './SessionDamageComposite'
import { SessionOcdrBars } from './SessionOcdrBars'
import { SessionFragCard } from './SessionFragCard'

interface Props {
  entry: SessionCompareEntry | null
  matches: SessionDetailMatchRow[]
  /** Colonne divisée (vue compacte / drawer) : donuts en % interne (pas d'étiquette externe). */
  compact?: boolean
  dense?: boolean
  participationSide?: 'left' | 'right'
  participationColor?: SemanticToken
  /** Bornes d'axe partagées A/B (mode comparaison) — fige les échelles pour comparabilité. */
  scale?: CompareScale
}

export function SessionChartStack({
  entry,
  matches,
  compact = false,
  dense = false,
  participationSide = 'right',
  participationColor = 'compare-a',
  scale,
}: Props) {
  const t = useSessionT()
  const locale = useAppShellStore((s) => s.locale)
  // Masquage par capability (décision produit : retrait silencieux, pas de carte vide).
  // team_mmr absent (Halo 5) → la carte « challenge MMR » disparaît entièrement.
  const hasTeamMmr = useCapability('team_mmr')

  const outcomeDonut = (
    <SessionOutcomeDonut title={t('session.detail.chart_outcomes_title')} matches={matches} compact={compact} />
  )
  const killsDonut = (
    <SessionKillsDonut
      title={t('session.detail.chart_kills_donut_title')}
      matches={matches}
      kda={entry?.kda ?? null}
      compact={compact}
    />
  )
  const modeBreakdown = (
    <SessionModeBreakdown
      title={t('session.detail.chart_mode_breakdown_title')}
      matches={matches}
      yMax={scale?.modeMaxCount}
    />
  )
  const placementBreakdown = (
    <SessionPlacementBreakdown
      title={t('session.detail.chart_placement_title')}
      matches={matches}
      yMax={scale?.placementMaxCount}
      axisMaxOverride={scale?.placementAxisMax}
    />
  )
  const fdaRadar = <SessionFdaRadar title={t('session.detail.chart_fda_per_game_title')} matches={matches} />
  const fragsRadar = <SessionFragsRadar title={t('session.detail.chart_frags_radar_title')} entry={entry} />
  const fdaBars = (
    <SessionFdaBars
      title={t('session.detail.chart_fda_per_minute_title')}
      matches={matches}
      mode="minute"
      yDomain={scale?.fdaMinute}
    />
  )
  const participation = (
    <SessionParticipationBars
      title={t('session.compare.participation_title')}
      entry={entry}
      axisSide={participationSide}
      colorToken={participationColor}
    />
  )
  const netScore = (
    <SessionNetScoreArea
      title={t('session.detail.chart_net_score_title')}
      matches={matches}
      yDomain={scale?.netScore}
    />
  )
  // Écart cumulé au FDA attendu (D2) — self-gate capability `expected_stats`
  // (null sur un titre sans FDA attendu, ex. Halo 5). Pleine largeur (pas de
  // wrapper grid) pour ne rien laisser d'affiché quand masqué.
  const fdaGap = (
    <SessionFdaGapCumulative title={t('session.detail.chart_fda_gap_title')} matches={matches} />
  )
  // Balance des dégâts cumulée (P3) — self-gate capability `damage_taken` (null
  // sur un titre sans dégâts subis, ex. Halo 5). Pleine largeur comme fdaGap.
  const netLives = (
    <SessionNetLivesCumulative
      title={t('session.detail.chart_net_lives_title')}
      matches={matches}
      yDomain={scale?.netLives}
    />
  )
  const mmr = hasTeamMmr ? (
    <SessionMmrDumbbell title={t('session.detail.chart_mmr_title')} matches={matches} />
  ) : null
  const perf = <SessionPerfTrend title={t('session.detail.chart_perf_title')} matches={matches} />
  const engagement = (
    <FeatureGate capability="engagement">
      <SessionEngagementChart
        title={t('session.detail.chart_engagement_title')}
        matches={matches}
        entry={entry}
        yDomain={scale?.engagement}
      />
      {/* Écart d'engagement cumulé (P4) — cumul du résidu pondéré par la durée. */}
      <SessionEngagementCumulative
        title={t('session.detail.chart_engagement_cumulative_title')}
        matches={matches}
        entry={entry}
        yDomain={scale?.engagementGap}
      />
    </FeatureGate>
  )
  const ocdr = (
    <SessionOcdrBars
      title={
        <span className="flex items-center gap-1.5">
          {t('session.compare.ocdr_title')}
          <InfoTooltip content={<EfficiencyTooltipText locale={locale} />} />
        </span>
      }
      matches={matches}
    />
  )
  const damage = <SessionDamageComposite title={t('session.detail.chart_damage_title')} matches={matches} />
  // Répartition des frags v2 (sunburst classe→rôle + « Détails des frags ») — alimentée par
  // l'agrégat de session (P5). Rend null si aucune donnée. Rendu Match view (compteur seul,
  // légende gauche, survol lié) ; EMPILÉ quand la colonne est étroite (`dense` drawer OU
  // `compact` = colonne principale rétrécie par le drawer de comparaison).
  const frags = <SessionFragCard entry={entry} stacked={dense || compact} />

  if (dense) {
    return (
      <>
        {outcomeDonut}
        {killsDonut}
        {modeBreakdown}
        {placementBreakdown}
        {fdaRadar}
        {fragsRadar}
        {netScore}
        {fdaBars}
        {fdaGap}
        {netLives}
        {participation}
        {mmr}
        {ocdr}
        {perf}
        {engagement}
        {damage}
        {frags}
      </>
    )
  }

  return (
    <>
      <div className="grid gap-6 xl:grid-cols-2">
        {outcomeDonut}
        {killsDonut}
      </div>
      <div className="grid gap-6 xl:grid-cols-2">
        {modeBreakdown}
        {placementBreakdown}
      </div>
      <div className="grid gap-6 xl:grid-cols-2">
        {fdaRadar}
        {fragsRadar}
      </div>
      <div className="grid gap-6 xl:grid-cols-2">
        {netScore}
        {fdaBars}
      </div>
      {fdaGap}
      {netLives}
      {participation}
      {mmr ? (
        <div className="grid gap-6 xl:grid-cols-2">
          {mmr}
          {ocdr}
        </div>
      ) : (
        ocdr
      )}
      {perf}
      {engagement}
      {damage}
      {frags}
    </>
  )
}
