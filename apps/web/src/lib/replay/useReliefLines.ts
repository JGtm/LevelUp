/**
 * useReliefLines — L'INTERRUPTEUR « TRAITS DU RELIEF », UNE SEULE VALEUR pour les trois fonds et les trois vues 2D
 * (rejeu, onglet Tactique, « Occupation du terrain ») : un OUI/NON stocké sous `replay-relief-lines`, lu et écrit des
 * trois côtés — allumer les traits au rejeu les allume partout. ALLUMÉS d'office ; le stockage ne retient que ce que
 * l'utilisateur a fait, absent ou refusé -> allumés.
 *
 * LA REPRISE DES CLÉS D'AVANT. Les traits étaient retenus PAR FOND (`replay-relief-lines.<fond>`, allumés d'office sur
 * le seul Relief). La première lecture reprend le choix que l'utilisateur avait fait, sans rien perdre : celui du fond
 * retenu s'il y en a un, sinon celui du Relief, du Plan, du Rendu, dans cet ordre. Les trois clés d'avant sont alors
 * effacées ; sans aucun choix d'avant, rien n'est écrit et les traits sont allumés. La reprise est idempotente : sans
 * clé d'avant, elle ne fait rien.
 *
 * Même patron que `usePersistedFlag` : l'abonnement fait bouger ensemble tous les composants qui lisent la clé, et la
 * persistance vient APRÈS la mise à jour d'état, depuis le gestionnaire d'événement — jamais depuis un updater
 * (cf. `usePersistedFlag`).
 */
import { useCallback, useEffect, useState } from 'react'

import { persistPreference, readStoredFlag, readStoredRaw, removePreference, subscribePreference } from './preferenceStore'

export const RELIEF_LINES_KEY = 'replay-relief-lines'
export const RELIEF_LINES_DEFAULT = true

/** Les clés d'avant (une par fond) et la clé du fond retenu, qui décide laquelle de ces valeurs prime. */
const LEGACY_PREFIX = 'replay-relief-lines.'
const LEGACY_FONDS = ['relief', 'plan', 'rendu'] as const
const BACKGROUND_STYLE_KEY = 'replay-background-style'

function legacyValue(fond: string): boolean | null {
  const raw = readStoredRaw(`${LEGACY_PREFIX}${fond}`)
  return raw === 'true' ? true : raw === 'false' ? false : null
}

/** Reprend le choix d'avant (par fond) dans la clé unique, puis efface les anciennes clés. Sans ancienne clé : rien. */
export function adoptLegacyReliefLines(): void {
  const present = LEGACY_FONDS.filter((fond) => readStoredRaw(`${LEGACY_PREFIX}${fond}`) !== null)
  if (present.length === 0) return
  if (readStoredRaw(RELIEF_LINES_KEY) === null) {
    const current = readStoredRaw(BACKGROUND_STYLE_KEY)
    const order = [...(current ? [current] : []), ...LEGACY_FONDS]
    const chosen = order.map(legacyValue).find((v) => v !== null)
    if (chosen !== undefined && chosen !== null) persistPreference(RELIEF_LINES_KEY, String(chosen))
  }
  for (const fond of present) removePreference(`${LEGACY_PREFIX}${fond}`)
}

/** Les traits du relief retenus, ou allumés. */
export function readReliefLines(): boolean {
  adoptLegacyReliefLines()
  return readStoredFlag(RELIEF_LINES_KEY, RELIEF_LINES_DEFAULT)
}

/** L'état des traits du relief, et sa bascule. */
export function useReliefLines(): [boolean, () => void] {
  const [value, setValue] = useState(readReliefLines)
  useEffect(() => subscribePreference(RELIEF_LINES_KEY, (raw) => setValue(raw === 'true')), [])
  const toggle = useCallback(() => {
    const next = !value
    setValue(next)
    persistPreference(RELIEF_LINES_KEY, String(next))
  }, [value])
  return [value, toggle]
}
