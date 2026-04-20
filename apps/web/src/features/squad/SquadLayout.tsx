/**
 * SquadLayout — layout partagé de la section Escouade.
 *
 * Gère la sélection des coéquipiers, le tableau, les KPIs et le drawer Comparer.
 * Expose les données sélectionnées via SquadContext pour les onglets enfants
 * (SquadSynergiesPage, SquadContributionsPage).
 *
 * Route parente : /players/$playerSlug/squad
 * Routes enfants : /squad/synergies · /squad/contributions
 */
import { createContext, useContext, useState } from 'react'
import { Outlet, useParams } from '@tanstack/react-router'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { useTeammates } from './queries'
import { PageHeader } from '@/components/shell/PageHeader'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import type {
  TeammateRow,
  TeammateKPIs,
  TeammatesQueryRequest,
} from '@/lib/api/types'
import { CompareDrawer } from '@/features/compare/CompareDrawer'
import { useComparePrefetch } from '@/features/compare/queries'

// ─── Context ──────────────────────────────────────────────────────────────────

interface SquadContextValue {
  selectedRows: TeammateRow[]
  soloReference: TeammateKPIs | null
}

const SquadContext = createContext<SquadContextValue>({
  selectedRows: [],
  soloReference: null,
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

// ─── Layout ───────────────────────────────────────────────────────────────────

export function SquadLayout() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const { filterContext, filterContextHash } = useGlobalFilterStore()
  const [selectedGts, setSelectedGts] = useState<string[]>([])
  const [compareTarget, setCompareTarget] = useState<string | null>(null)
  const prefetchCompare = useComparePrefetch(playerSlug)

  const request: TeammatesQueryRequest = {
    filters: filterContext,
    selected_gamertags: selectedGts.length > 0 ? selectedGts : undefined,
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

  const { teammates, solo_reference } = data
  const selectedRows = selectedGts
    .map((gt) => teammates.find((t) => t.gamertag === gt))
    .filter(Boolean) as TeammateRow[]

  return (
    <SquadContext.Provider value={{ selectedRows, soloReference: solo_reference }}>
      <div className="flex flex-col gap-6">
        <PageHeader
          title="Escouade"
          subtitle={`${teammates.length} coéquipiers · ${data.total_matches} matchs`}
        />

        {/* KPI rapides si coéquipier(s) sélectionné(s) */}
        <Card>
          <CardHeader>
            <CardTitle>Stats avec les coéquipiers sélectionnés</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            {selectedRows.length > 0 ? (
              <>
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
              </>
            ) : (
              <EmptyStateNotice
                title="Aucune sélection"
                description="Sélectionne jusqu'à 3 coéquipiers dans le tableau pour activer la comparaison."
              />
            )}
          </CardContent>
        </Card>

        {/* Contenu de l'onglet actif (Synergies ou Contributions) */}
        <Outlet />

        {/* Table principale coéquipiers */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              <span>Coéquipiers</span>
              <div className="flex items-center gap-3">
                <span className="text-sm font-normal text-muted-foreground">
                  {selectedGts.length}/{MAX_SELECTION} sélectionnés
                </span>
                {selectedGts.length > 0 && (
                  <button
                    className="text-xs text-muted-foreground hover:text-foreground"
                    onClick={() => setSelectedGts([])}
                  >
                    ✕ Effacer
                  </button>
                )}
              </div>
            </CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {teammates.length === 0 ? (
              <p className="p-6 text-center text-muted-foreground">
                Aucun coéquipier trouvé pour cette période.
              </p>
            ) : (
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
            )}
          </CardContent>
        </Card>

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
