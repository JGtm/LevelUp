/**
 * LeaderboardBlock — classement CSR des joueurs locaux + Waypoint.
 * Sprint 54-E.
 *
 * Usage :
 *   <LeaderboardBlock playerSlug={slug} />
 */
import { useState } from 'react'

const SUB_TIER_ROMAN = ['', 'I', 'II', 'III', 'IV', 'V', 'VI']
const toRoman = (n: number): string => SUB_TIER_ROMAN[n] ?? String(n)
import { useNavigate } from '@tanstack/react-router'
import { useLeaderboard } from './queries'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { EmptyStateCard } from '@/components/ui/empty-state'
import type { LeaderboardEntry } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

interface LeaderboardBlockProps {
  playerSlug: string
  defaultSeason?: string
  defaultPlaylist?: string
  /** E2.5 : callback appelé onMouseEnter pour prefetch compare */
  onHoverEntry?: (gamertag: string) => void
}

/** Ligne du classement. */
function LeaderboardRow({
  playerSlug,
  entry,
  onHover,
}: {
  playerSlug: string
  entry: LeaderboardEntry
  onHover?: (gamertag: string) => void
}) {
  const navigate = useNavigate()

  function goToExplorer() {
    void navigate({
      to: '/players/$playerSlug/explorer',
      params: { playerSlug },
      search: { mode: 'player', target: entry.gamertag },
    })
  }

  return (
    <tr
      className="border-b last:border-0 text-sm hover:bg-muted transition-colors"
      onMouseEnter={() => onHover?.(entry.gamertag)}
    >
      <td className="py-2 pr-4 text-center font-mono text-muted-foreground">
        {entry.rank}
      </td>
      <td className="py-2 pr-4 font-medium text-foreground">
        {entry.is_local ? (
          <span>
            {entry.gamertag}
            <Badge variant="secondary" className="ml-2 text-xs">Local</Badge>
          </span>
        ) : (
          <button
            type="button"
            className="hover:text-primary hover:underline transition-colors"
            onClick={goToExplorer}
            title={`Voir l'historique avec ${entry.gamertag}`}
          >
            {entry.gamertag}
          </button>
        )}
      </td>
      <td className="py-2 pr-4 text-center">
        <span className="inline-block px-2 py-0.5 rounded bg-accent text-accent-foreground text-xs font-semibold">
          {entry.tier}
          {entry.sub_tier > 0 ? ` ${toRoman(entry.sub_tier)}` : ''}
        </span>
      </td>
      <td className="py-2 text-right font-mono text-foreground">
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
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

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
              placeholder={t('common.leaderboard.season_placeholder')}
              className="border rounded px-2 py-1 w-28 focus:outline-none focus:ring-1 focus:ring-ring"
            />
            <input
              type="text"
              value={playlist}
              onChange={(e) => setPlaylist(e.target.value)}
              placeholder={t('common.leaderboard.playlist_placeholder')}
              className="border rounded px-2 py-1 w-32 focus:outline-none focus:ring-1 focus:ring-ring"
            />
          </div>
        </div>
        {data && (
          <p className="text-xs text-muted-foreground mt-1">
            {data.total} joueur{data.total > 1 ? 's' : ''} {t('common.leaderboard.season_prefix')}{' '}
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
              title={t('common.leaderboard.empty_title')}
              description={t('common.leaderboard.no_match_in_window')}
            />
          </div>
        )}

        {data && data.entries.length > 0 && (
          <table className="w-full">
            <thead>
              <tr className="text-xs text-muted-foreground border-b bg-muted">
                <th className="py-2 pr-4 text-center font-medium w-12">#</th>
                <th className="py-2 pr-4 text-left font-medium">Joueur</th>
                <th className="py-2 pr-4 text-center font-medium">Rang</th>
                <th className="py-2 text-right font-medium">CSR</th>
              </tr>
            </thead>
            <tbody>
              {data.entries.map((entry) => (
                <LeaderboardRow key={`${entry.xuid}-${entry.rank}`} playerSlug={playerSlug} entry={entry} onHover={onHoverEntry} />
              ))}
            </tbody>
          </table>
        )}
      </CardContent>
    </Card>
  )
}
