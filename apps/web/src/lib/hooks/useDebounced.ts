import { useEffect, useState } from 'react'

/**
 * useDebounced — retourne `value` après `delay` ms de stabilité.
 *
 * Chaque changement de `value` réarme le timer ; seule la dernière valeur d'une
 * rafale est propagée. Sert à débouncer une saisie avant de piloter une query
 * réseau (1 requête après la rafale, pas une par frappe).
 *
 * Hook canonique partagé (feedback-drawer, Explorer, …) — ne pas ré-implémenter
 * en local (règle CLAUDE.md n°6).
 */
export function useDebounced<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay)
    return () => clearTimeout(id)
  }, [value, delay])
  return debounced
}
