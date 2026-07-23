/**
 * actionJournalDisplay — logique PURE d'affichage du journal des actions (C2)
 * et du banner de dernier cycle post-sync (C1), séparée des composants pour
 * testabilité sans rendu (canon admin : *Display.ts + *Display.test.ts).
 */
import type { AdminManifestKey } from '@/lib/i18n/generated/admin'
import type { AdminActionJournalEntry, SchedulerSnapshot } from '@/lib/api/types'

// ─── C2 : « Dernière exécution » sous un bouton d'action globale ──────────────

export interface ActionRunDisplay {
  /** Clé i18n du déclencheur (manuel vs automatique). */
  triggerKey: AdminManifestKey
  /** Issue en échec → suffixe destructive. */
  failed: boolean
  /** Horodatage RFC3339 de la dernière exécution. */
  at: string
}

/**
 * describeActionRun — null si l'action n'a jamais tourné (→ « Jamais exécutée »),
 * sinon les éléments d'affichage de la dernière exécution.
 */
export function describeActionRun(entry: AdminActionJournalEntry | undefined): ActionRunDisplay | null {
  if (!entry || !entry.last_run_at) return null
  return {
    triggerKey: entry.trigger === 'tick' ? 'admin.actions.trigger_cron' : 'admin.actions.trigger_manual',
    failed: entry.outcome === 'error',
    at: entry.last_run_at,
  }
}

// ─── C1 : banner « Dernier cycle » (post-sync réhydraté au boot) ──────────────

export type LastCycleKind =
  | 'none' // aucun cycle connu (jamais tourné, aucune donnée)
  | 'live' // au moins un cycle depuis ce boot
  | 'previous' // snapshot réhydraté d'avant le redémarrage (daté)

export interface LastCycleDisplay {
  kind: LastCycleKind
  /** Horodatage du dernier cycle ('' si kind === 'none'). */
  at: string
}

/**
 * describeLastCycle — classe l'état du dernier cycle post-sync. last_cycle_at à
 * zéro (temps Go) sérialise en « 0001-… » : traité comme « aucun cycle connu ».
 */
export function describeLastCycle(snapshot: SchedulerSnapshot | undefined): LastCycleDisplay {
  const at = snapshot?.last_cycle_at
  const hasCycle = !!at && !at.startsWith('0001-')
  if (!hasCycle) return { kind: 'none', at: '' }
  return { kind: snapshot?.since_boot ? 'live' : 'previous', at: at as string }
}

/** Clé i18n du libellé du banner selon l'état (jamais appelée pour 'none'). */
export function lastCycleLabelKey(kind: Exclude<LastCycleKind, 'none'>): AdminManifestKey {
  return kind === 'live' ? 'admin.convergence.last_cycle' : 'admin.convergence.last_cycle_prev_boot'
}
