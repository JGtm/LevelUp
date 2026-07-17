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
import { formatPercentInt } from '@/lib/formatters'
import { kdRatioColor, winRateClass } from '@/lib/colors/outcomePalette'
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { useNavigate, useParams } from '@tanstack/react-router'
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { Tooltip } from '@/components/ui/tooltip'
import { formatMessage } from '@/lib/i18n/format'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import { tokenVar } from '@/lib/accessibility'
import { AllyEnemySplitBar, KDSplitBar } from '@/features/_shared/EncounterSplitBars'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import type { MatchEncounterBadge, MatchEncounterRow } from '@/lib/api/types'

interface Props {
  rows: MatchEncounterRow[]
  /** Locale pour formatRelative (défaut fr). */
  locale?: 'fr' | 'en'
  /**
   * Override de la navigation au clic sur un gamertag. Si fourni, remplace la
   * navigation par défaut (Explorer mode joueur scoped sur le playerSlug courant).
   * Permet à la page Carrière de réutiliser ce tableau hors contexte match-view.
   */
  onPlayerClick?: (gamertag: string) => void
  /**
   * Désactive le wrapper carte (border + title bar). Utile quand le composant
   * est embarqué dans une section qui fournit déjà son propre header h2 (ex.
   * page Carrière, où le user a explicitement demandé "pas de bloc").
   */
  hideCardWrapper?: boolean
}

function isSemanticToken(s: string): s is SemanticToken {
  return s.startsWith('narrative-') || s.startsWith('outcome-') || s.startsWith('perf-')
}

const ENCOUNTER_BADGE_TOOLTIPS: Record<string, { fr: string; en: string }> = {
  'narrative.encounter.ally_plus': {
    fr: 'Allié récurrent : bon taux de victoire ensemble',
    en: 'Recurring ally: good win rate together',
  },
  'narrative.encounter.tough_enemy': {
    fr: 'Dur à cuire : ratio frags/morts contre lui défavorable',
    en: 'Tough nut: unfavorable frags/deaths ratio against him',
  },
  'narrative.encounter.coriace': {
    fr: 'Coriace : faible taux de victoire face à ce joueur',
    en: 'Tough opponent: low win rate against him',
  },
  'narrative.encounter.ordinal': {
    fr: 'Total rencontres croisées (allié + ennemi)',
    en: 'Total cross encounters (ally + enemy)',
  },
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
        const tooltip = ENCOUNTER_BADGE_TOOLTIPS[badge.label_key]?.[locale]
        const badgeEl = <NarrativeBadge label={label} colorVar={colorVar} solid size="sm" />
        return tooltip ? (
          <Tooltip key={i} content={tooltip}>
            {badgeEl}
          </Tooltip>
        ) : (
          <span key={i}>{badgeEl}</span>
        )
      })}
    </span>
  )
}

function formatKDCross(kills: number | null | undefined, deaths: number | null | undefined): string {
  if (kills == null && deaths == null) return '—'
  return `${kills ?? 0}/${deaths ?? 0}`
}

function formatKDRatio(kills: number | null | undefined, deaths: number | null | undefined): string {
  if (kills == null || deaths == null) return '—'
  if (deaths === 0) return kills > 0 ? '∞' : '—'
  return (kills / deaths).toFixed(2)
}

// SplitBar / AllyEnemySplitBar / KDSplitBar : extraits vers
// features/_shared/EncounterSplitBars.tsx (dédup #6 — cf. import ci-dessus).

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

