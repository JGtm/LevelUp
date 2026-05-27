/**
 * SessionCompareMMR — MMR moyen team/enemy A vs B.
 * Chart 06 du mock session_compare.
 */
import type { SessionCompareEntry } from '@/lib/api/types'

export interface SessionCompareMMRProps {
  sessionA: SessionCompareEntry | null
  sessionB: SessionCompareEntry | null
  labels: {
    title: string
    teamMMR: string
    enemyMMR: string
    empty: string
  }
}

function fmt(v: number | null | undefined): string {
  return v != null ? v.toFixed(0) : '—'
}

function winnerClass(a: number | null | undefined, b: number | null | undefined, higherIsBetter = true): { a: string; b: string } {
  if (a == null || b == null) return { a: 'text-foreground', b: 'text-foreground' }
  if (Math.abs(a - b) < 1) return { a: 'text-foreground', b: 'text-foreground' }
  const aWins = higherIsBetter ? a > b : a < b
  return {
    a: aWins ? 'text-compare-a font-semibold' : 'text-foreground',
    b: aWins ? 'text-foreground' : 'text-compare-b font-semibold',
  }
}

export function SessionCompareMMR({
  sessionA,
  sessionB,
  labels,
}: SessionCompareMMRProps) {
  const hasData =
    sessionA?.avg_team_mmr != null ||
    sessionA?.avg_enemy_mmr != null ||
    sessionB?.avg_team_mmr != null ||
    sessionB?.avg_enemy_mmr != null

  if (!hasData) {
    return <p className="text-sm text-muted-foreground italic text-center py-4">{labels.empty}</p>
  }

  const teamColors = winnerClass(sessionA?.avg_team_mmr, sessionB?.avg_team_mmr)
  const enemyColors = winnerClass(sessionA?.avg_enemy_mmr, sessionB?.avg_enemy_mmr)

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b text-xs text-muted-foreground">
            <th className="py-2 pr-4 text-left">{labels.title}</th>
            <th className="py-2 pr-4 text-right text-compare-a">A</th>
            <th className="py-2 text-right text-compare-b">B</th>
          </tr>
        </thead>
        <tbody>
          <tr className="border-b last:border-0">
            <td className="py-2 pr-4 text-muted-foreground">{labels.teamMMR}</td>
            <td className={`py-2 pr-4 text-right tabular-nums ${teamColors.a}`}>
              {fmt(sessionA?.avg_team_mmr)}
            </td>
            <td className={`py-2 text-right tabular-nums ${teamColors.b}`}>
              {fmt(sessionB?.avg_team_mmr)}
            </td>
          </tr>
          <tr className="border-b last:border-0">
            <td className="py-2 pr-4 text-muted-foreground">{labels.enemyMMR}</td>
            <td className={`py-2 pr-4 text-right tabular-nums ${enemyColors.a}`}>
              {fmt(sessionA?.avg_enemy_mmr)}
            </td>
            <td className={`py-2 text-right tabular-nums ${enemyColors.b}`}>
              {fmt(sessionB?.avg_enemy_mmr)}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  )
}
