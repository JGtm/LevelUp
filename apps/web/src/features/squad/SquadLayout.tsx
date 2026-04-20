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
import { PageHeader } from '@/components/shell/PageHeader'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import { SessionScopeSelector } from './SessionScopeSelector'
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
const CHART_COLORS = ['#7C3AED', '#F59E0B', '#10B981']

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
  return (
    <div>
      <h3 className={`text-sm font-medium mb-2 ${color}`}>{title}</h3>
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <KPICard label="Matchs" value={kpis.match_count} />
        <KPICard label="Taux de victoire" value={kpis.win_rate * 100} unit="%" />
        <KPICard label="K/D" value={kpis.kd_ratio} />
        <KPICard label="Kills / match" value={kpis.kills_per_game} />
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
  const [selectedGts, setSelectedGts] = useState<string[]>([])
  const [compareTarget, setCompareTarget] = useState<string | null>(null)
  const [soloSession, setSoloSession] = useState<string | null>(null)
  const [squadSession, setSquadSession] = useState<string | null>(null)
  const prefetchCompare = useComparePrefetch(playerSlug)
  const matchRoute = useMatchRoute()
  const { data: settings } = useSettings()

  // Init depuis les amis par défaut dans les settings (une seule fois)
  useEffect(() => {
    if (settings?.friend_gamertags?.length && selectedGts.length === 0) {
      setSelectedGts(settings.friend_gamertags.slice(0, MAX_SELECTION))
    }
  }, [settings?.friend_gamertags])

  const request: TeammatesQueryRequest = {
    filters: filterContext,
    selected_gamertags: selectedGts.length > 0 ? selectedGts : undefined,
    picked_solo_session_label: soloSession,
    picked_squad_session_label: squadSession,
  }
  const { data, isLoading, isError, error } = useTeammates(
    playerSlug,
    request,
    filterContextHash,
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
      <div className="flex items-center justify-center min-h-64">
        <Spinner size="lg" />
      </div>
    )
  if (isError)
    return <div className="p-8 text-center text-destructive">Erreur : {String(error)}</div>
  if (!data) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader title="Escouade" subtitle="Analyse des coéquipiers et des synergies" />
        <div className="px-6">
          <EmptyStateCard
            title="Données d'escouade indisponibles"
            description="Aucune réponse exploitable n'a été renvoyée pour cette page. Vérifie les filtres ou la disponibilité des matchs partagés."
          />
        </div>
      </div>
    )
  }

  // Utiliser data.options pour le multiselect (coéquipiers fréquents)
  const availableOptions = data.options ?? []
  const teammates = data.teammates ?? []
  const solo_reference = data.solo_reference ?? null

  const selectedRows = selectedGts
    .map((gt) => teammates.find((t) => t.gamertag === gt))
    .filter(Boolean) as TeammateRow[]

  const synergiesRoute = '/players/$playerSlug/squad/synergies' as const
  const contributionsRoute = '/players/$playerSlug/squad/contributions' as const
  const isSynergies = !!matchRoute({ to: synergiesRoute, fuzzy: true })
  const isContributions = !!matchRoute({ to: contributionsRoute, fuzzy: true })

  return (
    <SquadContext.Provider value={{ selectedRows, soloReference: solo_reference, pageData: data ?? null }}>
      <div className="flex flex-col gap-6">
        <PageHeader
          title="Escouade"
          subtitle={`${availableOptions.length} coéquipiers fréquents · ${data.total_matches} matchs`}
        />

        {/* Sélecteur de session (solo / escouade) */}
        <SessionScopeSelector
          sessionLabels={data.session_labels ?? { solo: [], squad: [] }}
          soloSession={soloSession}
          squadSession={squadSession}
          onSoloChange={setSoloSession}
          onSquadChange={setSquadSession}
        />

        {/* Multiselect coéquipiers depuis les options */}
        {availableOptions.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center justify-between text-base">
                <span>Sélectionner des coéquipiers</span>
                <span className="text-sm font-normal text-muted-foreground">
                  {selectedGts.length}/{MAX_SELECTION}
                  {selectedGts.length > 0 && (
                    <button
                      className="ml-2 text-xs text-muted-foreground hover:text-foreground"
                      onClick={() => setSelectedGts([])}
                    >
                      ✕ Effacer
                    </button>
                  )}
                </span>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap gap-2">
                {availableOptions.map((opt) => {
                  const idx = selectedGts.indexOf(opt.gamertag)
                  const isSelected = idx >= 0
                  const color = isSelected ? CHART_COLORS[idx % CHART_COLORS.length] : undefined
                  return (
                    <button
                      key={opt.xuid ?? opt.gamertag}
                      onClick={() => toggleSelect(opt.gamertag)}
                      disabled={!isSelected && selectedGts.length >= MAX_SELECTION}
                      className={`flex items-center gap-1.5 rounded-full border px-3 py-1 text-sm transition-colors
                        ${isSelected
                          ? 'border-transparent text-white'
                          : 'border-border text-muted-foreground hover:border-primary hover:text-foreground disabled:opacity-40'
                        }`}
                      style={isSelected ? { backgroundColor: color } : undefined}
                    >
                      {isSelected && <span className="text-xs">✓</span>}
                      {opt.gamertag}
                      <span className="text-xs opacity-70">({opt.encounter_count})</span>
                    </button>
                  )
                })}
              </div>
            </CardContent>
          </Card>
        )}

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
