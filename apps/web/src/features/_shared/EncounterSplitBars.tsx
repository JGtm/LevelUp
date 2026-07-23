/**
 * EncounterSplitBars — barres proportionnelles allié/ennemi + frags/morts pour
 * les affichages d'affrontements (encounters).
 *
 * SOURCE UNIQUE (dédup #6, 2026-07-05) : `SplitBar` + `AllyEnemySplitBar` +
 * `KDSplitBar` étaient BYTE-IDENTIQUES dans `MatchEncountersTable.tsx` ET
 * `ExplorerEncounterBriefing.tsx` (cette dernière portait le commentaire
 * « copié depuis MatchEncountersTable.tsx »). Les 2 features importent d'ici.
 * Les libellés bilingues des tooltips vivent désormais en un seul endroit.
 */
import { Tooltip } from '@/components/ui/tooltip'
import { tokenCssVar } from '@/lib/accessibility'
import type { Locale } from '@/lib/i18n/locale'

function SplitBar({
  leftCount,
  rightCount,
  leftColor,
  rightColor,
  leftTooltip,
  rightTooltip,
}: {
  leftCount: number
  rightCount: number
  leftColor: string
  rightColor: string
  leftTooltip: string
  rightTooltip: string
}) {
  const total = leftCount + rightCount
  if (total === 0) return <span className="font-mono">—</span>
  const leftPct = Math.round((leftCount / total) * 100)
  return (
    <span className="inline-flex items-center gap-1 font-mono tabular-nums">
      <Tooltip content={leftTooltip}>
        <span style={{ color: leftColor }}>{leftCount}</span>
      </Tooltip>
      <span className="inline-flex h-2 w-12 border border-border overflow-hidden">
        <span style={{ width: `${leftPct}%`, backgroundColor: leftColor }} />
        <span style={{ flex: 1, backgroundColor: rightColor }} />
      </span>
      <Tooltip content={rightTooltip}>
        <span style={{ color: rightColor }}>{rightCount}</span>
      </Tooltip>
    </span>
  )
}

export function AllyEnemySplitBar({
  allyCount,
  enemyCount,
  locale,
}: {
  allyCount: number
  enemyCount: number
  locale: Locale
}) {
  const ttAlly = locale === 'en' ? `${allyCount} matches as ally` : `${allyCount} matchs en allié`
  const ttEnemy = locale === 'en' ? `${enemyCount} matches as enemy` : `${enemyCount} matchs en ennemi`
  return (
    <SplitBar
      leftCount={allyCount}
      rightCount={enemyCount}
      leftColor={tokenCssVar('team-ally')}
      rightColor={tokenCssVar('team-enemy')}
      leftTooltip={ttAlly}
      rightTooltip={ttEnemy}
    />
  )
}

export function KDSplitBar({
  kills,
  deaths,
  locale,
}: {
  kills: number
  deaths: number
  locale: Locale
}) {
  const ttKills = locale === 'en' ? `${kills} kills dealt` : `${kills} frags infligés`
  const ttDeaths = locale === 'en' ? `${deaths} deaths suffered` : `${deaths} morts subies`
  return (
    <SplitBar
      leftCount={kills}
      rightCount={deaths}
      leftColor={tokenCssVar('outcome-win')}
      rightColor={tokenCssVar('outcome-loss')}
      leftTooltip={ttKills}
      rightTooltip={ttDeaths}
    />
  )
}
