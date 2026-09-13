/**
 * GrilleForm.tsx — LA GRILLE ALIGNÉE (artefact 2ec1b8eb, forme `grille`).
 *
 * CE COMPOSANT NE RÉINVENTE RIEN : la grille « lignes alignées, une échelle et
 * un axe PAR COLONNE » existe déjà dans le dépôt (`components/charts/ValueGrid`,
 * employée par la vue match pour l'équipement et les objectifs). Une seconde
 * grille aurait divergé sur l'arrondi des bornes ou la position des graduations,
 * et deux écrans voisins auraient montré deux échelles différentes du même
 * compte.
 *
 * Ce fichier n'ajoute que l'adaptation : les lignes et colonnes de l'artefact
 * (sous-libellé, total en en-tête, hachure du non mesuré, intitulé d'axe) vers
 * les entrées de `buildValueGrid`.
 */
import { ValueGrid } from '@/components/charts/ValueGrid'
import { buildValueGrid } from '@/components/charts/valueGridModel'

/** Une ligne : son identité, son encre, et si elle est mesurée. */
export interface GrilleRow {
  key: string
  label: string
  sublabel?: string
  accent?: string
  /** Faux = ligne NON MESURÉE : rail hachuré, valeur en tiret. */
  measured?: boolean
}

/** Une colonne : sa grandeur, son total d'en-tête, son unité. */
export interface GrilleColumn {
  key: string
  label: string
  /** Le total écrit dans l'en-tête (déjà formaté). Absent = pas de total. */
  total?: string
  /** La colonne porte une durée (m:ss) plutôt qu'un compte. */
  duration?: boolean
  /** La colonne porte un POURCENTAGE : le milieu de son axe ne s'arrondit pas. */
  percent?: boolean
}

export interface GrilleFormProps {
  rows: GrilleRow[]
  columns: GrilleColumn[]
  /** La valeur d'une cellule. `null` = non mesuré. */
  value: (row: GrilleRow, column: GrilleColumn) => number | null
  /** L'encre d'une cellule. */
  ink: (row: GrilleRow, column: GrilleColumn) => string
  /** Le format d'une valeur de colonne. */
  format: (value: number, column: GrilleColumn) => string
  /** L'infobulle d'une cellule, qui reçoit la valeur déjà formatée. */
  tooltip: (row: GrilleRow, column: GrilleColumn, text: string) => string
  notMeasuredLabel: string
  axisTitle: string
  /** Largeur de la colonne des noms (px) — à élargir pour un sous-libellé long. */
  nameWidth?: number
}

export function GrilleForm({
  rows,
  columns,
  value,
  ink,
  format,
  tooltip,
  notMeasuredLabel,
  axisTitle,
  nameWidth,
}: GrilleFormProps) {
  const model = buildValueGrid({
    rows: rows.map((r) => ({
      key: r.key,
      label: r.label,
      sublabel: r.sublabel,
      accent: r.accent,
      // Un seul groupe : les filets de séparation de la primitive ne servent pas
      // ici (les lignes de ces grilles sont homogènes).
      group: 'formes',
      hint: r.sublabel != null ? `${r.label} — ${r.sublabel}` : r.label,
    })),
    columns: columns.map((c) => ({
      key: c.key,
      label: c.label,
      duration: c.duration,
      fractionalAxis: c.percent,
      // Le total vient de l'appelant (il connaît l'unité) : la primitive ne sait
      // pas qu'un « taux de rafle » ne s'additionne pas.
      showTotal: false,
    })),
    value: (r, c) => (rows[r].measured === false ? null : value(rows[r], columns[c])),
    format: (v, c) => format(v, columns[c]),
    color: (r, c) => ink(rows[r], columns[c]),
    tooltip: (r, c, text) =>
      rows[r].measured === false
        ? `${rows[r].label} — ${notMeasuredLabel}`
        : tooltip(rows[r], columns[c], text),
    notMeasured: '—',
  })
  // Le total d'en-tête est posé APRÈS la projection : `buildValueGrid` le
  // calculerait comme une somme de colonne, ce qui serait faux pour un taux.
  const withTotals = {
    ...model,
    columns: model.columns.map((c, i) => ({ ...c, totalText: columns[i].total ?? null })),
  }
  return (
    <ValueGrid model={withTotals} axisTitle={axisTitle} hatchNotMeasured nameWidth={nameWidth} />
  )
}
