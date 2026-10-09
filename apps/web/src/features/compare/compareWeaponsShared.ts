/**
 * compareWeaponsShared — LES TROIS OUTILS QUE LES DEUX FICHIERS DU PROFIL D'ARMES PARTAGENT.
 *
 * `CompareWeaponsSection` (classes + armes) et `CompareWeaponsRange` (portée) ont besoin des
 * mêmes encres, des mêmes formateurs et du même résolveur de libellé de rôle. Les recopier
 * donnerait deux palettes, deux façons d'écrire « 12,4 m » et deux résolutions de libellé qui
 * divergeraient au premier ajout de rôle — exactement ce que la règle des copies interdit.
 *
 * Fichier `.ts` et non `.tsx` : aucun JSX ici, seulement des constantes et deux hooks.
 */
import { useCallback, useMemo } from 'react'

import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { intlLocale } from '@/lib/formatters'
import { formatMessage } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import type { Locale } from '@/lib/i18n/locale'

import { roleLabel } from '@/components/charts/weaponRangeRoles'

/**
 * Les trois encres des joueurs — LES MÊMES que le reste de la page (barres, en-têtes).
 * Le graphe doit se lire comme la colonne d'à côté ; une quatrième palette ferait chercher
 * qui est qui à chaque bloc.
 */
export const TOKEN_A: SemanticToken = 'compare-a'
export const TOKEN_B: SemanticToken = 'compare-b'
export const TOKEN_C: SemanticToken = 'compare-c'

/** Le losange de médiane — la même encre que la Synthèse, pour la même forme. */
export const TOKEN_MEDIAN: SemanticToken = 'perf-tier-2'

/** Les deux joueurs d'UNE comparaison, nommés — `top` et `bottom` du graphe. */
export interface SideNames {
  a: string
  b: string
}

/** Formateurs locale-aware, partagés par les trois blocs. */
export function useWeaponFormats(locale: Locale) {
  return useMemo(() => {
    const loc = intlLocale(locale)
    const oneDecimal = new Intl.NumberFormat(loc, {
      minimumFractionDigits: 1,
      maximumFractionDigits: 1,
    })
    const count = new Intl.NumberFormat(loc)
    // `style: 'percent'` place l'unité selon la locale (espace insécable en FR, collé en EN) :
    // elle n'est jamais écrite à la main.
    const pct = new Intl.NumberFormat(loc, { style: 'percent', maximumFractionDigits: 0 })
    return {
      distance: (m: number) => `${oneDecimal.format(m)} m`,
      count: (n: number) => count.format(n),
      percent: (v: number) => pct.format(v / 100),
    }
  }, [locale])
}

/**
 * useRoleName — le libellé d'une clé de rôle, résolu par le manifeste des frags.
 *
 * JAMAIS UNE CLÉ BRUTE (D8) : `roleLabel` essaie `frags.role.<clé>`, puis `frags.class.<clé>`
 * (les clés qui sont leur propre rôle), puis rend la clé NUE — jamais son chemin de manifeste.
 */
export function useRoleName(locale: Locale) {
  return useCallback(
    (key: string) =>
      roleLabel(key, (manifestKey) => formatMessage(fragsManifest, manifestKey as never, locale)),
    [locale],
  )
}
