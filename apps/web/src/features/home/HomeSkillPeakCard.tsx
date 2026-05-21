/**
 * HomeSkillPeakCard — carte de skill peak (CSR/LUSR) avec badge image.
 *
 * Évolution mai 2026 : la source de vérité de l'état placement est désormais
 * `peak.measurement_matches_remaining` (10 → 0). Le backend renvoie un peak
 * même en placement (avec rating_value=0 et badge_image_url pointant sur
 * unranked_(10-N).png). Le front ne devine plus via `hasHistory` + `mode`.
 */
import type { HomeSkillPeakSummary } from '@/lib/api/types'
import { unrankedBadgeURL } from '@/lib/staticAssets'

export interface HomeSkillPeakCardProps {
  label: string
  peak: HomeSkillPeakSummary | null
  numberLocale: string
  testIdPrefix: string
  state: 'value' | 'placement' | 'neutral' | 'absent'
  detail: string
}

export function HomeSkillPeakCard({
  label,
  peak,
  numberLocale,
  testIdPrefix,
  state,
  detail,
}: HomeSkillPeakCardProps) {
  const isPlacement = state === 'placement'
  const hasValue = Boolean(peak)
  const showRatingValue = state === 'value' && peak !== null && peak.rating_value > 0

  return (
    <div
      data-testid={`${testIdPrefix}-card`}
      className={`flex h-full min-w-[11rem] items-center gap-3 rounded-2xl border px-4 py-3 shadow-[0_12px_30px_rgba(8,15,28,0.24)] backdrop-blur-sm ${
        hasValue ? 'border-border bg-card' : 'border-border bg-muted/30'
      }`}
    >
      {peak?.badge_image_url ? (
        <img
          data-testid={isPlacement ? `${testIdPrefix}-unranked` : `${testIdPrefix}-badge`}
          src={peak.badge_image_url}
          alt={isPlacement ? 'En placement' : label}
          className={`h-12 w-12 shrink-0 object-contain ${isPlacement ? 'opacity-80' : ''}`}
          loading="lazy"
          decoding="async"
        />
      ) : isPlacement ? (
        // Legacy path : backend antérieur à mai 2026 ne fournissait pas le
        // badge unranked dans le peak. Fallback sur l'image statique générique.
        <img
          data-testid={`${testIdPrefix}-unranked`}
          src={unrankedBadgeURL()}
          alt="En placement"
          className="h-12 w-12 shrink-0 object-contain opacity-80"
          loading="lazy"
          decoding="async"
        />
      ) : (
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-border bg-muted text-2xs font-semibold uppercase tracking-label-xl text-muted-foreground">
          {label.replace(/[^A-Z]/gi, '').slice(0, 4) || 'MMR'}
        </div>
      )}

      <div className="min-w-0">
        <p className="text-3xs uppercase tracking-label-xl text-muted-foreground">{label}</p>
        <p data-testid={`${testIdPrefix}-value`} className="mt-1 text-xl font-semibold text-foreground sm:text-2xl">
          {showRatingValue ? peak!.rating_value.toLocaleString(numberLocale, { maximumFractionDigits: 0 }) : '—'}
        </p>
        {(peak?.tier_label || detail) && (
          <p
            data-testid={peak?.tier_label && !isPlacement ? `${testIdPrefix}-tier` : `${testIdPrefix}-detail`}
            className="truncate text-xs text-muted-foreground"
          >
            {isPlacement ? detail : (peak?.tier_label ?? detail)}
          </p>
        )}
      </div>
    </div>
  )
}

/**
 * resolveSkillPeakState — lit l'état d'un skill peak depuis le summary backend.
 *
 * Source de vérité : `peak.measurement_matches_remaining` + `peak.placement_total`.
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
): Pick<HomeSkillPeakCardProps, 'state' | 'detail'> {
  if (peak) {
    const remaining = peak.measurement_matches_remaining ?? 0
    if (remaining > 0) {
      const total = peak.placement_total ?? 10
      const completed = Math.max(0, Math.min(total - 1, total - remaining))
      return { state: 'placement', detail: `En placement (${completed}/${total})` }
    }
    return { state: 'value', detail: '' }
  }
  if (hasHistory) {
    return mode === 'ranked'
      ? { state: 'placement', detail: 'En placement' }
      : { state: 'neutral', detail: 'Sans classement' }
  }
  return {
    state: 'absent',
    detail: mode === 'ranked' ? 'Aucune partie classée' : 'Aucune partie non classée',
  }
}
