/**
 * MedalDigest — résumé narratif médailles par joueur sur les matchs partagés.
 *
 * Rendu : une carte par joueur avec top 5 médailles en chips + stats footer +
 * grille déroulante de toutes les médailles.
 *
 * Avatar joueur : emblème Spartan lu depuis le cache TanStack Query
 * (queryKeys.home — zéro requête supplémentaire). Fallback : initiale dans
 * un cercle à la couleur du joueur. L'emblème est toujours disponible pour
 * le joueur principal (home page déjà en cache) ; pour les coéquipiers il
 * apparaît si leur home a déjà été visitée.
 *
 * Grille adaptative : 2 joueurs → 2 colonnes 50/50 ; 3-4 → auto-fill 240px.
 */
import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent } from '@/components/ui/card'
import { tokenCssVar } from '@/lib/accessibility'
import { dropShadowForDifficulty, boxShadowForDifficulty } from '@/lib/medalDifficulty'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { HomePageResponse, MedalDigestEntry, MedalDigestItem } from '@/lib/api/types'
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

/** Emblème rond du joueur — lit le cache home sans déclencher de requête. */
function PlayerAvatar({ gamertag, color }: { gamertag: string; color: string }) {
  const { data: emblemUrl } = useQuery({
    queryKey: queryKeys.home(gamertag),
    queryFn: () => api.get<HomePageResponse>(`/players/${gamertag}/pages/home`),
    select: (d: HomePageResponse) => d.spartan_identity?.emblem_image_url ?? null,
    enabled: false,
    staleTime: Infinity,
  })

  if (emblemUrl) {
    return (
      <img
        src={emblemUrl}
        alt={gamertag}
        className="h-8 w-8 rounded-full object-cover flex-shrink-0"
        style={{ boxShadow: `0 0 0 2px ${color}` }}
      />
    )
  }

  return (
    <span
      className="h-8 w-8 rounded-full flex items-center justify-center flex-shrink-0 text-xs font-bold"
      style={{ background: color, color: '#fff' }} // color-allow: blanc structurel sur fond joueur
    >
      {gamertag.charAt(0).toUpperCase()}
    </span>
  )
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
          background: 'var(--muted)', // color-allow: structurel — badge count neutre
          color: 'var(--foreground)',
          border: '1.5px solid var(--background)', // color-allow: structurel — séparation badge/icône
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
  const dropGlow = dropShadowForDifficulty(item.difficulty)
  const boxGlow = boxShadowForDifficulty(item.difficulty)
  return (
    <span className="relative flex flex-col items-center gap-0.5" title={tip}>
      {item.image_url ? (
        <img
          src={item.image_url}
          alt={item.label || String(item.medal_id)}
          className="h-10 w-10 object-contain"
          style={dropGlow ? { filter: dropGlow } : undefined}
        />
      ) : (
        <span
          className="h-10 w-10 flex items-center justify-center rounded-full bg-muted text-xs font-bold uppercase"
          style={boxGlow ? { boxShadow: boxGlow } : undefined}
        >
          {(item.label ?? '?').charAt(0)}
        </span>
      )}
      <span
        className="rounded-full px-1 text-[10px] font-bold leading-tight"
        style={{
          background: 'var(--muted)', // color-allow: structurel — badge count neutre
          color: 'var(--foreground)',
          border: '1.5px solid var(--background)', // color-allow: structurel — séparation badge/icône
          minWidth: '1.1rem',
          textAlign: 'center',
        }}
      >
        {item.total_count}
      </span>
    </span>
  )
}

const CATEGORY_ORDER = ['multikill', 'spree', 'skill', 'style', 'mode', 'proficiency']

function MedalExpandedGrid({
  medals,
  categoryLabels,
}: {
  medals: MedalDigestItem[]
  categoryLabels: SquadText['medals']['categoryLabels']
}) {
  const groups = new Map<string, MedalDigestItem[]>()
  for (const m of medals) {
    const key = m.category || 'other'
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key)!.push(m)
  }
  const orderedKeys = [
    ...CATEGORY_ORDER.filter((k) => groups.has(k)),
    ...[...groups.keys()].filter((k) => !CATEGORY_ORDER.includes(k)),
  ]
  return (
    <div className="pt-1 space-y-3">
      {orderedKeys.map((cat) => (
        <div key={cat}>
          <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
            {categoryLabels[cat as keyof typeof categoryLabels] ?? cat}
          </p>
          <div className="flex flex-wrap gap-3">
            {groups.get(cat)!.map((m) => (
              <MedalIconTile key={m.medal_id} item={m} />
            ))}
          </div>
        </div>
      ))}
    </div>
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
        <PlayerAvatar gamertag={entry.player} color={color} />
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
            <MedalExpandedGrid medals={entry.all_medals} categoryLabels={t.categoryLabels} />
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

  // 2 joueurs → 2 colonnes 50/50 pour occuper tout l'espace.
  // 3-4 joueurs → auto-fill 240px (3 colonnes sur écran large, 2×2 sur moyen).
  const gridCols =
    entries.length <= 2
      ? `repeat(${entries.length}, 1fr)`
      : 'repeat(auto-fill, minmax(240px, 1fr))'

  return (
    <Card>
      <CardContent className="pt-4 space-y-4">
        <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {t.title}
        </p>
        <div style={{ display: 'grid', gridTemplateColumns: gridCols, gap: '1rem' }}>
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
