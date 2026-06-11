/**
 * LeaderboardBlock — page Classement.
 *
 * Catégorie "csr-world" (défaut) : classement CSR mondial par playlist + saison
 * classée (snapshots scrapés depuis Halo Waypoint). Catégories de stats
 * (Frags/KDA/Précision…) : agrégation des joueurs croisés (données locales).
 *
 * Tableau triable (tri client), highlight du joueur local, clic gamertag →
 * Explorer. Source de la donnée CSR : Halo Waypoint (pas de proxy tiers).
 */
import { useMemo, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import type { ReactNode } from 'react'

import { useLeaderboard, useLeaderboardCatalog } from './queries'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { EmptyStateCard } from '@/components/ui/empty-state'
import type { LeaderboardEntry } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { csrRankImageURL } from '@/lib/staticAssets'
import { tokenCssVar } from '@/lib/accessibility'
import { skillDeltaScale } from '@/lib/accessibility/scales'
import { CombatYieldDisplay } from '@/components/ui/combat-yield-display'

const SUB_TIER_ROMAN = ['', 'I', 'II', 'III', 'IV', 'V', 'VI']
const toRoman = (n: number): string => SUB_TIER_ROMAN[n] ?? String(n)

const CSR_WORLD = 'csr-world'

// Masquage responsive : le mode enrichi compte jusqu'à 10 colonnes. #, joueur et CSR
// restent toujours visibles ; victoires/KDA apparaissent dès `sm` ; tier/parties/
// précision/rendement/Δrang dès `lg`. Mêmes classes appliquées en-tête ET cellules.
const COL_HIDE_SM = 'hidden sm:table-cell'
const COL_HIDE_LG = 'hidden lg:table-cell'

/**
 * Playlists classées (asset IDs stables) — FALLBACK si le catalogue dynamique
 * (snapshots réellement en base) est vide. Sert aussi de table de libellés
 * localisés : un id présent ici prime sur le display_name renvoyé par l'API.
 */
const PLAYLISTS: { id: string; label: string }[] = [
  { id: 'edfef3ac-9cbe-4fa2-b949-8f29deafd483', label: 'Arène classée' },
  { id: 'dcb2e24e-05fb-4390-8076-32a0cdb4326e', label: 'Assassin classé' },
  { id: 'fa5aa2a3-2428-4912-a023-e1eeea7b877c', label: 'Duo classé' },
  { id: '6233381c-fc96-40b9-b1ff-f6a4de72dd7a', label: 'Snipers classés' },
  { id: '57e417dd-7366-4dda-9bdd-2802151d5e81', label: 'Tactique classé' },
  { id: '71734db4-4b8e-4682-9206-62b6eff92582', label: 'Chacun pour soi classé' },
]

/** Saisons CSR récentes (FALLBACK + libellés localisés, cf. PLAYLISTS). */
const SEASONS: { id: string; label: string }[] = [
  { id: 'csrseason13-2', label: 'Infinite (13.2)' },
  { id: 'csrseason13-1', label: 'Saison 13.1' },
  { id: 'csrseason12-1', label: 'Shadows (12.1)' },
  { id: 'csrseason11-1', label: 'Last Stand (11.1)' },
]

const KNOWN_PLAYLIST_LABEL: Record<string, string> = Object.fromEntries(PLAYLISTS.map((p) => [p.id, p.label]))
const KNOWN_SEASON_LABEL: Record<string, string> = Object.fromEntries(SEASONS.map((s) => [s.id, s.label]))

type SelectorOption = { value: string; label: string }

/** Catégories disponibles (clés i18n alignées sur common.leaderboard.cat_*). */
const CATEGORIES: { id: string; key: CommonManifestKey }[] = [
  { id: CSR_WORLD, key: 'common.leaderboard.cat_csr_world' },
  { id: 'kills', key: 'common.leaderboard.cat_kills' },
  { id: 'kda', key: 'common.leaderboard.cat_kda' },
  { id: 'kdr', key: 'common.leaderboard.cat_kdr' },
  { id: 'kills_per_game', key: 'common.leaderboard.cat_kills_per_game' },
  { id: 'accuracy', key: 'common.leaderboard.cat_accuracy' },
  { id: 'damage', key: 'common.leaderboard.cat_damage' },
  { id: 'assists', key: 'common.leaderboard.cat_assists' },
]

interface LeaderboardBlockProps {
  playerSlug: string
  onHoverEntry?: (gamertag: string) => void
}

type SortDir = 'asc' | 'desc'

function tierLabel(entry: LeaderboardEntry): string {
  if (!entry.tier || entry.tier === 'Onyx') return 'Onyx'
  return entry.sub_tier > 0 ? `${entry.tier} ${toRoman(entry.sub_tier)}` : entry.tier
}

/** Formate la valeur d'une catégorie de stat. */
function formatStatValue(entry: LeaderboardEntry, locale: string): string {
  const v = entry.value ?? 0
  const intl = locale === 'en' ? 'en-US' : 'fr-FR'
  const decimals = entry.unit === '%' || /kd|per_game/.test(entry.category ?? '') ? 2 : 0
  return `${v.toLocaleString(intl, { minimumFractionDigits: decimals, maximumFractionDigits: decimals })}${entry.unit ?? ''}`
}

type Trend = 'up' | 'down' | 'stable'
const TREND_GLYPH: Record<Trend, string> = { up: '▲', down: '▼', stable: '=' }
// Tokens sémantiques de tendance (cf. KPIStrip) — jamais de hex direct.
const TREND_VAR: Record<Trend, string> = {
  up: '--narrative-trend-positive',
  down: '--narrative-trend-negative',
  stable: '--narrative-trend-neutral',
}
const isTrend = (v: unknown): v is Trend => v === 'up' || v === 'down' || v === 'stable'

/** Valeur d'une métrique suivie d'une flèche de tendance colorée (optionnelle). */
function MetricWithTrend({ text, trend, tooltip }: { text: string; trend?: string | null; tooltip?: string }) {
  return (
    <span className="inline-flex items-baseline justify-end gap-1">
      <span>{text}</span>
      {isTrend(trend) && (
        <span
          className="text-[10px] font-bold leading-none"
          style={{ color: `var(${TREND_VAR[trend]})` }}
          title={tooltip}
          aria-label={tooltip}
        >
          {TREND_GLYPH[trend]}
        </span>
      )}
    </span>
  )
}

const fmtPct = (v: number, locale: string): string =>
  `${(v * 100).toLocaleString(locale === 'en' ? 'en-US' : 'fr-FR', { minimumFractionDigits: 1, maximumFractionDigits: 1 })}%`

// renderCombatYield rend Rendement/Résistance EXACTEMENT comme la tuile de match
// (CombatYieldDisplay). Calcule oc/dr/dégâts-par-frag/dégâts-par-mort depuis les
// compteurs bruts agrégés (baseline 225). Affichage uniquement — pas d'agrégation.
function renderCombatYield(e: LeaderboardEntry): ReactNode {
  const dd = e.damage_dealt
  const dt = e.damage_taken
  const k = e.kills ?? 0
  const d = e.deaths ?? 0
  const a = e.assists ?? 0
  if (dd == null || dt == null || k <= 0 || d <= 0) {
    return <span className="text-muted-foreground">—</span>
  }
  const dmgPerKill = dd / (k + a / 3)
  const dmgPerDeath = dt / d
  return (
    <CombatYieldDisplay
      className="min-w-[170px]"
      offensiveConversion={dmgPerKill > 0 ? 225 / dmgPerKill : null}
      defensiveResistance={dmgPerDeath / 225}
      dmgPerKill={dmgPerKill}
      dmgPerDeath={dmgPerDeath}
      align="center"
    />
  )
}

export function LeaderboardBlock({ playerSlug, onHoverEntry }: LeaderboardBlockProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const navigate = useNavigate()

  const [category, setCategory] = useState(CSR_WORLD)
  const [playlist, setPlaylist] = useState(PLAYLISTS[0].id)
  const [season, setSeason] = useState(SEASONS[0].id)
  const [sortKey, setSortKey] = useState('rank')
  const [sortDir, setSortDir] = useState<SortDir>('asc')

  const isWorld = category === CSR_WORLD

  // Sélecteurs dynamiques : on liste les saisons/playlists qui ont réellement des
  // snapshots en base (catalogue). Fallback sur les listes codées en dur si le
  // catalogue est vide (avant le 1er snapshot). Libellé : on préfère un label
  // localisé connu (KNOWN_*), sinon le display_name renvoyé par l'API.
  const { data: catalog } = useLeaderboardCatalog(playerSlug)
  const seasonOptions: SelectorOption[] = useMemo(
    () =>
      catalog?.seasons?.length
        ? catalog.seasons.map((s) => ({ value: s.id, label: KNOWN_SEASON_LABEL[s.id] ?? s.display_name }))
        : SEASONS.map((s) => ({ value: s.id, label: s.label })),
    [catalog],
  )
  const playlistOptions: SelectorOption[] = useMemo(
    () =>
      catalog?.playlists?.length
        ? // Phase F : le backend renvoie le nom du catalogue mutualise (cascade
          // asset_translations[fr] > rankedplaylists FR > catalogue EN). On le prefere
          // au libelle code en dur (KNOWN_*), garde en fallback ultime.
          catalog.playlists.map((p) => ({ value: p.id, label: p.display_name || KNOWN_PLAYLIST_LABEL[p.id] || p.id }))
        : PLAYLISTS.map((p) => ({ value: p.id, label: p.label })),
    [catalog],
  )

  // Sélection effective dérivée au rendu (pas de setState dans un effet) : si le
  // choix courant n'est pas dans les options du catalogue, on retombe sur la 1re
  // option (la saison active est en tête côté API). Un choix utilisateur explicite
  // est toujours dans les options, donc préservé.
  const effectiveSeason = seasonOptions.some((o) => o.value === season) ? season : (seasonOptions[0]?.value ?? season)
  const effectivePlaylist = playlistOptions.some((o) => o.value === playlist)
    ? playlist
    : (playlistOptions[0]?.value ?? playlist)
  const { data, isLoading, isError, error } = useLeaderboard(playerSlug, {
    category,
    season: isWorld ? effectiveSeason : undefined,
    playlist: isWorld ? effectivePlaylist : undefined,
    limit: 200,
  })

  function goToExplorer(gamertag: string) {
    void navigate({
      to: '/players/$playerSlug/explorer',
      params: { playerSlug },
      search: { mode: 'player', target: gamertag },
    })
  }

  function toggleSort(key: string) {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir(key === 'rank' ? 'asc' : 'desc')
    }
  }

  const entries = useMemo(() => {
    const rows = data?.entries ?? []
    const sortValue = (e: LeaderboardEntry): number => {
      switch (sortKey) {
        case 'csr':
          return e.csr_value
        case 'value':
          return e.value ?? 0
        case 'matches':
          return e.matches_played ?? 0
        case 'world_matches':
          return e.match_count ?? 0
        case 'win_rate':
          return e.win_rate ?? 0
        case 'kda':
          return (e.kda ?? 0) / (e.match_count || 1)
        case 'accuracy':
          return (e.accuracy ?? 0) / (e.match_count || 1)
        default:
          return e.rank
      }
    }
    const sorted = [...rows].sort((a, b) => sortValue(a) - sortValue(b))
    return sortDir === 'asc' ? sorted : sorted.reverse()
  }, [data?.entries, sortKey, sortDir])

  // Colonnes enrichies affichées uniquement si au moins un joueur est backfillé
  // (sinon table CSR historique inchangée). Calculé une fois pour aligner en-tête + cellules.
  const hasEnrichment = isWorld && entries.some((e) => e.match_count != null)

  const sortIcon = (key: string): string => (sortKey === key ? (sortDir === 'asc' ? ' ▲' : ' ▼') : '')

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle className="text-base">
            {isWorld ? t('common.leaderboard.title_world') : t('common.leaderboard.title_community')}
          </CardTitle>
          <div className="flex flex-wrap gap-2 text-xs">
            <Selector
              label={t('common.leaderboard.category_label')}
              value={category}
              onChange={setCategory}
              options={CATEGORIES.map((c) => ({ value: c.id, label: t(c.key) }))}
            />
            {isWorld && (
              <>
                <Selector
                  label={t('common.leaderboard.playlist_label')}
                  value={effectivePlaylist}
                  onChange={setPlaylist}
                  options={playlistOptions}
                />
                <Selector
                  label={t('common.leaderboard.season_label')}
                  value={effectiveSeason}
                  onChange={setSeason}
                  options={seasonOptions}
                />
              </>
            )}
          </div>
        </div>
        <p className="mt-1 text-xs text-muted-foreground">
          {isWorld ? t('common.leaderboard.world_hint') : t('common.leaderboard.stats_hint')}
        </p>
      </CardHeader>

      <CardContent className="p-0">
        {isLoading && (
          <div className="flex justify-center py-8">
            <Spinner size="lg" />
          </div>
        )}

        {isError && (
          <div className="p-4">
            <EmptyStateCard
              title={t('common.leaderboard.load_error')}
              description={error?.message ?? t('common.leaderboard.load_error')}
            />
          </div>
        )}

        {data && data.entries.length === 0 && !isLoading && (
          <div className="p-4">
            <EmptyStateCard
              title={t('common.leaderboard.empty_title')}
              description={t('common.leaderboard.no_match_in_window')}
            />
          </div>
        )}

        {data && data.entries.length > 0 && (
          <table className="w-full">
            <thead>
              <tr className="border-b bg-muted text-xs text-muted-foreground">
                <SortableTh label="#" className="w-12 text-center" onClick={() => toggleSort('rank')} suffix={sortIcon('rank')} />
                <th className="py-2 pr-4 text-left font-medium">{t('common.leaderboard.col_player')}</th>
                {isWorld ? (
                  <>
                    <th className={`py-2 pr-4 text-center font-medium ${COL_HIDE_LG}`}>{t('common.leaderboard.col_tier')}</th>
                    <SortableTh label={t('common.leaderboard.col_csr')} className="text-right" onClick={() => toggleSort('csr')} suffix={sortIcon('csr')} />
                    {hasEnrichment && (
                      <>
                        <SortableTh label={t('common.leaderboard.col_matches')} className={`text-right ${COL_HIDE_LG}`} onClick={() => toggleSort('world_matches')} suffix={sortIcon('world_matches')} />
                        <SortableTh label={t('common.leaderboard.col_win_rate')} className={`text-right ${COL_HIDE_SM}`} onClick={() => toggleSort('win_rate')} suffix={sortIcon('win_rate')} />
                        <SortableTh label={t('common.leaderboard.col_kda')} className={`text-right ${COL_HIDE_SM}`} onClick={() => toggleSort('kda')} suffix={sortIcon('kda')} />
                        <SortableTh label={t('common.leaderboard.col_accuracy')} className={`text-right ${COL_HIDE_LG}`} onClick={() => toggleSort('accuracy')} suffix={sortIcon('accuracy')} />
                        <th className={`py-2 pr-4 text-center font-medium ${COL_HIDE_LG}`}>{t('common.leaderboard.col_combat')}</th>
                        <th className={`py-2 pr-4 text-right font-medium ${COL_HIDE_LG}`}>{t('common.leaderboard.col_rank_delta')}</th>
                      </>
                    )}
                  </>
                ) : (
                  <>
                    <SortableTh label={t('common.leaderboard.col_matches')} className="text-center" onClick={() => toggleSort('matches')} suffix={sortIcon('matches')} />
                    <SortableTh label={t('common.leaderboard.col_value')} className="text-right" onClick={() => toggleSort('value')} suffix={sortIcon('value')} />
                  </>
                )}
              </tr>
            </thead>
            <tbody>
              {entries.map((entry) => (
                <LeaderboardRow
                  key={`${entry.xuid || entry.gamertag}-${entry.rank}`}
                  entry={entry}
                  isWorld={isWorld}
                  hasEnrichment={hasEnrichment}
                  localLabel={t('common.leaderboard.local_badge')}
                  trendTooltip={t('common.leaderboard.trend_tooltip')}
                  rankDeltaTooltip={t('common.leaderboard.rank_delta_tooltip')}
                  locale={locale}
                  onHover={onHoverEntry}
                  onGamertagClick={goToExplorer}
                />
              ))}
            </tbody>
          </table>
        )}
      </CardContent>
    </Card>
  )
}

