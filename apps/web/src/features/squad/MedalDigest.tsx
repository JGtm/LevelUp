/**
 * MedalDigest — résumé narratif médailles par joueur sur les matchs partagés.
 *
 * Rendu : une carte par joueur avec top 5 médailles en chips + stats footer +
 * grille déroulante de toutes les médailles.
 *
 * Chaque chip affiche :
 *  - Icône médaille (img si image_url, sinon pill avec initial du label)
 *  - Badge count (toujours opaque/neutre pour accessibilité daltonienne)
 *  - Tooltip natif HTML : "Name: description"
 *
 * Couleurs de bordure par joueur alignées sur SQUAD_MAIN_PLAYER_TOKEN +
 * SQUAD_TEAMMATE_COLOR_TOKENS (tokenCssVar — pas de hex en JSX).
 */
import { useState } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { tokenCssVar } from '@/lib/accessibility'
import type { MedalDigestEntry, MedalDigestItem } from '@/lib/api/types'
import {
  SQUAD_MAIN_PLAYER_TOKEN,
  SQUAD_TEAMMATE_COLOR_TOKENS,
} from './colors'
import type { SquadText } from './i18n'

interface MedalDigestProps {
  entries: MedalDigestEntry[]
  mainPlayer: string
  t: SquadText['medals']
}

function playerColorVar(mainPlayer: string, player: string, allPlayers: string[]): string {
  if (player === mainPlayer) return tokenCssVar(SQUAD_MAIN_PLAYER_TOKEN)
  const idx = allPlayers.filter((p) => p !== mainPlayer).indexOf(player)
  const token = SQUAD_TEAMMATE_COLOR_TOKENS[idx] ?? SQUAD_TEAMMATE_COLOR_TOKENS[0]
  return tokenCssVar(token)
}

function medalTooltip(item: MedalDigestItem): string {
  const name = item.label || `#${item.medal_id}`
  return item.description ? `${name}: ${item.description}` : name
}

function MedalChip({ item }: { item: MedalDigestItem }) {
  const tip = medalTooltip(item)
  return (
    <span
      className="relative inline-flex items-center gap-1 rounded-full border border-border bg-muted px-2 py-0.5 text-xs"
      title={tip}
    >
      {item.image_url ? (
        <img
          src={item.image_url}
          alt={item.label || String(item.medal_id)}
          className="h-5 w-5 object-contain"
        />
      ) : (
        <span className="h-5 w-5 flex items-center justify-center rounded-full bg-muted-foreground/20 text-[10px] font-bold uppercase">
          {(item.label ?? '?').charAt(0)}
        </span>
      )}
      <span className="text-foreground/80 max-w-[7rem] truncate">{item.label || `#${item.medal_id}`}</span>
      <span
        className="rounded-full px-1 text-[10px] font-bold leading-tight"
        style={{
          background: tokenCssVar('muted'),
          color: tokenCssVar('foreground'),
          border: `1.5px solid ${tokenCssVar('background')}`,
          minWidth: '1.1rem',
          textAlign: 'center',
        }}
      >
        {item.total_count}
      </span>
    </span>
  )
}

function MedalIconTile({ item }: { item: MedalDigestItem }) {
  const tip = medalTooltip(item)
  return (
    <span className="relative flex flex-col items-center gap-0.5" title={tip}>
      {item.image_url ? (
        <img
          src={item.image_url}
          alt={item.label || String(item.medal_id)}
          className="h-10 w-10 object-contain"
        />
      ) : (
        <span className="h-10 w-10 flex items-center justify-center rounded-full bg-muted text-xs font-bold uppercase">
          {(item.label ?? '?').charAt(0)}
        </span>
      )}
      <span
        className="rounded-full px-1 text-[10px] font-bold leading-tight"
        style={{
          background: tokenCssVar('muted'),
          color: tokenCssVar('foreground'),
          border: `1.5px solid ${tokenCssVar('background')}`,
          minWidth: '1.1rem',
          textAlign: 'center',
        }}
      >
        {item.total_count}
      </span>
    </span>
  )
}

function PlayerMedalCard({
  entry,
  color,
  t,
}: {
  entry: MedalDigestEntry
  color: string
  t: SquadText['medals']
}) {
  const [expanded, setExpanded] = useState(false)
  return (
    <div
      className="flex flex-col gap-3 rounded-lg border border-border bg-card p-4"
      style={{ borderLeft: `4px solid ${color}` }}
    >
      <div className="flex items-center gap-2">
        <span className="h-2.5 w-2.5 rounded-full flex-shrink-0" style={{ background: color }} />
        <span className="font-semibold text-sm text-foreground truncate">{entry.player}</span>
      </div>

      {entry.top_medals.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {entry.top_medals.map((m) => (
            <MedalChip key={m.medal_id} item={m} />
          ))}
        </div>
      )}

      <div className="flex gap-4 text-xs text-muted-foreground border-t border-border pt-2">
        <span>
          <span className="font-medium text-foreground">{entry.distinct_types}</span>{' '}
          {t.statsDistinct}
        </span>
        <span>
          <span className="font-medium text-foreground">{entry.avg_per_match.toFixed(1)}</span>{' '}
          {t.statsAvg}
        </span>
        <span>
          <span className="font-medium text-foreground">{entry.peak_in_match}</span> {t.statsPeak}
        </span>
      </div>

      {entry.all_medals.length > entry.top_medals.length && (
        <>
          <button
            type="button"
            className="text-xs text-muted-foreground hover:text-foreground underline-offset-2 hover:underline text-left"
            onClick={() => setExpanded((e) => !e)}
          >
            {expanded ? t.collapseLabel : `${t.expandLabel} (${entry.all_medals.length})`}
          </button>
          {expanded && (
            <div className="flex flex-wrap gap-3 pt-1">
              {entry.all_medals.map((m) => (
                <MedalIconTile key={m.medal_id} item={m} />
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}

export function MedalDigest({ entries, mainPlayer, t }: MedalDigestProps) {
  if (!entries || entries.length === 0) {
    return null
  }

  const allPlayers = entries.map((e) => e.player)

  return (
    <Card>
      <CardContent className="pt-4 space-y-4">
        <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {t.title}
        </p>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))',
            gap: '1rem',
          }}
        >
          {entries.map((entry) => (
            <PlayerMedalCard
              key={entry.player}
              entry={entry}
              color={playerColorVar(mainPlayer, entry.player, allPlayers)}
              t={t}
            />
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
