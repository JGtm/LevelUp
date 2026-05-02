/**
 * MatchImpactBadgesBar — bandeau des badges d'impact du match.
 *
 * Affiche, au-dessus des onglets de la page match, les badges narratifs
 * calculés côté backend (analysis.ComputeMatchImpactFull) :
 *   - event-based : ⚡ Premier sang, 🪦 Première victime, 🎯 Finisseur,
 *     💀 Boulet, 🐌 Touriste, 🔫 Top Gun (timing affiché)
 *   - stat-based : 🥇 Bourreau, 🛡️ Héros silencieux, 🐍 Faux-frère
 *
 * Aligne le rendu sur la branche main (badges + horodatage). La résolution
 * xuid → gamertag se fait via le scoreboard du match.
 */
import { Badge } from '@/components/ui/badge'
import type { MatchImpactBadge, MatchScoreboardRow } from '@/lib/api/types'

interface BadgeMeta {
  icon: string
  /** order d'affichage (1 = en premier) */
  order: number
  /** style visuel — outline par défaut */
  variant?: 'outline' | 'secondary' | 'default'
}

const BADGE_META: Record<string, BadgeMeta> = {
  first_blood: { icon: '⚡', order: 1 },
  first_group_death: { icon: '🪦', order: 2 },
  clutch_finisher: { icon: '🎯', order: 3 },
  last_casualty: { icon: '💀', order: 4 },
  last_group_kill: { icon: '🐌', order: 5 },
  top_gun: { icon: '🔫', order: 6 },
  top_killer: { icon: '🥇', order: 7, variant: 'secondary' },
  silent_hero: { icon: '🛡️', order: 8, variant: 'secondary' },
  false_brother: { icon: '🐍', order: 9, variant: 'secondary' },
  // Alias — anciens keys backend (avant le portage analysis Go).
  tourist: { icon: '🐌', order: 5 },
  finisher: { icon: '🎯', order: 3 },
  first_victim: { icon: '🪦', order: 2 },
}

function formatTime(ms: number | null | undefined): string | null {
  if (ms == null || ms <= 0) return null
  const totalSec = Math.floor(ms / 1000)
  const m = Math.floor(totalSec / 60)
  const s = totalSec % 60
  return `${m}:${s.toString().padStart(2, '0')}`
}

function buildXUIDIndex(scoreboard: MatchScoreboardRow[]): Map<string, MatchScoreboardRow> {
  const idx = new Map<string, MatchScoreboardRow>()
  for (const row of scoreboard) idx.set(row.xuid, row)
  return idx
}

interface Props {
  badges: MatchImpactBadge[]
  scoreboard: MatchScoreboardRow[]
}

export function MatchImpactBadgesBar({ badges, scoreboard }: Props) {
  if (badges.length === 0) return null

  const xuidIndex = buildXUIDIndex(scoreboard)

  const sorted = [...badges].sort((a, b) => {
    const oa = BADGE_META[a.key]?.order ?? 99
    const ob = BADGE_META[b.key]?.order ?? 99
    if (oa !== ob) return oa - ob
    return (a.time_ms ?? 0) - (b.time_ms ?? 0)
  })

  return (
    <div className="border-b bg-background px-6 py-3">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-xs uppercase tracking-wide text-muted-foreground">
          Faits marquants
        </span>
        {sorted.map((b) => {
          const meta = BADGE_META[b.key]
          const player = b.player_xuid ? xuidIndex.get(b.player_xuid) : undefined
          const gamertag = player?.gamertag ?? null
          const isMe = player?.is_me ?? false
          const time = formatTime(b.time_ms)
          const variant = meta?.variant ?? 'outline'
          return (
            <Badge
              key={`${b.key}:${b.player_xuid ?? 'anon'}`}
              variant={variant}
              className={`gap-1 text-xs ${isMe ? 'ring-1 ring-primary/60' : ''}`}
              title={[b.label, gamertag, time && `à ${time}`].filter(Boolean).join(' · ')}
            >
              {meta?.icon && <span>{meta.icon}</span>}
              <span className="font-medium">{b.label}</span>
              {gamertag && (
                <span className="text-muted-foreground">· {gamertag}</span>
              )}
              {time && (
                <span className="text-muted-foreground tabular-nums">· {time}</span>
              )}
            </Badge>
          )
        })}
      </div>
    </div>
  )
}
