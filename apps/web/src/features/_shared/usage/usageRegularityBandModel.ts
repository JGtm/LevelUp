/**
 * usageRegularityBandModel.ts — LA FORME « bande de régularité » du bloc « usages
 * d'équipement, armes spéciales et objectifs » : une case par match mesuré, teintée par
 * l'écart à la parité de session.
 *
 * Extrait de `session-detail/usageLogic.ts` le 2026-09-09 (étape E5.1bis, scission de taille —
 * CLAUDE.md n°5) au moment du déménagement du bloc vers `features/_shared/usage/` (étape E5.1).
 *
 * Pur : aucun React, aucune couleur en dur.
 */
import type { SessionUsageMatchPoint } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { formatUsagePct } from './usageFormat'
import type { UsageText } from './usageI18n'

export type UsageBandTone = 'above' | 'near' | 'below' | 'unmeasured'

export interface UsageBandCell {
  matchId: string
  tone: UsageBandTone
  tooltip: string
}

/** Sous ±ε points de la parité, une case est « à la parité » (ni bonus ni déficit). */
const BAND_EPSILON_PT = 1

/**
 * buildRegularityBand — une case par match MESURÉ, teintée par l'écart de la part
 * d'équipe du joueur à la PARITÉ DE SESSION (le contrat ne publie pas la parité de
 * chaque match ; les comptes « au-dessus de la parité », eux, sont calculés côté Go
 * contre la parité de CHAQUE match — la légende du bloc les écrit tels quels).
 * Part absente sur un match (camp inconnu, dénominateur nul) → case « non mesurée ».
 */
export function buildRegularityBand(
  perMatch: SessionUsageMatchPoint[] | null | undefined,
  teamParityPct: number | null | undefined,
  t: UsageText,
  locale: Locale,
): UsageBandCell[] {
  return (perMatch ?? []).map((p, i) => {
    const share = p.player_share_of_team_pct
    if (share == null || teamParityPct == null) {
      return { matchId: p.match_id, tone: 'unmeasured', tooltip: t.bandTipUnmeasured(i + 1) }
    }
    const delta = share - teamParityPct
    const tone: UsageBandTone =
      delta > BAND_EPSILON_PT ? 'above' : delta < -BAND_EPSILON_PT ? 'below' : 'near'
    return {
      matchId: p.match_id,
      tone,
      tooltip: t.bandTipFmt(
        i + 1,
        formatUsagePct(share, locale),
        formatUsagePct(teamParityPct, locale),
      ),
    }
  })
}
