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
