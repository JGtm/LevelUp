/**
 * MatchEncountersTable — historique des rencontres avec les joueurs du match
 * (mock match_view.14).
 *
 * Spec visuelle : .ai/charts_specs/_generated/match_view/mock-echarts.html
 *   section "match_view.14 — Historique des rencontres (joueurs du match)".
 *
 * Colonnes :
 *  - Joueur (lien vers Explorer + badges narratifs ally_plus/tough_enemy/ordinal)
 *  - Rôle (badge allié / ennemi sur ce match courant)
 *  - Rencontres (count_together + breakdown "A:X | E:Y")
 *  - WR allié (winrate_as_ally, vert ≥ 50%)
 *  - WR ennemi (winrate_vs_enemy, vert ≥ 50% sinon orange)
 *  - K/D croisé (kills_dealt / deaths_suffered)
 *  - Vu pour la dernière fois (relative time depuis last_seen_at)
 *
 * Badges narratifs : labels FR/EN résolus via squadManifest (clés
 * narrative.encounter.{ally_plus, tough_enemy, ordinal}). Mêmes assets que la
 * page Explorer (réutilisation NarrativeBadge + tokens accessibilité).
 *
 * Construit avec TanStack Table v8.
 */
import { useMemo } from 'react'
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { useNavigate, useParams } from '@tanstack/react-router'
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { formatMessage } from '@/lib/i18n/format'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import { tokenVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import type { MatchEncounterBadge, MatchEncounterRow } from '@/lib/api/types'

interface Props {
  rows: MatchEncounterRow[]
  /** Locale pour formatRelative (défaut fr). */
  locale?: 'fr' | 'en'
}

function formatPercent(v: number | null | undefined): string {
  if (v == null || !Number.isFinite(v)) return '—'
  return `${Math.round(v * 100)}%`
}

function isSemanticToken(s: string): s is SemanticToken {
  return s.startsWith('narrative-') || s.startsWith('outcome-') || s.startsWith('perf-')
}

/**
 * Rendu d'une liste de badges narratifs (ally_plus / tough_enemy / ordinal).
 * Labels résolus via squadManifest (FR/EN), couleurs via tokens accessibilité.
 * Aligné sur l'implémentation de ExplorerPage.tsx (consistency cross-page).
 */
function EncounterBadgesInline({
  badges,
  locale,
}: {
  badges: MatchEncounterBadge[]
  locale: 'fr' | 'en'
}) {
  if (!badges.length) return null
  return (
    <span className="ml-2 inline-flex flex-wrap gap-1 align-middle">
      {badges.map((badge, i) => {
        const labelKey = badge.label_key as SquadManifestKey
        const ordinal =
          badge.detail && typeof badge.detail['ordinal'] === 'number'
            ? (badge.detail['ordinal'] as number)
            : undefined
        const label =
          ordinal !== undefined ? formatMessage(squadManifest, labelKey, locale, { ordinal }) : formatMessage(squadManifest, labelKey, locale)
        const colorVar = isSemanticToken(badge.color_token)
          ? tokenVar(badge.color_token as SemanticToken)
          : undefined
        return <NarrativeBadge key={i} label={label} colorVar={colorVar} size="sm" />
      })}
    </span>
  )
}

function percentClass(v: number | null | undefined): string {
  if (v == null || !Number.isFinite(v)) return ''
  return v >= 0.5 ? 'text-success font-bold' : 'text-warning font-bold'
}

function formatKDCross(kills: number | null | undefined, deaths: number | null | undefined): string {
  if (kills == null && deaths == null) return '—'
  return `${kills ?? 0}/${deaths ?? 0}`
}

function formatRelativeFR(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return '—'
  const diffMs = Date.now() - date.getTime()
  const minutes = Math.round(diffMs / 60_000)
  if (minutes < 1) return "à l'instant"
  if (minutes < 60) return `il y a ${minutes} min`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return hours <= 1 ? 'il y a 1 h' : `il y a ${hours} h`
  const days = Math.round(hours / 24)
  if (days === 1) return 'hier'
  if (days < 7) return `il y a ${days} j`
  const weeks = Math.round(days / 7)
  if (weeks < 5) return weeks <= 1 ? 'il y a 1 sem.' : `il y a ${weeks} sem.`
  const months = Math.round(days / 30)
  if (months < 12) return months <= 1 ? 'il y a 1 mois' : `il y a ${months} mois`
  const years = Math.round(days / 365)
  return years <= 1 ? 'il y a 1 an' : `il y a ${years} ans`
}

function formatRelativeEN(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return '—'
  const diffMs = Date.now() - date.getTime()
  const minutes = Math.round(diffMs / 60_000)
  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes} min ago`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours} h ago`
  const days = Math.round(hours / 24)
  if (days === 1) return 'yesterday'
  if (days < 7) return `${days} d ago`
  const weeks = Math.round(days / 7)
  if (weeks < 5) return `${weeks} w ago`
  const months = Math.round(days / 30)
  if (months < 12) return `${months} mo ago`
  const years = Math.round(days / 365)
  return years <= 1 ? '1 y ago' : `${years} y ago`
}

