import type { HomeSkillPeakSummary } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import type { HomeSkillPeakCardProps } from './HomeSkillPeakCard'

/**
 * resolveSkillPeakState — lit l'état d'un skill peak depuis le summary backend.
 *
 * Extrait de HomeSkillPeakCard.tsx pour que le module de composant n'exporte que
 * des composants (react-refresh/only-export-components).
 *
 * Phase 6 du plan pipeline CSR : depuis Season 3 (mars 2023) Halo utilise un
 * seuil 5 au lieu de 10. Le backend expose `placement_total` (5 ou 10) ; on
 * fallback à 10 pour les payloads legacy.
 *
 * Le `hasHistory` reste consulté en dégradation pour les responses backend
 * antérieures à mai 2026 (qui retournaient `peak=null` pour les joueurs en
 * placement). À supprimer une fois tous les clients à jour.
 */
export function resolveSkillPeakState(
  peak: HomeSkillPeakSummary | null,
  hasHistory: boolean,
  mode: 'ranked' | 'unranked',
  locale: Locale,
): Pick<HomeSkillPeakCardProps, 'state' | 'detail'> {
  const t = (key: keyof typeof commonManifest & string, vars?: Record<string, unknown>) =>
    formatMessage(commonManifest, key, locale, vars)
  if (peak) {
    const remaining = peak.measurement_matches_remaining ?? 0
    if (remaining > 0) {
      const total = peak.placement_total ?? 10
      const completed = Math.max(0, Math.min(total - 1, total - remaining))
      if (completed === 0) return { state: 'placement', detail: t('common.home.rank_unrated') }
      return {
        state: 'placement',
        detail: t('common.home.rank_placement_progress', { completed, total }),
      }
    }
    return { state: 'value', detail: '' }
  }
  if (hasHistory) {
    return mode === 'ranked'
      ? { state: 'placement', detail: t('common.home.rank_placement') }
      : { state: 'neutral', detail: t('common.home.unranked') }
  }
  return { state: 'absent', detail: t('common.home.rank_unrated') }
}
