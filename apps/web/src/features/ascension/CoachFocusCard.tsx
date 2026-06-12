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
 */
import { KpiCard } from '@/components/cards/KpiCard'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ProfileManifestKey } from '@/lib/i18n/generated/profile'
import { usePlayerProfile } from './profile/queries'
import { useProfileI18n } from './profile/useProfileI18n'

// FOCUS_THRESHOLD : amplitude mini de tendance pour émettre un cap (anti-bruit ;
// aligné sur TREND_THRESHOLD de PerformanceSection).
const FOCUS_THRESHOLD = 0.02

const STR = {
  fr: {
    title: 'Cap du moment',
    progression: 'en progression',
    consolidate: 'à consolider',
    up: (axis: string) => `Tu consolides ${axis}, continue sur ta lancée.`,
    down: (axis: string) => `${axis} mérite ton attention en ce moment.`,
    cta: 'Voir le défi →',
  },
  en: {
    title: 'Current focus',
    progression: 'improving',
    consolidate: 'to consolidate',
    up: (axis: string) => `You're consolidating ${axis} — keep it up.`,
    down: (axis: string) => `${axis} deserves your attention right now.`,
    cta: 'See the challenge →',
  },
}

export function CoachFocusCard({ playerSlug }: { playerSlug: string }) {
  const locale = useAppShellStore((s) => s.locale)
  const str = STR[locale === 'en' ? 'en' : 'fr']
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
        <p className="text-2xs uppercase tracking-wide text-muted-foreground">{str.title}</p>
        <p className="text-sm font-semibold text-foreground">
          {axis} · {positive ? str.progression : str.consolidate}
        </p>
        <p className="mt-0.5 text-xs text-muted-foreground">
          {positive ? str.up(axis) : str.down(axis)}
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
          {str.cta}
        </button>
      </div>
    </KpiCard>
  )
}
