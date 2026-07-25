/**
 * SortableTh — en-tête de colonne triable PARTAGÉ pour les tableaux HTML natifs
 * "hand-rolled" (pas TanStack Table) : petites tables admin + tableaux Carrière
 * (I16 — tri par en-têtes généralisé à tout l'app).
 *
 * Foyer canonique (règle ≤2 copies, CLAUDE.md n°6) : le state de tri (sortKey/
 * sortDir + fonction toggleSort) reste LOCAL à chaque consommateur (la logique de
 * comparaison diffère par colonne) — seul le RENDU de l'en-tête (bouton + flèche +
 * aria-sort) est centralisé ici. Garde-rail : sortable-th.guard.test.ts.
 *
 * Pattern DISTINCT de MatchScoreboard/MatchEncountersTable (features/match-view/
 * sortHeader.ts) : ces 2 tableaux TanStack cliquent directement le <th> (pas de
 * <button> imbriqué) — choix délibéré propre à leur contexte, pas un doublon à
 * fusionner ici.
 */
import type { ReactNode } from 'react'
import { InfoTooltip } from '@/components/ui/info-tooltip'

export interface SortableThProps {
  /** Libellé de la colonne (texte ou nœud). */
  label: ReactNode
  /** True si cette colonne porte le tri actif (sinon `dir` est ignoré). */
  active: boolean
  /** Sens du tri actif. */
  dir: 'asc' | 'desc'
  /** Active/bascule le tri sur cette colonne — logique côté appelant. */
  onClick: () => void
  /** Classes du <th> (alignement/padding) — reprend le style du tableau appelant. */
  className?: string
  /** Aide facultative (V72-04) : texte/nœud affiché au survol d'une icône ⓘ
   *  discrète APRÈS le libellé. L'icône est rendue en FRÈRE du bouton de tri
   *  (jamais à l'intérieur : `InfoTooltip` rend un `<button>`, imbriquer deux
   *  boutons est du HTML invalide). Absent = pas d'icône (rendu inchangé). */
  tooltip?: ReactNode
}

/** En-tête de colonne triable : bouton pleine cellule + flèche ▲/▼ (active only)
 *  + aria-sort porté par le <th>. Une icône ⓘ facultative (prop `tooltip`) est
 *  ajoutée à côté, sœur du bouton de tri (respecte l'alignement du <th>). */
export function SortableTh({ label, active, dir, onClick, className, tooltip }: SortableThProps) {
  const button = (
    <button
      type="button"
      onClick={onClick}
      className={`inline-flex items-center gap-0.5 whitespace-nowrap transition-colors hover:text-foreground${active ? ' text-foreground' : ''}`}
    >
      {label}
      {active && (
        <span aria-hidden="true" className="text-2xs leading-none">
          {dir === 'asc' ? '▲' : '▼'}
        </span>
      )}
    </button>
  )
  return (
    <th className={className} aria-sort={active ? (dir === 'asc' ? 'ascending' : 'descending') : 'none'}>
      {tooltip != null ? (
        <span className="inline-flex items-center gap-1">
          {button}
          <InfoTooltip content={tooltip} iconClass="w-3 h-3" />
        </span>
      ) : (
        button
      )}
    </th>
  )
}
