// cross-feature-allow: libellés de chaînes LUSR (`lusrChainLabel`) partagés avec l'onglet
// Carrière — une seule table de noms de chaînes dans l'app.
/**
 * labels — les libellés que la page Tendances résout : un type de partie et une ligne
 * d'indicateur.
 *
 * LES LIBELLÉS DE CHAMP NE SE RÉÉCRIVENT PAS. Une clé de champ canonique (`win_rate`,
 * `kda`, `enemy_mmr`...) se lit dans les mappings du titre, comme `useMetricLabel` : ce
 * module reprend ses deux primitives (`canonicalMetricKey`, `humanizeMetricKey`) au lieu de
 * tenir un dictionnaire (garde-rail `no-field-label-dictionary.test.ts`). Seules les clés
 * absentes du registre (`damage_balance`, `objective_*`...) passent par le manifeste.
 */
import { useCallback } from 'react'

import { lusrChainLabel } from '@/features/career/lusr-chains'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { canonicalMetricKey, humanizeMetricKey } from '@/lib/i18n/metricLabel'
import type { Locale } from '@/lib/i18n/locale'

import { getTendancesText } from './i18n'

/** Clés d'indicateur dont la variante est une chaîne LUSR, ou un groupe de file classée (CSR). */
const LUSR_VARIANT_KEY = 'lusr_value'
const CSR_VARIANT_KEY = 'csr_value'

/**
 * Libellé de la variante d'une ligne : la chaîne LUSR par son nom connu, le groupe de file
 * classée du CSR par le manifeste (la clé brute si le groupe est inconnu), sinon la variante
 * telle quelle (un gamertag).
 */
export function variantLabel(key: string, variant: string, locale: Locale): string {
  if (key === LUSR_VARIANT_KEY) return lusrChainLabel(variant, locale)
  if (key === CSR_VARIANT_KEY) return getTendancesText(locale).variantManifestLabel(variant) ?? variant
  return variant
}

/**
 * Libellé d'un type de partie (chaîne de performance) : le nom connu de la chaîne LUSR,
 * sinon le manifeste de la page, sinon la clé brute.
 */
export function gameTypeLabel(key: string, locale: Locale): string {
  const chaine = lusrChainLabel(key, locale)
  if (chaine !== key) return chaine
  return getTendancesText(locale).gameTypeManifestLabel(key) ?? key
}

/**
 * Renvoie la fonction qui écrit la ligne d'un indicateur. Hook : lit les mappings du titre
 * UNE fois pour toutes les lignes (un hook par ligne n'est pas possible dans une boucle).
 * Une ligne à variante s'écrit « libellé · variante » (cf. `variantLabel`).
 */
export function useIndicatorLabeler(locale: Locale): (key: string, variant?: string) => string {
  const { data } = useFieldMappings()
  return useCallback(
    (key, variant) => {
      const canonique = canonicalMetricKey(key)
      const base =
        getTendancesText(locale).indicatorLabel(key) ??
        data?.fields[canonique]?.label ??
        humanizeMetricKey(key)
      if (!variant) return base
      return `${base} · ${variantLabel(key, variant, locale)}`
    },
    [data, locale],
  )
}
