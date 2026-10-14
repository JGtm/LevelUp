/**
 * SquadPreviousSessionsSection — « Soirées précédentes » (Escouade › Emprise) : la rangée des deux
 * graphes qui comparent ce soir aux soirées précédentes de la composition, à droite de l'autre :
 *
 *   - « Contrôle des ressources, par soirée » (`HabitCard`, barres groupées), à gauche ;
 *   - « Rapport de force, soirée après soirée » (`ObjectiveEveningsCard`), à droite.
 *
 * Même rangée, mêmes hauteurs (cartes étirées, graphes de même hauteur). Une carte seule prend la
 * rangée entière. Aucune des deux : la section se retire, intertitre compris. Le rapport de force
 * suit la règle de l'objectif : sans match à objectif dans le périmètre (`hasSquadObjective`),
 * pas de carte.
 */
import { SectionTitle } from '@/components/ui/detail-section'
import type { SquadFormesBlock, SquadObjectiveHistory } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { EmpriseText } from './emprise/empriseStrings'
import { HabitCard } from './emprise/HabitCard'
import type { HabitView } from './emprise/habit.logic'
import { FORMES_CARDS_TEXT } from './formes/cardsI18n'
import { hasSquadObjective } from './objectif/objectif.logic'
import { ObjectiveEveningsCard } from './objectif/ObjectiveEveningsCard'
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
  const showEvenings = objectiveHistory != null && hasSquadObjective(objective)
  if (!showHabit && !showEvenings) return null
  return (
    <section className="space-y-2" data-testid="emprise-section-habitude">
      <SectionTitle>{et.sections.habitude}</SectionTitle>
      {/* Une carte seule prend la rangée entière. */}
      <div className="grid gap-4 lg:grid-cols-2 lg:[&>*:only-child]:col-span-2">
        {showHabit && <HabitCard view={habit} locale={locale} t={et} />}
        {showEvenings && (
          <ObjectiveEveningsCard
            history={objectiveHistory}
            familyLabel={(f) => families[f] ?? f}
            locale={locale}
            t={OBJECTIF_TEXT[locale]}
          />
        )}
      </div>
    </section>
  )
}
