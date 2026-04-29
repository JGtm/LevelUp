/**
 * MatchStatCards — composants KPI pour le Match View.
 *
 * C3 : StatExpectedCard — réel vs attendu (K/D/A)
 * C4 : MatchRankBadge   — delta rang CSR/LUSR après match
 * C5 : KdIndicatorCard  — K/D vs nemesis principal
 *
 * NATIVE_COMPONENTS items C3, C4, C5.
 */
import type { MatchViewRank, MatchExpectedStats, MatchNemesisRow } from '@/lib/api/types'
import { skillDeltaScale, kdScale } from '@/lib/accessibility/scales'
import { tokenCssVar } from '@/lib/accessibility'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

// ---------------------------------------------------------------------------
// C3 — StatExpectedCard (réel vs attendu)
// ---------------------------------------------------------------------------

interface StatExpectedCardProps {
  label: string
  actual: number | null
  expected: number | null
  lowerIsBetter?: boolean
  /** Si false, card grisée */
  hasData: boolean
}

export function StatExpectedCard({ label, actual, expected, lowerIsBetter = false, hasData }: StatExpectedCardProps) {
  const delta = actual != null && expected != null ? actual - expected : null
  const isFavorable =
    delta == null ? null : lowerIsBetter ? delta < 0 : delta > 0

  return (
    <div
      className={`rounded-lg border px-4 py-3 text-center ${
        hasData ? 'border-border bg-card' : 'border-border/40 bg-card/50 opacity-50'
      }`}
    >
      <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">{label}</p>
      <p className="text-2xl font-bold text-foreground">{actual ?? '—'}</p>
      {hasData && expected != null ? (
        <p className="text-xs text-muted-foreground mt-0.5">
          attendu {expected.toFixed(1)}{' '}
          {delta != null && (
            <span
              className="font-semibold"
              style={{
                color: isFavorable
                  ? tokenCssVar('divergent-pos')
                  : isFavorable === false
                    ? tokenCssVar('divergent-neg')
                    : undefined,
              }}
            >
              ({delta > 0 ? '+' : ''}{delta.toFixed(1)})
            </span>
          )}
        </p>
      ) : (
        <p className="text-xs text-muted-foreground mt-0.5">pas de données CSR</p>
      )}
    </div>
  )
}

interface ExpectedCardsSectionProps {
  kpis: { kills: number | null; deaths: number | null; assists: number | null }
  expectedStats: MatchExpectedStats
}

export function ExpectedCardsSection({ kpis, expectedStats }: ExpectedCardsSectionProps) {
  const { has_expected_data, expected_kills, expected_deaths, expected_assists } = expectedStats
  const { data: fieldMappings } = useFieldMappings()
  const labelOf = (key: string): string =>
    fieldMappings?.fields[key]?.label ?? key
  return (
    <div className="grid grid-cols-3 gap-3">
      <StatExpectedCard
        label={labelOf('kills')}
        actual={kpis.kills}
        expected={expected_kills}
        lowerIsBetter={false}
        hasData={has_expected_data}
      />
      <StatExpectedCard
        label={labelOf('deaths')}
        actual={kpis.deaths}
        expected={expected_deaths}
        lowerIsBetter={true}
        hasData={has_expected_data}
      />
      <StatExpectedCard
        label={labelOf('assists')}
        actual={kpis.assists}
        expected={expected_assists}
        lowerIsBetter={false}
        hasData={has_expected_data}
      />
    </div>
  )
}

// ---------------------------------------------------------------------------
// C4 — MatchRankBadge (delta rang CSR/LUSR)
// ---------------------------------------------------------------------------

interface MatchRankBadgeProps {
  rank: MatchViewRank
  hadBotTeammate?: boolean
}

export function MatchRankBadge({ rank, hadBotTeammate = false }: MatchRankBadgeProps) {
  if (rank.rating_type === 'none' || !rank.tier_label) return null

  const delta = rank.delta_value
  const deltaToken = delta != null ? skillDeltaScale(delta) : null
  const deltaColor = deltaToken === 'divergent-neutral' || deltaToken === null
    ? 'text-muted-foreground'
    : undefined
  const deltaStyle = deltaToken && deltaToken !== 'divergent-neutral'
    ? { color: tokenCssVar(deltaToken) }
    : undefined

  return (
    <div className="flex items-center gap-3 rounded-lg border border-border bg-card px-4 py-3">
      <div className="flex-1">
        <p className="text-xs text-muted-foreground uppercase tracking-wide">{rank.rating_type}</p>
        <div className="flex items-baseline gap-2 mt-0.5">
          <span className="text-lg font-bold text-foreground">{rank.tier_label}</span>
          {rank.numeric_value != null && (
            <span className="text-sm text-muted-foreground">{rank.numeric_value.toFixed(0)} pts</span>
          )}
        </div>
      </div>
      {delta != null && (
        <span
          className={`text-xl font-semibold ${deltaColor ?? ''}`}
          style={deltaStyle}
        >
          {delta > 0 ? '+' : ''}{delta.toFixed(0)}
        </span>
      )}
      {hadBotTeammate && (
        <span className="ml-2 rounded bg-warning/20 px-2 py-0.5 text-[10px] text-warning" title="Coéquipier bot présent dans ce match">
          🤖 bot
        </span>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// C5 — KdIndicatorCard (K/D vs nemesis)
// ---------------------------------------------------------------------------

interface KdIndicatorCardProps {
  nemesis: MatchNemesisRow | null
}

export function KdIndicatorCard({ nemesis }: KdIndicatorCardProps) {
  if (!nemesis) return null

  const kd = nemesis.killed_me > 0 ? nemesis.i_killed / nemesis.killed_me : nemesis.i_killed

  return (
    <div className="rounded-lg border border-border bg-card px-4 py-3">
      <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">
        K/D vs nemesis
      </p>
      <div className="flex items-baseline gap-2">
        <span
          className="text-2xl font-bold"
          style={{ color: tokenCssVar(kdScale(kd)) }}
        >{kd.toFixed(2)}</span>
        <span className="text-sm text-muted-foreground">vs {nemesis.gamertag}</span>
      </div>
      <p className="text-xs text-muted-foreground mt-0.5">
        {nemesis.i_killed} kills · {nemesis.killed_me} morts
      </p>
    </div>
  )
}
