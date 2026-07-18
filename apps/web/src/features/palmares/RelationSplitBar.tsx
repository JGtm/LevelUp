/**
 * RelationSplitBar — barre composite proportionnelle (deux segments) pleine
 * largeur, avec la LÉGENDE EN DESSOUS : valeur gauche (ex. « 34 frags ») alignée
 * à gauche, valeur droite (ex. « 78 morts ») alignée à droite. Reprend le langage
 * visuel du mock v2 (« Frags / morts » → outcome-win / outcome-loss).
 *
 * Aucune couleur hex : tout passe par tokenCssVar(token) (charte accessibilité).
 */
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

interface RelationSplitBarProps {
  leftValue: number
  rightValue: number
  leftToken: SemanticToken
  rightToken: SemanticToken
  /** Unité affichée sous le segment gauche (ex. « frags »). */
  leftLabel: string
  /** Unité affichée sous le segment droit (ex. « morts »). */
  rightLabel: string
  locale?: 'fr' | 'en'
}

export function RelationSplitBar({
  leftValue,
  rightValue,
  leftToken,
  rightToken,
  leftLabel,
  rightLabel,
  locale = 'fr',
}: RelationSplitBarProps) {
  const total = leftValue + rightValue
  const leftPct = total > 0 ? Math.round((leftValue / total) * 100) : 50
  return (
    <div className="flex flex-col gap-1">
      {/* barre pleine largeur */}
      <span
        className="inline-flex h-1.5 w-full overflow-hidden rounded-sm border border-border"
        role="presentation"
        aria-hidden="true"
      >
        <span style={{ width: `${leftPct}%`, backgroundColor: tokenCssVar(leftToken) }} />
        <span style={{ flex: 1, backgroundColor: tokenCssVar(rightToken) }} />
      </span>
      {/* légende en dessous : gauche / droite */}
      <div className="flex items-baseline justify-between font-mono text-xs">
        <span>
          <span className="font-semibold" style={{ color: tokenCssVar(leftToken) }}>
            {leftValue.toLocaleString(locale)}
          </span>{' '}
          <span className="text-muted-foreground">{leftLabel}</span>
        </span>
        <span>
          <span className="font-semibold" style={{ color: tokenCssVar(rightToken) }}>
            {rightValue.toLocaleString(locale)}
          </span>{' '}
          <span className="text-muted-foreground">{rightLabel}</span>
        </span>
      </div>
    </div>
  )
}
