/**
 * KpiCard — primitive carte KPI unifiée (fusion des types 2 et 4 du catalogue UI
 * d'harmonisation : « KPI card » accent fixe + « KPI card à accent dynamique »).
 *
 * Chrome commun : bordure + `bg-card` + coins arrondis + barre d'accent 3px en
 * haut. `overflow-hidden` clippe la barre aux coins arrondis. L'accent est soit
 * fixe (un token par métrique, type 2), soit calculé selon le contenu (type 4) —
 * c'est l'appelant qui décide en passant (ou non) le token.
 *
 * Contenu libre via `children` : le layout interne (label/valeur, barres
 * composites, alignement) reste à la charge de l'appelant. Pour remplir une
 * cellule de grille en hauteur, passer `className="flex h-full flex-col"` et
 * rendre le contenu interne `flex-1`.
 */
import type { ReactNode } from 'react'

import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'

interface KpiCardProps {
  /**
   * Token de la barre d'accent 3px. Fixe (par métrique, type 2) ou calculé selon
   * le contenu (type 4). Omis ⇒ pas d'accent.
   */
  accent?: SemanticToken
  /** Côté de la barre d'accent : `top` (défaut) ou `left` (bordure gauche 3px). */
  accentSide?: 'top' | 'left'
  /** Classes sur la carte racine (ex. `flex h-full flex-col` pour une cellule grid). */
  className?: string
  children: ReactNode
  testId?: string
}

export function KpiCard({ accent, accentSide = 'top', className = '', children, testId }: KpiCardProps) {
  const leftAccent = accent != null && accentSide === 'left'
  return (
    <div
      className={`overflow-hidden rounded-lg border border-border bg-card ${className}`}
      style={leftAccent ? { borderLeftWidth: '3px', borderLeftColor: tokenCssVar(accent) } : undefined}
      data-testid={testId}
    >
      {accent && accentSide === 'top' && (
        <div className="h-[3px] flex-none" style={{ backgroundColor: tokenCssVar(accent) }} />
      )}
      {children}
    </div>
  )
}
