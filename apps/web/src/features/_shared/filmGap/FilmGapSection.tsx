/**
 * FilmGapSection — une section tirée du film dont le film du périmètre n'est pas lu : l'intertitre reste,
 * avec un état vide plutôt qu'une section absente (qui laissait croire à un bug).
 *
 * Convention de l'application : un bloc parent (`Card`) qui porte l'état vide. L'analyse du film en
 * cours est dite par `AnalysisPendingSlot` (pointillé et indicateur tournant), le film non analysé par
 * `EmptyStateNotice` (pointillé). Tous deux DANS le bloc parent, jamais nus sous l'intertitre.
 *
 *   `FilmGapState`   l'état seul, pour un bloc parent qui existe déjà (une `SectionCard` titrée) ;
 *   `FilmGapCard`    le bloc parent et son état, pour une carte qui porte déjà son intertitre (Sessions) ;
 *   `FilmGapSection` l'intertitre, le bloc parent et son état (Escouade, Séries temporelles, Synthèse).
 */
import { AnalysisPendingSlot } from '@/components/ui/analysis-pending-slot'
import { Card, CardContent } from '@/components/ui/card'
import { SectionTitle } from '@/components/ui/detail-section'
import { EmptyStateNotice } from '@/components/ui/empty-state'

import type { FilmGap } from './filmGap.logic'
import type { FilmGapText } from './filmGapStrings'

/** L'état : « en cours » (indicateur tournant) ou « non analysé » (pointillé). */
export function FilmGapState({ gap, t }: { gap: FilmGap; t: FilmGapText }) {
  return gap === 'pending' ? (
    <AnalysisPendingSlot label={t.pending} />
  ) : (
    <EmptyStateNotice title={t.missingTitle} description={t.missingDescription} />
  )
}

/** Le bloc parent et son état. */
export function FilmGapCard({ gap, t, className }: { gap: FilmGap; t: FilmGapText; className?: string }) {
  return (
    <Card data-testid="film-gap" className={className}>
      <CardContent className="pt-4">
        <FilmGapState gap={gap} t={t} />
      </CardContent>
    </Card>
  )
}

export function FilmGapSection({ title, testId, gap, t }: { title: string; testId: string; gap: FilmGap; t: FilmGapText }) {
  return (
    <section className="space-y-2" data-testid={testId}>
      <SectionTitle>{title}</SectionTitle>
      <FilmGapCard gap={gap} t={t} />
    </section>
  )
}
