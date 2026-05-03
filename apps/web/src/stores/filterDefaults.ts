/**
 * filterDefaults.ts — constantes par défaut des filtres globaux.
 *
 * Fichier neutre sans imports de modules applicatifs : sert de source unique à
 * `globalFilterStore.ts` ET aux composants UI (`_filter_pills/_hooks.ts`,
 * `periodSessionNav.ts`) qui doivent connaître la valeur sans dépendre du store.
 *
 * Avant l'extraction : globalFilterStore -> periodSessionNav -> _hooks ->
 * globalFilterStore créait un cycle d'imports qui levait
 * "can't access lexical declaration 'DEFAULT_GAP_MINUTES' before initialization"
 * au boot.
 */

export const DEFAULT_GAP_MINUTES = 120
