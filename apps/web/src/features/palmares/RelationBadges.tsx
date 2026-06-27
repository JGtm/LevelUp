/**
 * RelationBadges — rendu des badges d'une relation (hub Communauté > Relations).
 *
 * Tous les badges de relation sont rendus en style "solid" (fond couleur saturée
 * + texte blanc) via NarrativeBadge(solid) — homogènes avec les autres surfaces
 * (Match View, Explorer, Compare). Les libellés sont résolus via squadManifest
 * (clés narrative.encounter.*). Les couleurs passent par les tokens accessibilité
 * (jamais de hex en dur).
 */
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { tokenVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { formatMessage } from '@/lib/i18n/format'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import type { RelationBadge } from '@/lib/api/types'

function isSemanticToken(s: string): s is SemanticToken {
  return s.startsWith('narrative-')
}

function resolveOrdinal(detail: RelationBadge['detail']): number | undefined {
  if (detail && typeof detail['ordinal'] === 'number') {
    return detail['ordinal'] as number
  }
  return undefined
}

/** resolveGame — nom d'affichage de l'autre titre (badge cross-jeu, Phase 3b). */
function resolveGame(detail: RelationBadge['detail']): string | undefined {
  if (detail && typeof detail['game'] === 'string') {
    return detail['game'] as string
  }
  return undefined
}

export function RelationBadges({
  badges,
  locale,
}: {
  badges: RelationBadge[] | null
  locale: 'fr' | 'en'
}) {
  if (!badges?.length) return null
  return (
    <span className="ml-2 inline-flex flex-wrap gap-1 align-middle">
      {badges.map((badge, i) => {
        const key = badge.label_key as SquadManifestKey
        const ordinal = resolveOrdinal(badge.detail)
        const game = resolveGame(badge.detail)
        const vars: Record<string, unknown> = {}
        if (ordinal !== undefined) vars['ordinal'] = ordinal
        if (game !== undefined) vars['game'] = game
        const label =
          Object.keys(vars).length > 0
            ? formatMessage(squadManifest, key, locale, vars)
            : formatMessage(squadManifest, key, locale)
        const colorVar = isSemanticToken(badge.color_token)
          ? tokenVar(badge.color_token as SemanticToken)
          : undefined
        return <NarrativeBadge key={i} label={label} colorVar={colorVar} solid size="sm" />
      })}
    </span>
  )
}
