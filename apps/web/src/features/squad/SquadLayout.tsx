/**
 * SquadLayout — layout partagé de la section Escouade.
 *
 * Gère la sélection des coéquipiers (via data.options), les KPI cards
 * et la navigation par onglets (Synergies / Contributions).
 * Expose les données sélectionnées via SquadContext pour les onglets enfants.
 *
 * Route parente : /players/$playerSlug/squad
 * Routes enfants : /squad/synergies · /squad/contributions
 */
import { createContext, useContext, useState, useEffect } from 'react'
import { Outlet, useParams, Link, useMatchRoute } from '@tanstack/react-router'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { useTeammates } from './queries'
import { useSettings } from '@/features/settings/queries'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { PageLoader } from '@/components/ui/spinner'
import { GamertagCombobox } from '@/components/ui/GamertagCombobox'
import { SessionScopeSelector } from './SessionScopeSelector'
import { getSeriesColors } from '@/lib/accessibility'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import type {
  TeammateRow,
  TeammateKPIs,
  TeammatesQueryRequest,
  TeammatesPageResponse,
} from '@/lib/api/types'
import { CompareDrawer } from '@/features/compare/CompareDrawer'
import { useComparePrefetch } from '@/features/compare/queries'

// ─── Context ──────────────────────────────────────────────────────────────────

interface SquadContextValue {
  selectedRows: TeammateRow[]
  soloReference: TeammateKPIs | null
  pageData: TeammatesPageResponse | null
}

const SquadContext = createContext<SquadContextValue>({
  selectedRows: [],
  soloReference: null,
  pageData: null,
})

export function useSquadContext(): SquadContextValue {
  return useContext(SquadContext)
}

// ─── Constante ────────────────────────────────────────────────────────────────

const MAX_SELECTION = 3
const CHART_COLORS = getSeriesColors(3, ['narrative-dominant', 'perf-tier-3', 'divergent-pos'])

// ─── KPI Block ────────────────────────────────────────────────────────────────

interface KPICardProps {
  label: string
  value: number | null
  unit?: string
}
function KPICard({ label, value, unit = '' }: KPICardProps) {
  const display =
    value == null ? '-' : `${Number.isInteger(value) ? value : value.toFixed(2)}${unit}`
  return (
    <div className="flex flex-col gap-1 rounded-lg border p-3">
      <span className="text-xs text-muted-foreground uppercase tracking-wide">{label}</span>
      <span className="text-xl font-bold">{display}</span>
    </div>
  )
}

