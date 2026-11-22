/**
 * useScopedCut — LA COUPE PAR ÉTAGE RETENUE, rangée avec sa portée : `null` = « Tout » (le défaut), et une autre
 * portée (une autre carte) repart de « Tout » sans effet ni remise à zéro à la main. Jamais persistée.
 */
import { useCallback, useState } from 'react'

export function useScopedCut(scopeKey: string): [number | null, (z: number | null) => void] {
  const [state, setState] = useState<{ key: string; cut: number | null }>({ key: scopeKey, cut: null })
  const set = useCallback((z: number | null) => setState({ key: scopeKey, cut: z }), [scopeKey])
  return [state.key === scopeKey ? state.cut : null, set]
}
