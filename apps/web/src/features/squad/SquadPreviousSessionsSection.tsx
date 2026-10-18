/**
 * SquadPreviousSessionsSection — « Soirées précédentes » (Escouade › Emprise) : la rangée des deux
 * graphes qui comparent ce soir aux soirées précédentes de la composition, à droite de l'autre :
 *
 *   - « Contrôle des ressources, par soirée » (`HabitCard`, barres groupées), à gauche ;
 *   - « Rapport de force, soirée après soirée » (`ObjectiveEveningsCard`), à droite.
 *
 * Même rangée, mêmes hauteurs (cartes étirées, graphes de même hauteur). Une carte seule prend la
 * rangée entière. Aucune des deux : la section se retire, intertitre compris. Le rapport de force
 * suit la règle de l'objectif : sans match à objectif dans le périmètre (`hasSquadObjective`), la carte
 * reste, titre gardé, avec « Aucun match à objectif sur ce périmètre » dans son cadre (« Objectif indisponible pour ce périmètre »
 * quand la lecture a échoué) ; elle ne se retire que si le titre ne mesure pas l'objectif
 * (`objectiveGap` nul).
 */
import { SectionTitle } from '@/components/ui/detail-section'
import { PAIR_ROW_CLASS } from '@/components/ui/pair-row'
import { ScopeEmptyState } from '@/components/ui/scope-empty'
import type { SquadFormesBlock, SquadObjectiveHistory } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { EmpriseText } from './emprise/empriseStrings'
import { HabitCard } from './emprise/HabitCard'
import type { HabitView } from './emprise/habit.logic'
import { FORMES_CARDS_TEXT } from './formes/cardsI18n'
import { hasSquadObjective, objectiveGap } from './objectif/objectif.logic'
import { ObjectiveEveningsCard } from './objectif/ObjectiveEveningsCard'
import { ObjectifFrame } from './objectif/ObjectifFrame'
import { OBJECTIF_TEXT } from './objectif/objectifStrings'

interface Props {
  habit: HabitView
  /** Le bloc de formes retenues : sans match à objectif, le rapport de force par soirée se retire. */
  objective: SquadFormesBlock | null | undefined
  objectiveHistory: SquadObjectiveHistory | null | undefined
  locale: Locale
  et: EmpriseText
}

export function SquadPreviousSessionsSection({ habit, objective, objectiveHistory, locale, et }: Props) {
  const families = FORMES_CARDS_TEXT[locale].families
  const showHabit = habit.kind !== 'none'
  const measured = hasSquadObjective(objective)
  const showEvenings = objectiveHistory != null && measured
  const gap = objectiveGap(objective)
  const showEveningsEmpty = gap != null
  if (!showHabit && !showEvenings && !showEveningsEmpty) return null
  return (
    <section className="space-y-2" data-testid="emprise-section-habitude">
      <SectionTitle>{et.sections.habitude}</SectionTitle>
      {/* Une carte seule prend la rangée entière. */}
      <div className={PAIR_ROW_CLASS}>
        {showHabit && <HabitCard view={habit} locale={locale} t={et} />}
        {showEvenings && (
          <ObjectiveEveningsCard
            history={objectiveHistory}
            familyLabel={(f) => families[f] ?? f}
            locale={locale}
            t={OBJECTIF_TEXT[locale]}
          />
        )}
        {gap && (
          <ObjectifFrame title={OBJECTIF_TEXT[locale].evenings.title} info={OBJECTIF_TEXT[locale].evenings.info} testId="objective-evenings-empty">
            <ScopeEmptyState kind={gap} locale={locale} />
          </ObjectifFrame>
        )}
      </div>
    </section>
  )
}
