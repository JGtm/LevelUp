/**
 * SynthesisFragCard — carte « Répartition des frags » v2 sur Synthesis : compose le
 * sunburst hiérarchique classe→rôle (FragSunburst) et, en dessous, le breakdown par
 * arme recoloré par classe (FragWeaponBreakdown). Remplace l'ancien donut « Frags par
 * type d'arme » (SynthesisRoleKillsDonut) — cf. .ai/V7/PLAN_FRAG_DISTRIBUTION_V2.md P2.1/P2.2.
 *
 * L'insight coach (angle mort armes lourdes / sur-dépendance) est PRÉSERVÉ : il reste
 * data-driven via weaponRoleInsight, qui dérive désormais les rôles des gun classes de
 * la FragDistribution (le DTO kills_by_role a été retiré en P7). Le sunburst et le
 * breakdown vivent dans components/charts (partagés) ; cette carte porte uniquement la
 * composition + le coach spécifiques à Synthesis.
 */
import { FragSunburst } from '@/components/charts/FragSunburst'
import { FragWeaponBreakdown } from '@/components/charts/FragWeaponBreakdown'
import type { FragDistribution, SynthesisWeaponKillEntry } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'
import { useAppShellStore } from '@/stores/appShellStore'
import { weaponRoleInsight } from './weaponRoleInsight'

type ManifestKey = keyof typeof synthesisManifest

interface Props {
  distribution?: FragDistribution | null
  weapons?: SynthesisWeaponKillEntry[]
}

export function SynthesisFragCard({ distribution, weapons }: Props) {
  const appLocale = useAppShellStore((s) => s.locale)

  // Insight coach data-driven (angle mort armes lourdes / sur-dépendance) : dérivé des
  // gun classes de la FragDistribution (arsenal réel ; weaponRoleInsight exclut déjà
  // les rôles hors-combat).
  const insight = weaponRoleInsight(distribution)
  const insightText = !insight
    ? null
    : insight.kind === 'blind_spot_power'
      ? formatMessage(synthesisManifest, 'synthesis.coach.blind_spot_power', appLocale)
      : formatMessage(synthesisManifest, 'synthesis.coach.over_reliance', appLocale, {
          pct: insight.pct,
          role: formatMessage(synthesisManifest, `synthesis.charts.role_${insight.role}` as ManifestKey, appLocale),
        })

  return (
    <div className="flex flex-col gap-3">
      <FragSunburst distribution={distribution} />
      {insightText && (
        <p className="w-full text-xs leading-snug text-muted-foreground">
          <span className="font-semibold text-foreground">
            {formatMessage(synthesisManifest, 'synthesis.coach.label', appLocale)}
            {' : '}
          </span>
          {insightText}
        </p>
      )}
      <FragWeaponBreakdown weapons={weapons} />
    </div>
  )
}
