/**
 * CoachFocusCard — « Cap du moment » (Phase A : coach soft-négatif / positif).
 *
 * Un seul headline adaptatif en tête de l'onglet Entraînement, qui consolide la
 * lecture de tendance (le TrendBadge de PerformanceSection est neutralisé en
 * parallèle). Dérive l'axe focal de la composante LUSR qui bouge le plus, et
 * bascule l'accent :
 *   - progression → accent `success` ;
 *   - soft-négatif → accent `info` (NEUTRE, jamais rouge/alerte) ;
 *   - tendance non significative → carte NON rendue (pas de bruit).
 *
 * Cadrage produit (universel, non culpabilisant) : « X mérite ton attention » /
 * « tu consolides X » — jamais « tu régresses ». Cf. PLAN_COACH_V3_GENERATION
 * Phase A (option B « Cap du moment »).
 *
 * i18n : manifest `profile.coach.*` (`lib/i18n/manifests/profile.toml`, ADR 0003)
 * — dernier composant coach migré depuis un objet FR/EN local (V721-07.3).
 */
import { KpiCard } from '@/components/cards/KpiCard'
import type { ProfileManifestKey } from '@/lib/i18n/generated/profile'
import { usePlayerProfile } from './profile/queries'
import { useProfileI18n } from './profile/useProfileI18n'

// FOCUS_THRESHOLD : amplitude mini de tendance pour émettre un cap (anti-bruit ;
// aligné sur TREND_THRESHOLD de PerformanceSection).
const FOCUS_THRESHOLD = 0.02

export function CoachFocusCard({ playerSlug }: { playerSlug: string }) {
  const { t } = useProfileI18n()
  const { data: profile } = usePlayerProfile(playerSlug)

  const components = profile?.lusr_components ?? []
  // Axe focal = la composante qui bouge le plus (au-dessus du seuil anti-bruit).
  let focal: (typeof components)[number] | undefined
  for (const c of components) {
    if (Math.abs(c.trend) < FOCUS_THRESHOLD) continue
    if (!focal || Math.abs(c.trend) > Math.abs(focal.trend)) focal = c
  }
  if (!focal) return null

  const positive = focal.trend > 0
  const axis = t(`profile.lusr.${focal.name}` as ProfileManifestKey)

  return (
    <KpiCard accent={positive ? 'success' : 'info'}>
      <div className="p-3">
        <p className="text-2xs uppercase tracking-wide text-muted-foreground">
          {t('profile.coach.title')}
        </p>
        <p className="text-sm font-semibold text-foreground">
          {axis} · {positive ? t('profile.coach.progression') : t('profile.coach.consolidate')}
        </p>
        <p className="mt-0.5 text-xs text-muted-foreground">
          {positive ? t('profile.coach.up', { axis }) : t('profile.coach.down', { axis })}
        </p>
        <button
          type="button"
          className="mt-1.5 text-xs font-medium text-foreground underline-offset-2 hover:underline"
          onClick={() =>
            document
              .getElementById('coach-proposals')
              ?.scrollIntoView({ behavior: 'smooth', block: 'start' })
          }
        >
          {t('profile.coach.cta')}
        </button>
      </div>
    </KpiCard>
  )
}
