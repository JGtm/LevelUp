/**
 * BriefingSectionCard — carte-section unifiée du bandeau de briefing Explorer.
 *
 * Chrome + en-tête bordurée calqués EXACTEMENT sur `ChartCard` (bloc « Tendance »,
 * la référence esthétique du plan) : carte `rounded-lg border border-border bg-card`
 * + en-tête `flex-none border-b border-border px-3 py-2 text-sm font-medium`. But
 * (item 7 du plan, décision P-6) : toutes les cartes-sections non-chart du briefing
 * (dimensions, classement, solo/escouade, séries, moments forts) partagent une seule
 * mise en forme d'en-tête, homogène avec le module « Tendance ».
 *
 * GARDE-RAIL ANTI-DIVERGENCE (CLAUDE.md §6 « ≤ 2 copies d'un même pattern ») :
 * l'en-tête bordurée du briefing existe désormais en 2 endroits canoniques —
 * `ChartCard` (module « Tendance », piloté par ECharts) et ce composant (cartes-
 * sections non-chart). TOUTE nouvelle carte-section du briefing (Phases 4/5/5b :
 * classement par type de rating, contexte solo/escouade, séries, moments forts)
 * DOIT passer par `BriefingSectionCard` — ne JAMAIS ré-inliner un titre
 * `text-3xs uppercase …` ni recopier l'en-tête bordurée à la main (la 3ᵉ copie
 * ferait re-diverger la mise en forme). Tokens sémantiques uniquement (aucune
 * couleur hex/Tailwind — skill `color-tokens`).
 */
import type { ReactNode } from 'react'

interface BriefingSectionCardProps {
  /**
   * Titre de section, déjà i18n-résolu en amont. Accepte un `ReactNode` pour
   * pouvoir injecter un `InfoTooltip` à côté du libellé (cf. `ChartCardProps.title`) ;
   * aucun tooltip n'est posé dans ce chantier (D-A).
   */
  title: ReactNode
  /** Classes sur la carte racine (ex. `h-full` pour remplir une cellule de grille). */
  className?: string
  /** Contenu de la carte (liste, blocs de valeurs…). */
  children: ReactNode
  testId?: string
}

export function BriefingSectionCard({
  title,
  className = '',
  children,
  testId,
}: BriefingSectionCardProps) {
  return (
    <div
      className={`rounded-lg border border-border bg-card ${className}`}
      data-testid={testId}
    >
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">
        {title}
      </div>
      <div className="p-3">{children}</div>
    </div>
  )
}
