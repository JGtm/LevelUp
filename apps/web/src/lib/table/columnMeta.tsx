/**
 * columnMeta — augmentation centralisée du type `ColumnMeta` de TanStack Table
 * (V72-04) + helper de rendu d'en-tête pour les tooltips explicatifs de colonnes.
 *
 * Les 8 tableaux TanStack de l'app rendent chacun leur propre `<thead>` avec
 * `flexRender`. Pour éviter d'inventer un champ ad hoc par tableau, on augmente
 * `ColumnMeta` avec `headerTooltip?: string` : n'importe quelle `ColumnDef` peut
 * porter `meta: { headerTooltip: t('...') }`, et chaque `<thead>` rend l'icône
 * ⓘ via `<HeaderInfoTooltip text={header.column.columnDef.meta?.headerTooltip} />`.
 *
 * NB : l'icône est rendue en FRÈRE du contrôle de tri (jamais À L'INTÉRIEUR d'un
 * `<button>` de tri) — `InfoTooltip` rend lui-même un `<button>`, et un bouton
 * imbriqué dans un bouton est du HTML invalide (le navigateur referme le bouton
 * externe). Le `stopPropagation` du wrapper évite qu'un clic sur l'icône ne
 * déclenche le tri dans les tableaux dont le `<th>` porte l'`onClick` de tri.
 */
import { type ReactNode } from 'react'
import { type RowData } from '@tanstack/react-table'
import { InfoTooltip } from '@/components/ui/info-tooltip'

// Augmentation de type globale : ajoute `headerTooltip` à toute `ColumnMeta`.
// On augmente `@tanstack/react-table` (pattern officiel TanStack v8) ; l'interface
// est effectivement déclarée dans `@tanstack/table-core` et re-exportée.
declare module '@tanstack/react-table' {
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  interface ColumnMeta<TData extends RowData, TValue> {
    /** Texte d'aide (1 phrase, FR/EN via manifest i18n) affiché au survol d'une
     *  icône ⓘ à côté du libellé de l'en-tête. Absent = pas d'icône. */
    headerTooltip?: string
  }
}

/**
 * HeaderInfoTooltip — icône ⓘ d'aide pour un en-tête de colonne. Rend `null`
 * quand `text` est absent, ce qui permet de l'appeler inconditionnellement dans
 * un `<thead>`. Icône réduite (w-3 h-3) adaptée à la densité des en-têtes.
 *
 * Le clic est stoppé au niveau du wrapper : dans les tableaux où le `<th>` (ou un
 * `<button>` frère) porte l'`onClick` de tri, cliquer l'icône ne doit PAS trier.
 */
export function HeaderInfoTooltip({ text }: { text?: string }): ReactNode {
  if (!text) return null
  return (
    <span className="inline-flex items-center" onClick={(e) => e.stopPropagation()}>
      <InfoTooltip content={text} iconClass="w-3 h-3" />
    </span>
  )
}
