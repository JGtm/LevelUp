/**
 * effectiveHp — barème PV-pour-tuer du titre COURANT (title-aware).
 *
 * 225 Halo Infinite (90 vie + 135 bouclier) · 115 Halo 5 (bouclier 70 + armure 45).
 * Source unique côté front pour les surfaces combat (rendement / résistance) qui
 * doivent afficher le barème du titre sélectionné au lieu du 225 câblé. Le défaut
 * est `ONE_LIFE_DAMAGE` (Infinite), aligné sur le fallback backend
 * (games.DefaultEffectiveHpToKill).
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { ONE_LIFE_DAMAGE } from '@/lib/charts/oneLifeDamageGradient'

/** PV-pour-tuer du titre courant, défaut Infinite (ONE_LIFE_DAMAGE). */
export function useEffectiveHpToKill(): number {
  return useAppShellStore(
    (s) =>
      s.availableTitles.find((t) => t.slug === s.currentTitleSlug)?.effective_hp_to_kill ??
      ONE_LIFE_DAMAGE,
  )
}

/** Remplace le jeton `{{HP}}` d'un libellé i18n par le barème PV du titre. */
export function substituteHpToken(text: string, hp: number): string {
  return text.replace(/\{\{HP\}\}/g, String(hp))
}

/** Frontière élite OC (80e percentile) par défaut — Halo Infinite. Miroir de
 *  games.DefaultOffensiveConversionP80 / analysis.OffensiveConversionP80. */
const DEFAULT_OFFENSIVE_CONVERSION_P80 = 0.9

/** Frontière élite OC (P80) du titre courant — repère de normalisation des barres
 *  de rendement (0.90 Infinite, 1.264 Halo 5). Défaut Infinite. */
export function useOffensiveConversionP80(): number {
  return useAppShellStore(
    (s) =>
      s.availableTitles.find((t) => t.slug === s.currentTitleSlug)?.offensive_conversion_p80 ??
      DEFAULT_OFFENSIVE_CONVERSION_P80,
  )
}
