/**
 * weaponRangeText — LES OUTILS DE TEXTE ET DE NOMBRE de la section « Portée des engagements ».
 *
 * Le traducteur typé sur le manifeste de la Synthèse, et les formateurs locale-aware que la
 * section, ses graphes et son tableau partagent : une distance écrite « 7,4 m » en français
 * et « 7.4 m » en anglais ne doit pas dépendre de qui l'écrit.
 *
 * Fichier à part parce que le tableau (`SynthesisWeaponRangeTable.tsx`) en dépend autant que
 * la section, et qu'un import croisé entre les deux composants aurait fermé un cycle.
 */
import { useMemo } from 'react'

import { intlLocale } from '@/lib/formatters'
import type { ManifestLocale } from '@/lib/i18n/format'
import type { synthesisManifest } from '@/lib/i18n/generated/synthesis'

export type SynthesisKey = keyof typeof synthesisManifest
export type Translate = (key: SynthesisKey, vars?: Record<string, unknown>) => string

/** Formateurs locale-aware de la section (distances, parts, effectifs). */
export interface RangeFormats {
  distance: (m: number) => string
  signedDistance: (m: number) => string
  percent: (v: number) => string
  count: (n: number) => string
}

export function useRangeFormats(locale: ManifestLocale): RangeFormats {
  return useMemo(() => {
    const numLoc = intlLocale(locale)
    const oneDecimal = new Intl.NumberFormat(numLoc, {
      minimumFractionDigits: 1,
      maximumFractionDigits: 1,
    })
    const signed = new Intl.NumberFormat(numLoc, {
      minimumFractionDigits: 1,
      maximumFractionDigits: 1,
      signDisplay: 'exceptZero',
    })
    // `style: 'percent'` place l'unité selon la locale (« 49 % » en FR, « 49% » en EN) :
    // l'espace insécable du français n'est pas à écrire à la main.
    const pct = new Intl.NumberFormat(numLoc, { style: 'percent', maximumFractionDigits: 0 })
    const count = new Intl.NumberFormat(numLoc)
    return {
      distance: (m) => `${oneDecimal.format(m)} m`,
      signedDistance: (m) => `${signed.format(m)} m`,
      percent: (v) => pct.format(v / 100),
      count: (n) => count.format(n),
    }
  }, [locale])
}
