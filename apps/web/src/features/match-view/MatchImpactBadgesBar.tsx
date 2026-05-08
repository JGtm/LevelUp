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

/**
 * Détecte si le `gamertag` retourné par l'API est en réalité un xuid brut
 * (cas où aucun alias n'existe en DB pour ce joueur — le resolver
 * `v_gamertag_lookup` côté backend retombe sur le xuid en dernier recours).
 * On évite alors d'afficher la chaîne illisible et on dégrade vers
 * "Joueur inconnu" / pas de nom.
 *
 * Heuristique : xuid Halo = numérique pur ≥ 15 chars OU format bot `bid(N.0)`.
 * Le bot prefix devrait être déjà rendu "343 Bot N" par la vue, donc en
 * pratique on attrape surtout les numériques purs.
 */
function isRawXUID(s: string | null | undefined): boolean {
  if (!s) return false
  if (/^bid\(/.test(s)) return true
  return /^\d{15,}$/.test(s)
}

interface Props {
  badges: MatchImpactBadge[] | null | undefined
  scoreboard: MatchScoreboardRow[] | null | undefined
}

export function MatchImpactBadgesBar({ badges, scoreboard }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const badgeI18n = getSquadText(locale).impact

  if (!badges || badges.length === 0) return null

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
        // Si le gamertag est en fait un xuid brut (alias absent de toute la
        // chaîne de résolution backend), on ne l'affiche pas — préférable à
        // un "Premier sang 2535472884034919".
        const gamertag = isRawXUID(rawGamertag) ? null : rawGamertag
        const isMe = player?.is_me ?? false
        const time = formatTime(b.time_ms)
        const description = badgeI18n.badgeDescriptions[b.key]
        const hasSubline = gamertag !== null || time !== null
        return (
          <Tooltip
            key={`${b.key}:${b.player_xuid ?? 'anon'}`}
            content={description ? <span>{description}</span> : null}
            className="w-full flex-1"
          >
            <div
              className={`flex h-full w-full flex-col justify-center gap-0.5 rounded-lg border bg-card px-3 py-2 ${
                isMe ? 'border-primary/60' : 'border-border'
              }`}
            >
              <div className="flex items-center gap-1.5">
                <BadgeIcon badgeKey={b.key} size={14} />
                <span className="text-sm font-medium text-foreground leading-none">
                  {b.label}
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
