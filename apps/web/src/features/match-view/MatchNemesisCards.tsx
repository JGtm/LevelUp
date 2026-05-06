/**
 * MatchNemesisCards — match_view.12 (Némésis et Souffre-douleur).
 *
 * Deux cartes côte à côte sur fond sombre :
 *   - Némésis : adversaire qui m'a le plus tué (max killed_me).
 *   - Souffre-douleur : adversaire que j'ai le plus tué (max i_killed).
 *
 * Source : `team_tab.nemesis` filtré pour ne garder que les adversaires
 * (team_side différent du joueur principal). Le scoreboard est utilisé
 * pour résoudre les team_side et écarter les coéquipiers (qui ne devraient
 * pas apparaître dans cette section narrative).
 */
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import type { MatchNemesisRow, MatchScoreboardRow } from '@/lib/api/types'
import type { MatchViewText } from './i18n'

interface Props {
  nemesis: MatchNemesisRow[]
  scoreboard: MatchScoreboardRow[]
  meXUID: string | null
  t: MatchViewText
}

export function MatchNemesisCards({ nemesis, scoreboard, meXUID, t }: Props) {
  const sbMe = meXUID ? scoreboard.find((r) => r.xuid === meXUID) : undefined
  const allyTeam = sbMe?.team_side ?? null
  const teamSideByXUID = new Map<string, string | null>()
  for (const r of scoreboard) teamSideByXUID.set(r.xuid, r.team_side ?? null)

  const enemyDuels = nemesis.filter((n) => {
    if (n.xuid === meXUID) return false
    if (allyTeam == null) return true
    return teamSideByXUID.get(n.xuid) !== allyTeam
  })

  if (enemyDuels.length === 0) {
    return null
  }

  const nemesisRow = pickMax(enemyDuels, (n) => n.killed_me)
  const bullyRow = pickMax(enemyDuels, (n) => n.i_killed)

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <NemesisCard
        title={t.combatNemesisTitle}
        accentToken="outcome-loss"
        row={nemesisRow}
        t={t}
      />
      <NemesisCard
        title={t.combatBullyTitle}
        accentToken="outcome-win"
        row={bullyRow}
        t={t}
      />
    </div>
  )
}

interface CardProps {
  title: string
  accentToken: SemanticToken
  row: MatchNemesisRow | null
  t: MatchViewText
}

function NemesisCard({ title, accentToken, row, t }: CardProps) {
  const accent = tokenCssVar(accentToken)
  return (
    <div
      className="relative overflow-hidden rounded-lg border border-border/60 p-5"
      style={{
        background: 'linear-gradient(135deg, rgb(15 18 24) 0%, rgb(24 27 36) 100%)',
        boxShadow: `inset 0 0 0 1px ${accent}33`,
      }}
    >
      <div
        className="absolute inset-y-0 left-0 w-1"
        style={{ background: accent }}
        aria-hidden="true"
      />
      <div className="pl-2">
        <div
          className="text-xs font-semibold uppercase tracking-wider"
          style={{ color: accent }}
        >
          {title}
        </div>
        <div className="mt-1 truncate text-2xl font-bold text-white">
          {row?.gamertag ?? t.combatNoNemesis}
        </div>
        <div className="mt-3 space-y-1 text-sm text-white/80">
          <div className="tabular-nums">
            {t.combatKilledMeFmt(row?.killed_me ?? 0)}
          </div>
          <div className="tabular-nums">
            {t.combatIKilledFmt(row?.i_killed ?? 0)}
          </div>
        </div>
      </div>
    </div>
  )
}

function pickMax<T>(rows: T[], score: (r: T) => number): T | null {
  if (rows.length === 0) return null
  let best = rows[0]
  let bestScore = score(best)
  for (let i = 1; i < rows.length; i++) {
    const s = score(rows[i])
    if (s > bestScore) {
      best = rows[i]
      bestScore = s
    }
  }
  return bestScore > 0 ? best : null
}
