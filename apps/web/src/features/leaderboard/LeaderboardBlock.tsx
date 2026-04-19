/**
 * LeaderboardBlock — classement CSR des joueurs locaux + Waypoint.
 * Sprint 54-E.
 *
 * Usage :
 *   <LeaderboardBlock playerSlug={slug} />
 */
import { useState } from 'react'
import { useLeaderboard } from './queries'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { EmptyStateCard } from '@/components/ui/empty-state'
import type { LeaderboardEntry } from '@/lib/api/types'

interface LeaderboardBlockProps {
  playerSlug: string
  defaultSeason?: string
  defaultPlaylist?: string
  /** E2.5 : callback appelé onMouseEnter pour prefetch compare */
  onHoverEntry?: (gamertag: string) => void
}

/** Ligne du classement. */
function LeaderboardRow({ entry, onHover }: { entry: LeaderboardEntry; onHover?: (gamertag: string) => void }) {
  return (
    <tr
      className="border-b last:border-0 text-sm hover:bg-gray-50 transition-colors"
      onMouseEnter={() => onHover?.(entry.gamertag)}
    >
      <td className="py-2 pr-4 text-center font-mono text-gray-500">
        {entry.rank}
      </td>
      <td className="py-2 pr-4 font-medium text-gray-800">
        {entry.gamertag}
        {entry.is_local && (
          <Badge variant="secondary" className="ml-2 text-xs">
            Local
          </Badge>
        )}
      </td>
      <td className="py-2 pr-4 text-center">
        <span className="inline-block px-2 py-0.5 rounded bg-indigo-100 text-indigo-800 text-xs font-semibold">
          {entry.tier}
          {entry.sub_tier > 0 ? ` ${entry.sub_tier}` : ''}
        </span>
      </td>
      <td className="py-2 text-right font-mono text-gray-700">
        {entry.csr_value.toLocaleString('fr-FR')}
      </td>
    </tr>
  )
}

/** Bloc complet du classement. */
export function LeaderboardBlock({
  playerSlug,
  defaultSeason,
  defaultPlaylist,
  onHoverEntry,
}: LeaderboardBlockProps) {
  const [season, setSeason] = useState(defaultSeason ?? '')
  const [playlist, setPlaylist] = useState(defaultPlaylist ?? '')

  const { data, isLoading, isError, error } = useLeaderboard(
    playerSlug,
    season || undefined,
    playlist || undefined,
  )

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">Classement CSR</CardTitle>
          <div className="flex gap-2 text-xs">
            <input
              type="text"
              value={season}
              onChange={(e) => setSeason(e.target.value)}
              placeholder="Saison (ex: Season5)"
              className="border rounded px-2 py-1 w-28 focus:outline-none focus:ring-1 focus:ring-indigo-400"
            />
            <input
              type="text"
              value={playlist}
              onChange={(e) => setPlaylist(e.target.value)}
              placeholder="Playlist (optionnel)"
              className="border rounded px-2 py-1 w-32 focus:outline-none focus:ring-1 focus:ring-indigo-400"
            />
          </div>
        </div>
        {data && (
          <p className="text-xs text-gray-500 mt-1">
            {data.total} joueur{data.total > 1 ? 's' : ''} — Saison :{' '}
            <span className="font-medium">{data.season_id || '—'}</span>
          </p>
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
              title="Erreur"
              description={error?.message ?? 'Impossible de charger le classement.'}
            />
          </div>
        )}

        {data && data.entries.length === 0 && (
          <div className="p-4">
            <EmptyStateCard
              title="Classement vide"
              description="Aucun joueur trouvé pour cette saison/playlist."
            />
          </div>
        )}

        {data && data.entries.length > 0 && (
          <table className="w-full">
            <thead>
              <tr className="text-xs text-gray-500 border-b bg-gray-50">
                <th className="py-2 pr-4 text-center font-medium w-12">#</th>
                <th className="py-2 pr-4 text-left font-medium">Joueur</th>
                <th className="py-2 pr-4 text-center font-medium">Rang</th>
                <th className="py-2 text-right font-medium">CSR</th>
              </tr>
            </thead>
            <tbody>
              {data.entries.map((entry) => (
                <LeaderboardRow key={`${entry.xuid}-${entry.rank}`} entry={entry} onHover={onHoverEntry} />
              ))}
            </tbody>
          </table>
        )}
      </CardContent>
    </Card>
  )
}
