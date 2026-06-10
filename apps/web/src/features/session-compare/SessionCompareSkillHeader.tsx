/**
 * SessionCompareSkillHeader — affichage du skill rating (LUSR/CSR) A vs B.
 * Chart 01 du mock session_compare.
 */
import type { SessionCompareEntry } from '@/lib/api/types'
import { displayRatingLabel, formatRankDelta } from '@/lib/formatters'

export interface SessionCompareSkillHeaderProps {
  sessionA: SessionCompareEntry | null
  sessionB: SessionCompareEntry | null
  labels: {
    title: string
    deltaLabel: string
    empty: string
  }
}

function SkillRatingCell({
  entry,
  side,
}: {
  entry: SessionCompareEntry | null
  side: 'a' | 'b'
}) {
  const colorClass = side === 'a' ? 'text-compare-a' : 'text-compare-b'

  if (!entry || entry.last_skill_rating == null) {
    return <td className="py-3 px-4 text-center text-muted-foreground text-sm">—</td>
  }

  const rating = Math.round(entry.last_skill_rating)
  const type = displayRatingLabel(entry.skill_rating_type) ?? ''
  const delta = entry.skill_rating_delta

  return (
    <td className={`py-3 px-4 text-center ${colorClass}`}>
      <div className="flex flex-col items-center gap-0.5">
        {type && (
          <span className="text-[10px] font-medium text-muted-foreground tracking-wide">
            {type}
          </span>
        )}
        <span className="text-xl font-bold tabular-nums">{rating}</span>
        {delta != null && (
          <span
            className={`text-xs font-medium ${
              delta > 0 ? 'text-success' : delta < 0 ? 'text-destructive' : 'text-muted-foreground'
            }`}
          >
            {formatRankDelta(delta, entry.skill_rating_type ?? '')}
          </span>
        )}
      </div>
    </td>
  )
}

export function SessionCompareSkillHeader({
  sessionA,
  sessionB,
  labels,
}: SessionCompareSkillHeaderProps) {
  const hasData = (sessionA?.last_skill_rating != null) || (sessionB?.last_skill_rating != null)

  if (!hasData) {
    return (
      <p className="text-sm text-muted-foreground italic text-center py-4">{labels.empty}</p>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full">
        <thead>
          <tr className="border-b text-xs text-muted-foreground">
            <th className="py-2 px-4 text-left">{labels.title}</th>
            <th className="py-2 px-4 text-center text-compare-a">A</th>
            <th className="py-2 px-4 text-center text-compare-b">B</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td className="py-3 px-4 text-sm text-muted-foreground">{labels.deltaLabel}</td>
            <SkillRatingCell entry={sessionA} side="a" />
            <SkillRatingCell entry={sessionB} side="b" />
          </tr>
        </tbody>
      </table>
    </div>
  )
}
