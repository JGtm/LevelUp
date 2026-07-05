/**
 * ExplorerTargetMedals — top médailles lifetime du joueur cible.
 *
 * Affiche les 18 médailles les plus gagnées (image + titre + compteur), avec un
 * expander discret pour voir le reste (cap 20 côté backend). La description
 * s'affiche en tooltip (title). Données : ExplorerTargetProfile.top_medals
 * (déjà triées par count décroissant) + images statiques /static/medals/.
 */
import { useState } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { dropShadowForDifficulty } from '@/lib/medalDifficulty'
import { MedalIcon } from '@/components/ui/MedalIcon'
import type { MedalDigestItem } from '@/lib/api/types'

const TOP_COUNT = 18

interface ExplorerTargetMedalsProps {
  medals: MedalDigestItem[]
}

export function ExplorerTargetMedals({ medals }: ExplorerTargetMedalsProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: ExplorerManifestKey, values?: Record<string, string | number>) =>
    formatMessage(explorerManifest, key, locale, values)
  const [expanded, setExpanded] = useState(false)

  if (medals.length === 0) return null

  const visible = expanded ? medals : medals.slice(0, TOP_COUNT)
  const hiddenCount = medals.length - TOP_COUNT

  return (
    <div className="rounded-lg border border-border bg-card" data-testid="explorer-target-medals">
      <div className="border-b border-border px-3 py-2 text-sm font-medium">
        {t('explorer.target_profile.top_medals_title')}
      </div>
      <div className="p-3">
        <ul className="grid grid-cols-2 gap-1.5 sm:grid-cols-3">
          {visible.map((m) => {
            // Glow par rareté UNIQUEMENT sur l'image (parité tuiles de match home /
            // MedalDigest) : drop-shadow qui épouse la forme du PNG transparent.
            const glow = dropShadowForDifficulty(m.difficulty)
            return (
              <li
                key={m.medal_id}
                className="flex items-center gap-1.5 rounded-md border border-border bg-muted/20 px-1.5 py-1.5"
                title={m.description || undefined}
              >
                {m.image_url || m.sprite_sheet ? (
                  <MedalIcon
                    imageUrl={m.image_url}
                    spriteSheet={m.sprite_sheet}
                    spriteLeft={m.sprite_left}
                    spriteTop={m.sprite_top}
                    spriteWidth={m.sprite_width}
                    spriteHeight={m.sprite_height}
                    label={m.label || `#${m.medal_id}`}
                    size={32}
                    className="flex-shrink-0 object-contain"
                    style={glow ? { filter: glow } : undefined}
                  />
                ) : (
                  <span
                    className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-muted-foreground/20 text-xs font-bold uppercase"
                    aria-hidden="true"
                  >
                    {(m.label ?? '?').charAt(0)}
                  </span>
                )}
                <span className="min-w-0 flex-1 truncate text-xs text-foreground">
                  {m.label || `#${m.medal_id}`}
                </span>
                <span className="flex-shrink-0 text-xs font-semibold tabular-nums text-muted-foreground">
                  ×{m.total_count.toLocaleString(intlLocale(locale))}
                </span>
              </li>
            )
          })}
        </ul>

        {hiddenCount > 0 && (
          <button
            type="button"
            onClick={() => setExpanded((v) => !v)}
            className="mt-2 text-xs font-medium text-muted-foreground underline-offset-2 hover:text-foreground hover:underline"
          >
            {expanded
              ? t('explorer.target_profile.medals_show_less')
              : t('explorer.target_profile.medals_show_more', { count: hiddenCount })}
          </button>
        )}
      </div>
    </div>
  )
}
