/**
 * scope-empty — le bloc qui reste AFFICHÉ quand aucun match du périmètre n'est concerné.
 *
 * Convention de l'application (état vide) : un bloc parent (`Card`) qui porte l'`EmptyStateNotice`
 * en pointillé, jamais le pointillé nu sous un intertitre. Trois niveaux, comme `FilmGap` :
 *
 *   `ScopeEmptyState`   le constat seul, pour un bloc parent qui existe déjà (carte titrée) ;
 *   `ScopeEmptyCard`    le bloc parent et son constat, pour un intertitre posé par l'appelant ;
 *   `ScopeEmptySection` l'intertitre, le bloc parent et le constat.
 *
 * Capability du titre absente : le bloc est RETIRÉ (fonction absente) ; ce module ne sert qu'à dire
 * « aucun match concerné ». Les textes vivent dans `scope-empty-text.ts` ; la garde
 * `scopeEmpty.guard.test.ts` interdit de les réécrire ailleurs.
 */
import type { ReactNode } from 'react'

import type { Locale } from '@/lib/i18n/locale'

import { Card, CardContent } from './card'
import { SectionTitle } from './detail-section'
import { EmptyStateNotice } from './empty-state'
import { scopeEmptyText, type ScopeEmptyKind } from './scope-empty-text'

interface StateProps {
  kind: ScopeEmptyKind
  locale: Locale
  className?: string
}

/** Le constat seul. */
export function ScopeEmptyState({ kind, locale, className }: StateProps) {
  const t = scopeEmptyText(kind, locale)
  return <EmptyStateNotice title={t.title} description={t.description} className={className} />
}

/** Le bloc parent et son constat. */
export function ScopeEmptyCard({ kind, locale, className, testId = 'scope-empty' }: StateProps & { testId?: string }) {
  return (
    <Card data-testid={testId} className={className}>
      <CardContent className="pt-4">
        <ScopeEmptyState kind={kind} locale={locale} />
      </CardContent>
    </Card>
  )
}

/** L'intertitre, le bloc parent et son constat. */
export function ScopeEmptySection({
  title,
  kind,
  locale,
  testId,
}: {
  title: ReactNode
  kind: ScopeEmptyKind
  locale: Locale
  testId: string
}) {
  return (
    <section className="space-y-2" data-testid={testId}>
      <SectionTitle>{title}</SectionTitle>
      <ScopeEmptyCard kind={kind} locale={locale} />
    </section>
  )
}
