/**
 * ExplorerBriefingStrip — bandeau de briefing au-dessus du tableau (mode Matchs).
 *
 * Lecture compacte du RÉSULTAT DE RECHERCHE : rangée socle de 4 à 6 tuiles KPI
 * (Matchs, Taux de victoire + micro-sparkline de tendance, FDA agrégat, Perf.
 * moyenne, et — conditionnelles — Classement et Séries) avec deltas vs baseline
 * personnelle. Les modules « Par… » + Moments forts sont rendus par
 * ExplorerBriefingModules sous le socle.
 *
 * Dégradation : briefing absent → rien ; low_sample → socle + mention
 * échantillon faible, aucun module. Aucune couleur hex : tokens sémantiques.
 */
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { useCapability } from '@/lib/capabilities/capabilities'
import { tokenCssVar } from '@/lib/accessibility'
import { kdaNetColor, winRateColor } from '@/lib/colors/outcomePalette'
import { formatDateRange, formatPercentInt } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ExplorerBriefing } from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { BriefingTile } from './BriefingTile'
import {
  deltaToken,
  formatSignedFixed,
  formatSignedPoints,
  isFullHistoryScope,
} from './ExplorerBriefing.logic'
import { ExplorerBriefingModules } from './ExplorerBriefingModules'
import { RankedTile, StreaksTile } from './ExplorerBriefingTiles'

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
  // Capability 'ranked' lue AVANT tout early-return (règle des hooks React).
  const hasRanked = useCapability('ranked')
  if (!briefing) return null

  const scope = briefing.scope
  const baseline = briefing.baseline
  const period = formatPeriod(briefing.period_start, briefing.period_end, locale)
  const matchesCount = scope?.matches ?? 0

  const wr = scope?.win_rate ?? null
  const kda = scope?.kda ?? null
  const perf = scope?.avg_perf ?? null
  const vs = t('explorer.briefing.vs_baseline')
  // Plein historique (scope == baseline) → deltas « vs habituel » nuls par
  // construction : on masque le fragment (valeur + flèche + libellé) sur le socle
  // ET les lignes de dimension (P-1). Sous filtre, comportement V1 inchangé.
  const fullHistory = isFullHistoryScope(scope?.matches, baseline?.matches)
  // Séries : tuile omise si les deux segments sont à zéro (DP-3).
  const streaks = briefing.streaks ?? null
  const showStreaks =
    streaks != null && ((streaks.best_win_streak ?? 0) > 0 || (streaks.worst_loss_streak ?? 0) > 0)

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

        {/* Taux de victoire + V-D-N + delta (sparkline retirée — DEC-SPARK V4) */}
        {scope && (
          <BriefingTile
            label={t('explorer.briefing.win_rate_label')}
            info={<InfoTooltip content={t('explorer.briefing.tip_win_rate')} iconClass="w-3.5 h-3.5" />}
            value={
              <span style={wr != null ? { color: winRateColor(wr) } : undefined}>
                {formatPercentInt(wr)}
              </span>
            }
            sub={
              <>
                {t('explorer.briefing.record_vdn', {
                  w: scope.wins,
                  l: scope.losses,
                  t: scope.ties,
                })}
                {baseline && !fullHistory && (
                  <>
                    {' · '}
                    <span
                      className="font-semibold"
                      style={{ color: tokenCssVar(deltaToken(baseline.delta_win_rate)) }}
                    >
                      {formatSignedPoints(baseline.delta_win_rate)}
                    </span>{' '}
                    {vs}
                  </>
                )}
              </>
            }
            accent={deltaToken(wr != null ? wr - 0.5 : null)}
          />
        )}

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

        {/* Perf. moyenne + delta */}
        {scope && (
          <BriefingTile
            label={t('explorer.briefing.perf_label')}
            info={<InfoTooltip content={t('explorer.briefing.tip_perf')} iconClass="w-3.5 h-3.5" />}
            value={perf != null ? perf.toFixed(0) : '—'}
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

        {/* Classement (tuile) — gaté capability 'ranked' + DTO présent (DP-2) */}
        {hasRanked && briefing.ranked != null && <RankedTile ranked={briefing.ranked} t={t} />}

        {/* Séries (tuile) — omise si les deux segments à zéro (DP-3) */}
        {showStreaks && streaks != null && <StreaksTile streaks={streaks} t={t} />}
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
