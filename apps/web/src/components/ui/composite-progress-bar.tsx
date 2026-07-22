import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { clampCompositeProgress } from './composite-progress-bar-labels'

export function CompositeProgressBar({
  value,
  fillTestId,
}: {
  value?: number | null
  fillTestId?: string
}) {
  const width = clampCompositeProgress(value)

  return (
    <div className="h-2 w-full overflow-hidden rounded-full bg-muted-foreground/25">
      <div
        data-testid={fillTestId}
        className="h-full rounded-full transition-all duration-300"
        style={{ width: `${width}%`, backgroundColor: width >= 100 ? tokenCssVar('success') : tokenCssVar('info') }}
      />
    </div>
  )
}
