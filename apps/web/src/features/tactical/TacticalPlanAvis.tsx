/**
 * TacticalPlanAvis — ce qui se pose SUR le fond du plan selon l'état de sa lecture, jamais À LA PLACE
 * du fond : l'avis (échec, carte hors du filtre, plan vide) et l'indicateur de lecture (premier
 * chargement, « Mise à jour… »).
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'

import type { TacticalText } from './i18n'
import type { EtatDuPlan } from './plan.logic'

/**
 * AvisSurLeFond — le message posé SUR le fond, jamais à sa place : l'échec (composition impossible
 * ou panne), la carte hors du filtre, ou l'état vide du plan (`children`, un titre seul).
 */
export function AvisSurLeFond({
  t,
  etat,
  inconnus,
  estompe,
  children,
}: {
  t: TacticalText
  etat: EtatDuPlan
  inconnus: string[] | null
  estompe: string
  children: string | null
}) {
  if (etat === 'echec') {
    return (
      <div className="absolute inset-0 grid place-items-center p-3" data-testid="tactical-plan-avis">
        {inconnus ? (
          <EmptyStateNotice
            title={t.unknownTeammateTitle}
            description={t.unknownTeammateDescription(inconnus.join(', '))}
            className="bg-card"
          />
        ) : (
          <p className="rounded-md border border-border bg-card px-2.5 py-1.5 text-center text-sm text-muted-foreground">
            {t.analysisErrorTitle}
          </p>
        )}
      </div>
    )
  }
  const titre = etat === 'hors_filtre' ? t.planEmptyNoMatchTitle : children
  if (!titre) return null
  return (
    <div
      className={`pointer-events-none absolute inset-0 grid place-items-center p-3${estompe}`}
      data-testid={etat === 'hors_filtre' ? 'tactical-carte-hors-filtre' : 'tactical-plan-vide'}
    >
      <p className="rounded-md border border-border bg-card px-2.5 py-1.5 text-center text-sm text-muted-foreground">{titre}</p>
    </div>
  )
}


/**
 * IndicateurDeLecture — ce qui se pose PAR-DESSUS le fond selon l'état de la lecture : l'indicateur
 * du premier chargement (visuel seul, sans texte d'attente : l'occupation est dite par `aria-busy`
 * sur le corps de la vue), ou la mention « Mise à jour… » d'une relecture. Jamais À LA PLACE du fond.
 */
export function IndicateurDeLecture({ t, etat }: { t: TacticalText; etat: EtatDuPlan }) {
  if (etat === 'attente') {
    return (
      <div className="absolute inset-0 flex items-center justify-center" data-testid="tactical-analysis-pending">
        <Spinner />
      </div>
    )
  }
  if (etat === 'relecture') {
    return (
      <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
        <p
          role="status"
          className="rounded-md border border-border bg-card px-3 py-1.5 text-sm text-foreground shadow-sm"
          data-testid="tactical-analysis-updating"
        >
          {t.analysisUpdating}
        </p>
      </div>
    )
  }
  return null
}

