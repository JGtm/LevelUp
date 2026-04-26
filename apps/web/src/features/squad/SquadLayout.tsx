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
 * Route parente : /players/$playerSlug/squad
 * Routes enfants : /squad/synergies · /squad/contributions
 */
import { useState, useEffect } from 'react'
import { Outlet, useParams, Link, useMatchRoute } from '@tanstack/react-router'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { useAppShellStore } from '@/stores/appShellStore'
import { useTeammates } from './queries'
import { useSettings } from '@/features/settings/queries'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { GamertagCombobox } from '@/components/ui/GamertagCombobox'
import { SquadSessionSelector } from './SquadSessionSelector'
import { getSeriesColors } from '@/lib/accessibility'
import { useFieldMappings, type FieldMappingsResponse } from '@/lib/i18n/fieldMappings'
import { getSquadText } from './i18n'
import { SQUAD_KPI_METRICS, type SquadMetric } from './metrics'
import { log } from './_logger'
import { SquadContext } from './SquadContext'
import type { TeammateRow, TeammateKPIs, TeammatesQueryRequest } from '@/lib/api/types'
import { CompareDrawer } from '@/features/compare/CompareDrawer'
import { useComparePrefetch } from '@/features/compare/queries'

// ─── Constantes ───────────────────────────────────────────────────────────────

const MAX_SELECTION = 3
const CHART_COLORS = getSeriesColors(3, ['narrative-dominant', 'perf-tier-3', 'divergent-pos'])

// ─── Helpers d'affichage ──────────────────────────────────────────────────────

/**
 * Filtre les métriques absentes du fields.toml du titre courant.
 * Émet un warn dédupliqué pour signaler la dégradation gracieuse.
 */
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

/**
 * Formate la valeur numérique selon le format de la métrique.
 * Retourne `-` pour `null`.
 */
function formatMetricValue(value: number | null, format: SquadMetric['format']): string {
  if (value == null) return '-'
  switch (format) {
    case 'integer':
      return String(Math.round(value))
    case 'percent':
      return `${value.toFixed(1)}%`
    case 'ratio':
      return value.toFixed(2)
    case 'per_game':
      return value.toFixed(2)
  }
}

/**
 * Compose le label affiché pour une métrique : libellé canonique du
 * FieldKey + suffixe d'unité quand pertinent (per_game).
 */
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

interface KPICardProps {
  label: string
  value: string
}
function KPICard({ label, value }: KPICardProps) {
  return (
    <div className="flex flex-col gap-1 rounded-lg border p-3">
      <span className="text-xs text-muted-foreground uppercase tracking-wide">{label}</span>
      <span className="text-xl font-bold">{value}</span>
    </div>
  )
}

interface KPIBlockProps {
  title: string
  kpis: TeammateKPIs
  color?: string
  perGameSuffix: string
}
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
  row,
  selectionIndex,
  onSelect,
  onCompare,
  onPrefetchCompare,
  intlLocale,
  openCompareLabel,
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
        {dotColor && (
          <span
            className="inline-block w-2 h-2 rounded-full"
            style={{ backgroundColor: dotColor }}
          />
        )}
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
          onClick={(e) => {
            e.stopPropagation()
            onCompare(row.gamertag)
          }}
          onMouseEnter={() => onPrefetchCompare(row.gamertag)}
          className="text-xs text-primary hover:underline"
        >
          {openCompareLabel}
        </button>
      </td>
    </tr>
  )
}

