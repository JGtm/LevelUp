/**
 * HomeSkillPeakCard — carte de skill peak (CSR/LUSR) avec badge image.
 *
 * P8.4 (revue 2026-04-29) : extrait de HomePage.tsx (HomeSkillPeakCard +
 * resolveSkillPeakState). Réduit la god page de ~80L.
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

  return (
    <div
      data-testid={`${testIdPrefix}-card`}
      className={`flex h-full min-w-[11rem] items-center gap-3 rounded-2xl border px-4 py-3 shadow-[0_12px_30px_rgba(8,15,28,0.24)] backdrop-blur-sm ${
        hasValue ? 'border-border bg-card' : 'border-border bg-muted/30'
      }`}
    >
      {peak?.badge_image_url ? (
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-border bg-muted p-1.5"> {/* color-allow: thématique Spartan UI */}
          <img
            data-testid={`${testIdPrefix}-badge`}
            src={peak.badge_image_url}
            alt={label}
            className="h-full w-full object-contain"
            loading="lazy"
            decoding="async"
          />
        </div>
      ) : (peak || isPlacement) ? (
        // Bug #1 : quand un peak existe (rating mais pas de tier_code stocké
        // en DB) ou que le joueur est en placement, on rend le badge unranked
        // générique au lieu de l'abréviation textuelle "MMR/LUSR".
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-border bg-muted p-1.5"> {/* color-allow: thématique Spartan UI */}
          <img
            data-testid={`${testIdPrefix}-unranked`}
            src={unrankedBadgeURL()}
            alt={isPlacement ? 'En placement' : label}
            className="h-full w-full object-contain opacity-80"
            loading="lazy"
            decoding="async"
          />
        </div>
      ) : (
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-border bg-muted text-[10px] font-semibold uppercase tracking-[0.24em] text-muted-foreground">
          {label.replace(/[^A-Z]/gi, '').slice(0, 4) || 'MMR'}
        </div>
      )}

      <div className="min-w-0">
        <p className="text-[11px] uppercase tracking-[0.24em] text-muted-foreground">{label}</p>
        <p data-testid={`${testIdPrefix}-value`} className="mt-1 text-xl font-semibold text-foreground sm:text-2xl">
          {peak ? peak.rating_value.toLocaleString(numberLocale, { maximumFractionDigits: 0 }) : '—'}
        </p>
        {(peak?.tier_label || detail) && (
          <p
            data-testid={peak?.tier_label ? `${testIdPrefix}-tier` : `${testIdPrefix}-detail`}
            className="truncate text-xs text-muted-foreground"
          >
            {peak?.tier_label ?? detail}
          </p>
        )}
      </div>
    </div>
  )
}

export function resolveSkillPeakState(
  peak: HomeSkillPeakSummary | null,
  hasHistory: boolean,
  mode: 'ranked' | 'unranked',
): Pick<HomeSkillPeakCardProps, 'state' | 'detail'> {
  if (peak) {
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
