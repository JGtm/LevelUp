/**
 * SessionMatrix — la matrice d'une ligne dépliée du tableau « Impact des coéquipiers ».
 *
 * Une session : un joueur par ligne ; un match par colonne (numéro centré, trait vert ou rouge dessous
 * selon l'issue, les rôles du joueur en icônes, un point quand il n'en a aucun) ; une colonne par rôle
 * (le nombre de fois où le joueur l'a eu, meilleur et pire de l'escouade surlignés) ; « Score » (net de
 * la session, même surlignage, couronne au premier, banane au dernier s'il est négatif, passager
 * clandestin au dernier sinon). La ligne « Toutes les sessions » n'a pas de matchs : mêmes colonnes de
 * rôles et de score. Aucun titre de groupe au-dessus des colonnes, aucun cadre dans le cadre.
 */
import type { CSSProperties, ReactNode } from 'react'

import { BadgeIcon } from '@/components/feedback/BadgeIcon'
import { Tooltip } from '@/components/ui/tooltip'
import { tokenCssVar } from '@/lib/accessibility'
import type { Locale } from '@/lib/i18n/locale'

import type { SquadText } from '../i18n'
import {
  formatSignedNet,
  roleCellStates,
  scoreCellStates,
  scoreMark,
  type CellState,
  type ImpactMatchView,
  type ImpactRowView,
  type ImpactSessionsView,
} from './impactSessions.logic'
import type { ImpactSessionsText } from './impactSessionsText'

const PLAYER_COL_PX = 200
const ROLE_COL_PX = 40
const SCORE_COL_PX = 110

/** Le fond et le texte du meilleur (vert) et du pire (rouge), au rendu des MVP / LVP du scoreboard de match. */
function highlightStyle(state: CellState): CSSProperties | undefined {
  if (state === 'neutral') return undefined
  const token = tokenCssVar(state === 'best' ? 'outcome-win' : 'outcome-loss')
  return {
    backgroundColor: `color-mix(in oklab, ${token} 26%, transparent)`,
    color: token,
    fontWeight: state === 'best' ? 600 : 500,
  }
}

/** Une icône de rôle avec son infobulle (nom et description d'aujourd'hui). */
function RoleIcon({ role, impact }: { role: string; impact: SquadText['impact'] }) {
  const description = impact.badgeDescriptions[role]
  return (
    <Tooltip
      content={
        <>
          <span className="font-semibold">{impact.badgeNames[role] ?? role}</span>
          {description && <p className="mt-0.5 text-muted-foreground">{description}</p>}
        </>
      }
    >
      <BadgeIcon badgeKey={role} size={18} />
    </Tooltip>
  )
}

/** Un joueur : sa pastille de couleur et son nom. */
export function PlayerName({ player, ink }: { player: string; ink: string }) {
  return (
    <span className="inline-flex min-w-0 items-center gap-2">
      <i className="inline-block size-2.5 shrink-0 rounded-[3px]" style={{ backgroundColor: ink }} aria-hidden />
      <span className="truncate">{player}</span>
    </span>
  )
}

/** Le numéro d'un match, centré, avec un trait vert ou rouge dessous selon l'issue. */
function MatchNumber({ match, text }: { match: ImpactMatchView; text: ImpactSessionsText }) {
  const token = match.outcome === 'win' ? 'outcome-win' : match.outcome === 'loss' ? 'outcome-loss' : 'outcome-dnf'
  return (
    <span
      className="mx-auto flex h-5 w-fit min-w-[22px] items-center justify-center text-xs font-semibold text-foreground"
      style={{ boxShadow: `inset 0 -2px 0 ${tokenCssVar(token)}` }}
      title={text.matchHeader(match.number, match.outcome)}
    >
      {match.number}
    </span>
  )
}

const CELL = 'border-l border-border/60 px-2.5 py-2 text-center'
const GROUP_START = 'border-l border-border px-2.5 py-2 text-center'

interface MatrixProps {
  row: ImpactRowView
  view: ImpactSessionsView
  impact: SquadText['impact']
  text: ImpactSessionsText
  locale: Locale
  inkOf: (player: string) => string
  ariaLabel: string
}

