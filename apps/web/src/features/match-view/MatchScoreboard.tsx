/**
 * MatchScoreboard — tableau de score du match avec highlight min/max et badges MVP/LVP.
 * A5 NATIVE_COMPONENTS — highlight vert (meilleur) / rouge (pire) par colonne,
 * inverti pour les colonnes négatives (deaths, damage_taken).
 * E1 : expansion de ligne avec PlayerDetailPanel.
 */
import { useState, useMemo } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { PlayerDetailPanel } from './PlayerDetailPanel'
import type { MatchScoreboardRow, MatchWeaponKill, MatchMedal, MatchCitation } from '@/lib/api/types'
import { useFieldMappings, type FieldMappingsResponse } from '@/lib/i18n/fieldMappings'

interface ColDef {
  key: keyof MatchScoreboardRow
  label: string
  inverted: boolean
  fmt?: (v: number) => string
}

/**
 * Construit les colonnes du scoreboard. Les libellés courts (K, D, A, HS, Tirs)
 * restent en abréviations FR pour des raisons d'espace, mais le `kda` colonne
 * récupère son libellé canonique depuis le backend (Phase D plan multi-titres).
 * Quand un futur titre redéfinira `kda` en `KAD` ou autre, le scoreboard
 * suivra automatiquement.
 */
function buildCols(fieldMappings?: FieldMappingsResponse): ColDef[] {
  const labelOf = (key: string): string =>
    fieldMappings?.fields[key]?.label ?? key
  return [
    { key: 'kills', label: labelOf('kills'), inverted: false },
    { key: 'deaths', label: labelOf('deaths'), inverted: true },
    { key: 'assists', label: labelOf('assists'), inverted: false },
    { key: 'headshot_kills', label: labelOf('headshot_kills'), inverted: false },
    { key: 'max_killing_spree', label: labelOf('max_killing_spree'), inverted: false },
    { key: 'perfect_kills', label: 'Perf', inverted: false },
    { key: 'power_weapon_kills', label: labelOf('power_weapon_kills'), inverted: false },
    { key: 'melee_kills', label: labelOf('melee_kills'), inverted: false },
    { key: 'shots_fired', label: labelOf('shots_fired'), inverted: false },
    { key: 'shots_hit', label: labelOf('shots_hit'), inverted: false },
    { key: 'kda', label: labelOf('kda'), inverted: false, fmt: (v) => v.toFixed(2) },
    { key: 'damage_dealt', label: labelOf('damage_dealt'), inverted: false, fmt: (v) => v.toFixed(0) },
    { key: 'damage_taken', label: labelOf('damage_taken'), inverted: true, fmt: (v) => v.toFixed(0) },
    { key: 'offensive_conversion', label: 'Off.', inverted: false, fmt: (v) => `${(v * 100).toFixed(0)}%` },
    { key: 'defensive_resistance', label: 'Déf.', inverted: false, fmt: (v) => `${(v * 100).toFixed(0)}%` },
    { key: 'damage_per_kill', label: 'Dmg/K', inverted: true, fmt: (v) => v.toFixed(0) },
  ]
}

type Extremes = { min: number | null; max: number | null }

function getExtremes(rows: MatchScoreboardRow[], key: keyof MatchScoreboardRow): Extremes {
  const vals = rows.map((r) => r[key] as number | null).filter((v): v is number => v != null)
  if (vals.length < 2) return { min: null, max: null }
  return { min: Math.min(...vals), max: Math.max(...vals) }
}

function cellClass(value: number | null, ex: Extremes, inverted: boolean): string {
  if (value == null || ex.min == null || ex.max == null || ex.min === ex.max) return ''
  const isBest = inverted ? value === ex.min : value === ex.max
  const isWorst = inverted ? value === ex.max : value === ex.min
  if (isBest) return 'bg-success/40 text-success font-semibold'
  if (isWorst) return 'bg-destructive/40 text-destructive'
  return ''
}

function getMvpLvpXuids(rows: MatchScoreboardRow[]): { mvp: string | null; lvp: string | null } {
  const withKills = rows.filter((r) => r.kills != null)
  if (withKills.length < 2) return { mvp: null, lvp: null }
  const sorted = [...withKills].sort((a, b) => (b.kills ?? 0) - (a.kills ?? 0))
  return { mvp: sorted[0].xuid, lvp: sorted[sorted.length - 1].xuid }
}

