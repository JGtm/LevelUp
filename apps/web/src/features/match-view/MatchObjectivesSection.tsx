/**
 * MatchObjectivesSection — section « Objectifs » du scoreboard Match View (V72-03).
 *
 * Affiche, PAR ÉQUIPE, les stats objectifs par joueur du mode joué (CTF / Zones
 * (Strongholds+KOTH) / Oddball / Stockpile / Extraction / VIP — V721-02 pour les 3
 * derniers) + une ligne « Total équipe » (agrégat à la LECTURE :
 * SUM cumulée, ou MAX pour les « meilleurs temps »). Colonnes pertinentes pilotées
 * par le mode détecté (data-driven). Durées formatées mm:ss.
 *
 * Double porte d'affichage :
 *   - capability `objective_stats` (titre) via useCapability — masquée sur un titre
 *     qui ne la déclare pas (Halo 5) ;
 *   - data-driven : aucune ligne ne porte de bloc objectif (Slayer) → mode == null →
 *     rien affiché (pas de section vide).
 *
 * Style aligné sur le scoreboard existant (colonnes compactes, tooltips d'en-tête
 * via HeaderLabelTooltip, tokens sémantiques — aucun hex). Le rendu ne réplique pas
 * l'identité complète d'équipe du scoreboard principal : couleur d'identité en
 * accent (bordure gauche + fond subtil), texte en --foreground pour le contraste.
 */
import { useMemo } from 'react'

import type { MatchScoreboardRow } from '@/lib/api/types'
import { useCapability } from '@/lib/capabilities/capabilities'
import { formatDurationMMSS } from '@/lib/formatters/duration'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { parseTeamSideID } from '@/lib/halo/teamNames'
import { displayPlayerName } from '@/lib/players/displayName'
import { HeaderLabelTooltip } from '@/lib/table/columnMeta'

import type { MatchViewText } from './i18n'
import { teamColorResolver } from './teamColor'
import {
  detectObjectiveMode,
  objectiveColsFor,
  objectiveTeamTotal,
  type ObjectiveColSpec,
} from './MatchScoreboard.logic'

interface Props {
  rows: MatchScoreboardRow[]
  /** team_side distincts du lobby (ex: "t0", "t1"), déjà triés par le parent. */
  teams: string[]
  /** team_side du joueur principal (is_me) — pour le repli couleur ally/enemy. */
  myTeamSide: string | null
  t: MatchViewText
}

/** Cellule d'une valeur objectif : « — » si nulle, mm:ss si durée, sinon l'entier. */
function fmtCell(v: number | null | undefined, col: ObjectiveColSpec): string {
  if (v == null) return '—'
  return col.duration ? formatDurationMMSS(v) : String(v)
}

/** Libellé + couleur d'identité d'une équipe (cascade backend → team_id → token). */
function teamVisual(
  teamRows: MatchScoreboardRow[],
  teamSide: string,
  isMyTeam: boolean,
  t: MatchViewText,
): { label: string; colorVar: string } {
  const teamID = parseTeamSideID(teamSide)
  const label = resolveTeamLabel(teamRows, teamSide, t)
  // Couleur d'identité : cascade UNIQUE (`teamColor.ts`), la même que le fil et le scoreboard.
  const colorVar = teamColorResolver(teamRows)(teamID, isMyTeam)
  return { label, colorVar }
}

export function MatchObjectivesSection({ rows, teams, myTeamSide, t }: Props) {
  const hasObjectiveCap = useCapability('objective_stats')
  const mode = useMemo(() => detectObjectiveMode(rows), [rows])

  // Double porte : capability titre absente OU aucun bloc objectif → rien.
  if (!hasObjectiveCap || mode == null) return null

  const cols = objectiveColsFor(mode)
  const colMeta = t.objectives.cols

  return (
    <section className="rounded-lg border-2 border-border" aria-label={t.objectives.title}>
      <h3 className="px-3 py-2 text-sm font-bold uppercase tracking-wider text-foreground">
        {t.objectives.title}
      </h3>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-3xs">
          <thead>
            <tr className="text-muted-foreground">
              <th className="border border-border border-b-2 px-2 py-1 text-left">
                {t.sbColPlayer}
              </th>
              {cols.map((c) => (
                <th key={String(c.key)} className="border border-border border-b-2 px-2 py-1 text-right">
                  <HeaderLabelTooltip text={colMeta[String(c.key)]?.tooltip} focusable>
                    <span className="inline-flex items-center justify-end gap-1">
                      {colMeta[String(c.key)]?.label ?? String(c.key)}
                    </span>
                  </HeaderLabelTooltip>
                </th>
              ))}
            </tr>
          </thead>
          {teams.map((side) => {
            const teamRows = rows.filter((r) => (r.team_side ?? '') === side && r.objective != null)
            if (teamRows.length === 0) return null
            const isMyTeam = side !== '' && side === myTeamSide
            const { label, colorVar } = teamVisual(teamRows, side, isMyTeam, t)
            return (
              <tbody key={side || 'unknown'}>
                <tr>
                  <th
                    colSpan={cols.length + 1}
                    className="border border-border px-3 py-1 text-left text-xs font-semibold text-foreground"
                    style={{
                      background: `color-mix(in oklab, ${colorVar} 18%, transparent)`,
                      borderLeft: `4px solid ${colorVar}`,
                    }}
                  >
                    {label}
                  </th>
                </tr>
                {teamRows.map((r, idx) => (
                  // Clé composite : le xuid peut être vide (joueurs H5 sans xuid) →
                  // plusieurs lignes key="" en collision. gamertag + index désambiguïsent.
                  <tr key={`${r.xuid}||${r.gamertag}||${idx}`} className={r.is_me ? 'bg-info/10' : ''}>
                    <td className="border border-border px-2 py-1 text-left">{displayPlayerName(r.gamertag, r.xuid)}</td>
                    {cols.map((c) => (
                      <td key={String(c.key)} className="border border-border px-2 py-1 text-right tabular-nums">
                        {fmtCell(r.objective?.[c.key], c)}
                      </td>
                    ))}
                  </tr>
                ))}
                <tr className="font-semibold text-foreground">
                  <td className="border border-border px-2 py-1 text-left">
                    {t.objectives.teamTotal}
                  </td>
                  {cols.map((c) => (
                    <td key={String(c.key)} className="border border-border px-2 py-1 text-right tabular-nums">
                      {fmtCell(objectiveTeamTotal(teamRows, c), c)}
                    </td>
                  ))}
                </tr>
              </tbody>
            )
          })}
        </table>
      </div>
    </section>
  )
}
