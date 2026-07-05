/**
 * SynthesisRoleKillsDonut — « Frags par type d'arme » (rôle de combat).
 *
 * PREMIER consommateur UI du registre d'armes (weaponregistry) : la donnée
 * kills_by_role est agrégée côté Go (frags par rôle de combat de l'arme),
 * title-agnostic (marche pour Halo Infinite ET Halo 5). Réutilise le donut SVG
 * partagé (KillTypesDonut), tokens color-blind friendly. Rend null si rien.
 */
import { KillTypesDonut, type DonutSlice } from '@/components/charts/KillTypesDonut'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import type { SynthesisRoleKillEntry } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import { weaponRoleInsight } from './weaponRoleInsight'

// Tokens DISTINCTS color-blind friendly par rôle (mapping stable → couleurs
// constantes d'un rendu à l'autre).
const ROLE_TOKENS: Record<string, SemanticToken> = {
  power: 'chart-series-1',
  precision: 'chart-series-2',
  automatic: 'chart-series-3',
  sniper: 'chart-series-4',
  sidearm: 'chart-series-5',
  shotgun: 'chart-series-6',
  special: 'chart-series-7',
  melee: 'chart-series-8',
  grenade: 'chart-series-8',
}

type ManifestKey = keyof typeof synthesisManifest

export function SynthesisRoleKillsDonut({ roles }: { roles?: SynthesisRoleKillEntry[] }) {
  const appLocale = useAppShellStore((s) => s.locale)
  const locale = intlLocale(appLocale)
  const title = formatMessage(synthesisManifest, 'synthesis.charts.role_kills_title', appLocale)

  // role_<key> existe dans le manifest pour les 9 rôles canoniques ; cast sûr
  // (formatMessage dégrade en affichant la clé si jamais absente).
  const slices: DonutSlice[] = (roles ?? [])
    .map((r) => ({
      label: formatMessage(synthesisManifest, `synthesis.charts.role_${r.role}` as ManifestKey, appLocale),
      count: r.kills,
      token: (ROLE_TOKENS[r.role] ?? 'chart-series-8') as SemanticToken,
    }))
    .filter((s) => s.count > 0)

  if (slices.length === 0) return null

  // Insight coach data-driven (angle mort armes lourdes / sur-dépendance).
  const insight = weaponRoleInsight(roles)
  const insightText = !insight
    ? null
    : insight.kind === 'blind_spot_power'
      ? formatMessage(synthesisManifest, 'synthesis.coach.blind_spot_power', appLocale)
      : formatMessage(synthesisManifest, 'synthesis.coach.over_reliance', appLocale, {
          pct: insight.pct,
          role: formatMessage(synthesisManifest, `synthesis.charts.role_${insight.role}` as ManifestKey, appLocale),
        })

  return (
    <div className="flex flex-col overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">{title}</div>
      <div className="flex w-full flex-col items-center gap-2 p-3">
        <KillTypesDonut slices={slices} locale={locale} />
        {insightText && (
          <p className="w-full text-xs leading-snug text-muted-foreground">
            <span className="font-semibold text-foreground">
              {formatMessage(synthesisManifest, 'synthesis.coach.label', appLocale)}
              {' : '}
            </span>
            {insightText}
          </p>
        )}
      </div>
    </div>
  )
}
