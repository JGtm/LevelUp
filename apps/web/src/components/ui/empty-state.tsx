import { Button } from './button'
import { Card, CardContent } from './card'

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
      className={`rounded-xl border border-dashed border-gray-200 bg-gray-50/80 px-4 py-5 text-center ${className}`}
    >
      <p className="text-sm font-semibold text-gray-800">{title}</p>
      <p className="mt-1 text-sm text-gray-500">{description}</p>
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

export function EmptyStateCard({ className = '', ...props }: EmptyStateCardProps) {
  return (
    <Card className={className}>
      <CardContent className="p-6">
        <EmptyStateNotice {...props} />
      </CardContent>
    </Card>
  )
}
