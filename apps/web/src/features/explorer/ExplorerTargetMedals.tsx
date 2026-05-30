/**
 * ExplorerTargetMedals — top médailles lifetime du joueur cible.
 *
 * Affiche les 5 médailles les plus gagnées (image + titre + compteur), avec un
 * expander discret pour voir jusqu'à 20. La description s'affiche en tooltip
 * (title). Données : ExplorerTargetProfile.top_medals (déjà triées par count
 * décroissant, cap 20 côté backend) + images statiques /static/medals/.
 */
import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { MedalDigestItem } from '@/lib/api/types'

const TOP_COUNT = 5

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
    <Card data-testid="explorer-target-medals">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-semibold">{t('explorer.target_profile.top_medals_title')}</CardTitle>
      </CardHeader>
      <CardContent className="pt-0">
        <ul className="grid grid-cols-1 gap-1.5 sm:grid-cols-2">
          {visible.map((m) => (
            <li
              key={m.medal_id}
              className="flex items-center gap-2 rounded-md border border-border bg-muted/20 px-2 py-1.5"
              title={m.description || undefined}
            >
              {m.image_url ? (
                <img
                  src={m.image_url}
                  alt=""
                  aria-hidden="true"
                  className="h-9 w-9 flex-shrink-0 object-contain"
                  loading="lazy"
                  decoding="async"
                />
              ) : (
                <span
                  className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full bg-muted-foreground/20 text-xs font-bold uppercase"
                  aria-hidden="true"
                >
                  {(m.label ?? '?').charAt(0)}
                </span>
              )}
              <span className="min-w-0 flex-1 truncate text-sm text-foreground">
                {m.label || `#${m.medal_id}`}
              </span>
              <span className="flex-shrink-0 text-sm font-semibold tabular-nums text-muted-foreground">
                ×{m.total_count.toLocaleString(locale === 'en' ? 'en-US' : 'fr-FR')}
              </span>
            </li>
          ))}
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
      </CardContent>
    </Card>
  )
}
