/**
 * MatchToolsCard — « Outils de destruction » du joueur de la page sur la Vue match (plan
 * PLAN_MATCHVIEW_EMPRISE_2026-10-06, carte B, D9) : une barre par outil (arme, grenade, mêlée, objet
 * explosif, chute), la pastille de la classe devant l'outil, le compte au bout, à l'encre du joueur.
 *
 * Le graphe de l'Escouade et de Sessions (`SquadWeaponKillsChart` + `buildSquadToolRows`) sur le bloc
 * `combat_tab.weapon_tools` du match ; libellés des natures par la source unique `toolKindLabels`.
 */
import { useMemo } from 'react'

import { InfoTooltip } from '@/components/ui/info-tooltip'
import { buildSquadToolRows, toolKindLabels } from '@/features/squad/charts/squadFragTools'
import { getSquadPlayerColors } from '@/features/squad/colors'
import { SquadWeaponKillsChart } from '@/features/squad/SquadWeaponKillsChart'
import type { SquadWeaponTools } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { MATCH_EMPRISE_TEXT } from './matchEmpriseText'

export function MatchToolsCard({ tools, locale }: { tools: SquadWeaponTools | null | undefined; locale: Locale }) {
  const t = MATCH_EMPRISE_TEXT[locale].squad
  const rows = useMemo(() => buildSquadToolRows(tools, { locale, labels: toolKindLabels(locale, t.weaponKills) }), [tools, locale, t.weaponKills])
  const player = tools?.players?.[0] ?? ''
  const colors = useMemo(() => getSquadPlayerColors(player, []), [player])
  return (
    <SquadWeaponKillsChart
      title={
        <span className="flex items-center gap-1.5">
          {t.weaponKills.title}
          <InfoTooltip content={t.weaponKills.info} />
        </span>
      }
      emptyMessage={t.empty.noBlockData}
      data={rows}
      colorByPlayer={colors}
      valueText={t.weaponKills.killsShare}
      valueLabel="count"
    />
  )
}
