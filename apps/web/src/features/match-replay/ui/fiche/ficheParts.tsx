/**
 * ficheParts — LES PIÈCES COMMUNES DES SECTIONS DE LA FICHE : la section et son titre, la capsule « estimation »,
 * la liste libellé / valeur et sa petite note. Des composants de rendu, sans lecture : ce que la section écrit lui
 * est donné. Les cotes sont `COTES_FICHE_INSTANT`.
 *
 * L'ENCRE : libellés et notes en encre discrète, valeurs en encre du thème, filets à la bordure. Aucun littéral de
 * couleur.
 */
import type { CSSProperties, ReactNode } from 'react'

import { COTES_FICHE_INSTANT as C } from '../../model/cotesFicheInstant'

/** Les deux formes de titre : le titre d'une section (`sub`), et celui d'un groupe séparé par un filet (`groupe`, en capitales). */
type TitleKind = 'sub' | 'groupe'

export interface FicheSectionProps {
  title: string
  /** Un mot à côté du titre : la capsule « estimation », ou l'instant lu d'un groupe. */
  aside?: ReactNode
  kind?: TitleKind
  children: ReactNode
}

const TITLE_STYLE: Record<TitleKind, CSSProperties> = {
  sub: { fontSize: C.section.titrePx },
  groupe: { fontSize: C.section.titreGroupePx },
}

const TITLE_CLASS: Record<TitleKind, string> = {
  sub: 'font-semibold text-foreground',
  groupe: 'font-semibold uppercase tracking-[.2px] text-muted-foreground',
}

/**
 * Une section de la fiche : son titre (et ce qui le suit), puis ses lignes. Un groupe est séparé de ce qui
 * précède par un filet.
 */
export function FicheSection({ title, aside, kind = 'sub', children }: FicheSectionProps) {
  const section = (
    <section aria-label={title} className="flex flex-col" style={{ gap: C.section.ecartPx }}>
      <div className="flex items-baseline" style={{ gap: C.section.titreEcartPx }}>
        <span className={TITLE_CLASS[kind]} style={TITLE_STYLE[kind]}>
          {title}
        </span>
        {aside}
      </div>
      {children}
    </section>
  )
  if (kind === 'sub') return section
  return (
    <div className="border-t border-border" style={{ paddingTop: C.section.groupeMargeHautePx }}>
      {section}
    </div>
  )
}

/** La capsule « estimation » : un mot encadré, à côté d'un titre ou d'une ligne. */
export function EstimateCapsule({ text, hint }: { text: string; hint: string }) {
  return (
    <span
      title={hint}
      className="inline-grid place-items-center self-center border border-border font-normal normal-case text-muted-foreground"
      style={{
        height: C.capsule.hauteurPx,
        paddingInline: C.capsule.margeIntPx,
        borderRadius: C.capsule.rayonPx,
        fontSize: C.capsule.taillePx,
      }}
    >
      {text}
    </span>
  )
}

/** Le texte discret qui suit une valeur (« · 2e sur 8 », « · 61 / 142 »). */
export function Note({ children }: { children: ReactNode }) {
  return <small className="font-normal text-muted-foreground"> · {children}</small>
}

/** La liste libellé à gauche, valeur à droite. Ses lignes sont des `FicheRow`. */
export function FicheRows({ children }: { children: ReactNode }) {
  return (
    <dl
      className="grid"
      style={{
        gridTemplateColumns: 'minmax(0, 1fr) auto',
        rowGap: C.ligne.ecartRangeesPx,
        columnGap: C.ligne.ecartColonnesPx,
        fontSize: C.ligne.taillePx,
      }}
    >
      {children}
    </dl>
  )
}

export function FicheRow({ label, aside, children, title }: {
  label: string
  /** Ce qui suit le libellé : la capsule « estimation » d'une ligne. */
  aside?: ReactNode
  children: ReactNode
  title?: string
}) {
  return (
    <>
      <dt className="flex items-center text-muted-foreground" style={{ gap: C.ligne.ecartLibellePx }}>
        {label}
        {aside}
      </dt>
      <dd className="text-right text-foreground tabular-nums" title={title}>
        {children}
      </dd>
    </>
  )
}
