/**
 * detail-section — LE TITRE DE SECTION SANS CARTE, et il n'y en a qu'un.
 *
 * POURQUOI CETTE PRIMITIVE EXISTE (règle CLAUDE.md n°6, ≤ 2 copies). Le gabarit d'un titre
 * de section hors carte — un `h3 text-base font-semibold text-foreground` posé au-dessus de
 * son contenu — vivait dans un helper LOCAL à la page match
 * (`features/match-view/DetailSection.tsx`, deux consommateurs) et était RECOPIÉ en toutes
 * lettres ailleurs : deux sections de la page Synergies (impact des coéquipiers, médailles),
 * la section Performance des Contributions, les deux titres des formes retenues, et le
 * tableau « Détail des matchs » du détail de session — celui-là en `h2`, ce qui cassait en
 * plus la hiérarchie des titres de la page. Six copies pour un seul gabarit : à la
 * troisième, la règle impose un helper partagé ET un garde-rail.
 *
 * UNE FACTORISATION SANS GARDE-RAIL RE-DIVERGE (même règle) : `detail-section.guard.test.ts`
 * interdit, dans `features/**`, tout `<h2>`/`<h3>` qui réécrit `text-base font-semibold` à la
 * main, hors allowlist datée des surfaces non migrées.
 *
 * DEUX EXPORTS PARCE QU'IL Y A DEUX BESOINS, et c'est délibéré :
 * - `DetailSection` monte la section complète (`space-y-4` + titre + corps) — le cas de la
 *   page match ;
 * - `SectionTitle` ne monte QUE le titre, pour les sections qui posent déjà leur propre
 *   conteneur et son espacement (`space-y-3`), ou qui intercalent un bandeau entre le titre
 *   et le corps. Leur imposer le conteneur de `DetailSection` aurait changé l'espacement de
 *   six rendus d'un coup : le gabarit du TITRE est commun, la mise en page reste à
 *   l'appelant.
 *
 * Aucune couleur en dur (règle n°12) : `text-foreground` est un jeton sémantique.
 */
import type { ReactNode } from 'react'

/** Concaténation de classes — le projet n'embarque ni `clsx` ni `tailwind-merge`. */
function joinClasses(base: string, extra?: string): string {
  return extra ? `${base} ${extra}` : base
}

export interface SectionTitleProps {
  /** Libellé du titre. `ReactNode` : certains titres portent une infobulle ou un badge. */
  children: ReactNode
  /**
   * Classes supplémentaires sur le `h3`. Réservé aux titres qui ajoutent du chrome AUTOUR
   * du gabarit (filet bas, marge haute, mise en ligne d'une infobulle) — jamais pour
   * redéfinir la taille ou la graisse, qui font le gabarit.
   */
  className?: string
  /** Ancre du titre, quand une navigation interne le vise. */
  id?: string
}

/** Le titre de section nu : le gabarit, et rien d'autre. */
export function SectionTitle({ children, className, id }: SectionTitleProps) {
  return (
    <h3 id={id} className={joinClasses('text-base font-semibold text-foreground', className)}>
      {children}
    </h3>
  )
}

export interface DetailSectionProps {
  /** Libellé du titre (voir `SectionTitleProps.children`). */
  title: ReactNode
  /** Corps de la section, posé tel quel sous le titre. */
  children: ReactNode
  /** Classes supplémentaires sur la `<section>` (l'espacement par défaut est `space-y-4`). */
  className?: string
  /** Classes supplémentaires sur le titre — passées telles quelles à `SectionTitle`. */
  titleClassName?: string
}

/** Section titrée sans carte : titre type-1 + contenu groupé. */
export function DetailSection({ title, children, className, titleClassName }: DetailSectionProps) {
  return (
    <section className={joinClasses('space-y-4', className)}>
      <SectionTitle className={titleClassName}>{title}</SectionTitle>
      {children}
    </section>
  )
}
