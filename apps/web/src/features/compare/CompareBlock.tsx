/**
 * CompareBlock — LE GABARIT D'UN BLOC DE LA PAGE FACE-À-FACE, et il n'y en a qu'un.
 *
 * # POURQUOI CE COMPOSANT EXISTE (2026-09-17, retour du gate visuel)
 *
 * Le markup « carte à en-tête » (`rounded-lg border bg-card`, un bandeau de titre séparé par
 * un filet, un corps espacé) vivait INLINE dans `CategoryColumn` et `CategoryMirrorSection`.
 * Le passage du profil d'armes en cartes en aurait fait QUATRE copies — la règle n°6 du dépôt
 * impose de centraliser à la troisième. Une carte recopiée dérive : c'est déjà arrivé sur le
 * `<h2>`, dont les deux exemplaires étaient restés grisés et en capitales quand la section
 * neuve, elle, était en couleur de premier plan.
 *
 * # CE QU'IL POSE, ET CE QU'IL NE DÉCIDE PAS
 *
 * Il pose le chrome et le titre ; tout le reste vient de l'appelant. Il ne choisit pas
 * l'espacement interne du corps (`p-3` seul, sans `space-y`) : une colonne de métriques et une
 * grille de colonnes d'armes n'ont pas la même respiration, et l'imposer ici obligerait chaque
 * appelant à la défaire.
 */
import type { ReactNode } from 'react'

export interface CompareBlockProps {
  /** Libellé du bandeau, déjà localisé — ce composant n'i18n rien. */
  title: string
  children: ReactNode
}

export function CompareBlock({ title, children }: CompareBlockProps) {
  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="border-b border-border px-3 py-2">
        <h2 className="text-sm font-semibold">{title}</h2>
      </div>
      <div className="p-3">{children}</div>
    </div>
  )
}
