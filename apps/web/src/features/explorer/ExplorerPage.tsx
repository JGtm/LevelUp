/**
 * ExplorerPage — page Explorer (recherche + filtres).
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { PageHeader } from '@/components/shell/PageHeader'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { GamertagSearchInput } from './GamertagSearchInput'
import { useExplorerMatches, useExplorerPlayer } from './queries'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { CompareDrawer } from '@/features/compare/CompareDrawer'
import { useComparePrefetch } from '@/features/compare/queries'
import type { ExplorerMatchFilters } from '@/lib/api/types'

type SearchMode = 'matches' | 'player'

export function ExplorerPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const filterContextHash = useGlobalFilterStore((s) => s.filterContextHash)

  const [mode, setMode] = useState<SearchMode>('matches')
  const [targetGamertag, setTargetGamertag] = useState('')
  const [compareOpen, setCompareOpen] = useState(false)
  const prefetchCompare = useComparePrefetch(playerSlug)

  // Filtres cascade mode Matchs
  const [dateFilter, setDateFilter] = useState('')
  const [squadScope, setSquadScope] = useState<'' | 'all' | 'solo' | 'squad'>('')
  const [expType, setExpType] = useState('')
  const [playlistFilter, setPlaylistFilter] = useState('')
  const [modeFilter, setModeFilter] = useState('')
  const [mapFilter, setMapFilter] = useState('')

  const matchFilters: ExplorerMatchFilters = {
    selected_date: dateFilter || null,
    squad_scope: squadScope || undefined,
    experience_type: expType || null,
    playlist: playlistFilter || null,
    mode: modeFilter || null,
    map: mapFilter || null,
  }

  const matchesQuery = useExplorerMatches(
    playerSlug,
    { filters: filterContext, match_filters: matchFilters },
    filterContextHash,
  )

  const playerQuery = useExplorerPlayer(
    playerSlug,
    { target_gamertag: targetGamertag },
  )

  return (
    <div className="flex flex-col">
      <PageHeader title="Explorer" subtitle="Recherche de matchs et de joueurs" />

      <div className="p-6 space-y-6">
        {/* Onglets mode */}
        <div className="flex gap-2">
          <Button
            variant={mode === 'matches' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setMode('matches')}
          >
            Matchs
          </Button>
          <Button
            variant={mode === 'player' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setMode('player')}
          >
            Joueur
          </Button>
        </div>

        {/* Mode Joueur */}
        {mode === 'player' && (
          <div className="space-y-4">
            <GamertagSearchInput onSelect={setTargetGamertag} />

            {!targetGamertag && (
              <Card>
                <CardContent className="py-4">
                  <EmptyStateNotice
                    title="Aucun joueur sélectionné"
                    description="Recherche un gamertag pour afficher les matchs communs et le bilan face à ce joueur."
                  />
                </CardContent>
              </Card>
            )}

            {targetGamertag && playerQuery.isLoading && (
              <div className="flex justify-center py-8">
                <Spinner label="Recherche en cours…" />
              </div>
            )}

            {targetGamertag && playerQuery.isError && (
              <Card>
                <CardContent className="py-4">
                  <EmptyStateNotice
                    title="Recherche indisponible"
                    description="Le backend n'a pas pu renvoyer les informations de ce joueur. Réessaie avec un autre gamertag ou relance la requête."
                  />
                </CardContent>
              </Card>
            )}

            {targetGamertag && !playerQuery.isLoading && !playerQuery.isError && !playerQuery.data && (
              <Card>
                <CardContent className="py-4">
                  <EmptyStateNotice
                    title="Aucun résultat joueur"
                    description="La recherche n'a renvoyé aucune charge utile exploitable pour ce gamertag."
                  />
                </CardContent>
              </Card>
            )}

            {targetGamertag && playerQuery.data && (
              <Card>
                <CardContent className="py-4 space-y-2">
                  <div className="flex items-center justify-between">
                    <p className="font-semibold text-gray-900">{playerQuery.data.target.gamertag || targetGamertag}</p>
                    {/* C4.1 : CTA Comparer avec prefetch onMouseEnter */}
                    <Button
                      size="sm"
                      variant="outline"
                      onMouseEnter={() => prefetchCompare(playerQuery.data.target.gamertag || targetGamertag)}
                      onClick={() => setCompareOpen(true)}
                    >
                      Comparer
                    </Button>
                  </div>
                  <div className="grid grid-cols-3 gap-4 text-sm">
                    <div>
                      <p className="text-xs text-gray-500">Matchs ensemble</p>
                      <p className="font-bold text-purple-700">{playerQuery.data.summary.matches_together}</p>
                    </div>
                    <div>
                      <p className="text-xs text-gray-500">Victoires</p>
                      <p className="font-bold text-green-600">{playerQuery.data.summary.wins_together}</p>
                    </div>
                    <div>
                      <p className="text-xs text-gray-500">Défaites</p>
                      <p className="font-bold text-red-500">{playerQuery.data.summary.losses_together}</p>
                    </div>
                  </div>

                  {/* Matchs communs */}
                  <div className="mt-4">
                    <p className="mb-2 text-xs font-medium text-gray-500 uppercase tracking-wide">
                      Matchs communs récents
                    </p>
                    {playerQuery.data.common_matches.length > 0 ? (
                      <div className="space-y-1">
                        {playerQuery.data.common_matches.slice(0, 5).map((m) => (
                          <div key={m.match_id} className="flex items-center justify-between rounded-md bg-gray-50 px-3 py-1.5 text-sm">
                            <span>{m.map_ui} · {m.mode_ui}</span>
                            <Badge
                              variant={
                                m.outcome_label.toLowerCase().includes('victoire') ? 'success' :
                                m.outcome_label.toLowerCase().includes('défaite') ? 'destructive' : 'secondary'
                              }
                            >
                              {m.outcome_label}
                            </Badge>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <EmptyStateNotice
                        title="Aucun match commun récent"
                        description="Ce joueur n'a aucun match commun visible avec le scope actuel."
                      />
                    )}
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        )}

        {/* Mode Matchs */}
        {mode === 'matches' && (
          <div className="space-y-4">
            {/* Filtres cascade */}
            <Card>
              <CardContent className="py-3">
                <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-6">
                  <div>
                    <label className="block text-xs text-gray-500 mb-1">Date</label>
                    <input
                      type="date"
                      value={dateFilter}
                      onChange={(e) => setDateFilter(e.target.value)}
                      className="w-full rounded border border-gray-300 px-2 py-1 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-gray-500 mb-1">Contexte</label>
                    <select
                      value={squadScope}
                      onChange={(e) => setSquadScope(e.target.value as '' | 'all' | 'solo' | 'squad')}
                      className="w-full rounded border border-gray-300 px-2 py-1 text-sm"
                    >
                      <option value="">Tous</option>
                      <option value="solo">Solo</option>
                      <option value="squad">Escouade</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-xs text-gray-500 mb-1">Type d'expérience</label>
                    <input
                      type="text"
                      placeholder="PvP, PvE…"
                      value={expType}
                      onChange={(e) => setExpType(e.target.value)}
                      className="w-full rounded border border-gray-300 px-2 py-1 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-gray-500 mb-1">Playlist</label>
                    <input
                      type="text"
                      placeholder="Filtrer playlist…"
                      value={playlistFilter}
                      onChange={(e) => setPlaylistFilter(e.target.value)}
                      className="w-full rounded border border-gray-300 px-2 py-1 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-gray-500 mb-1">Mode</label>
                    <input
                      type="text"
                      placeholder="Filtrer mode…"
                      value={modeFilter}
                      onChange={(e) => setModeFilter(e.target.value)}
                      className="w-full rounded border border-gray-300 px-2 py-1 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-gray-500 mb-1">Carte</label>
                    <input
                      type="text"
                      placeholder="Filtrer carte…"
                      value={mapFilter}
                      onChange={(e) => setMapFilter(e.target.value)}
                      className="w-full rounded border border-gray-300 px-2 py-1 text-sm"
                    />
                  </div>
                </div>
                {(dateFilter || squadScope || expType || playlistFilter || modeFilter || mapFilter) && (
                  <div className="mt-2 flex justify-end">
                    <button
                      className="text-xs text-purple-600 hover:underline"
                      onClick={() => { setDateFilter(''); setSquadScope(''); setExpType(''); setPlaylistFilter(''); setModeFilter(''); setMapFilter('') }}
                    >
                      Réinitialiser les filtres
                    </button>
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Résultats */}
            <div className="space-y-2">
            {matchesQuery.isLoading ? (
              <div className="flex justify-center py-16">
                <Spinner label="Chargement des matchs…" />
              </div>
            ) : matchesQuery.isError ? (
              <div className="rounded-lg border border-red-100 bg-red-50 p-6 text-center">
                <p className="text-red-600">Impossible de charger les matchs.</p>
                <button onClick={() => matchesQuery.refetch()} className="mt-2 text-sm text-purple-600 underline">
                  Réessayer
                </button>
              </div>
            ) : matchesQuery.data ? (
              <>
                <p className="text-sm text-gray-500">
                  {matchesQuery.data.summary?.total_matches?.toLocaleString('fr-FR') ?? '?'} matchs trouvés
                </p>
                <div className="overflow-x-auto rounded-lg border border-gray-200 bg-white">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-gray-100 bg-gray-50 text-xs font-medium text-gray-500">
                        <th className="px-4 py-2.5 text-left">Date</th>
                        <th className="px-4 py-2.5 text-left">Carte / Mode</th>
                        <th className="px-4 py-2.5 text-left">Résultat</th>
                        <th className="px-4 py-2.5 text-left">Score</th>
                        <th className="px-4 py-2.5 text-left">Type</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-50">
                      {matchesQuery.data.table.items.map((row) => (
                        <tr key={row.match_id} className="hover:bg-purple-50/30 transition-colors">
                          <td className="px-4 py-2 text-gray-400">
                            {new Date(row.start_time).toLocaleDateString('fr-FR')}
                          </td>
                          <td className="px-4 py-2">
                            <span className="font-medium text-gray-800">{row.map_ui}</span>
                            <span className="ml-1 text-xs text-gray-400">· {row.mode_ui}</span>
                          </td>
                          <td className="px-4 py-2">
                            <Badge
                              variant={
                                row.outcome_label.toLowerCase().includes('victoire') ? 'success' :
                                row.outcome_label.toLowerCase().includes('défaite') ? 'destructive' : 'secondary'
                              }
                            >
                              {row.outcome_label}
                            </Badge>
                          </td>
                          <td className="px-4 py-2 text-gray-700">{row.score_label}</td>
                          <td className="px-4 py-2 text-gray-500">{row.experience_type_label}</td>
                        </tr>
                      ))}
                      {matchesQuery.data.table.items.length === 0 && (
                        <tr>
                          <td colSpan={5} className="px-4 py-8 text-center text-gray-400">
                            Aucun match trouvé.
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              </>
            ) : (
              <Card>
                <CardContent className="py-4">
                  <EmptyStateNotice
                    title="Résultats indisponibles"
                    description="La requête n'a renvoyé aucun tableau de matchs exploitable pour ce scope."
                  />
                </CardContent>
              </Card>
            )}
            </div>
          </div>
        )}
      </div>

      {/* C4.1 : CompareDrawer déclenché depuis la card résultats joueur */}
      <CompareDrawer
        playerSlug={playerSlug}
        open={compareOpen}
        onClose={() => setCompareOpen(false)}
      />
    </div>
  )
}
