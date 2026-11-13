/**
 * usePersistedFlag — UN interrupteur persisté du rejeu : son état, et la bascule qui l'écrit.
 * Extrait de `useReplaySettings.ts` pour que les réglages de calques puissent le partager sans
 * cycle d'import.
 */
import { useCallback, useEffect, useState } from 'react'

import { persistPreference, readStoredFlag, subscribePreference } from './replayPreferences'

/**
 * usePersistedFlag — UN interrupteur persisté : son état, et la bascule qui l'écrit.
 *
 * CENTRALISÉ ICI (CLAUDE.md n°6, « à la 3e copie, centraliser ») : les calques de visée, de
 * zones, de noms, de carte de chaleur, d'effets de tirs et d'effets de mort partagent EXACTEMENT
 * le même corps — état initial lu du stockage, bascule qui persiste la nouvelle valeur. Six
 * copies de six lignes se seraient mises à diverger (l'une oubliant la persistance, l'autre
 * la forme fonctionnelle du setState).
 *
 * L'ÉCRITURE NE SE FAIT JAMAIS DEPUIS L'UPDATER de `setValue`, et c'est un invariant, pas un
 * style. `persistPreference` prévient SYNCHRONEMENT tous les abonnés de la clé, qui appellent
 * `setValue` à leur tour : le faire depuis l'updater, c'est déclencher une mise à jour d'état
 * en pleine phase de rendu, ré-entrante sur ce hook — et sur l'AUTRE instance de la même clé
 * quand elle vit chez un ancêtre (défaut vécu sur la clé des fiches compactes, aujourd'hui
 * supprimée : le tiroir et la page lisaient la même clé). React y répondait en gardant
 * l'ancienne valeur une fois sur deux : la bascule « revenait en arrière » toute seule, et
 * StrictMode, qui invoque les updaters deux fois pour débusquer exactement cette impureté,
 * doublait le symptôme.
 *
 * Donc : `next` se calcule ICI, hors updater, à partir de la valeur rendue ; `setValue` reçoit
 * une valeur nue ; la persistance vient APRÈS, depuis le gestionnaire d'événement. `value` est
 * de ce fait une dépendance de la bascule — c'est le prix, et il est juste : une bascule qui ne
 * connaît pas la valeur qu'elle inverse n'existe pas.
 *
 * EXPORTÉ depuis le 2026-08-28 : le repli de la frise est une préférence persistée comme les
 * autres, mais elle ne passe pas par le tiroir (`useReplayTimeline` la lit directement). Faire
 * une seconde copie du corps pour cette seule raison aurait rouvert exactement la divergence
 * que cette centralisation a fermée.
 */
export function usePersistedFlag(key: string, fallback: boolean): [boolean, () => void] {
  const [value, setValue] = useState(() => readStoredFlag(key, fallback))
  // L'ABONNEMENT REND LA CLÉ PARTAGEABLE : deux composants qui lisent la même préférence
  // bougent ensemble (cf. la note de `subscribePreference`). Sans lui, la bascule du tiroir
  // ne toucherait que sa propre copie de l'état.
  useEffect(() => subscribePreference(key, (raw) => setValue(raw === 'true')), [key])
  const toggle = useCallback(() => {
    const next = !value
    setValue(next)
    persistPreference(key, String(next))
  }, [key, value])
  return [value, toggle]
}
