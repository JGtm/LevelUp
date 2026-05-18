/**
 * SessionOutcomeTape — bande séquentielle des outcomes d'une session.
 *
 * Wrapper de `OutcomeSequenceTape` (charts/) : convertit `SessionDetailMatchRow[]`
 * en `OutcomePoint[]` ordonnés chronologiquement, résout les libellés via
 * `useFieldMappings.outcomes` (TOML). Aucun libellé hardcodé.
 */
import { useMemo } from 'react'

import {
  OutcomeSequenceTape,
  type OutcomePoint,
  type OutcomeSequenceLabels,
} from '@/components/charts/OutcomeSequenceTape'
import type { SessionDetailMatchRow } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

import { outcomeIntToKey } from './_shared'

interface Props {
  matches: SessionDetailMatchRow[]
  height?: number
}

export function SessionOutcomeTape({ matches, height = 90 }: Props) {
  const { data: fieldMappings } = useFieldMappings()

  const points: OutcomePoint[] = useMemo(() => {
    return [...matches]
      .sort((a, b) => a.start_time.localeCompare(b.start_time))
      .map((m) => {
        const key = outcomeIntToKey(m.outcome)
        if (!key) return null
        return {
          outcome: key,
          matchId: m.match_id,
          map: m.pair_name,
          mode: m.playlist_name,
        } satisfies OutcomePoint
      })
      .filter((p): p is OutcomePoint => p !== null)
  }, [matches])

  // Libellés résolus via fieldMappings.outcomes — pas de string FR/EN hardcodée.
  // Fallback ultime sur la clé canonique pour ne pas casser le rendu si TOML absent.
  const labels: OutcomeSequenceLabels = {
    win: fieldMappings?.outcomes?.win?.label ?? 'win',
    loss: fieldMappings?.outcomes?.loss?.label ?? 'loss',
    tie: fieldMappings?.outcomes?.tie?.label ?? 'tie',
    dnf: fieldMappings?.outcomes?.dnf?.label ?? 'dnf',
  }

  if (points.length === 0) return null

  return <OutcomeSequenceTape matches={points} labels={labels} height={height} />
}
