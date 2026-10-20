import { useState, type Dispatch, type SetStateAction } from 'react'

/**
 * useServerDraft — la copie éditable d'un objet servi par une requête (réglages, etc.).
 *
 * La copie se réaligne chaque fois que la requête livre un NOUVEL objet (ajustement pendant le
 * rendu, pattern React « valeur précédente », sans effet) : un refetch réaligne l'état local,
 * une édition locale reste en place jusqu'au prochain objet servi.
 *
 * La valeur précédente part de `undefined`, JAMAIS de l'objet servi : quand la requête est déjà
 * en cache au montage (la coquille lit les réglages avant toute page), partir de l'objet en cache
 * sautait la première copie et chaque contrôle montrait son défaut au lieu de la valeur
 * enregistrée. Hook canonique : garde-rail `useServerDraft.guard.test.ts` (aucune copie locale du
 * motif).
 */
export function useServerDraft<T extends object>(
  served: T | undefined,
): [Partial<T>, Dispatch<SetStateAction<Partial<T>>>] {
  const [draft, setDraft] = useState<Partial<T>>(() => served ?? {})
  const [previous, setPrevious] = useState<T | undefined>(undefined)
  if (served && served !== previous) {
    setPrevious(served)
    setDraft(served)
  }
  return [draft, setDraft]
}
