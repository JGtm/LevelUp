/**
 * MatchImpactBadgesBar — bandeau des badges d'impact du match.
 *
 * Affiche, au-dessus des onglets de la page match, les badges narratifs
 * calculés côté backend (analysis.ComputeMatchImpactFull) :
 *   - event-based : Premier sang, Première victime, Finisseur, Boulet,
 *     Touriste, Top Gun (timing affiché)
 *   - stat-based : Bourreau, Héros silencieux, Faux-frère
 *
 * Pictos : Fluent Emoji Flat (cf. components/feedback/BadgeIcon).
 * Aligne le rendu sur la branche main (badges + horodatage). La résolution
 * xuid → gamertag se fait via le scoreboard du match.
 */
import { Badge } from '@/components/ui/badge'
import { BadgeIcon } from '@/components/feedback/BadgeIcon'
import { Tooltip } from '@/components/ui/tooltip'
import { useAppShellStore } from '@/stores/appShellStore'
import { getSquadText } from '@/features/squad/i18n'
import type { MatchImpactBadge, MatchScoreboardRow } from '@/lib/api/types'

interface BadgeMeta {
  /** order d'affichage (1 = en premier) */
  order: number
  /** style visuel — outline par défaut */
  variant?: 'outline' | 'secondary' | 'default'
}

const BADGE_META: Record<string, BadgeMeta> = {
  first_blood: { order: 1 },
  first_group_death: { order: 2 },
  clutch_finisher: { order: 3 },
  last_casualty: { order: 4 },
  last_group_kill: { order: 5 },
  top_gun: { order: 6 },
  top_killer: { order: 7, variant: 'secondary' },
  silent_hero: { order: 8, variant: 'secondary' },
  false_brother: { order: 9, variant: 'secondary' },
  // Alias — anciens keys backend (avant le portage analysis Go).
  tourist: { order: 5 },
  finisher: { order: 3 },
  first_victim: { order: 2 },
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
  const locale = useAppShellStore((s) => s.locale)
  const badgeI18n = getSquadText(locale).impact

  if (badges.length === 0) return null

  const xuidIndex = buildXUIDIndex(scoreboard)

  const sorted = [...badges].sort((a, b) => {
    const oa = BADGE_META[a.key]?.order ?? 99
    const ob = BADGE_META[b.key]?.order ?? 99
    if (oa !== ob) return oa - ob
    return (a.time_ms ?? 0) - (b.time_ms ?? 0)
  })

  return (
    <div className="rounded-lg border border-border bg-card px-4 py-3">
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
          const description = badgeI18n.badgeDescriptions[b.key]
          return (
            <Tooltip
              key={`${b.key}:${b.player_xuid ?? 'anon'}`}
              content={description ? <span>{description}</span> : null}
            >
              <Badge
                variant={variant}
                className={`gap-1 text-xs ${isMe ? 'ring-1 ring-primary/60' : ''}`}
              >
                <BadgeIcon badgeKey={b.key} size={14} />
                <span className="font-medium">{b.label}</span>
                {gamertag && (
                  <span className="text-muted-foreground">· {gamertag}</span>
                )}
                {time && (
                  <span className="text-muted-foreground tabular-nums">· {time}</span>
                )}
              </Badge>
            </Tooltip>
          )
        })}
      </div>
    </div>
  )
}
