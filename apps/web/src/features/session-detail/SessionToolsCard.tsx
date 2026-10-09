/**
 * SessionToolsCard — carte B de « Frags et usages » (plan PLAN_SESSIONS_EMPRISE_2026-10-06, §3) :
 * « Outils de destruction » du joueur de la page, une barre par outil, à la couleur de la CLASSE de
 * l'outil (celle de la Répartition des frags, comme avant la reconstruction de la page), la pastille de
 * la classe devant l'outil ; aucun gamertag dans l'infobulle : la page ne montre qu'un joueur.
 *
 * Le graphe de l'Escouade (`SquadWeaponKillsChart` + `buildSquadToolRows`) sur le bloc
 * `weapon_tools` de la session (D6). Pleine page : tous les outils, le compte au bout. Vue compacte
 * (maquette `makeTools`, `cp`) : les six premiers, « Non attribué » exclu, la part de TOUS les frags du joueur
 * au bout (dénominateur : la somme des lignes du serveur, reliquat compris — la feuille de match).
 */
import { useMemo } from 'react'

import { InfoTooltip } from '@/components/ui/info-tooltip'
import { buildSquadToolRows, toolKindLabels } from '@/features/squad/charts/squadFragTools'
import { getSquadPlayerColors } from '@/features/squad/colors'
import { SquadWeaponKillsChart } from '@/features/squad/SquadWeaponKillsChart'
import type { SquadWeaponTools } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { SESSION_TOOLS_TOP } from './sessionEmprise.logic'
import type { SessionCardTexts } from './sessionEmpriseText'

interface Props {
  tools: SquadWeaponTools | null | undefined
  /** Le nom du joueur de la page, tel que les lignes d'outils le portent. */
  player: string
  locale: Locale
  texts: SessionCardTexts
  compact: boolean
}

export function SessionToolsCard({ tools, player, locale, texts, compact }: Props) {
  const t = texts.squad
  // Les natures sans nom de registre : la source unique `toolKindLabels` (manifeste `frags`, textes
  // de l'Escouade pour les deux catégories de source du film).
  const rows = useMemo(
    () => buildSquadToolRows(tools, { locale, labels: toolKindLabels(locale, t.weaponKills), top: compact ? SESSION_TOOLS_TOP : undefined }),
    [tools, locale, compact, t.weaponKills],
  )
  const colors = useMemo(() => getSquadPlayerColors(player, []), [player])
  const shareTotals = useMemo(() => {
    if (!compact) return undefined
    const all = (tools?.lines ?? []).reduce((a, l) => a + (l.kills_by_player[player] ?? 0), 0)
    return { [player]: all }
  }, [compact, tools, player])
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
      valueLabel={compact ? 'share' : 'count'}
      shareTotals={shareTotals}
      minLabelShare={compact ? 0 : undefined}
      soloByClass
    />
  )
}
