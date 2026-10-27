/**
 * useSynthesisFragCharts — état + données partagés des graphes frags de Synthesis, remontés
 * ici parce que le sunburst (rangée 1) et le breakdown (rangée 2) vivent sur DEUX rangées
 * distinctes mais partagent le survol LIÉ (hoveredClass) + la même « Détails des frags ».
 *
 * Construit « Détails des frags » = armes (guns, per-arme) + détail mêlée/grenade/capacités
 * depuis la distribution (autoritatif, sans recouvrement — même logique que le match view,
 * 2e et dernière copie autorisée). Fournit aussi le texte de l'insight coach (Synthesis).
 */
import { useState } from 'react'

import { buildFragDetailBreakdown } from '@/components/charts/fragDetailBreakdown'
import type { FragDistribution, SynthesisWeaponKillEntry } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'
import { useAppShellStore } from '@/stores/appShellStore'
import { weaponRoleInsight } from './weaponRoleInsight'

type ManifestKey = keyof typeof synthesisManifest

export interface SynthesisFragCharts {
  hovered: string | null
  setHovered: (c: string | null) => void
  breakdown: SynthesisWeaponKillEntry[]
  detailTitle: string
  insightText: string | null
}

export function useSynthesisFragCharts(
  distribution?: FragDistribution | null,
  weapons?: SynthesisWeaponKillEntry[],
): SynthesisFragCharts {
  const appLocale = useAppShellStore((s) => s.locale)
  const [hovered, setHovered] = useState<string | null>(null)

  const classLabel = (c: string) => formatMessage(fragsManifest, `frags.class.${c}` as never, appLocale)
  const roleLabel = (r: string) => formatMessage(fragsManifest, `frags.role.${r}` as never, appLocale)
  const detailTitle = formatMessage(fragsManifest, 'frags.charts.detail_title', appLocale)

  const breakdown = buildFragDetailBreakdown(distribution, weapons ?? [], { roleLabel, classLabel })

  const insight = weaponRoleInsight(distribution)
  const insightText = !insight
    ? null
    : insight.kind === 'blind_spot_power'
      ? formatMessage(synthesisManifest, 'synthesis.coach.blind_spot_power', appLocale)
      : formatMessage(synthesisManifest, 'synthesis.coach.over_reliance', appLocale, {
          pct: insight.pct,
          role: formatMessage(synthesisManifest, `synthesis.charts.role_${insight.role}` as ManifestKey, appLocale),
        })

  return { hovered, setHovered, breakdown, detailTitle, insightText }
}
