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
}

export function EmptyStateNotice({
  title,
  description,
  actionLabel,
  onAction,
  className = '',
}: EmptyStateNoticeProps) {
  return (
    <div
      className={`rounded-xl border border-dashed border-border bg-muted/80 px-4 py-5 text-center ${className}`}
    >
      <p className="text-sm font-semibold text-foreground">{title}</p>
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
