/**
 * RelationBadges — rendu des badges d'une relation (hub Communauté > Relations).
 *
 * Deux styles servis par le backend (analysis/relations) :
 *  - "tinted" : badges narratifs existants (ordinal / ally_plus / tough_enemy /
 *    coriace) → NarrativeBadge (fond teinté, aligné Carrière / Match View).
 *  - "solid"  : nouveaux badges (duo_gagnant / cameleon / de_longue_date /
 *    recrue / proie_favorite) → fond plein + texte blanc.
 *
 * Les labels sont résolus via squadManifest (clés narrative.encounter.*). Les
 * couleurs passent par les tokens accessibilité (jamais de hex en dur).
 */
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { tokenVar, tokenCssVar } from '@/lib/accessibility'
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

/** SolidBadge — badge à fond plein + texte blanc (nouveaux badges Phase 1). */
function SolidBadge({ label, colorToken }: { label: string; colorToken: string }) {
  const bg = isSemanticToken(colorToken) ? tokenCssVar(colorToken as SemanticToken) : undefined
  return (
    <span
      className="inline-flex items-center rounded-full px-2 py-0.5 text-2xs font-semibold leading-none text-primary-foreground"
      style={bg ? { backgroundColor: bg } : undefined}
      data-testid="relation-badge-solid"
    >
      {label}
    </span>
  )
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
        const label =
          ordinal !== undefined
            ? formatMessage(squadManifest, key, locale, { ordinal })
            : formatMessage(squadManifest, key, locale)

        if (badge.style === 'solid') {
          return <SolidBadge key={i} label={label} colorToken={badge.color_token} />
        }
        const colorVar = isSemanticToken(badge.color_token)
          ? tokenVar(badge.color_token as SemanticToken)
          : undefined
        return <NarrativeBadge key={i} label={label} colorVar={colorVar} size="sm" />
      })}
    </span>
  )
}