interface Props {
  rows: MatchScoreboardRow[]
  /** Armes du joueur principal — pour le détail E1 */
  weaponKills?: MatchWeaponKill[]
  /** Médailles du joueur principal — pour le détail E1 */
  medals?: MatchMedal[]
  /** Citations du joueur principal — pour le détail E1 */
  citations?: MatchCitation[]
}

export function MatchScoreboard({ rows, weaponKills, medals, citations }: Props) {
  const [expandedXuid, setExpandedXuid] = useState<string | null>(null)
  const { data: fieldMappings } = useFieldMappings()
  const cols = useMemo(() => buildCols(fieldMappings), [fieldMappings])
  const extremes = Object.fromEntries(cols.map((c) => [c.key, getExtremes(rows, c.key)]))
  const { mvp, lvp } = getMvpLvpXuids(rows)

  // Grouper par team_side
  const teams = Array.from(new Set(rows.map((r) => r.team_side ?? ''))).sort()

  function renderTeam(side: string) {
    const teamRows = rows.filter((r) => (r.team_side ?? '') === side)
    const label = side === '' ? 'Joueurs' : side === '0' || side.toLowerCase().includes('eagle') ? 'Mon équipe' : 'Équipe adverse'

    return (
      <div key={side} className="mb-4">
        <p className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">{label}</p>
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b border-border text-muted-foreground">
                <th className="pb-1 text-left pr-2">Joueur</th>
                {cols.map((c) => (
                  <th key={String(c.key)} className="pb-1 text-right pr-2">{c.label}</th>
                ))}
                <th className="pb-1 text-right">Vie moy.</th>
                <th className="pb-1 text-right pl-2">{fieldMappings?.fields['outcome']?.label ?? 'Résultat'}</th>
              </tr>
            </thead>
            <tbody>
              {teamRows.map((row) => {
                const isExpanded = expandedXuid === row.xuid
                return (
                  <>
                    <tr
                      key={row.xuid}
                      className={`cursor-pointer border-b border-border hover:bg-accent transition-colors ${row.is_me ? 'bg-info/10' : ''}`}
                      onClick={() => setExpandedXuid(isExpanded ? null : row.xuid)}
                    >
                      <td className="py-1 pr-2 font-medium text-foreground whitespace-nowrap">
                        <span className="mr-1 text-muted-foreground">{isExpanded ? '▾' : '▸'}</span>
                        {row.gamertag}
                        {row.xuid === mvp && (
                          <span className="ml-1 rounded px-1 py-0 text-[10px] font-bold bg-warning/80 text-foreground">MVP</span>
                        )}
                        {row.xuid === lvp && (
                          <span className="ml-1 rounded px-1 py-0 text-[10px] font-bold bg-muted text-muted-foreground">LVP</span>
                        )}
                      </td>
                      {cols.map((c) => {
                        const val = row[c.key] as number | null
                        const formatted = val == null ? '—' : c.fmt ? c.fmt(val) : String(val)
                        return (
                          <td key={String(c.key)} className={`py-1 pr-2 text-right font-mono ${cellClass(val, extremes[String(c.key)] as Extremes, c.inverted)}`}>
                            {formatted}
                          </td>
                        )
                      })}
                      <td className="py-1 text-right text-muted-foreground">
                        {row.avg_life_seconds != null
                          ? `${row.avg_life_seconds.toFixed(0)}s`
                          : (row.average_life ?? '—')}
                      </td>
                      <td className="py-1 pl-2 text-right text-muted-foreground whitespace-nowrap">{row.outcome_label}</td>
                    </tr>
                    {isExpanded && (
                      <tr key={`${row.xuid}-detail`}>
                        <td colSpan={cols.length + 3} className="p-0">
                          <PlayerDetailPanel
                            row={row}
                            weaponKills={row.is_me ? weaponKills : undefined}
                            medals={row.is_me ? medals : undefined}
                            citations={row.is_me ? citations : undefined}
                          />
                        </td>
                      </tr>
                    )}
                  </>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>
    )
  }

  return (
    <Card>
      <CardContent className="py-4">
        <p className="mb-3 text-sm font-semibold text-foreground">Scoreboard</p>
        {teams.map((side) => renderTeam(side))}
      </CardContent>
    </Card>
  )
}
