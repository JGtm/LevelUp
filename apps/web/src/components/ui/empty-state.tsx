import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { Button } from './button'
import { Card } from './card'

interface EmptyStateBaseProps {
  title: string
  description: string
  actionLabel?: string
  onAction?: () => void
}

interface EmptyStateNoticeProps extends EmptyStateBaseProps {
  className?: string
  /**
   * Ton sémantique. `neutral` (défaut) : placeholder gris pointillé (rien à
   * signaler de particulier). `success` : état vide qui EST un succès (aucun
   * problème restant) — bordure + coche en token `success`, perceptible d'un
   * coup d'œil plutôt qu'un gris atone.
   */
  tone?: 'neutral' | 'success'
}

export function EmptyStateNotice({
  title,
  description,
  actionLabel,
  onAction,
  className = '',
  tone = 'neutral',
}: EmptyStateNoticeProps) {
  const success = tone === 'success'
  return (
    <div
      className={`rounded-xl border px-4 py-5 text-center ${
        success ? '' : 'border-dashed border-border bg-muted/80'
      } ${className}`}
      style={success ? { borderColor: tokenCssVar('success') } : undefined}
    >
      {success && (
        <span
          aria-hidden
          className="mx-auto mb-2 flex h-8 w-8 items-center justify-center"
          style={{ color: tokenCssVar('success') }}
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            className="h-7 w-7"
          >
            <circle cx="12" cy="12" r="9" />
            <path d="M8.5 12.5l2.5 2.5 4.5-5" />
          </svg>
        </span>
      )}
      <p
        className="text-sm font-semibold text-foreground"
        style={success ? { color: tokenCssVar('success') } : undefined}
      >
        {title}
      </p>
      <p className="mt-1 text-sm text-muted-foreground">{description}</p>
      {actionLabel && onAction && (
        <div className="mt-4">
          <Button variant="outline" size="sm" onClick={onAction}>
            {actionLabel}
          </Button>
        </div>
      )}
    </div>
  )
}

interface EmptyStateCardProps extends EmptyStateBaseProps {
  className?: string
}

export function EmptyStateCard({
  className = '',
  title,
  description,
  actionLabel,
  onAction,
}: EmptyStateCardProps) {
  return (
    <Card className={className}>
      <div className="p-8 text-center">
        <p className="text-sm font-semibold text-foreground">{title}</p>
        <p className="mt-2 text-sm text-muted-foreground">{description}</p>
        {actionLabel && onAction && (
          <div className="mt-6">
            <Button variant="outline" size="sm" onClick={onAction}>
              {actionLabel}
            </Button>
          </div>
        )}
      </div>
    </Card>
  )
}
