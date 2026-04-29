/**
 * SquadLayout — layout partagé de la section Escouade.
 *
 * Gère la sélection des coéquipiers (via data.options), les KPI cards et la
 * navigation par onglets (Synergies / Contributions). Expose les données
 * sélectionnées via SquadContext pour les onglets enfants.
 *
 * Multi-titres : tous les libellés métier passent par useFieldMappings
 * (fields.toml du titre courant) ; les strings UI passent par getSquadText
 * (FR/EN). La liste des KPIs (SQUAD_KPI_METRICS) dégrade gracefully quand
 * un FieldKey est absent du titre courant.
 *
 * Barre de filtres unifiée (sticky top-12, NavL2 masqué sur /squad) :
 *   [joueur actif] [coéquipiers▾] [Filtres▾] [Période▾] [Sessions escouade▾]
 *   [N matchs] [Analyser] [Réinitialiser]
 *
 * Route parente : /players/$playerSlug/squad
 * Routes enfants : /squad/synergies · /squad/contributions · /squad/v2
 */
import { useState, useEffect, useMemo, useRef } from 'react'
import { Outlet, useParams, Link, useMatchRoute } from '@tanstack/react-router'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { useAppShellStore } from '@/stores/appShellStore'
import { useTeammates } from './queries'
import { useSettings } from '@/features/settings/queries'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { GamertagCombobox } from '@/components/ui/GamertagCombobox'
import { SessionMultiSelect } from '@/components/ui/SessionMultiSelect'
import { AddFriendModal } from '@/features/friends/AddFriendFlow'
import { getSeriesColors, tokenCssVar } from '@/lib/accessibility'
import { useFieldMappings, type FieldMappingsResponse } from '@/lib/i18n/fieldMappings'
import { getSquadText } from './i18n'
import { SQUAD_KPI_METRICS, type SquadMetric } from './metrics'
import { log } from './_logger'
import { SquadContext } from './SquadContext'
import type { TeammateRow, TeammateKPIs, TeammatesQueryRequest } from '@/lib/api/types'
import { CompareDrawer } from '@/features/compare/CompareDrawer'
import { useComparePrefetch } from '@/features/compare/queries'
import {
  FiltresPill,
  PeriodePill,
  DEFAULT_CASCADE,
  DEFAULT_PERIOD,
  DEFAULT_SESSIONS,
  computePendingHash,
} from '@/components/shell/FilterOmnibar'

// ─── Constantes ───────────────────────────────────────────────────────────────

const MAX_SELECTION = 3
const CHART_COLORS = getSeriesColors(3, ['narrative-dominant', 'perf-tier-3', 'divergent-pos'])

function formatError(err: unknown): string {
  if (err == null) return 'Erreur inconnue'
  if (err instanceof Error) return err.message
  if (typeof err === 'string') return err
  if (typeof err === 'object') {
    const e = err as { message?: unknown; status?: unknown; statusText?: unknown }
    if (typeof e.message === 'string') return e.message
    if (typeof e.statusText === 'string') {
      return typeof e.status === 'number' ? `${e.status} ${e.statusText}` : e.statusText
    }
    try { return JSON.stringify(err) } catch { return 'Erreur non sérialisable' }
  }
  return String(err)
}

// ─── Helpers d'affichage ──────────────────────────────────────────────────────

function filterAvailableMetrics(
  metrics: readonly SquadMetric[],
  mappings: FieldMappingsResponse | undefined,
  surface: string,
): SquadMetric[] {
  if (!mappings) return [...metrics]
  const slug = mappings.title_slug
  return metrics.filter((m) => {
    const present = !!mappings.fields[m.key]
    if (!present) {
      log.warn(
        `field_missing:${slug}:${m.key}:${surface}`,
        `FieldKey "${m.key}" absent du fields.toml de ${slug} (surface=${surface}) — métrique masquée.`,
      )
    }
    return present
  })
}

function formatMetricValue(value: number | null, format: SquadMetric['format']): string {
  if (value == null) return '-'
  switch (format) {
    case 'integer': return String(Math.round(value))
    case 'percent': return `${value.toFixed(1)}%`
    case 'ratio': return value.toFixed(2)
    case 'per_game': return value.toFixed(2)
  }
}

function composeMetricLabel(
  metric: SquadMetric,
  mappings: FieldMappingsResponse | undefined,
  perGameSuffix: string,
): string {
  const baseLabel = mappings?.fields[metric.key]?.label ?? metric.key
  if (metric.format === 'per_game') return `${baseLabel}${perGameSuffix}`
  return baseLabel
}

