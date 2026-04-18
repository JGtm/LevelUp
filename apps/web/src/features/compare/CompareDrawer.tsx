/**
 * CompareDrawer — tiroir latéral de comparaison joueur vs joueur.
 * Sprint 54-C.
 *
 * Usage :
 *   <CompareDrawer playerSlug={slug} open={open} onClose={() => setOpen(false)} />
 */
import { useState } from 'react'
import { useCompare } from './queries'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { EmptyStateCard } from '@/components/ui/empty-state'
import type { CompareMetricRow, NormalizedPlayerStats } from '@/lib/api/types'

interface CompareDrawerProps {
  playerSlug: string
  open: boolean
  onClose: () => void
}

/** Carte de stats pour un joueur dans la comparaison. */
function PlayerStatsCard({
  stats,
  side,
}: {
  stats: NormalizedPlayerStats
  side: 'A' | 'B'
}) {
  const colorClass = side === 'A' ? 'text-blue-700' : 'text-orange-700'
  const bgClass =
    side === 'A' ? 'bg-blue-50 border-blue-200' : 'bg-orange-50 border-orange-200'

  return (
    <Card className={`border ${bgClass}`}>
      <CardHeader className="pb-2">
        <CardTitle className={`text-sm ${colorClass}`}>
          Joueur {side} — {stats.gamertag}
          {stats.is_local && (
            <Badge variant="secondary" className="ml-2 text-xs">
              Local
            </Badge>
          )}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-1 text-xs text-gray-700">
        <p>
          <span className="font-medium">Matchs :</span> {stats.matches_played}
        </p>
        <p>
          <span className="font-medium">Victoires :</span>{' '}
          {(stats.win_rate * 100).toFixed(1)} %
        </p>
        <p>
          <span className="font-medium">K/D :</span> {stats.kd_ratio.toFixed(2)}
        </p>
        <p>
          <span className="font-medium">Précision :</span>{' '}
          {(stats.accuracy * 100).toFixed(1)} %
        </p>
        <p>
          <span className="font-medium">Kills/match :</span>{' '}
          {stats.avg_kills.toFixed(1)}
        </p>
      </CardContent>
    </Card>
  )
}

/** Ligne de métrique dans le tableau comparatif. */
function MetricRowComp({ row }: { row: CompareMetricRow }) {
  const winnerA = row.winner === 'a'
  const winnerB = row.winner === 'b'

  return (
    <tr className="border-b last:border-0 text-sm">
      <td className={`py-2 pr-4 ${winnerA ? 'font-semibold text-blue-700' : 'text-gray-700'}`}>
        {String(row.value_a)}
      </td>
      <td className="py-2 px-2 text-gray-500 text-center text-xs">{row.label_fr}</td>
      <td className={`py-2 pl-4 text-right ${winnerB ? 'font-semibold text-orange-700' : 'text-gray-700'}`}>
        {String(row.value_b)}
      </td>
    </tr>
  )
}

/** Tiroir principal de comparaison. */
export function CompareDrawer({ playerSlug, open, onClose }: CompareDrawerProps) {
  const [gamertagB, setGamertagB] = useState('')
  const { mutate, data, isPending, isError, error, reset } = useCompare(playerSlug)

  if (!open) return null

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!gamertagB.trim()) return
    reset()
    mutate({ gamertag_b: gamertagB.trim() })
  }

  return (
    <>
      {/* Overlay */}
      <div
        className="fixed inset-0 bg-black/40 z-40"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Tiroir */}
      <aside className="fixed right-0 top-0 h-full w-full max-w-lg bg-white shadow-xl z-50 flex flex-col overflow-y-auto">
        {/* En-tête */}
        <div className="flex items-center justify-between p-4 border-b">
          <h2 className="text-lg font-semibold">Comparer avec un joueur</h2>
          <button
            onClick={onClose}
            className="text-gray-500 hover:text-gray-700 text-xl font-bold"
            aria-label="Fermer"
          >
            ✕
          </button>
        </div>

        {/* Formulaire de recherche */}
        <form onSubmit={handleSubmit} className="p-4 border-b space-y-2">
          <label className="block text-sm font-medium text-gray-700">
            Gamertag du joueur à comparer
          </label>
          <div className="flex gap-2">
            <input
              type="text"
              value={gamertagB}
              onChange={(e) => setGamertagB(e.target.value)}
              placeholder="Ex : HaloPlayer123"
              className="flex-1 border rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
            />
            <button
              type="submit"
              disabled={isPending || !gamertagB.trim()}
              className="px-4 py-2 bg-blue-600 text-white rounded text-sm font-medium hover:bg-blue-700 disabled:opacity-50"
            >
              {isPending ? '…' : 'Comparer'}
            </button>
          </div>
        </form>

        {/* Corps */}
        <div className="flex-1 p-4 space-y-4">
          {isPending && (
            <div className="flex justify-center py-8">
              <Spinner size="lg" />
            </div>
          )}

          {isError && (
            <EmptyStateCard
              title="Erreur"
              message={error?.message ?? 'Impossible de récupérer la comparaison.'}
            />
          )}

          {data && (
            <>
              {/* Cartes des deux joueurs */}
              <div className="grid grid-cols-2 gap-3">
                <PlayerStatsCard stats={data.player_a} side="A" />
                <PlayerStatsCard stats={data.player_b} side="B" />
              </div>

              {/* Tableau des métriques */}
              {data.metrics.length > 0 && (
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm">Comparaison détaillée</CardTitle>
                  </CardHeader>
                  <CardContent className="p-0">
                    <table className="w-full">
                      <thead>
                        <tr className="text-xs text-gray-500 border-b">
                          <th className="py-2 pr-4 text-left font-medium text-blue-600">
                            {data.player_a.gamertag}
                          </th>
                          <th className="py-2 px-2 text-center font-medium">Métrique</th>
                          <th className="py-2 pl-4 text-right font-medium text-orange-600">
                            {data.player_b.gamertag}
                          </th>
                        </tr>
                      </thead>
                      <tbody>
                        {data.metrics.map((row) => (
                          <MetricRowComp key={row.metric} row={row} />
                        ))}
                      </tbody>
                    </table>
                  </CardContent>
                </Card>
              )}
            </>
          )}

          {!isPending && !isError && !data && (
            <p className="text-sm text-gray-500 text-center py-8">
              Entrez un gamertag pour lancer la comparaison.
            </p>
          )}
        </div>
      </aside>
    </>
  )
}
