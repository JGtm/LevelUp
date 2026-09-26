/**
 * MatchNemesisCards — match_view.12 (Némésis et Souffre-douleur).
 *
 * Deux cartes côte à côte sur fond bg-card (aligné sur les autres blocs) :
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
import { themedIconSrc } from '@/lib/themedIcon'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'
import type { MatchViewText } from './i18n'

interface Props {
  nemesis: MatchNemesisRow[] | null | undefined
  scoreboard: MatchScoreboardRow[] | null | undefined
  meXUID: string | null
  t: MatchViewText
}

export function MatchNemesisCards({ nemesis, scoreboard, meXUID, t }: Props) {
  const theme = useSettingsDraftStore((state) => state.localUiPrefs.theme)
  const sb = scoreboard ?? []
  const nem = nemesis ?? []
  const sbMe = meXUID ? sb.find((r) => r.xuid === meXUID) : undefined
  const allyTeam = sbMe?.team_side ?? null
  const teamSideByXUID = new Map<string, string | null>()
  for (const r of sb) teamSideByXUID.set(r.xuid, r.team_side ?? null)

  const enemyDuels = nem.filter((n) => {
    if (n.xuid === meXUID) return false
    if (allyTeam == null) return true
    return teamSideByXUID.get(n.xuid) !== allyTeam
  })

  // Pas de `return null` quand il n'y a aucun duel : on garde les deux cartes
  // titrées (Némésis / Souffre-douleur) qui affichent leur état vide
  // (`combatNoNemesis`) au lieu de faire disparaître la section.
  const nemesisRow = pickMax(enemyDuels, (n) => n.killed_me)
  const bullyRow = pickMax(enemyDuels, (n) => n.i_killed)

  return (
    <div className="grid grid-cols-2 gap-4">
      <NemesisCard
        title={t.combatNemesisTitle}
        accentToken="outcome-loss"
        row={nemesisRow}
        statLine={nemesisRow ? t.combatKilledMeFmt(nemesisRow.killed_me) : null}
        logoSrc={themedIconSrc('nemesis', theme)}
        t={t}
      />
      <NemesisCard
        title={t.combatBullyTitle}
        accentToken="outcome-win"
        row={bullyRow}
        statLine={bullyRow ? t.combatIKilledFmt(bullyRow.i_killed) : null}
        logoSrc={themedIconSrc('victim', theme)}
        t={t}
      />
    </div>
  )
}

interface CardProps {
  title: string
  accentToken: SemanticToken
  row: MatchNemesisRow | null
  /** null = etat vide (aucun duel) : la ligne de stat est masquee plutot que d afficher un 0 absurde. */
  statLine: string | null
  logoSrc: string
  t: MatchViewText
}

function NemesisCard({ title, accentToken, row, statLine, logoSrc, t }: CardProps) {
  const accent = tokenCssVar(accentToken)
  return (
    <div
      className="relative overflow-hidden rounded-lg border border-border/60 bg-card p-5"
      style={{
        boxShadow: `inset 0 0 0 1px ${accent}33`,
      }}
    >
      <div
        className="absolute inset-y-0 left-0 w-1"
        style={{ background: accent }}
        aria-hidden="true"
      />
      <div className="flex items-center gap-3 pl-2">
        <img
          src={logoSrc}
          alt=""
          aria-hidden
          className="h-12 w-12 shrink-0 object-contain opacity-75"
        />
        <div className="min-w-0 flex-1">
          <div
            className="text-xs font-semibold uppercase tracking-wider"
            style={{ color: accent }}
          >
            {title}
          </div>
          <div className="mt-1 truncate text-2xl font-bold text-white">
            {row?.gamertag ?? t.combatNoNemesis}
          </div>
          {statLine !== null && (
            <div className="mt-3 text-sm text-white/80">
              <div className="tabular-nums">{statLine}</div>
            </div>
          )}
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