/** En-tête de colonne triable. */
function SortableTh({ label, className, onClick, suffix }: { label: string; className?: string; onClick: () => void; suffix: string }) {
  return (
    <th className={`py-2 pr-4 font-medium ${className ?? ''}`}>
      <button type="button" onClick={onClick} className="transition-colors hover:text-foreground">
        {label}
        {suffix}
      </button>
    </th>
  )
}

/** Ligne du classement. */
function LeaderboardRow({
  entry,
  isWorld,
  hasEnrichment,
  localLabel,
  trendTooltip,
  rankDeltaTooltip,
  locale,
  onHover,
  onGamertagClick,
}: {
  entry: LeaderboardEntry
  isWorld: boolean
  hasEnrichment: boolean
  localLabel: string
  trendTooltip: string
  rankDeltaTooltip: string
  locale: string
  onHover?: (gamertag: string) => void
  onGamertagClick: (gamertag: string) => void
}) {
  // Accent podium : top-3 en gras + couleur pleine (vs muted), sans nouvelle couleur
  // (tokens sémantiques foreground/muted existants — pas de hex ni classe de teinte).
  const isPodium = entry.rank <= 3
  const rankClass = isPodium ? 'font-bold text-foreground' : 'text-muted-foreground'
  const playerCell: ReactNode = (
    <span className="inline-flex items-center gap-2">
      {isWorld && (
        <img
          src={csrRankImageURL(entry.tier, entry.sub_tier)}
          alt={entry.tier}
          className="h-6 w-6 shrink-0 object-contain"
          loading="lazy"
        />
      )}
      <button
        type="button"
        className="transition-colors hover:text-primary hover:underline"
        onClick={() => onGamertagClick(entry.gamertag)}
        title={`Explorer : ${entry.gamertag}`}
      >
        {entry.gamertag}
      </button>
      {entry.is_local && (
        <Badge variant="secondary" className="text-xs">
          {localLabel}
        </Badge>
      )}
    </span>
  )

  return (
    <tr
      className={`border-b text-sm transition-colors last:border-0 hover:bg-muted ${entry.is_local ? 'bg-accent/40' : ''}`}
      onMouseEnter={() => onHover?.(entry.gamertag)}
    >
      <td className={`py-2 pr-4 text-center font-mono ${rankClass}`}>{entry.rank}</td>
      <td className="py-2 pr-4 font-medium text-foreground">{playerCell}</td>
      {isWorld ? (
        <>
          <td className={`py-2 pr-4 text-center text-xs text-muted-foreground ${COL_HIDE_LG}`}>{tierLabel(entry)}</td>
          <td className="py-2 pr-4 text-right font-mono text-foreground">
            {entry.csr_value.toLocaleString(locale === 'en' ? 'en-US' : 'fr-FR')}
          </td>
          {hasEnrichment && (
            <>
              <td className={`py-2 pr-4 text-right font-mono text-muted-foreground ${COL_HIDE_LG}`}>{entry.match_count ?? '—'}</td>
              <td className={`py-2 pr-4 text-right font-mono text-foreground ${COL_HIDE_SM}`}>
                {entry.win_rate != null ? <MetricWithTrend text={fmtPct(entry.win_rate, locale)} trend={entry.win_rate_trend} tooltip={trendTooltip} /> : '—'}
              </td>
              <td className={`py-2 pr-4 text-right font-mono text-foreground ${COL_HIDE_SM}`}>
                {entry.kda != null && entry.match_count ? (
                  <MetricWithTrend text={(entry.kda / entry.match_count).toFixed(2)} trend={entry.kda_trend} tooltip={trendTooltip} />
                ) : (
                  '—'
                )}
              </td>
              <td className={`py-2 pr-4 text-right font-mono text-foreground ${COL_HIDE_LG}`}>
                {entry.accuracy != null && entry.match_count
                  ? `${(entry.accuracy / entry.match_count).toLocaleString(locale === 'en' ? 'en-US' : 'fr-FR', { minimumFractionDigits: 1, maximumFractionDigits: 1 })}%`
                  : '—'}
              </td>
              <td className={`py-2 pr-4 ${COL_HIDE_LG}`}>{renderCombatYield(entry)}</td>
              <td className={`py-2 pr-4 text-right font-mono ${COL_HIDE_LG}`}>
                {entry.rank_delta != null && entry.rank_delta !== 0 ? (
                  <span style={{ color: tokenCssVar(skillDeltaScale(entry.rank_delta)) }} title={rankDeltaTooltip}>
                    {entry.rank_delta > 0 ? '+' : ''}
                    {entry.rank_delta}
                  </span>
                ) : (
                  <span className="text-muted-foreground">—</span>
                )}
              </td>
            </>
          )}
        </>
      ) : (
        <>
          <td className="py-2 pr-4 text-center font-mono text-muted-foreground">{entry.matches_played ?? 0}</td>
          <td className="py-2 pr-4 text-right font-mono text-foreground">{formatStatValue(entry, locale)}</td>
        </>
      )}
    </tr>
  )
}

/** Sélecteur natif compact avec libellé accessible. */
function Selector({
  label,
  value,
  onChange,
  options,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  options: { value: string; label: string }[]
}) {
  return (
    <select
      aria-label={label}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="rounded border border-border bg-transparent px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
    >
      {options.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  )
}
