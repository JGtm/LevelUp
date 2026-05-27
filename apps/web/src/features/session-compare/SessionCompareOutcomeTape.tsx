/**
 * SessionCompareOutcomeTape — bandes séquentielles A et B empilées.
 * Utilise OutcomeSequenceTape pour chaque session, libellés via fieldMappings TOML.
 */
import { useMemo } from 'react'

import {
  OutcomeSequenceTape,
  type OutcomePoint,
  type OutcomeSequenceLabels,
} from '@/components/charts/OutcomeSequenceTape'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import type { SessionCompareEntry } from '@/lib/api/types'

export interface SessionCompareOutcomeTapeProps {
  sessionA: SessionCompareEntry | null
  sessionB: SessionCompareEntry | null
  labels: {
    title: string
    sessionA: string
    sessionB: string
    empty: string
  }
  height?: number
}

const OUTCOME_INT_KEY: Record<number, 'win' | 'loss' | 'tie' | 'dnf'> = {
  2: 'win',
  1: 'tie',
  3: 'loss',
  4: 'dnf',
}

function toOutcomePoints(entry: SessionCompareEntry): OutcomePoint[] {
  return [...(entry.matches ?? [])]
    .sort((a, b) => a.start_time.localeCompare(b.start_time))
    .flatMap<OutcomePoint>((m) => {
      const key = m.outcome != null ? OUTCOME_INT_KEY[m.outcome] : undefined
      if (!key) return []
      return [{ outcome: key, matchId: m.match_id, map: m.pair_name, mode: m.playlist_name }]
    })
}

export function SessionCompareOutcomeTape({
  sessionA,
  sessionB,
  labels,
  height = 72,
}: SessionCompareOutcomeTapeProps) {
  const { data: fieldMappings } = useFieldMappings()

  const outcomeLabels: OutcomeSequenceLabels = {
    win: fieldMappings?.outcomes?.win?.label ?? 'win',
    loss: fieldMappings?.outcomes?.loss?.label ?? 'loss',
    tie: fieldMappings?.outcomes?.tie?.label ?? 'tie',
    dnf: fieldMappings?.outcomes?.dnf?.label ?? 'dnf',
  }

  const pointsA = useMemo(() => (sessionA ? toOutcomePoints(sessionA) : []), [sessionA])
  const pointsB = useMemo(() => (sessionB ? toOutcomePoints(sessionB) : []), [sessionB])

  const hasData = pointsA.length > 0 || pointsB.length > 0
  if (!hasData) {
    return <p className="text-sm text-muted-foreground italic text-center py-4">{labels.empty}</p>
  }

  return (
    <div className="space-y-3">
      {pointsA.length > 0 && (
        <div>
          <p className="mb-1 text-xs font-semibold text-compare-a">{labels.sessionA}</p>
          <OutcomeSequenceTape matches={pointsA} labels={outcomeLabels} height={height} />
        </div>
      )}
      {pointsB.length > 0 && (
        <div>
          <p className="mb-1 text-xs font-semibold text-compare-b">{labels.sessionB}</p>
          <OutcomeSequenceTape matches={pointsB} labels={outcomeLabels} height={height} />
        </div>
      )}
    </div>
  )
}
