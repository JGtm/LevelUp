/**
 * Délai du débrief — logique d'affichage du tableau de la file de construction : quelles
 * colonnes, quelle étape dans chaque colonne, et comment écrire un écart depuis la fin du
 * match. Le calcul des délais est fait par le serveur (analysis.DebriefDelay).
 */
import type { MatchDelay, MatchDelayStep } from '@/lib/api/types'
import type { AdminManifestKey } from '@/lib/i18n/generated/admin'

type StepName = MatchDelayStep['step']

/** Une colonne du tableau : sa clé de libellé et les étapes qu'elle affiche (la première présente). */
export interface DebriefColumn {
  key: AdminManifestKey
  steps: StepName[]
}

/**
 * Les colonnes, dans l'ordre du chemin d'un match. La mise en file (ouvrier) et le début de la
 * construction locale partagent une colonne : un match ne passe que par l'une des deux.
 */
export const DEBRIEF_COLUMNS: DebriefColumn[] = [
  { key: 'admin.buildqueue.delays_col_detected', steps: ['detected'] },
  { key: 'admin.buildqueue.delays_col_stats', steps: ['stats_written'] },
  { key: 'admin.buildqueue.delays_col_killsource', steps: ['killsource_written'] },
  { key: 'admin.buildqueue.delays_col_replay', steps: ['replay_enqueued', 'replay_build_started'] },
  { key: 'admin.buildqueue.delays_col_claimed', steps: ['replay_claimed'] },
  { key: 'admin.buildqueue.delays_col_artifact', steps: ['artifact_stored'] },
  { key: 'admin.buildqueue.delays_col_derivations', steps: ['derivations_written'] },
  { key: 'admin.buildqueue.delays_col_citations', steps: ['citations_written'] },
]

/** L'étape affichée dans une colonne pour ce match, ou undefined. */
export function stepForColumn(delay: MatchDelay, column: DebriefColumn): MatchDelayStep | undefined {
  const steps = delay.steps ?? []
  for (const name of column.steps) {
    const found = steps.find((s) => s.step === name)
    if (found) return found
  }
  return undefined
}

/** Écart depuis la fin du match, « +m:ss » (« −m:ss » avant la fin), « — » sans référence. */
export function formatOffset(seconds: number | undefined | null): string {
  if (seconds === undefined || seconds === null || Number.isNaN(seconds)) return '—'
  const sign = seconds < 0 ? '−' : '+'
  const total = Math.round(Math.abs(seconds))
  const minutes = Math.floor(total / 60)
  const rest = total % 60
  return `${sign}${minutes}:${String(rest).padStart(2, '0')}`
}

/** Statut d'objectif du match : tenu, dépassé, ou incomplet (données encore attendues). */
export function objectiveStatus(delay: MatchDelay): 'met' | 'missed' | 'pending' {
  if (delay.objective_met === undefined || delay.objective_met === null) return 'pending'
  return delay.objective_met ? 'met' : 'missed'
}
