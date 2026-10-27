/**
 * empriseStrings.ts — les textes de l'onglet « Emprise » de l'Escouade (lot L5 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) : titres (§3 du plan), aides ⓘ (trois phrases au
 * plus) et libellés de la maquette `.ai/archive/V7.5/MAQUETTE_ONGLET_TACTIQUE_ESCOUADE_2026-09-26.html`.
 * Fichier à part du dictionnaire de la feature (précédent : `objectif/objectifStrings.ts`).
 *
 * Le type est déclaré une fois (`i18n/contract.ts`) ; chaque langue vit dans son fichier
 * (`i18n/fr.ts`, `i18n/en.ts`) et le satisfait : une langue à laquelle il manque un champ ne
 * compile pas. Ce module y ajoute les mots des véhicules (`vehicleStrings.ts`).
 */
import type { Locale } from '@/lib/i18n/locale'
import type { BaseEmpriseText, EmpriseText } from './i18n/contract'
import { duration } from './i18n/format'
import { EN_TEXT } from './i18n/en'
import { FR_TEXT } from './i18n/fr'
import { buildVehicleText, type VehicleText } from './vehicleStrings'

export type { EmpriseText, ExposureText, ResourceText } from './i18n/contract'

const VT = buildVehicleText(duration)
const withVehicles = (b: BaseEmpriseText, v: VehicleText): EmpriseText => ({
  ...b, vehicles: v,
  resources: { ...b.resources, vehicle: v.resource },
  production: { ...b.production, exposure: { ...b.production.exposure, aboard_ms: v.aboard } },
})
export const EMPRISE_TEXT: Record<Locale, EmpriseText> = { fr: withVehicles(FR_TEXT, VT.fr), en: withVehicles(EN_TEXT, VT.en) }