// ─── KPI Block ────────────────────────────────────────────────────────────────

interface KPICardProps { label: string; value: string }
function KPICard({ label, value }: KPICardProps) {
  return (
    <div className="flex flex-col gap-1 rounded-lg border p-3">
      <span className="text-xs text-muted-foreground uppercase tracking-wide">{label}</span>
      <span className="text-xl font-bold">{value}</span>
    </div>
  )
}

interface KPIBlockProps { title: string; kpis: TeammateKPIs; color?: string; perGameSuffix: string }
function KPIBlock({ title, kpis, color = 'text-muted-foreground', perGameSuffix }: KPIBlockProps) {
  const { data: mappings } = useFieldMappings()
  const metrics = filterAvailableMetrics(SQUAD_KPI_METRICS, mappings, 'kpi')
  if (metrics.length === 0) return null
  return (
    <div>
      <h3 className={`text-sm font-medium mb-2 ${color}`}>{title}</h3>
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        {metrics.map((m) => (
          <KPICard
            key={m.key}
            label={composeMetricLabel(m, mappings, perGameSuffix)}
            value={formatMetricValue(m.extract(kpis), m.format)}
          />
        ))}
      </div>
    </div>
  )
}

// ─── Ligne coéquipier ─────────────────────────────────────────────────────────

interface TeammateRowItemProps {
  row: TeammateRow
  selectionIndex: number
  onSelect: () => void
  onCompare: (gamertag: string) => void
  onPrefetchCompare: (gamertag: string) => void
  intlLocale: string
  openCompareLabel: string
}
function TeammateRowItem({
  row, selectionIndex, onSelect, onCompare, onPrefetchCompare, intlLocale, openCompareLabel,
}: TeammateRowItemProps) {
  const isSelected = selectionIndex >= 0
  const wr = (row.with_kpis.win_rate * 100).toFixed(0)
  const kd = row.with_kpis.kd_ratio?.toFixed(2) ?? '-'
  const dotColor = isSelected ? CHART_COLORS[selectionIndex % CHART_COLORS.length] : undefined
  return (
    <tr
      onClick={onSelect}
      className={`cursor-pointer transition-colors hover:bg-muted ${isSelected ? 'bg-primary/10' : ''}`}
    >
      <td className="px-4 py-3 font-medium flex items-center gap-2">
        {dotColor && <span className="inline-block w-2 h-2 rounded-full" style={{ backgroundColor: dotColor }} />}
        {row.gamertag}
      </td>
      <td className="px-4 py-3 text-center">{row.encounter_count}</td>
      <td className="px-4 py-3 text-center">{row.with_kpis.wins}</td>
      <td className="px-4 py-3 text-center">{wr}%</td>
      <td className="px-4 py-3 text-center">{kd}</td>
      <td className="px-4 py-3 text-center text-xs text-muted-foreground">
        {row.last_seen_at ? new Date(row.last_seen_at).toLocaleDateString(intlLocale) : '-'}
      </td>
      <td className="px-4 py-3 text-center">
        <button
          onClick={(e) => { e.stopPropagation(); onCompare(row.gamertag) }}
          onMouseEnter={() => onPrefetchCompare(row.gamertag)}
          className="text-xs text-primary hover:underline"
        >
          {openCompareLabel}
        </button>
      </td>
    </tr>
  )
}

// ─── Composant principal ──────────────────────────────────────────────────────

