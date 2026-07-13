/**
 * MatchImpactBadgesBar — bandeau des badges d'impact du match.
 *
 * Affiche, au-dessus des onglets de la page match, les badges narratifs
 * calculés côté backend (analysis.ComputeMatchImpactFull) :
 *   - event-based : Premier sang, Première victime, Finisseur, Boulet,
 *     Touriste, Top Gun (timing affiché)
 *   - stat-based : Bourreau, Héros silencieux, Faux-frère
 *
 * Valence visuelle (option C) :
 *   - Toujours  : strip gauche 3px à 30 % sur fond `bg-card` (aligné sur les
 *     autres blocs de la page — pas de fond teinté)
 *   - is_me     : strip 4px à 100 % + titre coloré
 *
 * Pictos : Fluent Emoji Flat (cf. components/feedback/BadgeIcon).
 * Aligne le rendu sur la branche main (badges + horodatage). La résolution
 * xuid → gamertag se fait via le scoreboard du match.
 */
import { BadgeIcon } from '@/components/feedback/BadgeIcon'
import { hexToRgba } from '@/components/charts/_utils'
import { Tooltip } from '@/components/ui/tooltip'
import { resolveToken } from '@/lib/accessibility'
import { useAppShellStore } from '@/stores/appShellStore'
import { getSquadText } from '@/features/squad/i18n'
import type { MatchImpactBadge, MatchScoreboardRow } from '@/lib/api/types'
import type { MatchViewText } from './i18n'

type Valence = 'positive' | 'negative' | 'neutral'

interface BadgeMeta {
  order: number
  variant?: 'outline' | 'secondary' | 'default'
  hideTime?: boolean
  valence?: Valence
}

const BADGE_META: Record<string, BadgeMeta> = {
  first_blood:       { order: 1, valence: 'positive' },
  first_group_death: { order: 2, valence: 'negative' },
  clutch_finisher:   { order: 3, valence: 'positive' },
  last_casualty:     { order: 4, valence: 'negative' },
  last_group_kill:   { order: 5, valence: 'positive' },
  top_gun:           { order: 6, valence: 'positive' },
  top_killer:        { order: 7, variant: 'secondary', valence: 'positive' },
  silent_hero:       { order: 8, variant: 'secondary', valence: 'positive' },
  false_brother:     { order: 9, variant: 'secondary', valence: 'negative' },
  kamikaze:          { order: 10, variant: 'secondary', hideTime: true, valence: 'negative' },
  // Alias — anciens keys backend (avant le portage analysis Go).
  tourist:           { order: 5, valence: 'neutral' },
  finisher:          { order: 3, valence: 'positive' },
  first_victim:      { order: 2, valence: 'negative' },
}

function valenceColorToken(v: Valence | undefined): 'success' | 'destructive' | null {
  if (v === 'positive') return 'success'
  if (v === 'negative') return 'destructive'
  return null
}

function formatTime(ms: number | null | undefined): string | null {
  if (ms == null || ms <= 0) return null
  const totalSec = Math.floor(ms / 1000)
  const m = Math.floor(totalSec / 60)
  const s = totalSec % 60
  return `${m}m${s.toString().padStart(2, '0')}s`
}

function buildXUIDIndex(scoreboard: MatchScoreboardRow[]): Map<string, MatchScoreboardRow> {
  const idx = new Map<string, MatchScoreboardRow>()
  for (const row of scoreboard) idx.set(row.xuid, row)
  return idx
}

function isRawXUID(s: string | null | undefined): boolean {
  if (!s) return false
  if (/^bid\(/.test(s)) return true
  return /^\d{15,}$/.test(s)
}

interface Props {
  badges: MatchImpactBadge[] | null | undefined
  scoreboard: MatchScoreboardRow[] | null | undefined
  t: MatchViewText
}

export function MatchImpactBadgesBar({ badges, scoreboard, t }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const badgeI18n = getSquadText(locale).impact

  if (!badges || badges.length === 0) {
    // Placeholder muet (style carte badge) au lieu de disparaître : la colonne
    // garde sa place à côté du graphe Frags cumulés.
    return (
      <div className="flex h-full min-h-[80px] items-center justify-center rounded-lg border border-border bg-card px-3 py-2 text-center text-xs text-muted-foreground">
        {t.impactBadgesNoData}
      </div>
    )
  }

  const xuidIndex = buildXUIDIndex(scoreboard ?? [])

  const sorted = [...badges].sort((a, b) => {
    const oa = BADGE_META[a.key]?.order ?? 99
    const ob = BADGE_META[b.key]?.order ?? 99
    if (oa !== ob) return oa - ob
    return (a.time_ms ?? 0) - (b.time_ms ?? 0)
  })

  return (
    <div className="flex h-full flex-col gap-2">
      {sorted.map((b) => {
        const player = b.player_xuid ? xuidIndex.get(b.player_xuid) : undefined
        const rawGamertag = player?.gamertag ?? null
        const gamertag = isRawXUID(rawGamertag) ? null : rawGamertag
        const isMe = player?.is_me ?? false
        const time = BADGE_META[b.key]?.hideTime ? null : formatTime(b.time_ms)
        const description = badgeI18n.badgeDescriptions[b.key]
        const hasSubline = gamertag !== null || time !== null

        const valence = BADGE_META[b.key]?.valence
        const colorToken = valenceColorToken(valence)
        const resolvedHex = colorToken ? resolveToken(colorToken) : null

        const cardStyle: React.CSSProperties = resolvedHex ? {
          boxShadow: isMe
            ? `inset 4px 0 0 0 ${hexToRgba(resolvedHex, 1)}`
            : `inset 3px 0 0 0 ${hexToRgba(resolvedHex, 0.3)}`,
        } : {}

        const titleStyle: React.CSSProperties = (isMe && resolvedHex)
          ? { color: resolvedHex }
          : {}

        return (
          <Tooltip
            key={`${b.key}:${b.player_xuid ?? 'anon'}`}
            content={description ? <span>{description}</span> : null}
            className="w-full flex-1"
          >
            <div
              className={`flex h-full w-full flex-col justify-center gap-1.5 rounded-lg border bg-card px-3 py-2 ${
                isMe ? 'border-primary/60' : 'border-border'
              }`}
              style={cardStyle}
            >
              <div className="flex items-center gap-1.5">
                <BadgeIcon badgeKey={b.key} size={14} />
                <span className="text-sm font-medium leading-none" style={titleStyle}>
                  {t.impactBadgeNames[b.key] ?? b.label}
                </span>
              </div>
              {hasSubline && (
                <p className="text-xs text-muted-foreground leading-none tabular-nums">
                  {gamertag ?? ''}
                  {gamertag && time ? ' · ' : ''}
                  {time ?? ''}
                </p>
              )}
            </div>
          </Tooltip>
        )
      })}
    </div>
  )
}
