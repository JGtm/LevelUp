/**
 * useCounterSnapshot — hook canonique du pattern « baseline roulante » (A8.2).
 *
 * Snapshot clé→count persisté en localStorage : au 1er run de la session, la
 * baseline vient du localStorage (comparaison inter-sessions) ; ensuite chaque
 * nouveau run (generated_at différent) compare au run PRÉCÉDENT — pas au
 * snapshot figé au mount (bug corrigé 2026-06-10 : un refetch intra-session
 * comparait à l'état pré-mount et pouvait masquer une régression).
 *
 * Factorise les 3 copies historiques (data-quality, convergence, invariants).
 * Garde-rail : counters-snapshot.guard.test.ts interdit readCountersSnapshot(
 * hors de countersTrend.ts et de ce hook.
 */
import { useEffect, useRef, useState } from 'react'

import {
  readCountersSnapshot,
  writeCountersSnapshot,
  type CountersSnapshot,
} from './countersTrend'

/**
 * Retourne la baseline de comparaison (« visite précédente ») pour la réponse
 * courante. `generatedAt` pilote la rotation ; `buildSnapshot` construit le
 * snapshot de la réponse courante (appelé seulement quand generatedAt change —
 * identité de fonction libre, lue via ref).
 */
export function useCounterSnapshot(
  storageKey: string,
  generatedAt: string | undefined,
  buildSnapshot: () => CountersSnapshot,
): CountersSnapshot {
  const [previous, setPrevious] = useState<CountersSnapshot>(() => readCountersSnapshot(storageKey))
  const lastRunRef = useRef<{ generatedAt: string; snapshot: CountersSnapshot } | null>(null)
  const buildRef = useRef(buildSnapshot)
  buildRef.current = buildSnapshot

  useEffect(() => {
    if (!generatedAt) return
    const last = lastRunRef.current
    if (last && last.generatedAt === generatedAt) return
    const snap = buildRef.current()
    if (last) {
      setPrevious(last.snapshot)
    }
    lastRunRef.current = { generatedAt, snapshot: snap }
    writeCountersSnapshot(storageKey, snap)
  }, [generatedAt, storageKey])

  return previous
}
