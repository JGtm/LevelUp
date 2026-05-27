/**
 * SessionCompareOCDR — Rendement offensif / Résistance défensive A vs B.
 * Affiche avg_oc et avg_dr sous forme de tableau comparatif avec CombatYieldBar visuelle.
 */
import { CombatYieldBar } from '@/components/ui/combat-yield-bar'
import type { SessionCompareEntry } from '@/lib/api/types'

export interface SessionCompareOCDRProps {
  sessionA: SessionCompareEntry | null
  sessionB: SessionCompareEntry | null
  labels: {
    title: string
    empty: string
  }
}

function fmt(v: number | null | undefined, decimals = 2): string {
  return v != null ? v.toFixed(decimals) : '—'
}

function winnerClass(
  a: number | null | undefined,
  b: number | null | undefined,
  higherIsBetter = true,
): { a: string; b: string } {
  if (a == null || b == null) return { a: 'text-foreground', b: 'text-foreground' }
  if (Math.abs(a - b) < 0.005) return { a: 'text-foreground', b: 'text-foreground' }
  const aWins = higherIsBetter ? a > b : a < b
  return {
    a: aWins ? 'text-compare-a font-semibold' : 'text-foreground',
    b: aWins ? 'text-foreground' : 'text-compare-b font-semibold',
  }
}

export function SessionCompareOCDR({ sessionA, sessionB, labels }: SessionCompareOCDRProps) {
  const hasData =
    sessionA?.avg_oc != null ||
    sessionA?.avg_dr != null ||
    sessionB?.avg_oc != null ||
    sessionB?.avg_dr != null

  if (!hasData) {
    return (
      <p className="text-sm text-muted-foreground italic text-center py-4">{labels.empty}</p>
    )
  }

  const ocColors = winnerClass(sessionA?.avg_oc, sessionB?.avg_oc, true)
  const drColors = winnerClass(sessionA?.avg_dr, sessionB?.avg_dr, true)

  return (
    <div className="space-y-4">
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
            <td className="py-2 pr-4 text-muted-foreground">OC</td>
            <td className={`py-2 pr-4 text-right tabular-nums ${ocColors.a}`}>
              {fmt(sessionA?.avg_oc)}
            </td>
            <td className={`py-2 text-right tabular-nums ${ocColors.b}`}>
              {fmt(sessionB?.avg_oc)}
            </td>
          </tr>
          <tr className="border-b last:border-0">
            <td className="py-2 pr-4 text-muted-foreground">DR</td>
            <td className={`py-2 pr-4 text-right tabular-nums ${drColors.a}`}>
              {fmt(sessionA?.avg_dr)}
            </td>
            <td className={`py-2 text-right tabular-nums ${drColors.b}`}>
              {fmt(sessionB?.avg_dr)}
            </td>
          </tr>
        </tbody>
      </table>

      {/* Visualisation barres OC/DR côte à côte */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        {sessionA && (sessionA.avg_oc != null || sessionA.avg_dr != null) && (
          <div className="flex flex-col items-center gap-1">
            <p className="text-xs font-semibold text-compare-a">A</p>
            <CombatYieldBar
              offensiveConversion={sessionA.avg_oc}
              defensiveResistance={sessionA.avg_dr}
            />
          </div>
        )}
        {sessionB && (sessionB.avg_oc != null || sessionB.avg_dr != null) && (
          <div className="flex flex-col items-center gap-1">
            <p className="text-xs font-semibold text-compare-b">B</p>
            <CombatYieldBar
              offensiveConversion={sessionB.avg_oc}
              defensiveResistance={sessionB.avg_dr}
            />
          </div>
        )}
      </div>
    </div>
  )
}
