/**
 * ExplorerBriefingStrip — bandeau de briefing au-dessus du tableau (mode Matchs).
 *
 * Lecture compacte du RÉSULTAT DE RECHERCHE : rangée socle de 4 à 8 tuiles KPI
 * (Matchs, Taux de victoire avec ruban V-D-N + tooltip, FDA, Perf. moyenne colorée,
 * Durée totale, Pic FDA — puis en cascade par priorité, au plus 2 : Meilleure série,
 * Pic rang, Pic MMR) avec deltas vs baseline personnelle. Les modules « Par… »
 * (dimensions, « Par contexte » + Classement par chaîne) sont rendus par
 * ExplorerBriefingModules sous le socle. En low_sample : seules Matchs / Taux de
 * victoire / FDA / Perf.
 *
 * Dégradation : briefing absent → rien ; low_sample → socle réduit + mention
 * échantillon faible, aucun module. Aucune couleur hex : tokens sémantiques.
 */
import type { ReactNode } from 'react'

import { InfoTooltip } from '@/components/ui/info-tooltip'
import { tokenCssVar } from '@/lib/accessibility'
import { kdaNetColor } from '@/lib/colors/outcomePalette'
import { getPerfColor } from '@/lib/perf-color'
import { formatDateRange } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ExplorerBriefing } from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { BriefingTile } from './BriefingTile'
import { deltaToken, formatSignedFixed, isFullHistoryScope } from './ExplorerBriefing.logic'
import { ExplorerBriefingModules } from './ExplorerBriefingModules'
import {
  DurationTile,
  PeakKdaTile,
  PeakMmrTile,
  PeakRankTile,
  StreaksTile,
  WinRateTile,
} from './ExplorerBriefingTiles'

type T = (key: ExplorerManifestKey, values?: Record<string, string | number>) => string

interface Props {
  briefing: ExplorerBriefing | null | undefined
  t: T
}

function formatPeriod(
  start: string | null | undefined,
  end: string | null | undefined,
  locale: string,
): string | undefined {
  if (!start) return undefined
  // Intervalle daté COMPLET (année incluse) via le helper canonique formatDateRange —
  // factorise mois/année (« 3–12 mars 2025 ») ; date simple si end absent/égal.
  return formatDateRange(start, end, locale === 'en' ? 'en-US' : 'fr-FR')
}

export function ExplorerBriefingStrip({ briefing, t }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  if (!briefing) return null

  const scope = briefing.scope
  const baseline = briefing.baseline
  const period = formatPeriod(briefing.period_start, briefing.period_end, locale)
  const matchesCount = scope?.matches ?? 0

  const kda = scope?.kda ?? null
  const perf = scope?.avg_perf ?? null
  const vs = t('explorer.briefing.vs_baseline')
  // Plein historique (scope == baseline) → deltas « vs habituel » nuls par
  // construction : on masque le fragment (valeur + libellé) sur le socle ET les
  // lignes de dimension (P-1). Sous filtre, comportement V1 inchangé.
  const fullHistory = isFullHistoryScope(scope?.matches, baseline?.matches)
  // Meilleure série : conditionnelle omise si les deux segments sont à zéro (DP-3).
  const streaks = briefing.streaks ?? null
  const showStreaks =
    streaks != null && ((streaks.best_win_streak ?? 0) > 0 || (streaks.worst_loss_streak ?? 0) > 0)

  const lowSample = !!briefing.low_sample
  const peakRanks = scope?.peak_ranks ?? []
  const peakMmr = scope?.peak_team_mmr ?? null

  // Cascade des tuiles conditionnelles (DEC-TILES) : collectées par PRIORITÉ
  // décroissante (Meilleure série > Pic rang > Pic MMR), au plus 2 rendues → socle
  // plafonné à 8 (6 base hors low_sample + 2). Omises entièrement en low_sample.
  const conditionalTiles: ReactNode[] = []
  if (scope && !lowSample) {
    if (showStreaks && streaks != null) {
      conditionalTiles.push(<StreaksTile key="streaks" streaks={streaks} t={t} />)
    }
    if (peakRanks.length > 0) {
      conditionalTiles.push(<PeakRankTile key="peak-rank" ranks={peakRanks} t={t} />)
    }
    if (peakMmr != null) {
      conditionalTiles.push(<PeakMmrTile key="peak-mmr" value={peakMmr} t={t} />)
    }
  }
  const cappedConditionals = conditionalTiles.slice(0, 2)

  return (
    <div className="space-y-2">
      <div className="grid gap-2 grid-cols-2 sm:[grid-template-columns:repeat(auto-fit,minmax(150px,1fr))]">
        {/* Matchs + période */}
        <BriefingTile
          label={t('explorer.briefing.matches_label')}
          value={String(matchesCount)}
          sub={period}
          accent="outcome-draw"
        />

        {/* Taux de victoire (hero : ruban V-D-N + tooltip des 4 issues, DEC-TILES) */}
        {scope && <WinRateTile scope={scope} baseline={baseline} fullHistory={fullHistory} t={t} />}

        {/* FDA agrégat + delta */}
        {scope && (
          <BriefingTile
            label={t('explorer.briefing.fda_label')}
            info={<InfoTooltip content={t('explorer.briefing.tip_fda')} iconClass="w-3.5 h-3.5" />}
            value={
              <span style={kda != null ? { color: kdaNetColor(kda) } : undefined}>
                {kda != null ? kda.toFixed(2) : '—'}
              </span>
            }
            sub={
              baseline && !fullHistory ? (
                <>
                  <span
                    className="font-semibold"
                    style={{ color: tokenCssVar(deltaToken(baseline.delta_kda)) }}
                  >
                    {formatSignedFixed(baseline.delta_kda, 2)}
                  </span>{' '}
                  {vs}
                </>
              ) : undefined
            }
            accent={deltaToken(kda)}
          />
        )}

        {/* Perf. moyenne colorée (DEC-PERF) + delta */}
        {scope && (
          <BriefingTile
            label={t('explorer.briefing.perf_label')}
            info={<InfoTooltip content={t('explorer.briefing.tip_perf')} iconClass="w-3.5 h-3.5" />}
            value={
              perf != null ? (
                <span style={{ color: getPerfColor(perf) }}>{perf.toFixed(0)}</span>
              ) : (
                '—'
              )
            }
            sub={
              baseline && !fullHistory && baseline.delta_perf != null ? (
                <>
                  <span
                    className="font-semibold"
                    style={{ color: tokenCssVar(deltaToken(baseline.delta_perf)) }}
                  >
                    {formatSignedFixed(baseline.delta_perf, 0)}
                  </span>{' '}
                  {vs}
                </>
              ) : undefined
            }
          />
        )}

        {/* Durée totale + Pic FDA : tuiles de base hors low_sample */}
        {scope && !lowSample && <DurationTile seconds={scope.total_duration_seconds} t={t} />}
        {scope && !lowSample && <PeakKdaTile value={scope.peak_kda} t={t} />}

        {/* Conditionnelles en cascade (Meilleure série > Pic rang > Pic MMR, au plus 2) */}
        {cappedConditionals}
      </div>

      {/* Échantillon faible : socle seul + mention ; sinon modules conditionnels. */}
      {briefing.low_sample ? (
        <p className="text-2xs text-muted-foreground">
          {t('explorer.briefing.low_sample', { n: matchesCount })}
        </p>
      ) : (
        <ExplorerBriefingModules briefing={briefing} t={t} hideDelta={fullHistory} />
      )}
    </div>
  )
}
