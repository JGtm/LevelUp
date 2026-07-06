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
import { SEASONS } from './seasons.i18n'
import {
  columnHighlightStyle,
  computeColumnExtremes,
  dmgPerKill,
  dmgPerDeath,
  type ColumnExtremes,
} from './LeaderboardBlock.highlight'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { EmptyStateCard } from '@/components/ui/empty-state'
import type { LeaderboardEntry } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { intlLocale } from '@/lib/formatters'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { csrRankImageURL } from '@/lib/staticAssets'
import { tokenCssVar } from '@/lib/accessibility'
import { skillDeltaScale } from '@/lib/accessibility/scales'

const CSR_WORLD = 'csr-world'

// Masquage responsive : le mode enrichi compte jusqu'à 10 colonnes. #, joueur et CSR
// restent toujours visibles ; victoires/KDA apparaissent dès `sm` ; tier/parties/
// précision/rendement/Δrang dès `lg`. Mêmes classes appliquées en-tête ET cellules.
const COL_HIDE_SM = 'hidden sm:table-cell'
const COL_HIDE_LG = 'hidden lg:table-cell'

/**
 * Playlists classées (asset IDs stables) — FALLBACK bilingue si le catalogue
 * dynamique (snapshots réellement en base) est vide. Dès que le catalogue est
 * présent, on s'appuie sur `display_name` renvoyé par l'API (déjà résolu selon la
 * locale via le header X-LevelUp-Locale côté backend).
 */
const PLAYLISTS: { id: string; fr: string; en: string }[] = [
  { id: 'edfef3ac-9cbe-4fa2-b949-8f29deafd483', fr: 'Arène classée', en: 'Ranked Arena' },
  { id: 'dcb2e24e-05fb-4390-8076-32a0cdb4326e', fr: 'Assassin classé', en: 'Ranked Slayer' },
  { id: 'fa5aa2a3-2428-4912-a023-e1eeea7b877c', fr: 'Duo classé', en: 'Ranked Doubles' },
  { id: '6233381c-fc96-40b9-b1ff-f6a4de72dd7a', fr: 'Snipers classés', en: 'Ranked Snipers' },
  { id: '57e417dd-7366-4dda-9bdd-2802151d5e81', fr: 'Tactique classé', en: 'Ranked Tactical' },
  { id: '71734db4-4b8e-4682-9206-62b6eff92582', fr: 'Chacun pour soi classé', en: 'Ranked FFA' },
]

// SEASONS (libellés de saison) déplacé dans ./seasons.i18n.ts — dict i18n local
// (whitelist du linter no-hardcoded-fields ; noms propres de saison sans source catalogue).

const KNOWN_SEASON_LABEL: Record<string, string> = Object.fromEntries(SEASONS.map((s) => [s.id, s.label]))

type SelectorOption = { value: string; label: string; enriched?: boolean }

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