export function SessionMatrix({ row, view, impact, text, locale, inkOf, ariaLabel }: MatrixProps) {
  const scoreStates = scoreCellStates(row.players)
  const statesByRole = Object.fromEntries(
    view.roleOrder.map((role) => [
      role,
      roleCellStates(row.players.map((p) => p.counts[role] ?? 0), view.costlyRoles.has(role)),
    ]),
  )
  return (
    <table className="w-full min-w-[900px] table-fixed border-collapse text-sm" aria-label={ariaLabel}>
      <colgroup>
        <col style={{ width: PLAYER_COL_PX }} />
        {row.matches.map((m) => (
          <col key={m.matchId} />
        ))}
        {view.roleOrder.map((role) => (
          <col key={role} style={{ width: ROLE_COL_PX }} />
        ))}
        <col style={{ width: SCORE_COL_PX }} />
      </colgroup>
      <thead>
        <tr className="border-b border-border/60 text-xs font-medium text-muted-foreground">
          <th scope="col" className="py-2 pl-9 pr-2.5 text-left font-medium">
            {text.colPlayer}
          </th>
          {row.matches.map((m, i) => (
            <th key={m.matchId} scope="col" className={`${i === 0 ? GROUP_START : CELL} font-medium`}>
              <MatchNumber match={m} text={text} />
            </th>
          ))}
          {view.roleOrder.map((role, i) => (
            <th key={role} scope="col" className={`${i === 0 ? GROUP_START : CELL} font-medium`}>
              <span className="flex justify-center">
                <RoleIcon role={role} impact={impact} />
              </span>
            </th>
          ))}
          <th scope="col" className={`${GROUP_START} font-medium`}>
            {text.colScore}
          </th>
        </tr>
      </thead>
      <tbody>
        {row.players.map((p, pi) => (
          <tr key={p.player} className={pi > 0 ? 'border-t border-border/60' : undefined}>
            <th scope="row" className="py-2 pl-9 pr-2.5 text-left font-normal">
              <PlayerName player={p.player} ink={inkOf(p.player)} />
            </th>
            {row.matches.map((m, i) => (
              <td key={m.matchId} className={i === 0 ? GROUP_START : CELL}>
                <MatchRoles roles={p.rolesByMatch[m.matchId]} impact={impact} noRole={text.noRole} />
              </td>
            ))}
            {view.roleOrder.map((role, i) => {
              const count = p.counts[role] ?? 0
              return (
                <td
                  key={role}
                  className={`${i === 0 ? GROUP_START : CELL} tabular-nums`}
                  style={highlightStyle(statesByRole[role][pi])}
                >
                  {count > 0 ? count : <span className="text-muted-foreground">—</span>}
                </td>
              )
            })}
            <td className={GROUP_START} style={highlightStyle(scoreStates[pi])}>
              <ScoreCell
                value={p.matchesPlayed > 0 ? formatSignedNet(p.points, locale) : null}
                mark={scoreMark(scoreStates[pi], p.points)}
                impact={impact}
              />
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

/** Les rôles d'un joueur sur un match, en icônes ; un point quand il n'en a aucun. */
function MatchRoles({ roles, impact, noRole }: { roles: string[] | undefined; impact: SquadText['impact']; noRole: string }) {
  if (!roles || roles.length === 0) {
    return (
      <span className="text-muted-foreground" title={noRole}>
        ·
      </span>
    )
  }
  return (
    <span className="flex flex-wrap justify-center gap-0.5" data-testid="squad-impact-cell">
      {roles.map((role, i) => (
        <RoleIcon key={`${role}-${i}`} role={role} impact={impact} />
      ))}
    </span>
  )
}

const MARK_LABEL: Record<'champion' | 'maillon_faible' | 'passager_clandestin', keyof SquadText['impact']> = {
  champion: 'badgeChampion',
  maillon_faible: 'badgeWeakLink',
  passager_clandestin: 'badgeStowaway',
}

/** Le score d'un joueur et sa marque (couronne, banane ou passager clandestin). */
function ScoreCell({
  value,
  mark,
  impact,
}: {
  value: string | null
  mark: ReturnType<typeof scoreMark>
  impact: SquadText['impact']
}): ReactNode {
  return (
    <span className="inline-flex items-center justify-center gap-1.5 font-semibold tabular-nums">
      {value ?? <span className="font-normal text-muted-foreground">—</span>}
      {mark && <BadgeIcon badgeKey={mark} size={16} title={String(impact[MARK_LABEL[mark]])} />}
    </span>
  )
}