export function MatchEncountersTable({ rows, locale = 'fr' }: Props) {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug?: string }
  const navigate = useNavigate()
  const formatRelative = locale === 'en' ? formatRelativeEN : formatRelativeFR

  const labels = useMemo(
    () =>
      locale === 'en'
        ? {
            title: 'Encounter history (match players)',
            empty: 'No prior encounters with these players.',
            player: 'Player',
            role: 'Role',
            roleAlly: 'ally',
            roleEnemy: 'enemy',
            encounters: 'Encounters',
            wrAlly: 'WR as ally',
            wrEnemy: 'WR as enemy',
            kdCross: 'Cross K/D',
            lastSeen: 'Last seen',
          }
        : {
            title: 'Historique des rencontres',
            empty: 'Aucune rencontre antérieure avec ces joueurs.',
            player: 'Joueur',
            role: 'Rôle',
            roleAlly: 'allié',
            roleEnemy: 'ennemi',
            encounters: 'Rencontres',
            wrAlly: 'Taux de victoire allié',
            wrEnemy: 'Taux de victoire ennemi',
            kdCross: 'ratio F/D croisé',
            lastSeen: 'Vu pour la dernière fois',
          },
    [locale],
  )

  function goToExplorer(gamertag: string) {
    if (!playerSlug) return
    void navigate({
      to: '/players/$playerSlug/explorer',
      params: { playerSlug },
      search: { mode: 'player', target: gamertag },
    })
  }

  const columns = useMemo<ColumnDef<MatchEncounterRow>[]>(
    () => [
      {
        id: 'player',
        header: labels.player,
        cell: (ctx) => {
          const r = ctx.row.original
          // Pas de lien Explorer pour les bots (xuid 'bid(...)' sans historique cross-match).
          const linkable = playerSlug && !r.is_bot
          return (
            <span className="whitespace-nowrap">
              {linkable ? (
                <button
                  type="button"
                  className="font-semibold text-info hover:underline"
                  onClick={() => goToExplorer(r.gamertag)}
                >
                  {r.gamertag}
                </button>
              ) : (
                <span className={`font-semibold ${r.is_bot ? 'text-muted-foreground italic' : 'text-foreground'}`}>{r.gamertag}</span>
              )}
              {r.is_bot && (
                <span className="ml-1 rounded px-1 py-0 text-[10px] font-bold bg-muted text-muted-foreground uppercase tracking-wide">Bot</span>
              )}
              {r.badges && r.badges.length > 0 && (
                <EncounterBadgesInline badges={r.badges} locale={locale} />
              )}
            </span>
          )
        },
      },
      {
        id: 'role',
        header: labels.role,
        cell: (ctx) => {
          const r = ctx.row.original
          const cls = r.is_ally
            ? 'bg-success/30 text-success'
            : 'bg-destructive/30 text-destructive'
          const txt = r.is_ally ? labels.roleAlly : labels.roleEnemy
          return (
            <span className={`inline-block rounded-full px-2 py-0.5 text-[0.75em] ${cls}`}>{txt}</span>
          )
        },
      },
      {
        id: 'encounters',
        header: labels.encounters,
        cell: (ctx) => {
          const r = ctx.row.original
          const ally = r.ally_count
          const enemy = r.enemy_count
          const breakdown =
            ally != null && enemy != null
              ? ` (A:${ally} | E:${enemy})`
              : ''
          return (
            <span className="font-mono">
              {r.count_together}
              {breakdown && (
                <span className="ml-1 text-[0.8em] text-muted-foreground">{breakdown}</span>
              )}
            </span>
          )
        },
      },
      {
        id: 'wr_ally',
        header: labels.wrAlly,
        cell: (ctx) => {
          const v = ctx.row.original.winrate_as_ally
          return <span className={`font-mono ${percentClass(v)}`}>{formatPercent(v)}</span>
        },
      },
      {
        id: 'wr_enemy',
        header: labels.wrEnemy,
        cell: (ctx) => {
          const v = ctx.row.original.winrate_vs_enemy
          return <span className={`font-mono ${percentClass(v)}`}>{formatPercent(v)}</span>
        },
      },
      {
        id: 'kd_cross',
        header: labels.kdCross,
        cell: (ctx) => {
          const r = ctx.row.original
          return <span className="font-mono">{formatKDCross(r.kills_dealt, r.deaths_suffered)}</span>
        },
      },
      {
        id: 'last_seen',
        header: labels.lastSeen,
        cell: (ctx) => {
          const ts = ctx.row.original.last_seen_at
          return (
            <span className="text-[0.85em] text-muted-foreground">
              {ts ? formatRelative(ts) : '—'}
            </span>
          )
        },
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [labels, playerSlug, formatRelative],
  )

  const table = useReactTable<MatchEncounterRow>({
    data: rows,
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  if (!rows || rows.length === 0) {
    return (
      <div className="rounded-lg border border-border bg-card overflow-hidden">
        <div className="border-b border-border px-3 py-2 text-sm font-medium">
          {labels.title}
        </div>
        <div className="p-3">
          <p className="text-xs text-muted-foreground">{labels.empty}</p>
        </div>
      </div>
    )
  }

  return (
    // Wrapper aligné sur ChartCard (Engagement) : carte rounded + barre titre.
    <div className="rounded-lg border border-border bg-card overflow-hidden">
      <div className="border-b border-border px-3 py-2 text-sm font-medium">
        {labels.title}
      </div>
      <div className="p-3">
        <div className="overflow-x-auto">
          {/*
            Grille complète : table border 2px (extérieur), cellules border 1px,
            header border-b-2 plus marqué. border-collapse fait que la bordure
            partagée prend la plus large.
          */}
          <table className="w-full border-2 border-border border-collapse text-xs">
            <thead>
              {table.getHeaderGroups().map((hg) => (
                <tr key={hg.id} className="text-muted-foreground">
                  {hg.headers.map((h, idx) => (
                    <th
                      key={h.id}
                      className={`border border-border border-b-2 px-2 pb-1 pt-1 ${idx === 0 ? 'text-left' : 'text-right'}`}
                    >
                      {flexRender(h.column.columnDef.header, h.getContext())}
                    </th>
                  ))}
                </tr>
              ))}
            </thead>
            <tbody>
              {table.getRowModel().rows.map((row) => (
                <tr key={row.id} className="hover:bg-accent/40 transition-colors">
                  {row.getVisibleCells().map((cell, idx) => (
                    <td
                      key={cell.id}
                      className={`border border-border px-2 py-1.5 ${idx === 0 ? 'text-left' : 'text-right'}`}
                    >
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