export function MatchEncountersTable({ rows, locale = 'fr', onPlayerClick, hideCardWrapper = false }: Props) {
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
            kdCross: 'K/D',
            ratio: 'Ratio',
            ratioTooltip: 'Kill/Death ratio: kills dealt ÷ deaths suffered across all shared matches',
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
            kdCross: 'F/D',
            ratio: 'Ratio',
            ratioTooltip: 'Ratio frags/morts : frags infligés ÷ morts subies sur l’ensemble des matchs communs',
            lastSeen: 'Vu pour la dernière fois',
          },
    [locale],
  )

  function goToExplorer(gamertag: string) {
    if (onPlayerClick) {
      onPlayerClick(gamertag)
      return
    }
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
          const linkable = !r.is_bot && (Boolean(playerSlug) || Boolean(onPlayerClick))
          const displayGamertag = r.gamertag
          return (
            <span className="whitespace-nowrap">
              {linkable ? (
                <button
                  type="button"
                  className="font-semibold text-info hover:underline"
                  onClick={() => goToExplorer(r.gamertag)}
                >
                  {displayGamertag}
                </button>
              ) : (
                <span className={`font-semibold ${r.is_bot ? 'text-muted-foreground italic' : 'text-foreground'}`}>{displayGamertag}</span>
              )}
              {r.is_bot && (
                <span className="ml-1 rounded px-1 py-0 text-2xs font-bold bg-muted text-muted-foreground uppercase tracking-wide">Bot</span>
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
          // Pill Rôle harmonisée sur NarrativeBadge (même composant + rendu solid
          // que les badges de la colonne Joueur et de la page Relations). Couleurs
          // via les tokens encounter validés WCAG AA sur texte blanc
          // (cf. wcagContrast.test.ts) : vert ally-plus = allié, rouge tough-enemy
          // = ennemi. team-ally/team-enemy sont volontairement exclus ici car
          // configurables par l'utilisateur → le texte blanc forcé du mode solid
          // ne garantirait plus le contraste.
          const colorVar = tokenVar(
            r.is_ally ? 'narrative-encounter-ally-plus' : 'narrative-encounter-tough-enemy',
          )
          const txt = r.is_ally ? labels.roleAlly : labels.roleEnemy
          return <NarrativeBadge label={txt} colorVar={colorVar} solid size="sm" />
        },
      },
      {
        id: 'encounters',
        header: labels.encounters,
        cell: (ctx) => {
          const r = ctx.row.original
          if (r.ally_count != null && r.enemy_count != null) {
            return <AllyEnemySplitBar allyCount={r.ally_count} enemyCount={r.enemy_count} locale={locale} />
          }
          return <span className="font-mono">{r.count_together}</span>
        },
      },
      {
        id: 'wr_ally',
        header: labels.wrAlly,
        cell: (ctx) => {
          const v = ctx.row.original.winrate_as_ally
          return <span className={`font-mono ${winRateClass(v)}`}>{formatPercentInt(v)}</span>
        },
      },
      {
        id: 'wr_enemy',
        header: labels.wrEnemy,
        cell: (ctx) => {
          const v = ctx.row.original.winrate_vs_enemy
          return <span className={`font-mono ${winRateClass(v)}`}>{formatPercentInt(v)}</span>
        },
      },
      {
        id: 'kd_cross',
        header: labels.kdCross,
        cell: (ctx) => {
          const r = ctx.row.original
          if (r.kills_dealt != null && r.deaths_suffered != null) {
            return <KDSplitBar kills={r.kills_dealt} deaths={r.deaths_suffered} locale={locale} />
          }
          return <span className="font-mono">{formatKDCross(r.kills_dealt, r.deaths_suffered)}</span>
        },
      },
      {
        id: 'ratio',
        header: () => (
          <Tooltip content={labels.ratioTooltip}>
            <span className="cursor-help border-b border-dashed border-current">{labels.ratio}</span>
          </Tooltip>
        ),
        cell: (ctx) => {
          const r = ctx.row.original
          const color = kdRatioColor(r.kills_dealt, r.deaths_suffered)
          return (
            <span className="font-mono font-bold" style={color ? { color } : undefined}>
              {formatKDRatio(r.kills_dealt, r.deaths_suffered)}
            </span>
          )
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
    [labels, playerSlug, formatRelative, onPlayerClick],
  )

  const table = useReactTable<MatchEncounterRow>({
    data: rows,
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  if (!rows || rows.length === 0) {
    if (hideCardWrapper) {
      return <p className="text-xs text-muted-foreground">{labels.empty}</p>
    }
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

  // Mode hideCardWrapper : juste la table dans un overflow-x-auto, sans
  // la barre de titre (utile quand le composant parent fournit déjà un h2).
  if (hideCardWrapper) {
    return (
      <div className="overflow-x-auto">
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
    )
  }

  return (
    // Wrapper aligné sur ChartCard (Engagement) : carte rounded + barre titre.
    <div className="rounded-lg border border-border bg-card overflow-hidden">
      <div className="border-b border-border px-3 py-2 text-sm font-medium">
        {labels.title}
      </div>
      <div className="overflow-x-auto">
        {/*
          Grille complète : table border 2px (extérieur), cellules border 1px,
          header border-b-2 plus marqué. border-collapse fait que la bordure
          partagée prend la plus large.
        */}
        <table className="w-full border-collapse text-xs">
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
  )
}