export function SquadLayout() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const {
    filterContext,
    filterContextHash,
    resolvedContext,
    setFilterContext,
    resetFilters,
  } = useGlobalFilterStore()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const storageKey = `squad-teammates-${playerSlug}`

  // ── Sélection coéquipiers ────────────────────────────────────────────────
  const [selectedGts, setSelectedGtsRaw] = useState<string[]>(() => {
    try {
      const stored = localStorage.getItem(storageKey)
      return stored ? (JSON.parse(stored) as string[]) : []
    } catch { return [] }
  })
  const setSelectedGts = (next: string[] | ((prev: string[]) => string[])) => {
    setSelectedGtsRaw((prev) => {
      const value = typeof next === 'function' ? next(prev) : next
      try { localStorage.setItem(storageKey, JSON.stringify(value)) } catch { /* ignore */ }
      return value
    })
  }
  const confirmedGts = selectedGts
  const [compareTarget, setCompareTarget] = useState<string | null>(null)
  const [addFriendGamertag, setAddFriendGamertag] = useState<string | null>(null)
  const prefetchCompare = useComparePrefetch(playerSlug)
  const matchRoute = useMatchRoute()
  const { data: settings } = useSettings()

  // ── Filtre multi-sessions escouade (persisté, appliqué immédiatement) ────
  const sessionStorageKey = `squad-sessions-${playerSlug}`
  const [pickedSquadSessionLabels, setPickedSquadSessionLabelsRaw] = useState<string[]>(() => {
    try {
      const stored = localStorage.getItem(sessionStorageKey)
      return stored ? (JSON.parse(stored) as string[]) : []
    } catch { return [] }
  })
  const applySessionLabels = (labels: string[]) => {
    setPickedSquadSessionLabelsRaw(labels)
    try { localStorage.setItem(sessionStorageKey, JSON.stringify(labels)) } catch { /* ignore */ }
  }

  // ── Filtres global pending (période + cascade) — commités via Analyser ──
  const [pending, setPending] = useState(() => filterContext)
  const lastSyncedHash = useRef(filterContextHash)
  useEffect(() => {
    if (filterContextHash !== lastSyncedHash.current) {
      lastSyncedHash.current = filterContextHash
      setPending(filterContext)
    }
  }, [filterContextHash, filterContext])

  const [activePopover, setActivePopover] = useState<'filtres' | 'periode' | null>(null)
  const togglePopover = (which: 'filtres' | 'periode') =>
    setActivePopover((cur) => (cur === which ? null : which))
  const closeAll = () => setActivePopover(null)

  const pendingCascade = pending.cascade ?? DEFAULT_CASCADE
  const pendingPeriod = pending.period

  function setPendingPeriod(p: typeof DEFAULT_PERIOD) {
    const isPeriodSet = !!(p?.start_date || p?.end_date)
    setPending((prev) => ({
      ...prev,
      period: p,
      filter_mode: isPeriodSet ? 'period' : 'sessions',
      sessions: isPeriodSet ? DEFAULT_SESSIONS : prev.sessions,
    }))
  }
  function setPendingCascade(c: typeof DEFAULT_CASCADE) {
    setPending((prev) => ({ ...prev, cascade: c }))
  }

  const isDirty = filterContextHash !== computePendingHash(pending)
  const cascadeCount = (['playlists', 'modes', 'maps', 'experience_types'] as const)
    .reduce((n, k) => n + ((pendingCascade[k] as string[] | undefined)?.length ?? 0), 0)

  function handleAnalyser() {
    setFilterContext(pending)
    lastSyncedHash.current = computePendingHash(pending)
  }

  const totalAfter = resolvedContext?.counts?.total_matches_after_filters ?? null
  const rawAvailable = resolvedContext?.available_options
  const available = useMemo(() => {
    if (!rawAvailable) return undefined
    const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
    const filterUUIDs = (opts: { label: string; value: string }[]) =>
      opts.filter((o) => !UUID_RE.test(o.label.trim()))
    return {
      playlists: filterUUIDs(rawAvailable.playlists),
      modes: filterUUIDs(rawAvailable.modes),
      maps: filterUUIDs(rawAvailable.maps),
      experience_types: filterUUIDs(rawAvailable.experience_types),
    }
  }, [rawAvailable])


  // ── Init coéquipiers depuis settings ────────────────────────────────────
  useEffect(() => {
    if (settings?.friend_gamertags?.length && selectedGts.length === 0) {
      setSelectedGts(settings.friend_gamertags.slice(0, MAX_SELECTION))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [settings?.friend_gamertags])

  // ── Requête TeammatesService ─────────────────────────────────────────────
  const request: TeammatesQueryRequest = {
    filters: filterContext,
    selected_gamertags: confirmedGts.length > 0 ? confirmedGts : undefined,
    picked_squad_session_labels: pickedSquadSessionLabels.length > 0 ? pickedSquadSessionLabels : undefined,
  }
  const { data, isLoading, isError, error } = useTeammates(
    playerSlug,
    request,
    filterContextHash,
    confirmedGts,
    pickedSquadSessionLabels,
  )

  const toggleSelect = (gamertag: string) => {
    setSelectedGts((prev) => {
      if (prev.includes(gamertag)) return prev.filter((g) => g !== gamertag)
      if (prev.length >= MAX_SELECTION) return prev
      return [...prev, gamertag]
    })
  }

  // Sessions escouade (stables : LoadSynthesisMatches charge TOUT l'historique,
  // indépendamment de la période filtrée).
  const squadSessions = data?.session_labels?.squad ?? []

  // ── Routes actives ───────────────────────────────────────────────────────
  const synergiesRoute = '/players/$playerSlug/squad/synergies' as const
  const contributionsRoute = '/players/$playerSlug/squad/contributions' as const
  const v2Route = '/players/$playerSlug/squad/v2' as const
  const isSynergies = !!matchRoute({ to: synergiesRoute, fuzzy: true })
  const isContributions = !!matchRoute({ to: contributionsRoute, fuzzy: true })
  const isV2 = !!matchRoute({ to: v2Route, fuzzy: true })

  // ── Gestion chargement / erreur ──────────────────────────────────────────
  // La barre de filtres (sticky) est toujours rendue; seul le contenu est
  // conditionnel pour ne pas faire disparaître les contrôles lors d'un refetch.
  const availableOptions = data?.options ?? []
  const teammates = data?.teammates ?? []
  const selectedRows = confirmedGts
    .map((gt) => teammates.find((r) => r.gamertag.toLowerCase() === gt.toLowerCase()))
    .filter(Boolean) as TeammateRow[]

  if (confirmedGts.length > 0 && !isLoading && selectedRows.length === 0) {
    log.warn(
      `invalid_selection:${playerSlug}`,
      `Aucun gamertag confirmé n'a matché un teammate côté backend (player=${playerSlug}).`,
      { confirmedGts },
    )
  }

  return (
    <SquadContext.Provider
      value={{ selectedRows, confirmedGamertags: confirmedGts, pageData: data ?? null }}
    >
      {/* ─── Barre de filtres unifiée (sticky top-12, remplace NavL2/FilterOmnibar) ─── */}
      <div className="sticky top-12 z-30 border-b border-border" style={{ background: 'var(--background)' }}>
        <div className="flex min-h-10 items-center gap-1.5 overflow-visible px-4 py-1.5">

          {/* Joueur actif — pill colorée fixe, non supprimable */}
          <span
            className="inline-flex shrink-0 items-center rounded-full px-2.5 py-0.5 text-sm font-medium"
            style={{ backgroundColor: tokenCssVar('compare-a'), color: '#fff' }}
            title={playerSlug}
          >
            <span className="max-w-[7rem] truncate">{playerSlug}</span>
          </span>

          {/* Coéquipiers (multi-select compact inline, jusqu'à 3) */}
          <GamertagCombobox
            compact
            selected={selectedGts}
            onChange={setSelectedGts}
            max={MAX_SELECTION}
            frequentOptions={availableOptions}
            colors={CHART_COLORS}
            excludeGamertag={playerSlug}
            placeholder={t.selection.placeholder(availableOptions.length)}
            onAddAsFriend={setAddFriendGamertag}
          />

          {/* Séparateur */}
          <div className="mx-0.5 h-5 w-px shrink-0 bg-border" aria-hidden />

          {/* Filtres cascade (playlists / modes / cartes / expérience) */}
          {available && (
            <FiltresPill
              open={activePopover === 'filtres'}
              onToggle={() => togglePopover('filtres')}
              onClose={closeAll}
              available={available}
              cascade={pendingCascade}
              cascadeCount={cascadeCount}
              onSetCascade={setPendingCascade}
            />
          )}

          {/* Période */}
          <PeriodePill
            open={activePopover === 'periode'}
            onToggle={() => togglePopover('periode')}
            onClose={closeAll}
            period={pendingPeriod}
            onSetPeriod={setPendingPeriod}
          />

          {/* Sessions escouade (multi-select par label) */}
          {squadSessions.length > 0 && (
            <SessionMultiSelect
              sessions={squadSessions}
              selected={pickedSquadSessionLabels}
              onChange={applySessionLabels}
              locale={locale}
              triggerClassName="flex items-center gap-1.5 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium hover:bg-muted whitespace-nowrap transition-colors"
            />
          )}

          <div className="flex-1" />

          {/* Compteur dynamique */}
          {totalAfter !== null && (
            <span className="shrink-0 text-xs text-muted-foreground" aria-live="polite">
              {totalAfter} match{totalAfter > 1 ? 's' : ''}
            </span>
          )}

          {/* Analyser */}
          <button
            type="button"
            onClick={handleAnalyser}
            className={[
              'shrink-0 rounded-md px-2.5 py-1 text-xs font-medium transition-colors',
              isDirty
                ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                : 'border border-input bg-background text-muted-foreground hover:bg-muted',
            ].join(' ')}
          >
            Analyser
          </button>

          {/* Réinitialiser */}
          <button
            type="button"
            onClick={() => {
              resetFilters()
              applySessionLabels([])
            }}
            className="shrink-0 text-xs text-muted-foreground transition-colors hover:text-destructive"
            title="Réinitialiser tous les filtres"
          >
            ↺ Réinitialiser
          </button>
        </div>
      </div>

      {/* ─── Contenu ─────────────────────────────────────────────────────────── */}
      {isLoading && (
        <div className="flex items-center justify-center p-12 text-sm text-muted-foreground">
          Chargement…
        </div>
      )}

      {!isLoading && isError && (
        <div className="p-6 text-center text-destructive">
          {t.errors.loadError(formatError(error))}
        </div>
      )}

      {!isLoading && !isError && !data && (
        <div className="p-6">
          <EmptyStateCard title={t.empty.noDataTitle} description={t.empty.noDataDescription} />
        </div>
      )}

      {!isLoading && !isError && data && (
        <div className="flex flex-col gap-6 p-6">
          {/* KPI block si coéquipier(s) sélectionné(s) */}
          {selectedRows.length > 0 && (
            <Card>
              <CardHeader><CardTitle>{t.title.statsWith}</CardTitle></CardHeader>
              <CardContent className="flex flex-col gap-4">
                {selectedRows.map((row, i) => (
                  <KPIBlock
                    key={row.gamertag}
                    title={t.table.withTeammate(row.gamertag)}
                    kpis={row.with_kpis}
                    color={`text-[${CHART_COLORS[i % CHART_COLORS.length]}]`}
                    perGameSuffix={t.units.perGame}
                  />
                ))}
              </CardContent>
            </Card>
          )}

          {/* Navigation onglets */}
          <div className="border-b">
            <nav className="flex gap-0">
              <Link
                to="/players/$playerSlug/squad/synergies"
                params={{ playerSlug }}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${isSynergies ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
              >
                {t.nav.synergies}
              </Link>
              <Link
                to="/players/$playerSlug/squad/contributions"
                params={{ playerSlug }}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${isContributions ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
              >
                {t.nav.contributions}
              </Link>
              <Link
                to="/players/$playerSlug/squad/v2"
                params={{ playerSlug }}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${isV2 ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
              >
                {t.nav.v2}
              </Link>
            </nav>
          </div>

          <Outlet />

          {/* Table coéquipiers */}
          {teammates.length > 0 && (
            <Card>
              <CardHeader><CardTitle>{t.title.allTeammates}</CardTitle></CardHeader>
              <CardContent className="p-0">
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead className="bg-muted border-b">
                      <tr>
                        <th className="px-4 py-3 text-left">{t.table.gamertag}</th>
                        <th className="px-4 py-3 text-center">{t.table.matches}</th>
                        <th className="px-4 py-3 text-center">{t.table.wins}</th>
                        <th className="px-4 py-3 text-center">{t.table.winPct}</th>
                        <th className="px-4 py-3 text-center">{t.table.kd}</th>
                        <th className="px-4 py-3 text-center">{t.table.lastSeen}</th>
                        <th className="px-4 py-3 text-center">{t.table.actions}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y">
                      {teammates.map((row) => {
                        const idx = selectedGts.indexOf(row.gamertag)
                        return (
                          <TeammateRowItem
                            key={row.xuid ?? row.gamertag}
                            row={row}
                            selectionIndex={idx}
                            onSelect={() => toggleSelect(row.gamertag)}
                            onCompare={(gt) => setCompareTarget(gt)}
                            onPrefetchCompare={prefetchCompare}
                            intlLocale={t.intlLocale}
                            openCompareLabel={t.table.openCompare}
                          />
                        )
                      })}
                    </tbody>
                  </table>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      )}

      {compareTarget && (
        <CompareDrawer
          playerSlug={playerSlug}
          open={!!compareTarget}
          onClose={() => setCompareTarget(null)}
        />
      )}

      {addFriendGamertag && (
        <AddFriendModal
          gamertag={addFriendGamertag}
          open={!!addFriendGamertag}
          onClose={() => setAddFriendGamertag(null)}
          locale={locale}
        />
      )}
    </SquadContext.Provider>
  )
}
