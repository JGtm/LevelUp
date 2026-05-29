/**
 * page-scope/serialize — helpers purs de (dé)sérialisation entre l'état
 * applicatif riche (Set<string>, chaînes) et la forme plate sérialisable en
 * query params d'URL.
 *
 * Convention csv : valeurs jointes par ','. Les vocabulaires concernés côté
 * Explorer (playlists, modes, maps, types d'expérience, tiers) sont contrôlés
 * et ne contiennent pas de virgule. Une valeur vide → `undefined` : le param
 * est alors omis de l'URL (et `JSON.stringify` l'omet du miroir localStorage),
 * ce qui garde l'URL propre.
 *
 * Module pur, sans dépendance React/Router : testable sans DOM.
 */

/** Set → "a,b,c". Set vide ou absent → undefined (param omis). */
export function setToCsv(set: ReadonlySet<string> | undefined | null): string | undefined {
  if (!set || set.size === 0) return undefined
  return [...set].join(',')
}

/** "a,b,c" → Set. Valeur non-chaîne ou vide → Set vide. */
export function csvToSet(value: unknown): Set<string> {
  if (typeof value !== 'string' || value.length === 0) return new Set()
  return new Set(value.split(',').filter(Boolean))
}

/** Chaîne non vide → elle-même ; sinon undefined (param omis). */
export function strOrUndef(value: string | undefined | null): string | undefined {
  return value ? value : undefined
}
