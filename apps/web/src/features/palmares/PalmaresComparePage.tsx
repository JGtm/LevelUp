import { useParams } from '@tanstack/react-router'

import { Card, CardContent } from '@/components/ui/card'
import { CompareSurface } from '@/features/compare/CompareSurface'

import { PalmaresShell } from './PalmaresShell'

export function PalmaresComparePage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }

  return (
    <PalmaresShell playerSlug={playerSlug} activeTab="compare">
      <Card>
        <CardContent className="pt-6">
          <CompareSurface playerSlug={playerSlug} />
        </CardContent>
      </Card>
    </PalmaresShell>
  )
}
