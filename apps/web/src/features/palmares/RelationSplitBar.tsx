/**
 * RelationSplitBar — barre composite proportionnelle (deux segments) + les deux
 * valeurs colorées. Reprend le langage visuel du mock v2 :
 *   - « Frags / morts »  → segments outcome-win / outcome-loss
 *   - « Rencontres »      → segments team-ally / team-enemy
 *
 * Aucune couleur hex : tout passe par tokenCssVar(token) (charte accessibilité).
 */
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

interface RelationSplitBarProps {
  label: string
  leftValue: number
  rightValue: number
  leftToken: SemanticToken
  rightToken: SemanticToken
  locale?: 'fr' | 'en'
}

export function RelationSplitBar({
  label,
  leftValue,
  rightValue,
  leftToken,
  rightToken,
  locale = 'fr',
}: RelationSplitBarProps) {
  const total = leftValue + rightValue
  const leftPct = total > 0 ? Math.round((leftValue / total) * 100) : 50
  return (
    <div className="flex items-center gap-2 font-mono text-xs">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span
        className="inline-flex h-1.5 w-16 shrink-0 overflow-hidden rounded-sm border border-border"
        role="presentation"
        aria-hidden="true"
      >
        <span style={{ width: `${leftPct}%`, backgroundColor: tokenCssVar(leftToken) }} />
        <span style={{ flex: 1, backgroundColor: tokenCssVar(rightToken) }} />
      </span>
      <span className="font-semibold" style={{ color: tokenCssVar(leftToken) }}>
        {leftValue.toLocaleString(locale)}
      </span>
      <span className="text-muted-foreground">·</span>
      <span className="font-semibold" style={{ color: tokenCssVar(rightToken) }}>
        {rightValue.toLocaleString(locale)}
      </span>
    </div>
  )
}
