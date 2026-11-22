/**
 * TacticalPlanMentions — LA MENTION DES CALQUES QUI NE PORTENT PAS LA HAUTEUR, au pied du plan : quand une coupe par
 * étage est choisie, chaque calque affiché que la coupe ne filtre pas le dit (« {calque} : ce calque ne porte pas la
 * hauteur, il reste affiché sur tous les étages. »). Rien sans coupe, rien quand tous les calques affichés suivent la coupe.
 */
import type { TacticalText } from './i18n'
import { mentionSansHauteur, type CalqueSansHauteur } from './reliefPlan.logic'

export function TacticalPlanMentions({ t, calques }: { t: TacticalText; calques: readonly CalqueSansHauteur[] }) {
  if (calques.length === 0) return null
  return (
    <ul
      className="mx-3 mb-3 -mt-1 flex flex-col gap-1 text-[11.5px] text-muted-foreground"
      data-testid="tactical-plan-sans-hauteur"
      aria-live="polite"
    >
      {calques.map((calque) => (
        <li key={calque} className="flex items-center gap-1.5">
          <svg className="size-3 shrink-0" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.1" strokeLinejoin="round" aria-hidden>
            <path d="M6 1.5 11 4 6 6.5 1 4z" />
            <path d="M1 6.3 6 8.8l5-2.5M1 8.6 6 11l5-2.4" />
          </svg>
          {mentionSansHauteur(t, calque)}
        </li>
      ))}
    </ul>
  )
}
