/**
 * EmpriseFilmGapSection — une section de l'onglet Emprise dont le film du périmètre n'est pas lu :
 * l'intertitre reste, avec un état vide plutôt qu'une section absente (qui laissait croire à un bug).
 *
 * Convention de l'application : un bloc parent (`Card`) qui porte l'état vide. L'analyse du film en
 * cours est dite par `AnalysisPendingSlot` (pointillé et indicateur tournant), le film non analysé
 * par `EmptyStateNotice` (pointillé). Tous deux DANS le bloc parent, jamais nus sous l'intertitre.
 */
import { AnalysisPendingSlot } from '@/components/ui/analysis-pending-slot'
import { Card, CardContent } from '@/components/ui/card'
import { SectionTitle } from '@/components/ui/detail-section'
import { EmptyStateNotice } from '@/components/ui/empty-state'

import type { FilmGapText } from './filmGapStrings'

export function EmpriseFilmGapSection({
  title,
  testId,
  pending,
  t,
}: {
  title: string
  testId: string
  /** Vrai : l'analyse du film tourne pour au moins un match du périmètre. */
  pending: boolean
  t: FilmGapText
}) {
  return (
    <section className="space-y-2" data-testid={testId}>
      <SectionTitle>{title}</SectionTitle>
      <Card data-testid="emprise-film-gap">
        <CardContent className="pt-4">
          {pending ? (
            <AnalysisPendingSlot label={t.pending} />
          ) : (
            <EmptyStateNotice title={t.missingTitle} description={t.missingDescription} />
          )}
        </CardContent>
      </Card>
    </section>
  )
}
