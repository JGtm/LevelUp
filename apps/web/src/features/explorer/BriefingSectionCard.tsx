/**
 * BriefingSectionCard — carte-section unifiée du bandeau de briefing Explorer.
 *
 * Chrome + en-tête bordurée : carte `rounded-lg border border-border bg-card`
 * + en-tête `flex-none border-b border-border px-3 py-2 text-sm font-medium`. But
 * (item 7 du plan, décision P-6) : les cartes-sections du briefing partagent une
 * seule mise en forme d'en-tête.
 *
 * PÉRIMÈTRE (V3, compaction du bandeau) : seules les cartes de la rangée « Par… »
 * passent par ce wrapper — les dimensions (carte/mode/sélection) et la carte
 * « Par contexte ». Les autres blocs ont quitté le format carte : Classement et
 * Séries sont des tuiles du socle (ExplorerBriefingTiles), la Tendance une
 * micro-sparkline (Strip), les Moments forts une bande nue (DominanceBand).
 *
 * GARDE-RAIL ANTI-DIVERGENCE (CLAUDE.md §6 « ≤ 2 copies d'un même pattern ») :
 * l'en-tête bordurée existe en 2 endroits canoniques — `ChartCard` (charts ECharts)
 * et ce composant (cartes-sections « Par… »). TOUTE carte-section « Par… » DOIT
 * passer par `BriefingSectionCard` — ne JAMAIS ré-inliner un titre
 * `text-3xs uppercase …` ni recopier l'en-tête bordurée à la main (la 3ᵉ copie
 * ferait re-diverger la mise en forme). Tokens sémantiques uniquement (aucune
 * couleur hex/Tailwind — skill `color-tokens`).
 */
import type { ReactNode } from 'react'

interface BriefingSectionCardProps {
  /**
   * Titre de section, déjà i18n-résolu en amont. Accepte un `ReactNode` pour
   * pouvoir injecter un `InfoTooltip` à côté du libellé (cf. `ChartCardProps.title`) —
   * les cartes « Par… » (dimensions, « Par contexte ») y accolent un tooltip de
   * légende (V3, amendement tooltips).
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
