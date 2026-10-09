/**
 * La hachure d'une valeur NON MESURÉE : un motif neutre du thème, jamais un jeton de
 * donnée — une absence n'est pas une valeur. Partagée par la grille de valeurs
 * (`ValueGrid`, option `hatchNotMeasured`) et la barre d'assistances de la tuile de match
 * (part des frags sans information d'assistance).
 */
import type { CSSProperties } from 'react'

export const NOT_MEASURED_HATCH: CSSProperties = {
  backgroundImage:
    'repeating-linear-gradient(45deg, transparent 0px, transparent 3px, var(--muted-foreground) 3px, var(--muted-foreground) 4px)',
  opacity: 0.3,
}