export function SquadLayout() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const { filterContext, filterContextHash } = useGlobalFilterStore()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const storageKey = `squad-teammates-${playerSlug}`

  const [selectedGts, setSelectedGts] = useState<string[]>(() => {
    try {
      const stored = localStorage.getItem(storageKey)
      return stored ? (JSON.parse(stored) as string[]) : []
    } catch {
      return []
    }
  })
  const [confirmedGts, setConfirmedGts] = useState<string[]>(() => {
    try {
      const stored = localStorage.getItem(storageKey)
      return stored ? (JSON.parse(stored) as string[]) : []
    } catch {
      return []
    }
  })
  const [compareTarget, setCompareTarget] = useState<string | null>(null)
  const [squadSession, setSquadSession] = useState<string | null>(null)
  const prefetchCompare = useComparePrefetch(playerSlug)
  const matchRoute = useMatchRoute()
  const { data: settings } = useSettings()

  // Init depuis les amis par défaut dans les settings si localStorage vide
  useEffect(() => {
    if (settings?.friend_gamertags?.length && selectedGts.length === 0) {
      const initial = settings.friend_gamertags.slice(0, MAX_SELECTION)
      setSelectedGts(initial)
      setConfirmedGts(initial)
      localStorage.setItem(storageKey, JSON.stringify(initial))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [settings?.friend_gamertags])

  const isDirty =
    JSON.stringify([...selectedGts].sort()) !== JSON.stringify([...confirmedGts].sort())

  const handleConfirm = () => {
    setConfirmedGts(selectedGts)
    localStorage.setItem(storageKey, JSON.stringify(selectedGts))
  }

  const request: TeammatesQueryRequest = {
    filters: filterContext,
    selected_gamertags: confirmedGts.length > 0 ? confirmedGts : undefined,
    picked_squad_session_label: squadSession,
  }
  const { data, isLoading, isError, error } = useTeammates(
    playerSlug,
    request,
    filterContextHash,
    confirmedGts,
  )

  const toggleSelect = (gamertag: string) => {
    setSelectedGts((prev) => {
      if (prev.includes(gamertag)) return prev.filter((g) => g !== gamertag)
      if (prev.length >= MAX_SELECTION) return prev
      return [...prev, gamertag]
    })
  }

  if (isLoading) return null
  if (isError)
    return (
      <div className="p-6 text-center text-destructive">{t.errors.loadError(String(error))}</div>
    )
  if (!data) {
    return (
      <div className="p-6">
        <EmptyStateCard
          title={t.empty.noDataTitle}
          description={t.empty.noDataDescription}
        />
      </div>
    )
  }

  const availableOptions = data.options ?? []
  const teammates = data.teammates ?? []

  const selectedRows = confirmedGts
    .map((gt) => teammates.find((titem) => titem.gamertag === gt))
    .filter(Boolean) as TeammateRow[]

  // Détection du cas "sélection invalide" — un gamertag confirmé n'a matché
  // aucune ligne backend. Cf. teammates_service.go: buildTeammateRowWithMatches
  // retourne nil quand le gamertag n'est pas trouvé dans LoadTopTeammates.
  if (confirmedGts.length > 0 && selectedRows.length === 0) {
    log.warn(
      `invalid_selection:${playerSlug}`,
      `Aucun gamertag confirmé n'a matché un teammate côté backend (player=${playerSlug}).`,
      { confirmedGts },
    )
  }

  const synergiesRoute = '/players/$playerSlug/squad/synergies' as const
  const contributionsRoute = '/players/$playerSlug/squad/contributions' as const
  const isSynergies = !!matchRoute({ to: synergiesRoute, fuzzy: true })
  const isContributions = !!matchRoute({ to: contributionsRoute, fuzzy: true })

  return (
    <SquadContext.Provider
      value={{ selectedRows, confirmedGamertags: confirmedGts, pageData: data ?? null }}
    >
      <div className="flex flex-col gap-6 p-6">
        {/* Sélecteur de session escouade */}
        <SquadSessionSelector
          sessionLabels={data.session_labels ?? { solo: [], squad: [] }}
          squadSession={squadSession}
          onSquadChange={setSquadSession}
        />

        {/* Sélecteur coéquipiers avec fuzzy search */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{t.title.teammates}</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <GamertagCombobox
              selected={selectedGts}
              onChange={setSelectedGts}
              max={MAX_SELECTION}
              frequentOptions={availableOptions}
              colors={CHART_COLORS}
              excludeGamertag={playerSlug}
              placeholder={t.selection.placeholder(availableOptions.length)}
            />
            <div className="flex items-center justify-between gap-3 pt-1">
              <span className="text-xs text-muted-foreground">
                {isDirty
                  ? t.selection.dirty
                  : selectedGts.length === 0
                    ? t.selection.prompt
                    : t.selection.applied(confirmedGts.length)}
              </span>
              <button
                onClick={handleConfirm}
                disabled={selectedGts.length === 0 && confirmedGts.length === 0}
                className="shrink-0 rounded-md bg-primary px-4 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {t.selection.apply}
              </button>
            </div>
          </CardContent>
        </Card>

        {/* KPI block si coéquipier(s) sélectionné(s) — plus de "Référence solo" */}
        {selectedRows.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>{t.title.statsWith}</CardTitle>
            </CardHeader>
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
              className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors
                ${isSynergies
                  ? 'border-primary text-primary'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
                }`}
            >
              {t.nav.synergies}
            </Link>
            <Link
              to="/players/$playerSlug/squad/contributions"
              params={{ playerSlug }}
              className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors
                ${isContributions
                  ? 'border-primary text-primary'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
                }`}
            >
              {t.nav.contributions}
            </Link>
          </nav>
        </div>

        {/* Contenu de l'onglet actif */}
        <Outlet />

        {/* Table coéquipiers (secondaire, repliable si sélection active) */}
        {teammates.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>{t.title.allTeammates}</CardTitle>
            </CardHeader>
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

        {compareTarget && (
          <CompareDrawer
            playerSlug={playerSlug}
            open={!!compareTarget}
            onClose={() => setCompareTarget(null)}
          />
        )}
      </div>
    </SquadContext.Provider>
  )
}
