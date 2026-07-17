/**
 * ExplorerBriefingStrip — bandeau de briefing au-dessus du tableau (mode Matchs).
 *
 * Lecture compacte du RÉSULTAT DE RECHERCHE : rangée socle de 4 KPI (Matchs,
 * Bilan, FDA agrégat, Perf. moyenne) avec deltas vs baseline personnelle, puis
 * frise des résultats. Les modules conditionnels (dimensions, tendance, classé)
 * sont rendus par ExplorerBriefingModules (Lot C) sous le socle.
 *
 * Dégradation : briefing absent → rien ; low_sample → socle + mention
 * échantillon faible, aucun module. Aucune couleur hex : tokens sémantiques.
 */
import { KpiCard } from '@/components/cards/KpiCard'
import { OutcomeSequenceTape, type OutcomePoint } from '@/components/charts/OutcomeSequenceTape'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { kdaNetColor, winRateColor } from '@/lib/colors/outcomePalette'
import { formatDateRange, formatPercentInt } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ExplorerBriefing } from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import {
  formatSignedFixed,
  formatSignedPoints,
  isFullHistoryScope,
  outcomeCodeToValue,
  signOf,
} from './ExplorerBriefing.logic'
import { ExplorerBriefingModules } from './ExplorerBriefingModules'

type T = (key: ExplorerManifestKey, values?: Record<string, string | number>) => string

interface Props {
  briefing: ExplorerBriefing | null | undefined
  t: T
}

/** Token de couleur d'un delta signé (positif = gagnant, négatif = perdant, nul = neutre). */
function deltaToken(v: number | null | undefined): SemanticToken {
  const s = signOf(v)
  return s > 0 ? 'outcome-win' : s < 0 ? 'outcome-loss' : 'outcome-draw'
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

interface TileProps {
  label: string
  value: React.ReactNode
  sub?: React.ReactNode
  accent?: SemanticToken
}

function BriefingTile({ label, value, sub, accent }: TileProps) {
  return (
    <KpiCard accent={accent} className="h-full">
      <div className="px-3 py-2">
        <p className="text-3xs uppercase tracking-wide text-muted-foreground">{label}</p>
        <div className="mt-0.5 text-xl font-bold tabular-nums leading-tight text-foreground">
          {value}
        </div>
        {sub && <div className="mt-0.5 text-2xs text-muted-foreground">{sub}</div>}
      </div>
    </KpiCard>
  )
}

export function ExplorerBriefingStrip({ briefing, t }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  if (!briefing) return null

  const scope = briefing.scope
  const baseline = briefing.baseline
  const period = formatPeriod(briefing.period_start, briefing.period_end, locale)
  const matchesCount = scope?.matches ?? briefing.outcome_sequence?.length ?? 0

  const tapePoints: OutcomePoint[] = (briefing.outcome_sequence ?? []).map((o) => ({
    outcome: outcomeCodeToValue(o.outcome_code),
    matchId: o.match_id,
  }))

  const wr = scope?.win_rate ?? null
  const kda = scope?.kda ?? null
  const perf = scope?.avg_perf ?? null
  const vs = t('explorer.briefing.vs_baseline')
  // Plein historique (scope == baseline) → deltas « vs habituel » nuls par
  // construction : on masque le fragment (valeur + flèche + libellé) sur le socle
  // ET les lignes de dimension (P-1). Sous filtre, comportement V1 inchangé.
  const fullHistory = isFullHistoryScope(scope?.matches, baseline?.matches)

  return (
    <div className="space-y-2">
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        {/* Matchs + période */}
        <BriefingTile
          label={t('explorer.briefing.matches_label')}
          value={String(matchesCount)}
          sub={period}
          accent="outcome-draw"
        />

        {/* Bilan : taux de victoire + V-D-N + delta */}
        {scope && (
          <BriefingTile
            label={t('explorer.briefing.win_rate_label')}
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
      </div>

      {/* Frise des résultats */}
      {tapePoints.length > 0 && (
        <OutcomeSequenceTape
          matches={tapePoints}
          height={64}
          labels={{
            win: t('explorer.briefing.series_win'),
            loss: t('explorer.briefing.series_loss'),
            tie: t('explorer.briefing.series_tie'),
            dnf: t('explorer.briefing.series_dnf'),
          }}
        />
      )}

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
