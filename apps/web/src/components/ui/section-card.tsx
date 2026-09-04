/**
 * SectionCard — LE GABARIT D'UNE CARTE DE SECTION, et il n'y en a qu'un.
 *
 * POURQUOI CETTE PRIMITIVE EXISTE (règle CLAUDE.md n°6, ≤ 2 copies). Le chrome d'une carte
 * de section — cadre arrondi, fond de carte, bandeau de titre souligné — était écrit en
 * toutes lettres dans CINQ fichiers, et il avait DIVERGÉ : `ChartCard` et le `PaneCard` des
 * médailles/citations posaient `rounded-lg border border-border bg-card` avec un bandeau
 * `border-b border-border px-3 py-2 text-sm font-medium`, tandis que quatre sections de la
 * page match (distance des frags, objectifs, bilan d'équipement, contrôle des socles)
 * posaient un `border-2` sans fond et un titre `text-sm font-bold uppercase tracking-wider`.
 * Deux gabarits pour la même chose, ce sont deux cartes différentes dans la même colonne.
 * L'utilisateur a désigné le bloc « Citations » comme la référence (2026-09-03) : c'est ce
 * chrome-là qui est repris ici, à l'identique.
 *
 * UNE FACTORISATION SANS GARDE-RAIL RE-DIVERGE (même règle) : `section-card.guard.test.ts`
 * interdit le retour du littéral `border-2 border-border` sur une balise `<section>` dans
 * les deux features concernées.
 *
 * CE QUE CETTE PRIMITIVE NE FAIT PAS, et c'est délibéré : elle n'enveloppe PAS le corps.
 * Les sections migrées posent elles-mêmes leur `overflow-x-auto`, leurs paragraphes ou leur
 * corps à hauteur plancher — un padding imposé ici aurait décalé quatre rendus d'un coup.
 * Le chrome est commun, la mise en page du contenu reste à l'appelant.
 */
import type { ReactNode } from 'react'

export interface SectionCardProps {
  /** Libellé du bandeau de titre. */
  title: string
  /**
   * Nom accessible de la section (`aria-label`). Absent = section SANS nom accessible :
   * l'ARIA ne l'expose alors pas comme région, exactement comme le `<div>` qu'elle
   * remplace. Les sections qui se cherchent par leur nom dans les tests le passent.
   */
  label?: string
  /**
   * Habillage optionnel du LIBELLÉ de titre : reçoit le libellé, rend le nœud posé dans le
   * bandeau. Sert au titre porteur d'une infobulle d'en-tête (contrôle des socles). Absent
   * = le libellé nu.
   */
  titleAdornment?: (label: string) => ReactNode
  /** Corps de la carte, posé tel quel sous le bandeau. */
  children: ReactNode
  /**
   * Pied de carte : les notes de mesure (dénominateurs, réserves de couverture). Posé tel
   * quel après le corps, sans chrome ajouté — les notes existantes portent déjà le leur.
   */
  footer?: ReactNode
  /** Classes supplémentaires sur la section (rare : la carte est volontairement uniforme). */
  className?: string
}

/** SectionCard pose le chrome commun ; tout le reste vient de l'appelant. */
export function SectionCard({
  title,
  label,
  titleAdornment,
  children,
  footer,
  className = '',
}: SectionCardProps) {
  // `flex flex-col` : reprise EXACTE du gabarit de référence (PaneCard) — un corps en
  // `flex-1` peut ainsi remplir la cellule de grille que sa voisine étire.
  return (
    <section
      className={`relative flex flex-col rounded-lg border border-border bg-card${className ? ` ${className}` : ''}`}
      aria-label={label}
    >
      <h3 className="flex-none border-b border-border px-3 py-2 text-sm font-medium">
        {titleAdornment ? titleAdornment(title) : title}
      </h3>
      {children}
      {footer}
    </section>
  )
}