/** Formate la valeur d'une catégorie de stat. */
function formatStatValue(entry: LeaderboardEntry, locale: ManifestLocale): string {
  const v = entry.value ?? 0
  const intl = intlLocale(locale)
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

const fmtPct = (v: number, locale: ManifestLocale): string =>
  `${(v * 100).toLocaleString(intlLocale(locale), { minimumFractionDigits: 1, maximumFractionDigits: 1 })}%`

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
  // Suffixe « (archivée) » sur les saisons non enrichies (classement CSR scrappé mais
  // pas de stats détaillées — historique au-delà de l'horizon API). Calculé hors useMemo
  // pour rester stable (dépend de la locale via t), et passé en dépendance.
  const archivedBadge = t('common.leaderboard.season_archived_badge')
  const seasonOptions: SelectorOption[] = useMemo(
    () =>
      catalog?.seasons?.length
        ? catalog.seasons.map((s) => {
            const base = KNOWN_SEASON_LABEL[s.id] ?? s.display_name
            return { value: s.id, label: s.enriched ? base : `${base} (${archivedBadge})`, enriched: s.enriched }
          })
        : SEASONS.map((s) => ({ value: s.id, label: s.label, enriched: true })),
    [catalog, archivedBadge],
  )
  const playlistOptions: SelectorOption[] = useMemo(
    () =>
      catalog?.playlists?.length
        ? // Le backend renvoie un display_name DÉJÀ localisé (header X-LevelUp-Locale :
          // cascade asset_translations[locale] > rankedplaylists locale > canonique > id).
          // On l'utilise directement, sans table FR codée en dur.
          catalog.playlists.map((p) => ({ value: p.id, label: p.display_name || p.id }))
        : // Catalogue vide (avant le 1er snapshot) : fallback bilingue local.
          PLAYLISTS.map((p) => ({ value: p.id, label: locale === 'en' ? p.en : p.fr })),
    [catalog, locale],
  )

  // Sélection effective dérivée au rendu (pas de setState dans un effet) : si le
  // choix courant n'est pas dans les options du catalogue, on retombe sur la 1re
  // option (la saison active est en tête côté API). Un choix utilisateur explicite
  // est toujours dans les options, donc préservé.
  const effectiveSeason = seasonOptions.some((o) => o.value === season) ? season : (seasonOptions[0]?.value ?? season)
  const effectivePlaylist = playlistOptions.some((o) => o.value === playlist)
    ? playlist
    : (playlistOptions[0]?.value ?? playlist)
  // Saison active non enrichie → bandeau « classement seul » (la table dégrade déjà :
  // hasEnrichment=false masque les colonnes stats détaillées).
  const selectedSeasonArchived =
    isWorld && seasonOptions.some((o) => o.value === effectiveSeason && o.enriched === false)
  const { data, isLoading, isError, error } = useLeaderboard(playerSlug, {
    category,
    season: isWorld ? effectiveSeason : undefined,
    playlist: isWorld ? effectivePlaylist : undefined,
    // Top-100 : profondeur d'enrichissement du cron world (WorldLeaderboardTopN).
    // NE PAS remonter à 200 — les rangs 101+ ne sont pas enrichis automatiquement
    // (cron scrape 200 mais n'agrège que le top-100) → cellules vides à l'écran.
    limit: 100,
  })

  function goToExplorer(gamertag: string, xuid: string) {
    void navigate({
      to: '/players/$playerSlug/explorer',
      params: { playerSlug },
      // On transmet le xuid (connu de la ligne) pour que l'Explorer affiche le
      // profil live même si le joueur n'est pas dans les données locales (sinon
      // ResolveXUIDByGamertag échoue côté backend pour un joueur du classement
      // mondial jamais croisé).
      search: { mode: 'player', target: gamertag, targetXuid: xuid || undefined },
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
          return e.cumulative_match_count ?? e.match_count ?? 0
        case 'kills':
          return e.kills ?? 0
        case 'deaths':
          return e.deaths ?? 0
        case 'assists':
          return e.assists ?? 0
        case 'win_rate':
          return e.win_rate ?? 0
        case 'kda':
          return (e.kda ?? 0) / (e.match_count || 1)
        case 'accuracy':
          return (e.accuracy ?? 0) / (e.match_count || 1)
        case 'dmg_per_kill':
          return dmgPerKill(e) ?? 0
        case 'dmg_per_death':
          return dmgPerDeath(e) ?? 0
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

  // Extrêmes {min,max} par colonne pour la mise en valeur best/worst (parité
  // scoreboard : meilleur en vert, pire en rouge ; cf. LeaderboardBlock.highlight).
  const extremes = useMemo(() => computeColumnExtremes(entries), [entries])

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
        {selectedSeasonArchived && (
          <p className="mt-1 text-xs italic text-muted-foreground">{t('common.leaderboard.archived_season_note')}</p>
        )}
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

        {data && entries.length === 0 && !isLoading && (
          <div className="p-4">
            <EmptyStateCard
              title={t('common.leaderboard.empty_title')}
              description={t('common.leaderboard.no_match_in_window')}
            />
          </div>
        )}

        {data && entries.length > 0 && (
          <div className="overflow-x-auto border border-border">
            <table className="w-full">
            <thead>
              <tr className="border-b bg-muted text-[11px] uppercase tracking-wide text-muted-foreground divide-x divide-border">
                <SortableTh label="#" className="w-12 text-center" onClick={() => toggleSort('rank')} suffix={sortIcon('rank')} />
                <th className="px-3 py-2 text-left font-medium">{t('common.leaderboard.col_player')}</th>
                {isWorld ? (
                  <>
                    <th className="px-3 py-2 text-center font-medium">{t('common.leaderboard.col_tier')}</th>
                    <SortableTh label={t('common.leaderboard.col_csr')} className="text-center" onClick={() => toggleSort('csr')} suffix={sortIcon('csr')} />
                    {hasEnrichment && (
                      <>
                        <SortableTh label={t('common.leaderboard.col_kda')} className={`text-center ${COL_HIDE_SM}`} onClick={() => toggleSort('kda')} suffix={sortIcon('kda')} />
                        <SortableTh label={t('common.leaderboard.col_frags')} className={`text-center ${COL_HIDE_SM}`} onClick={() => toggleSort('kills')} suffix={sortIcon('kills')} />
                        <SortableTh label={t('common.leaderboard.col_deaths')} className={`text-center ${COL_HIDE_SM}`} onClick={() => toggleSort('deaths')} suffix={sortIcon('deaths')} />
                        <SortableTh label={t('common.leaderboard.col_assists')} className={`text-center ${COL_HIDE_SM}`} onClick={() => toggleSort('assists')} suffix={sortIcon('assists')} />
                        <SortableTh label={t('common.leaderboard.col_win_rate')} className={`text-center ${COL_HIDE_SM}`} onClick={() => toggleSort('win_rate')} suffix={sortIcon('win_rate')} />
                        <SortableTh label={t('common.leaderboard.col_matches')} className={`text-center ${COL_HIDE_LG}`} onClick={() => toggleSort('world_matches')} suffix={sortIcon('world_matches')} />
                        <SortableTh label={t('common.leaderboard.col_accuracy')} className={`text-center ${COL_HIDE_LG}`} onClick={() => toggleSort('accuracy')} suffix={sortIcon('accuracy')} />
                        <SortableTh label={t('common.leaderboard.col_dmg_per_kill')} className={`text-center ${COL_HIDE_LG}`} onClick={() => toggleSort('dmg_per_kill')} suffix={sortIcon('dmg_per_kill')} />
                        <SortableTh label={t('common.leaderboard.col_dmg_per_death')} className={`text-center ${COL_HIDE_LG}`} onClick={() => toggleSort('dmg_per_death')} suffix={sortIcon('dmg_per_death')} />
                        <th className={`px-3 py-2 text-center font-medium ${COL_HIDE_LG}`}>{t('common.leaderboard.col_rank_delta')}</th>
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
                  extremes={extremes}
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
          </div>
        )}
      </CardContent>
    </Card>
  )
}

/** En-tête de colonne triable. */
function SortableTh({ label, className, onClick, suffix }: { label: string; className?: string; onClick: () => void; suffix: string }) {
  return (
    <th className={`px-3 py-2 font-medium ${className ?? ''}`}>
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
  extremes,
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
  extremes: ColumnExtremes
  localLabel: string
  trendTooltip: string
  rankDeltaTooltip: string
  locale: ManifestLocale
  onHover?: (gamertag: string) => void
  onGamertagClick: (gamertag: string, xuid: string) => void
}) {
  const intl = intlLocale(locale)
  // Accent podium : top-3 en gras (tokens foreground/muted, pas de hex).
  const isPodium = entry.rank <= 3
  const rankClass = isPodium ? 'font-bold text-primary' : 'text-muted-foreground'
  // Valeurs PAR MATCH (FDA/Précision) — mêmes calculs que l'affichage (égalité ===
  // exacte avec les extrêmes pour le highlight best/worst).
  const fda = entry.kda != null && entry.match_count ? entry.kda / entry.match_count : null
  const acc = entry.accuracy != null && entry.match_count ? entry.accuracy / entry.match_count : null
  const dpk = dmgPerKill(entry)
  const dpd = dmgPerDeath(entry)

  const playerCell: ReactNode = (
    <span className="inline-flex items-center gap-2">
      <button
        type="button"
        className="transition-colors hover:text-primary hover:underline"
        onClick={() => onGamertagClick(entry.gamertag, entry.xuid)}
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
      className={`divide-x divide-border border-b text-sm transition-colors last:border-0 hover:bg-muted ${entry.is_local ? 'bg-accent/40' : isPodium ? 'bg-primary/5' : 'even:bg-muted/30'}`}
      onMouseEnter={() => onHover?.(entry.gamertag)}
    >
      <td className={`px-3 py-2 text-center font-mono ${rankClass}`}>{entry.rank}</td>
      <td className="px-3 py-2 font-medium text-foreground">{playerCell}</td>
      {isWorld ? (
        <>
          <td className="px-3 py-2 text-center">
            <img
              src={csrRankImageURL(entry.tier, entry.sub_tier)}
              alt={entry.tier}
              className="mx-auto h-5 w-5 object-contain"
              loading="lazy"
            />
          </td>
          <td
            className="px-3 py-2 text-center font-mono text-foreground"
            style={columnHighlightStyle('csr', entry.csr_value, extremes)}
          >
            {entry.csr_value.toLocaleString(intl)}
          </td>
          {hasEnrichment && (
            <>
              <td
                className={`px-3 py-2 text-center font-mono text-foreground ${COL_HIDE_SM}`}
                style={columnHighlightStyle('fda', fda, extremes)}
              >
                {fda != null ? <MetricWithTrend text={fda.toFixed(2)} trend={entry.kda_trend} tooltip={trendTooltip} /> : '—'}
              </td>
              <td
                className={`px-3 py-2 text-center font-mono text-foreground ${COL_HIDE_SM}`}
                style={columnHighlightStyle('kills', entry.kills ?? null, extremes)}
              >
                {entry.kills?.toLocaleString(intl) ?? '—'}
              </td>
              <td
                className={`px-3 py-2 text-center font-mono text-foreground ${COL_HIDE_SM}`}
                style={columnHighlightStyle('deaths', entry.deaths ?? null, extremes)}
              >
                {entry.deaths?.toLocaleString(intl) ?? '—'}
              </td>
              <td
                className={`px-3 py-2 text-center font-mono text-foreground ${COL_HIDE_SM}`}
                style={columnHighlightStyle('assists', entry.assists ?? null, extremes)}
              >
                {entry.assists?.toLocaleString(intl) ?? '—'}
              </td>
              <td
                className={`px-3 py-2 text-center font-mono text-foreground ${COL_HIDE_SM}`}
                style={columnHighlightStyle('winRate', entry.win_rate ?? null, extremes)}
              >
                {entry.win_rate != null ? (
                  <MetricWithTrend text={fmtPct(entry.win_rate, locale)} trend={entry.win_rate_trend} tooltip={trendTooltip} />
                ) : (
                  '—'
                )}
              </td>
              <td className={`px-3 py-2 text-center font-mono text-muted-foreground ${COL_HIDE_LG}`}>
                {(entry.cumulative_match_count ?? entry.match_count)?.toLocaleString(intl) ?? '—'}
              </td>
              <td
                className={`px-3 py-2 text-center font-mono text-foreground ${COL_HIDE_LG}`}
                style={columnHighlightStyle('accuracy', acc, extremes)}
              >
                {acc != null ? `${acc.toLocaleString(intl, { minimumFractionDigits: 1, maximumFractionDigits: 1 })}%` : '—'}
              </td>
              <td
                className={`px-3 py-2 text-center font-mono text-foreground ${COL_HIDE_LG}`}
                style={columnHighlightStyle('dmgKill', dpk, extremes)}
              >
                {dpk != null ? Math.round(dpk).toLocaleString(intl) : '—'}
              </td>
              <td
                className={`px-3 py-2 text-center font-mono text-foreground ${COL_HIDE_LG}`}
                style={columnHighlightStyle('dmgDeath', dpd, extremes)}
              >
                {dpd != null ? Math.round(dpd).toLocaleString(intl) : '—'}
              </td>
              <td className={`px-3 py-2 text-center font-mono ${COL_HIDE_LG}`}>
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
          <td className="px-3 py-2 text-center font-mono text-muted-foreground">{entry.matches_played ?? 0}</td>
          <td className="px-3 py-2 text-right font-mono text-foreground">{formatStatValue(entry, locale)}</td>
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