interface KPIBlockProps {
  title: string
  kpis: TeammateKPIs
  color?: string
}
function KPIBlock({ title, kpis, color = 'text-muted-foreground' }: KPIBlockProps) {
  const { data: fieldMappings } = useFieldMappings()
  const labelOf = (key: string, fallback: string): string =>
    fieldMappings?.fields[key]?.label ?? fallback
  const kills = labelOf('kills', 'Kills')
  return (
    <div>
      <h3 className={`text-sm font-medium mb-2 ${color}`}>{title}</h3>
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <KPICard label={labelOf('total_matches_played', 'Matchs')} value={kpis.match_count} />
        <KPICard label={labelOf('win_rate', 'Taux de victoire')} value={kpis.win_rate * 100} unit="%" />
        <KPICard label={labelOf('kdr', 'K/D')} value={kpis.kd_ratio} />
        <KPICard label={`${kills} / match`} value={kpis.kills_per_game} />
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
}
function TeammateRowItem({
  row,
  selectionIndex,
  onSelect,
  onCompare,
  onPrefetchCompare,
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
        {row.last_seen_at ? new Date(row.last_seen_at).toLocaleDateString('fr-FR') : '-'}
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
          Comparer
        </button>
      </td>
    </tr>
  )
}

export function SquadLayout() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const { filterContext, filterContextHash } = useGlobalFilterStore()
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
  const [soloSession, setSoloSession] = useState<string | null>(null)
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
    picked_solo_session_label: soloSession,
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

  if (isLoading)
    return (
      <PageLoader />
    )
  if (isError)
    return <div className="p-8 text-center text-destructive">Erreur : {String(error)}</div>
  if (!data) {
    return (
      <div className="px-6">
        <EmptyStateCard
          title="Données d'escouade indisponibles"
          description="Aucune réponse exploitable n'a été renvoyée pour cette page. Vérifie les filtres ou la disponibilité des matchs partagés."
        />
      </div>
    )
  }

  // Utiliser data.options pour le multiselect (coéquipiers fréquents)
  const availableOptions = data.options ?? []
  const teammates = data.teammates ?? []
  const solo_reference = data.solo_reference ?? null

  const selectedRows = confirmedGts
    .map((gt) => teammates.find((t) => t.gamertag === gt))
    .filter(Boolean) as TeammateRow[]

  const synergiesRoute = '/players/$playerSlug/squad/synergies' as const
  const contributionsRoute = '/players/$playerSlug/squad/contributions' as const
  const isSynergies = !!matchRoute({ to: synergiesRoute, fuzzy: true })
  const isContributions = !!matchRoute({ to: contributionsRoute, fuzzy: true })

  return (
    <SquadContext.Provider value={{ selectedRows, soloReference: solo_reference, pageData: data ?? null }}>
      <div className="flex flex-col gap-6">
        {/* Sélecteur de session (solo / escouade) */}
        <SessionScopeSelector
          sessionLabels={data.session_labels ?? { solo: [], squad: [] }}
          soloSession={soloSession}
          squadSession={squadSession}
          onSoloChange={setSoloSession}
          onSquadChange={setSquadSession}
        />

        {/* Sélecteur coéquipiers avec fuzzy search */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Coéquipiers comparés</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <GamertagCombobox
              selected={selectedGts}
              onChange={setSelectedGts}
              max={MAX_SELECTION}
              frequentOptions={availableOptions}
              colors={CHART_COLORS}
              excludeGamertag={playerSlug}
              placeholder={`Rechercher parmi ${availableOptions.length} coéquipiers…`}
            />
            <div className="flex items-center justify-between gap-3 pt-1">
              <span className="text-xs text-muted-foreground">
                {isDirty
                  ? 'Sélection modifiée — clique sur Appliquer pour mettre à jour les graphiques.'
                  : selectedGts.length === 0
                    ? 'Sélectionne jusqu\'à 3 coéquipiers puis clique sur Appliquer.'
                    : `${confirmedGts.length} coéquipier${confirmedGts.length > 1 ? 's' : ''} comparé${confirmedGts.length > 1 ? 's' : ''}.`}
              </span>
              <button
                onClick={handleConfirm}
                disabled={selectedGts.length === 0 && confirmedGts.length === 0}
                className="shrink-0 rounded-md bg-primary px-4 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                Appliquer
              </button>
            </div>
          </CardContent>
        </Card>

        {/* KPI block si coéquipier(s) sélectionné(s) */}
        {selectedRows.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>Stats avec les coéquipiers sélectionnés</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              {selectedRows.map((row, i) => (
                <KPIBlock
                  key={row.gamertag}
                  title={`Avec ${row.gamertag}`}
                  kpis={row.with_kpis}
                  color={`text-[${CHART_COLORS[i % CHART_COLORS.length]}]`}
                />
              ))}
              {solo_reference && (
                <KPIBlock title="Référence solo" kpis={solo_reference} color="text-info" />
              )}
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
              Synergies
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
              Contributions
            </Link>
          </nav>
        </div>

        {/* Contenu de l'onglet actif */}
        <Outlet />

        {/* Table coéquipiers (secondaire, repliable si sélection active) */}
        {teammates.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>Tous les coéquipiers</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead className="bg-muted border-b">
                    <tr>
                      <th className="px-4 py-3 text-left">Gamertag</th>
                      <th className="px-4 py-3 text-center">Matchs</th>
                      <th className="px-4 py-3 text-center">Victoires</th>
                      <th className="px-4 py-3 text-center">Win%</th>
                      <th className="px-4 py-3 text-center">K/D</th>
                      <th className="px-4 py-3 text-center">Dernière rencontre</th>
                      <th className="px-4 py-3 text-center">Actions</th>
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
